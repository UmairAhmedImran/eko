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
		id := args[0]
		database := db.InitDB()
		var path string
		database.QueryRow("SELECT path FROM snapshots WHERE id=?", id).Scan(&path)
		err := snapshot.RestoreSnapshot(path)
		if err != nil {
			return err
		}
		fmt.Println("Restored:", id)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}
