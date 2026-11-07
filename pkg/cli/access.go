package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/localauth"
)

// studioLoginURL tautan masuk Studio: engine menukar token dengan cookie lalu membuka cockpit.
func studioLoginURL(port int, token string) string {
	return fmt.Sprintf("http://localhost:%d/auth?t=%s", port, url.QueryEscape(token))
}

var openPort int

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Buka Studio di browser (masuk otomatis dengan token lokal)",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := localauth.Token()
		if err != nil {
			return err
		}
		port := config.GetConfig().Port
		if cmd.Flags().Changed("port") {
			port = openPort
		}
		u := studioLoginURL(port, token)
		if err := openURL(u); err != nil {
			fmt.Println("Buka tautan ini di browser:", u)
			return nil
		}
		fmt.Println("Studio dibuka di browser. Pastikan `vibeasik serve` berjalan.")
		return nil
	},
}

var rotateToken bool

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Tampilkan token akses lokal (untuk konfigurasi MCP HTTP)",
	Long: `Token akses lokal wajib dikirim setiap pemanggil API/MCP engine (header X-Vibeasik-Token).
Disimpan di ~/.vibeasik/token. --rotate membuat token baru: sesi Studio & konfigurasi MCP HTTP lama tidak berlaku.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			t   string
			err error
		)
		if rotateToken {
			t, err = localauth.Rotate()
			if err == nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "Token baru dibuat. Restart `vibeasik serve` agar memakainya.")
			}
		} else {
			t, err = localauth.Token()
		}
		if err != nil {
			return err
		}
		fmt.Println(t)
		return nil
	},
}

func init() {
	openCmd.Flags().IntVarP(&openPort, "port", "p", 0, "Port engine (bawaan dari konfigurasi)")
	tokenCmd.Flags().BoolVar(&rotateToken, "rotate", false, "Buat token baru")
	rootCmd.AddCommand(openCmd, tokenCmd)
}
