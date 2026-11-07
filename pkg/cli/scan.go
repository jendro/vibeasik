package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/scanner"
)

var (
	jsonOutput bool
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a directory and inspect detected Backend/Frontend bundles and stacks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]
		res, err := scanner.ScanDirectory(targetPath)
		if err != nil {
			return err
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("==========================================================")
		fmt.Printf("📂 Project Directory: %s\n", res.RootPath)
		fmt.Printf("🏷️  Suggested Name:   %s (%s)\n", res.SuggestedName, res.SuggestedSlug)
		fmt.Printf("🏗️  Topology Type:    %s\n", res.ProjectType)
		if res.GitBranch != "" {
			fmt.Printf("🌿 Git Branch:       %s\n", res.GitBranch)
		}
		fmt.Println("==========================================================")

		fmt.Printf("Found %d Component(s):\n\n", len(res.Components))
		for i, c := range res.Components {
			portInfo := "no port"
			if c.Port > 0 {
				portInfo = fmt.Sprintf("Port %d", c.Port)
			}
			fmt.Printf(" [%d] %s (%s)\n", i+1, c.Name, c.Role)
			fmt.Printf("     Path:      %s\n", c.RelativePath)
			fmt.Printf("     Stack:     %s / %s (%s)\n", c.Language, c.Framework, c.PackageManager)
			fmt.Printf("     Dev Cmd:   %s [%s]\n", c.DevCommand, portInfo)
			if c.EnvFile != "" {
				fmt.Printf("     Env:       %s\n", c.EnvFile)
			}
			fmt.Println()
		}

		fmt.Println("Folder Tree Preview:")
		fmt.Print(res.DirectoryTree)
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Output results in JSON format")
	rootCmd.AddCommand(scanCmd)
}
