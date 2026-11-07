package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/account"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Masuk dengan akun vibeasik (membuka fitur sesuai paket)",
	RunE: func(cmd *cobra.Command, args []string) error {
		host, _ := os.Hostname()
		dc, err := account.StartLogin("vibeasik " + Edition + " · " + host)
		if err != nil {
			return err
		}
		fmt.Printf("Buka %s lalu masukkan kode:\n\n    %s\n\n", dc.VerificationURI, dc.UserCode)
		_ = openURL(dc.VerificationURIComplete)
		interval := time.Duration(max(dc.Interval, 2)) * time.Second
		deadline := time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(interval)
			st, err := account.PollLogin(dc.DeviceCode)
			if errors.Is(err, account.ErrPending) {
				continue
			}
			if err != nil {
				return err
			}
			fmt.Printf("✅ Masuk sebagai %s · paket %s\n", st.Email, st.Plan)
			return nil
		}
		return errors.New("waktu login habis; jalankan ulang `vibeasik login`")
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Keluar dan cabut perangkat ini dari akun",
	RunE: func(cmd *cobra.Command, args []string) error {
		account.Activate()
		if err := account.Logout(); err != nil {
			return err
		}
		fmt.Println("Keluar. Fitur berbayar terkunci kembali.")
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Tampilkan akun, paket, dan fitur aktif",
	RunE: func(cmd *cobra.Command, args []string) error {
		account.Activate()
		if account.Current().LoggedIn {
			if err := account.Refresh(); err != nil {
				fmt.Println("⚠", err)
			}
		}
		st := account.Current()
		if !st.LoggedIn {
			fmt.Printf("Belum login (edisi %s). Jalankan `vibeasik login`.\n", Edition)
			return nil
		}
		fmt.Printf("Akun:    %s\nPaket:   %s\nPerangkat: %s\nServer:  %s\n", st.Email, st.Plan, st.DeviceID, st.CloudURL)
		if st.UsableUntil != nil {
			fmt.Printf("Berlaku: sampai %s (termasuk masa tenggang offline)\n", st.UsableUntil.Format("2006-01-02 15:04"))
		}
		if len(st.Features) > 0 {
			fmt.Printf("Fitur:   %s\n", strings.Join(st.Features, ", "))
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(loginCmd, logoutCmd, whoamiCmd) }
