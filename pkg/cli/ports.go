package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/runner"
)

var (
	portsEnsureUnique bool
	portsProjectID    uint
)

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Inspect port registry or ensure unique collision-free ports for local development",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}

		if portsEnsureUnique {
			updated, err := runner.EnsureUniquePorts(db, portsProjectID)
			if err != nil {
				return err
			}
			fmt.Printf("✅ Berhasil mengalokasikan port unik untuk %d komponen:\n", len(updated))
			for _, c := range updated {
				fmt.Printf("   • [%d] %s (Project %d) -> run_port: :%d (port bawaan: :%d)\n", c.ID, c.Name, c.ProjectID, c.RunPort, c.Port)
			}
			return nil
		}

		svc := runner.NewService(db)
		entries, err := svc.Registry()
		if err != nil {
			return err
		}

		fmt.Println("==========================================================================================")
		fmt.Printf("%-4s | %-20s | %-22s | %-8s | %-8s | %s\n", "ID", "PROJECT", "COMPONENT", "DECLARED", "RUN_PORT", "STATUS")
		fmt.Println("------------------------------------------------------------------------------------------")
		for _, e := range entries {
			status := "idle"
			if e.Listening {
				status = "🟢 LIVE"
			}
			fmt.Printf("%-4d | %-20s | %-22s | :%-7d | :%-7d | %s\n", e.ComponentID, truncate(e.ProjectName, 20), truncate(e.ComponentName, 22), e.DeclaredPort, e.RunPort, status)
		}
		fmt.Println("==========================================================================================")
		return nil
	},
}

func init() {
	portsCmd.Flags().BoolVar(&portsEnsureUnique, "ensure-unique", false, "Otomatis alokasikan port unik lokal bebas bentrok untuk semua komponen")
	portsCmd.Flags().UintVar(&portsProjectID, "project", 0, "Filter untuk project ID tertentu (0 = seluruh proyek)")
	rootCmd.AddCommand(portsCmd)
}
