// Package cli berisi perintah baris vibeasik. Build Official (vibeasik-pro) memakai paket yang sama:
// main-nya mengimpor modul Pro (yang mendaftar ke registry) lalu memanggil cli.Execute.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibeasik/vibeasik/pkg/account"
)

// Version diisi saat rilis: -ldflags "-X github.com/vibeasik/vibeasik/pkg/cli.Version=vX.Y.Z".
var Version = "dev"

// Edition "Community" atau "Official"; build Official menggantinya sebelum Execute.
var Edition = "Community"

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "vibeasik",
	Short: "vibeasik — lapisan standar untuk kode buatan AI",
	Long: `vibeasik berjalan di laptop atau dev server Anda: mencatat proyek, mengaudit kode buatan AI,
mengatur port & dev server, dan menjadi server MCP untuk agent coding (Claude Code, Cursor, Antigravity, …).`,
	SilenceUsage: true,
	// Setiap perintah (serve, mcp stdio, …) memakai entitlement akun yang tersimpan.
	PersistentPreRun: func(cmd *cobra.Command, args []string) { account.Activate() },
	// Tanpa perintah: tampilkan bantuan. Bila dibuka dengan klik ganda di Windows, jalankan engine dan
	// buka Studio, agar jendela konsol tidak langsung tertutup.
	RunE: func(cmd *cobra.Command, args []string) error {
		if !launchedByDoubleClick() {
			return cmd.Help()
		}
		openBrowser = true
		err := serveCmd.RunE(serveCmd, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nvibeasik berhenti:", err)
			fmt.Fprint(os.Stderr, "Tekan Enter untuk menutup jendela ini...")
			_, _ = fmt.Scanln()
		}
		return err
	},
}

// Execute menjalankan CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// AddCommand memungkinkan build Official menambah perintah Pro.
func AddCommand(cmds ...*cobra.Command) { rootCmd.AddCommand(cmds...) }

// DBPath nilai flag --db (kosong = lokasi bawaan), untuk perintah tambahan dari build Official.
func DBPath() string { return dbPath }

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path berkas database SQLite (bawaan ~/.vibeasik/projects.db)")
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Tampilkan versi",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("vibeasik %s (%s)\n", Version, Edition)
		},
	})
}
