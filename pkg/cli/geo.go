package cli

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/traffic/geo"
)

var geoOnly string

var geoCmd = &cobra.Command{
	Use:   "geo",
	Short: "Basis data lokasi & jaringan IP offline (DB-IP Lite) untuk Traffic Analyzer",
	Long: `Kota, ISP, dan jenis jaringan pengunjung dibaca dari basis data offline di mesin ini;
IP pengunjung tidak dikirim ke pihak lain. Berkas disimpan di folder geo/ pada direktori data vibeasik.

IP Geolocation by DB-IP (https://db-ip.com), lisensi CC BY 4.0.`,
}

var geoUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Unduh basis data ASN (±9 MB) dan City (±121 MB) bulan ini",
	RunE: func(cmd *cobra.Command, args []string) error {
		if geoOnly != "all" && geoOnly != "asn" && geoOnly != "city" {
			return fmt.Errorf("--only harus all, asn, atau city")
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		u := &geo.Updater{Dir: geo.Dir(), Log: func(f string, a ...any) { fmt.Printf("  "+f+"\n", a...) }}
		start := time.Now()
		err := u.Update(ctx, geoOnly, geo.Default().Release)
		geo.Default().Reload()
		printGeoStatus()
		if err != nil {
			return fmt.Errorf("sebagian unduhan gagal: %w", err)
		}
		fmt.Printf("selesai dalam %s. Engine yang sedang berjalan memuat ulang berkas baru otomatis.\n", time.Since(start).Round(time.Second))
		return nil
	},
}

var geoStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Tampilkan berkas basis data geo yang terpasang",
	Run:   func(cmd *cobra.Command, args []string) { printGeoStatus() },
}

var geoLookupCmd = &cobra.Command{
	Use:   "lookup <ip>...",
	Short: "Cari kota, ISP, dan jenis jaringan satu atau beberapa IP",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db := geo.Default()
		for _, ip := range args {
			in := db.Lookup(ip)
			if in.Empty() {
				fmt.Printf("%-18s (tidak ditemukan atau bukan IP publik)\n", ip)
				continue
			}
			fmt.Printf("%-18s %s %s | AS%d %s | %s\n", ip, in.Country, in.City, in.ASN, in.Org, in.NetType)
		}
	},
}

func printGeoStatus() {
	dir := geo.Dir()
	fmt.Println("folder:", dir)
	for _, s := range geo.Sources {
		fi, err := os.Stat(filepath.Join(dir, s.File))
		if err != nil {
			fmt.Printf("  %-22s belum terpasang (jalankan: vibeasik geo update)\n", s.File)
			continue
		}
		fmt.Printf("  %-22s %6.1f MB, diperbarui %s\n", s.File, float64(fi.Size())/(1<<20), fi.ModTime().Format("2006-01-02"))
	}
	st := geo.Default().Status()
	if !st.Ready() {
		fmt.Println("status: belum ada basis data yang termuat; kota/ISP tidak akan terisi")
		return
	}
	fmt.Printf("status: city=%v asn=%v (`serve` memperbarui otomatis tiap bulan; matikan dengan %s=0)\n", st.CityReady, st.ASNReady, geo.AutoEnv)
}

func init() {
	geoUpdateCmd.Flags().StringVar(&geoOnly, "only", "all", "Basis data yang diunduh: all, asn, atau city")
	geoCmd.AddCommand(geoUpdateCmd, geoStatusCmd, geoLookupCmd)
	rootCmd.AddCommand(geoCmd)
}
