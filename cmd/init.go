package cmd

import (
	"eko/internal/db"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize eko project",
	RunE: func(cmd *cobra.Command, args []string) error {
		os.MkdirAll(".eko/snapshots", 0755)
		database := db.InitDB()
		defer database.Close()
		if err := db.MigrateDB(database); err != nil {
			return fmt.Errorf("error initializing database schema: %w", err)
		}
		fmt.Println("Eko initialized.")

		// Check if a .git directory exists
		if info, err := os.Stat(".git"); err == nil && info.IsDir() {
			fmt.Println("Tip: A Git repository was detected. Eko runs independently of Git but automatically ignores the .git directory.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
