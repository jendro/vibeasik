package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/activity"
	"github.com/vibeasik/vibeasik/pkg/api"
	"github.com/vibeasik/vibeasik/pkg/codemap"
	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/knowledge"
	"github.com/vibeasik/vibeasik/pkg/registry"
	"github.com/vibeasik/vibeasik/pkg/standard"
	"github.com/vibeasik/vibeasik/pkg/traffic"
)

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

		server := api.NewServer(db)
		// Pantau sesi agent coding di laptop ini (Claude Code, Antigravity, Copilot).
		activity.StartLocalWatchers(ctx)
		// Standar vibeasik: blok port per proyek, env lokal/server, todo standar.
		standard.Init(db)
		codemap.StartAutoSync(ctx, db, codemap.AutoSyncInterval)
		// Traffic Analyzer: tarik statistik aplikasi live sesuai interval tiap sumber.
		traffic.StartSyncer(ctx, db)
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
		log.Printf("   • Studio:  http://localhost:%d\n", listenPort)
		if listenHost != "127.0.0.1" && listenHost != "localhost" {
			for _, ip := range config.GetLocalIPs() {
				log.Printf("   • LAN:     http://%s:%d\n", ip, listenPort)
			}
		}
		log.Printf("   • MCP:     http://localhost:%d/mcp\n", listenPort)
		log.Printf("   • DB:      %s\n", cfg.DBDriver)
		log.Println("==================================================================")

		if openBrowser {
			go func() {
				time.Sleep(350 * time.Millisecond)
				_ = openURL(fmt.Sprintf("http://localhost:%d", listenPort))
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

		return server.Start(addr)
	},
}

func init() {
	cfg := config.GetConfig()
	serveCmd.Flags().StringVarP(&host, "host", "H", cfg.Host, "Alamat bind (0.0.0.0 untuk akses LAN)")
	serveCmd.Flags().IntVarP(&port, "port", "p", cfg.Port, "Port engine")
	serveCmd.Flags().BoolVarP(&openBrowser, "open", "o", false, "Buka Studio di browser setelah start")
	rootCmd.AddCommand(serveCmd)
}
