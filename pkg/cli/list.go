package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/models"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed projects and bundles",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}

		var projects []models.Project
		if err := db.Preload("Components").Find(&projects).Error; err != nil {
			return err
		}

		if len(projects) == 0 {
			fmt.Println("No projects registered yet. Use 'vibeasik scan <path>' or the web UI to add one.")
			return nil
		}

		fmt.Println("==========================================================================================")
		fmt.Printf("%-4s | %-24s | %-12s | %-8s | %s\n", "ID", "PROJECT NAME", "TYPE", "HEALTH", "COMPONENTS & PORTS")
		fmt.Println("------------------------------------------------------------------------------------------")
		for _, p := range projects {
			var compParts []string
			for _, c := range p.Components {
				portStr := ""
				if c.Port > 0 {
					portStr = fmt.Sprintf(":%d", c.Port)
				}
				compParts = append(compParts, fmt.Sprintf("%s%s (%s)", c.Name, portStr, c.Role))
			}
			comps := strings.Join(compParts, " • ")
			if comps == "" {
				comps = "none"
			}
			fmt.Printf("%-4d | %-24s | %-12s | %3d/100  | %s\n", p.ID, truncate(p.Name, 24), p.ProjectType, p.HealthScore, comps)
		}
		fmt.Println("==========================================================================================")

		return nil
	},
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func init() {
	rootCmd.AddCommand(listCmd)
}
