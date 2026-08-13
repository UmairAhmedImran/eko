package cmd

import (
	"encoding/json"
	"fmt"

	"eko/internal/api"
	"eko/internal/db"

	"github.com/spf13/cobra"
)

var (
	diffJSON bool
	diffFull bool
)

var diffCmd = &cobra.Command{
	Use:     "diff [id1] [id2]",
	Short:   "Compare two snapshots",
	Long:    `Compare and show differences between two snapshots.`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		target1 := args[0]
		target2 := args[1]

		database := db.InitDB()
		defer database.Close()

		_, path1, err := db.ResolveSnapshot(database, target1)
		if err != nil {
			return err
		}

		_, path2, err := db.ResolveSnapshot(database, target2)
		if err != nil {
			return err
		}

		diffs, err := api.BuildDiff(path1, path2)
		if err != nil {
			return fmt.Errorf("diff: failed to build diff: %w", err)
		}

		if diffJSON {
			data, err := json.Marshal(diffs)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		if len(diffs) == 0 {
			fmt.Println("No differences found.")
			return nil
		}

		if diffFull {
			for _, df := range diffs {
				if df.Original == "" {
					fmt.Printf("--- %s (Added) ---\n+++ Content\n%s\n", df.Name, df.Modified)
				} else if df.Modified == "" {
					fmt.Printf("--- %s (Deleted) ---\n--- Content\n%s\n", df.Name, df.Original)
				} else {
					fmt.Printf("--- %s (Modified) ---\n--- Original\n%s\n+++ Modified\n%s\n", df.Name, df.Original, df.Modified)
				}
			}
		} else {
			for _, df := range diffs {
				if df.Original == "" {
					fmt.Printf("  Added:    %s\n", df.Name)
				} else if df.Modified == "" {
					fmt.Printf("  Deleted:  %s\n", df.Name)
				} else {
					fmt.Printf("  Modified: %s\n", df.Name)
				}
			}
		}

		return nil
	},
}

func init() {
	diffCmd.Flags().BoolVar(&diffJSON, "json", false, "output diff as JSON")
	diffCmd.Flags().BoolVarP(&diffFull, "full", "v", false, "show full before/after content")
	rootCmd.AddCommand(diffCmd)
}
