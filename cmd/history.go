package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"eko/internal/db"

	"github.com/spf13/cobra"
)

var (
	jsonOutput    bool
	verboseOutput bool
)

type historyEntry struct {
	ID        string `json:"id"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"created_at"`
	Summary   string `json:"summary,omitempty"`
}

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "Show snapshots",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		rows, err := database.Query("SELECT id, message, created_at, summary FROM snapshots ORDER BY created_at DESC, rowid DESC")
		if err != nil {
			// Fallback for older schemas without message or summary columns
			rows, err = database.Query("SELECT id, created_at FROM snapshots")
			if err != nil {
				return err
			}
		}
		defer rows.Close()

		entries := []historyEntry{}
		cols, _ := rows.Columns()

		for rows.Next() {
			var entry historyEntry
			if len(cols) >= 4 {
				var msg sql.NullString
				var sum sql.NullString
				if err := rows.Scan(&entry.ID, &msg, &entry.CreatedAt, &sum); err != nil {
					return err
				}
				if msg.Valid {
					entry.Message = msg.String
				}
				if sum.Valid {
					entry.Summary = sum.String
				}
			} else {
				if err := rows.Scan(&entry.ID, &entry.CreatedAt); err != nil {
					return err
				}
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
			if verboseOutput || entry.Summary != "" {
				fmt.Printf("%s %s - %s\n", entry.ID, entry.CreatedAt, entry.Message)
				if entry.Summary != "" {
					fmt.Printf("  ✦ Summary: %s\n", entry.Summary)
				}
			} else if entry.Message != "" && entry.Message != "snapshot" {
				fmt.Printf("%s %s - %s\n", entry.ID, entry.CreatedAt, entry.Message)
			} else {
				fmt.Println(entry.ID, entry.CreatedAt)
			}
		}

		return nil
	},
}

func init() {
	historyCmd.Flags().BoolVar(&jsonOutput, "json", false, "output history as JSON")
	historyCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "show verbose history with detailed AI summaries")
	rootCmd.AddCommand(historyCmd)
}
