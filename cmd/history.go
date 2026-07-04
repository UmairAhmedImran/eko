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
	Run: func(cmd *cobra.Command, args []string) {
		database := db.InitDB()
		defer database.Close()

		rows, err := database.Query("SELECT id, created_at FROM snapshots")
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		entries := []historyEntry{}
		for rows.Next() {
			var entry historyEntry
			if err := rows.Scan(&entry.ID, &entry.CreatedAt); err != nil {
				panic(err)
			}
			entries = append(entries, entry)
		}

		if jsonOutput {
			data, err := json.Marshal(entries)
			if err != nil {
				panic(err)
			}
			fmt.Println(string(data))
			return
		}

		for _, entry := range entries {
			fmt.Println(entry.ID, entry.CreatedAt)
		}
	},
}

func init() {
	historyCmd.Flags().BoolVar(&jsonOutput, "json", false, "output history as JSON")
	rootCmd.AddCommand(historyCmd)
}
