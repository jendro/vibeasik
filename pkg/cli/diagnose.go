package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/database"
	"github.com/vibeasik/vibeasik/pkg/diagnostics"
	"github.com/vibeasik/vibeasik/pkg/models"
	"github.com/vibeasik/vibeasik/pkg/scanner"
)

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose [project_id or path]",
	Short: "Run automated health inspection and architectural diagnostics on a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		db, err := database.InitDB(dbPath)
		if err != nil {
			return err
		}

		var project models.Project
		// Check if target is numeric ID
		if id, err := strconv.Atoi(target); err == nil {
			if err := db.Preload("Components").First(&project, id).Error; err != nil {
				return fmt.Errorf("project ID %d not found in database", id)
			}
		} else {
			// Scan path directly as temporary project
			scanRes, err := scanner.ScanDirectory(target)
			if err != nil {
				return err
			}
			project = models.Project{
				Name:        scanRes.SuggestedName,
				RootPath:    scanRes.RootPath,
				ProjectType: scanRes.ProjectType,
				GitBranch:   scanRes.GitBranch,
			}
			for _, cp := range scanRes.Components {
				project.Components = append(project.Components, models.Component{
					Name:         cp.Name,
					Role:         cp.Role,
					RelativePath: cp.RelativePath,
					AbsolutePath: cp.AbsolutePath,
					Language:     cp.Language,
					Framework:    cp.Framework,
					Port:         cp.Port,
					DevCommand:   cp.DevCommand,
					EnvFile:      cp.EnvFile,
					EnvSummary:   cp.EnvSummary,
				})
			}
		}

		var allProjects []models.Project
		db.Preload("Components").Find(&allProjects)

		report, err := diagnostics.AnalyzeProject(&project, allProjects)
		if err != nil {
			return err
		}

		fmt.Println("==========================================================")
		fmt.Printf("🔍 Codebase Diagnostics Report: %s\n", project.Name)
		fmt.Printf("🎯 Health Score: %d/100\n", report.Score)
		fmt.Printf("📝 Summary: %s\n", report.Summary)
		fmt.Println("==========================================================")

		var findings []models.Finding
		_ = json.Unmarshal([]byte(report.FindingsJSON), &findings)

		if len(findings) == 0 {
			fmt.Println("✅ No issues found! Clean codebase.")
		} else {
			fmt.Printf("Detected %d Findings:\n\n", len(findings))
			for i, f := range findings {
				icon := "ℹ️"
				if f.Severity == "CRITICAL" {
					icon = "🚨"
				} else if f.Severity == "WARNING" {
					icon = "⚠️"
				}
				fmt.Printf(" [%d] %s [%s] %s\n", i+1, icon, f.Severity, f.Title)
				fmt.Printf("     Issue:      %s\n", f.Description)
				fmt.Printf("     Suggestion: %s\n\n", f.Suggestion)
			}
		}

		var opts []models.Optimization
		_ = json.Unmarshal([]byte(report.Optimizations), &opts)
		if len(opts) > 0 {
			fmt.Println("💡 Actionable Optimization Opportunities:")
			for _, o := range opts {
				fmt.Printf(" • [%s] %s: %s\n", o.Category, o.Title, o.Description)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
}
