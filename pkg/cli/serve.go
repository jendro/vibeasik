package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/api"
	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/knowledge"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
	"github.com/vibeasik/vibeasik/pkg/localauth"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/setup"
	"github.com/vibeasik/vibeasik/pkg/standard"
	"github.com/vibeasik/vibeasik/pkg/telemetry"
	"github.com/vibeasik/vibeasik/pkg/traffic"
	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
	"github.com/vibeasik/vibeasik/pkg/update"
)

var restarting atomic.Bool

var (
	host        string
	port        int
	openBrowser bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Jalankan engine: REST API, Studio, dan server MCP HTTP",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetConfig()
		listenHost, listenPort := cfg.Host, cfg.Port
		if cmd.Flags().Changed("host") {
			listenHost = host
		}
		if cmd.Flags().Changed("port") {
			listenPort = port
		}

		db, err := database.InitDB(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		if extra := registry.Models(); len(extra) > 0 {
			if err := db.AutoMigrate(extra...); err != nil {
				return fmt.Errorf("migrasi model modul gagal: %w", err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		registry.Edition = Edition
		// Akun vibeasik (diaktifkan di PersistentPreRun): entitlement diperbarui berkala selama serve.
		account.StartRefresher(ctx)
		// Pusat Pengetahuan: tarik paket baru tiap hari (gagal jaringan diabaikan; pengetahuan lokal tetap jalan).
		knowledgesync.StartAutoPull(ctx, db)
		server := api.NewServer(db)
		lan := listenHost != "127.0.0.1" && listenHost != "localhost" && listenHost != "::1"
		server.SetLAN(lan)
		if restartSelf != nil {
			// Hentikan dev server dan engine dengan rapi sebelum proses diganti biner/konfigurasi baru.
			server.SetRestart(func() error {
				if err := restartCheck(); err != nil {
					return err // jangan hentikan engine bila proses baru tak bisa dijalankan
				}
				restarting.Store(true)
				cancel()
				return server.Shutdown() // Start() kembali; exec dilakukan di bawah, di goroutine utama
			})
		}
		// Pembaruan: cek saat start dan tiap 6 jam; pasang otomatis di latar bila diizinkan (build Official).
		upd := update.New(Version, Edition)
		server.SetUpdater(upd)
		upd.Background(ctx, update.Auto, 6*time.Hour)
		// Ping pemakaian ringan (OS, versi, zona waktu, status akun; tanpa isi proyek). Matikan: VIBEASIK_NO_TELEMETRY=1.
		telemetry.Start(ctx, account.CloudURL(), func() telemetry.Info {
			var n int64
			db.Model(&models.Project{}).Count(&n)
			return telemetry.Info{
				Version: Version, Edition: Edition, DBDriver: cfg.DBDriver,
				LoggedIn: account.Current().LoggedIn, AccountChoice: setup.Load().AccountChoice, Projects: int(n),
			}
		})
		token, err := localauth.Token()
		if err != nil {
			return fmt.Errorf("token akses lokal: %w", err)
		}
		// Pantau sesi agent coding di laptop ini (Claude Code, Antigravity, Copilot).
		activity.StartLocalWatchers(ctx)
		// Standar vibeasik: blok port per proyek, env lokal/server, todo standar.
		standard.Init(db)
		codemap.StartAutoSync(ctx, db, codemap.AutoSyncInterval)
		// Traffic Analyzer: tarik statistik aplikasi live sesuai interval tiap sumber.
		traffic.StartSyncer(ctx, db)
		// Basis data lokasi & jaringan IP (DB-IP Lite): unduh bila belum ada atau sudah >30 hari; matikan: VIBEASIK_GEO_AUTO=0.
		geo.StartAutoUpdate(ctx, geo.Default())
		// Pusat Pengetahuan: sinkron patokan & pengetahuan, tautan, klaster, pangkas berkala.
		knowledge.Start(ctx, db)

		// Worker modul (Pro & tambahan) yang aktif untuk akun ini.
		for _, m := range registry.All() {
			if st, ok := m.(registry.Starter); ok && registry.Enabled(m) {
				if err := st.Start(ctx, db); err != nil {
					log.Printf("[registry] modul %s gagal start: %v", m.Name(), err)
				}
			}
		}

		addr := fmt.Sprintf("%s:%d", listenHost, listenPort)
		log.Println("==================================================================")
		log.Printf("🚀 vibeasik %s (%s)\n", Version, Edition)
		studioURL := studioLoginURL(listenPort, token)
		log.Printf("   • Studio:  %s\n", studioURL)
		if lan {
			log.Println("   ⚠ Engine dibuka ke LAN: siapa pun di jaringan ini yang punya token bisa memakainya.")
			for _, ip := range config.GetLocalIPs() {
				log.Printf("   • LAN:     http://%s:%d/auth?t=<token>\n", ip, listenPort)
			}
		}
		log.Printf("   • MCP:     http://localhost:%d/mcp (header %s; token: `vibeasik token`)\n", listenPort, localauth.Header)
		log.Printf("   • DB:      %s\n", cfg.DBDriver)
		log.Println("==================================================================")

		if openBrowser {
			go func() {
				time.Sleep(350 * time.Millisecond)
				_ = openURL(studioURL)
			}()
		}

		// Hentikan dev server yang dijalankan vibeasik agar tidak jadi proses yatim yang memegang port.
		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
			log.Println("🛑 Menghentikan dev server yang dijalankan vibeasik...")
			cancel()
			_ = server.Shutdown()
		}()

		err = server.Start(addr)
		if restarting.Load() {
			time.Sleep(300 * time.Millisecond) // beri waktu dev server berhenti dan port dilepas
			return restartSelf()
		}
		return err
	},
}

func init() {
	cfg := config.GetConfig()
	serveCmd.Flags().StringVarP(&host, "host", "H", cfg.Host, "Alamat bind (0.0.0.0 untuk akses LAN)")
	serveCmd.Flags().IntVarP(&port, "port", "p", cfg.Port, "Port engine")
	serveCmd.Flags().BoolVarP(&openBrowser, "open", "o", false, "Buka Studio di browser setelah start")
	rootCmd.AddCommand(serveCmd)
}
