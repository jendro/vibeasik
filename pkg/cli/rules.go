package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/guidelines"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Daftar rule audit bawaan per kategori",
	RunE: func(cmd *cobra.Command, args []string) error {
		cats, err := guidelines.LoadAllGuidelines()
		if err != nil {
			return err
		}
		sort.Slice(cats, func(i, j int) bool { return cats[i].FilePath < cats[j].FilePath })
		total := 0
		for _, c := range cats {
			fmt.Printf("%-40s %3d rule\n", c.Category, len(c.Guidelines))
			total += len(c.Guidelines)
		}
		fmt.Printf("\n%d kategori, %d rule\n", len(cats), total)
		return nil
	},
}

func init() { rootCmd.AddCommand(rulesCmd) }
