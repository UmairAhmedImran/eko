package cmd

import (
	"eko/internal/db"
	"eko/internal/snapshot"
	"fmt"

	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:     "restore [id]",
	Short:   "Restore snapshot",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		database := db.InitDB()
		defer database.Close()

		id, path, err := db.ResolveSnapshot(database, target)
		if err != nil {
			return err
		}

		err = snapshot.RestoreSnapshot(path)
		if err != nil {
			return err
		}
		fmt.Printf("Restored: %s (tag: %s)\n", id, target)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
