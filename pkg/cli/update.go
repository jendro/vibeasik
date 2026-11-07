package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/update"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Periksa dan pasang versi vibeasik terbaru",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		u := update.New(Version, Edition)
		info := u.Check(ctx)
		switch {
		case info.Error != "":
			return fmt.Errorf("%s", info.Error)
		case info.Reason != "" && !info.Available:
			fmt.Printf("vibeasik %s: pembaruan tidak berlaku (%s).\n", Version, info.Reason)
			return nil
		case !info.Available:
			fmt.Printf("vibeasik %s sudah versi terbaru.\n", Version)
			return nil
		}
		fmt.Printf("Versi baru tersedia: %s (terpasang %s).\n", info.Latest, Version)
		if updateCheckOnly {
			return nil
		}
		if !info.CanApply {
			fmt.Printf("%s\nUnduh: %s\n", info.Reason, info.URL)
			return nil
		}
		fmt.Println("→ Mengunduh dan memverifikasi…")
		v, err := u.Apply(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("✅ vibeasik %s terpasang. Jalankan ulang engine (`vibeasik serve`) untuk memakainya.\n", v)
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Hanya periksa, jangan pasang")
	rootCmd.AddCommand(updateCmd)
}
