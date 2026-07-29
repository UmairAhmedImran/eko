package cmd

import (
	"encoding/json"
	"fmt"

	"eko/internal/db"

	"github.com/spf13/cobra"
)

var jsonOutput bool

type historyEntry struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "Show snapshots",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		rows, err := database.Query("SELECT id, created_at FROM snapshots")
		if err != nil {
			return err
		}
		defer rows.Close()

		entries := []historyEntry{}
		for rows.Next() {
			var entry historyEntry
			if err := rows.Scan(&entry.ID, &entry.CreatedAt); err != nil {
				return err
			}
			entries = append(entries, entry)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating history rows: %w", err)
		}

		if jsonOutput {
			data, err := json.Marshal(entries)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		for _, entry := range entries {
			fmt.Println(entry.ID, entry.CreatedAt)
		}

		return nil
	},
}

func init() {
	historyCmd.Flags().BoolVar(&jsonOutput, "json", false, "output history as JSON")
	rootCmd.AddCommand(historyCmd)
}
