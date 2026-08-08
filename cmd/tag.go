package cmd

import (
	"eko/internal/db"
	"fmt"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:     "tag <snapshot-id> <tag-name>",
	Short:   "Assign a human-readable tag/alias to a snapshot",
	Long:    `Tag assigns a human-readable alias (e.g., 'v1.0', 'pre-refactor', 'stable-checkpoint') to a snapshot ID so you can restore or summarize using human names.`,
	Args:    cobra.ExactArgs(2),
	Example: `  eko tag 8c9d1a2f pre-refactor`,
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		snapID := args[0]
		tagName := args[1]

		database := db.InitDB()
		defer database.Close()

		id, _, err := db.ResolveSnapshot(database, snapID)
		if err != nil {
			return err
		}

		if err := db.SaveTag(database, id, tagName); err != nil {
			return fmt.Errorf("failed to assign tag %q to snapshot %s: %w", tagName, id, err)
		}

		fmt.Printf("Tagged snapshot %s as %q\n", id, tagName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)
}
