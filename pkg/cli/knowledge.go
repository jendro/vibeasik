package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/knowledgesync"
)

var knowledgeCmd = &cobra.Command{Use: "knowledge", Short: "Pusat Pengetahuan: tarik paket, kirim sinyal & usulan"}

var forcePull bool

var knowledgePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Tarik paket pengetahuan terbaru dari server vibeasik (free tanpa login; full untuk paket Pro)",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}
		res, err := knowledgesync.Pull(db, forcePull)
		if err != nil {
			return err
		}
		if res.UpToDate {
			fmt.Printf("Sudah terbaru: paket %s (%s)\n", res.Version, res.Tier)
			return nil
		}
		fmt.Printf("Paket %s (%s): %d baru, %d diperbarui, %d dilewati (sudah ada), %d tautan\n",
			res.Version, res.Tier, res.Added, res.Updated, res.Skipped, res.Links)
		return nil
	},
}

var knowledgeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Versi paket terpasang dan status sinkronisasi",
	RunE: func(cmd *cobra.Command, args []string) error {
		st := knowledgesync.LoadState()
		if st.Version == "" {
			fmt.Println("Belum ada paket. Jalankan `vibeasik knowledge pull`.")
			return nil
		}
		fmt.Printf("Paket:  %s (%s), %d node\nDitarik: %s\n", st.Version, st.Tier, st.Nodes, st.PulledAt.Format(time.RFC3339))
		fmt.Printf("Usulan terkirim: %d\n", len(st.Contributed))
		return nil
	},
}

var knowledgeSignalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Kirim angka helpful/harmful node paket (anonim, tanpa teks; wajib login)",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}
		n, err := knowledgesync.SendSignals(db)
		if err != nil {
			return err
		}
		fmt.Printf("%d sinyal terkirim.\n", n)
		return nil
	},
}

var contribYes string

var knowledgeContributeCmd = &cobra.Command{
	Use:   "contribute",
	Short: "Tinjau usulan pengetahuan; kirim yang Anda setujui dengan --send <id,id,...>",
	Long: `Menampilkan node buatan Anda (promosi, review QA, riset celah) yang aman dibagikan — sudah disensor
dan tanpa path/email. Tidak ada yang dikirim kecuali Anda memilihnya lewat --send.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}
		if contribYes == "" {
			cands := knowledgesync.Candidates(db)
			for _, c := range cands {
				fmt.Printf("#%d [%s/%s] %s\n    %s\n", c.ID, c.Domain, c.Kind, c.Title, shortText(c.Content, 160))
			}
			fmt.Printf("\n%d usulan. Kirim dengan: vibeasik knowledge contribute --send 12,15\n", len(cands))
			return nil
		}
		var ids []uint
		for _, p := range strings.Split(contribYes, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && v > 0 {
				ids = append(ids, uint(v))
			}
		}
		n, err := knowledgesync.Contribute(db, ids)
		if err != nil {
			return err
		}
		fmt.Printf("%d usulan terkirim untuk dikurasi.\n", n)
		return nil
	},
}

func shortText(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func init() {
	knowledgePullCmd.Flags().BoolVar(&forcePull, "force", false, "Impor ulang walau versi sama")
	knowledgeContributeCmd.Flags().StringVar(&contribYes, "send", "", "Daftar id usulan yang disetujui untuk dikirim")
	knowledgeCmd.AddCommand(knowledgePullCmd, knowledgeStatusCmd, knowledgeSignalsCmd, knowledgeContributeCmd)
	rootCmd.AddCommand(knowledgeCmd)
}
