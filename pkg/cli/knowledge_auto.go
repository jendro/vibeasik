package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/account"
	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
)

var consentYes bool

var knowledgeConsentCmd = &cobra.Command{
	Use:   "consent",
	Short: "Setujui Syarat & Ketentuan Pusat Pengetahuan (diperlukan untuk menarik paket dan berkontribusi)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if knowledgesync.HasConsent() {
			st := knowledgesync.LoadState()
			fmt.Printf("Sudah disetujui (versi %s, %s).\n", st.ConsentVersion, st.ConsentAt.Format("2006-01-02"))
			return nil
		}
		fmt.Println(knowledgesync.ConsentText)
		if !consentYes {
			fmt.Print("\nKetik \"setuju\" untuk menyetujui: ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(line), "setuju") {
				fmt.Println("Tidak disetujui. Pusat Pengetahuan tetap nonaktif.")
				return nil
			}
		}
		if err := knowledgesync.Consent(); err != nil {
			return err
		}
		fmt.Println("Disetujui. Jalankan `vibeasik knowledge pull` untuk menarik paket.")
		if account.DeviceToken() == "" {
			fmt.Println("Catatan: kiriman hanya berjalan setelah `vibeasik login`.")
		}
		return nil
	},
}

var knowledgeAutoCmd = &cobra.Command{
	Use:   "auto [status|on|ask|off|preview]",
	Short: "Kirim otomatis pengetahuan berguna: lihat status, pratinjau, atau (Pro) ubah mode",
	Long: `Node pengetahuan yang terbukti berguna dikirim otomatis ke server vibeasik untuk dikurasi (bawaan menyala).

  status    mode yang berlaku, persetujuan, dan jumlah kiriman
  preview   daftar node yang AKAN dikirim dan yang dilewati beserta alasannya (tidak mengirim apa pun)
  on|ask|off  mengubah mode; hanya untuk paket Pro (Free terkunci pada auto)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sub := "status"
		if len(args) == 1 {
			sub = strings.ToLower(args[0])
		}
		switch sub {
		case "status":
			return autoStatus()
		case "preview":
			return autoPreview()
		case "on", "ask", "off":
			m := knowledgesync.Mode(sub)
			if sub == "on" {
				m = knowledgesync.ModeAuto
			}
			if err := knowledgesync.SetMode(m); err != nil {
				return err
			}
			fmt.Printf("Mode kirim: %s\n", m)
			return nil
		}
		return fmt.Errorf("perintah %q tidak dikenal (status, preview, on, ask, off)", sub)
	},
}

func autoStatus() error {
	mode := knowledgesync.EffectiveMode()
	lock := "dapat diubah (Pro)"
	if !knowledgesync.CanControl() {
		lock = "terkunci pada paket Free"
	}
	fmt.Printf("Mode kirim : %s (%s)\n", mode, lock)
	if knowledgesync.HasConsent() {
		st := knowledgesync.LoadState()
		fmt.Printf("Persetujuan: ya (versi %s, %s)\n", st.ConsentVersion, st.ConsentAt.Format("2006-01-02"))
	} else {
		fmt.Println("Persetujuan: belum. Jalankan `vibeasik knowledge consent`.")
	}
	if account.DeviceToken() == "" {
		fmt.Println("Akun       : belum login (`vibeasik login`); kiriman belum berjalan")
	}
	l := knowledgesync.LoadLedger()
	fmt.Printf("Terkirim   : %d total, %d hari ini (batas %d/hari)\n", len(l.Sent), l.SentOn(time.Now()), knowledgesync.MaxPerDay)
	return nil
}

func autoPreview() error {
	db, err := database.InitDB(dbPath)
	if err != nil {
		return err
	}
	el := knowledgesync.AutoEligible(db, time.Now())
	fmt.Printf("Akan dikirim (%d):\n", len(el.OK))
	for _, c := range el.OK {
		fmt.Printf("  #%d [%s/%s] %s\n      %s\n", c.ID, c.Domain, c.Kind, c.Title, shortText(c.Content, 140))
	}
	fmt.Printf("\nDilewati (%d):\n", len(el.Skipped))
	for _, s := range el.Skipped {
		fmt.Printf("  #%d  %s\n", s.LocalID, s.Reason)
	}
	fmt.Printf("\nBatas harian %d node. Tidak ada yang dikirim oleh perintah ini.\n", knowledgesync.MaxPerDay)
	return nil
}

var knowledgeLedgerCmd = &cobra.Command{
	Use:   "ledger",
	Short: "Riwayat kiriman pengetahuan dan alasan node dilewati",
	RunE: func(cmd *cobra.Command, args []string) error {
		l := knowledgesync.LoadLedger()
		fmt.Printf("Terkirim (%d):\n", len(l.Sent))
		for _, s := range l.Sent {
			id := ""
			if s.ServerID != 0 {
				id = fmt.Sprintf(" server#%d", s.ServerID)
			}
			fmt.Printf("  %s  lokal#%d%s  [%s]  %s\n", s.SentAt.Format("2006-01-02 15:04"), s.LocalID, id, s.Mode, s.Title)
		}
		fmt.Printf("\nDilewati (%d):\n", len(l.Skipped))
		for _, s := range l.Skipped {
			fmt.Printf("  %s  lokal#%d  %s\n", s.At.Format("2006-01-02"), s.LocalID, s.Reason)
		}
		return nil
	},
}

func init() {
	knowledgeConsentCmd.Flags().BoolVar(&consentYes, "yes", false, "Setujui tanpa bertanya (setelah Anda membaca ketentuannya)")
	knowledgeCmd.AddCommand(knowledgeConsentCmd, knowledgeAutoCmd, knowledgeLedgerCmd)
}
