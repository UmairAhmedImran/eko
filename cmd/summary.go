package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"eko/internal/ai"
	"eko/internal/db"

	"github.com/spf13/cobra"
)

var (
	summaryJSON     bool
	summaryProvider string
	summarySave     bool
)

type snapshotRow struct {
	ID        string
	Message   string
	Path      string
	CreatedAt string
	Summary   sql.NullString
}

var summaryCmd = &cobra.Command{
	Use:     "summary [snapshot-id] [target-snapshot-id]",
	Aliases: []string{"summarize"},
	Short:   "Generate AI-powered change summary for snapshots",
	Long: `Summary generates AI-powered change summaries of snapshot differences.

Without arguments, it compares the latest snapshot against its preceding snapshot.
With one argument <id>, it compares that snapshot against its predecessor.
With two arguments <id1> <id2>, it compares snapshot <id1> to <id2>.`,
	Example: `  # Summarize changes introduced in the latest snapshot
  eko summary

  # Summarize changes introduced in a specific snapshot
  eko summary 3b7f2a1e

  # Summarize changes between two snapshots
  eko summary 3b7f2a1e 8c9d1a2f

  # Output summary in JSON format using heuristic provider
  eko summary --json --provider heuristic

  # Save the generated AI summary to the database for the snapshot
  eko summary 3b7f2a1e --save`,
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		snapshots, err := getSnapshotsList(database)
		if err != nil {
			return err
		}

		if len(snapshots) == 0 {
			return fmt.Errorf("no snapshots found in project")
		}

		var fromPath, toPath string
		var targetID string

		if len(args) == 0 {
			if len(snapshots) == 1 {
				toPath = snapshots[0].Path
				targetID = snapshots[0].ID
			} else {
				// Latest snapshot vs predecessor
				toPath = snapshots[0].Path
				fromPath = snapshots[1].Path
				targetID = snapshots[0].ID
			}
		} else if len(args) == 1 {
			targetID = args[0]
			idx := findSnapshotIndex(snapshots, targetID)
			if idx == -1 {
				return fmt.Errorf("snapshot not found: %s", targetID)
			}
			toPath = snapshots[idx].Path
			if idx+1 < len(snapshots) {
				fromPath = snapshots[idx+1].Path
			}
		} else {
			id1, id2 := args[0], args[1]
			idx1 := findSnapshotIndex(snapshots, id1)
			if idx1 == -1 {
				return fmt.Errorf("snapshot not found: %s", id1)
			}
			idx2 := findSnapshotIndex(snapshots, id2)
			if idx2 == -1 {
				return fmt.Errorf("snapshot not found: %s", id2)
			}
			fromPath = snapshots[idx1].Path
			toPath = snapshots[idx2].Path
			targetID = snapshots[idx2].ID
		}

		ctx := context.Background()
		result, err := ai.GenerateSnapshotSummary(ctx, fromPath, toPath, summaryProvider)
		if err != nil {
			return fmt.Errorf("failed to generate AI summary: %w", err)
		}

		if summarySave && targetID != "" {
			if err := db.SaveSummary(database, targetID, result.Summary); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save summary to db: %v\n", err)
			}
		}

		if summaryJSON {
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("✦ Snapshot Change Summary [%s]\n", targetID)
		fmt.Printf("Provider: %s\n", result.ProviderUsed)
		fmt.Printf("Files Changed: %d (+%d / -%d lines)\n\n", result.FilesChanged, result.TotalInsertions, result.TotalDeletions)
		fmt.Println(result.Summary)

		return nil
	},
}

func getSnapshotsList(database *sql.DB) ([]snapshotRow, error) {
	rows, err := database.Query("SELECT id, message, path, created_at, summary FROM snapshots ORDER BY created_at DESC, rowid DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []snapshotRow
	for rows.Next() {
		var s snapshotRow
		if err := rows.Scan(&s.ID, &s.Message, &s.Path, &s.CreatedAt, &s.Summary); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func findSnapshotIndex(list []snapshotRow, id string) int {
	for i, s := range list {
		if s.ID == id {
			return i
		}
	}
	return -1
}

func init() {
	summaryCmd.Flags().BoolVarP(&summaryJSON, "json", "j", false, "output summary details in JSON format")
	summaryCmd.Flags().StringVarP(&summaryProvider, "provider", "p", "auto", "AI provider to use (auto, heuristic, openai, gemini)")
	summaryCmd.Flags().BoolVarP(&summarySave, "save", "s", false, "save generated summary to snapshot database record")
	rootCmd.AddCommand(summaryCmd)
}
