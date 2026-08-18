package cmd

import (
	"context"
	"eko/internal/ai"
	"eko/internal/db"
	"eko/internal/snapshot"
	"eko/internal/util"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	saveMessage  string
	saveAI       bool
	saveAIProv   string
	saveWithEnv  bool
	saveProgress bool
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save project snapshot",
	Long: `Save creates a new snapshot of the current project state into the CAS object store.

Each snapshot generates a lightweight manifest file and stores unique file blobs in
.eko/objects/, compressed with gzip and deduplicated across all snapshots.`,
	Example: `  # Save a snapshot of the current project state
  eko save

  # Save with a custom message
  eko save -m "fixed db lock issue"

  # Save and auto-generate an AI change summary
  eko save --ai

  # Save with AI summary using a specific provider
  eko save --ai --provider heuristic`,
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		// Get previous snapshot path before creating a new one
		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		if saveWithEnv {
			fmt.Println("Warning: Capturing environment variables may store sensitive credentials (API keys, passwords, etc.) in the snapshot.")
		}

		// Set up progress bar if enabled and stdout is a TTY
		var onProgress func()
		showProgress := saveProgress && util.IsTTY(os.Stderr)
		if showProgress {
			fileCount, err := snapshot.CountFiles()
			if err == nil && fileCount > 0 {
				prog := util.NewProgress(fileCount, os.Stderr, "Saving snapshot...")
				prog.Start()
				defer prog.Stop()
				onProgress = prog.Increment
			}
		}

		id, path, err := snapshot.CreateSnapshot(database, saveWithEnv, onProgress)
		if err != nil {
			return err
		}

		var summaryText string
		if saveAI {
			ctx := context.Background()
			res, err := ai.GenerateSnapshotSummary(ctx, prevPath, path, saveAIProv)
			if err == nil && res != nil {
				summaryText = res.Summary
				if saveMessage == "snapshot" {
					saveMessage = res.Summary
				}
			}
		}

		if _, err := database.Exec(
			"INSERT INTO snapshots(id, message, path, summary) VALUES (?, ?, ?, ?)",
			id,
			saveMessage,
			path,
			summaryText,
		); err != nil {
			// CreateSnapshot has already written the manifest, but the failed row
			// insert leaves no supported way to list or restore it. Remove that
			// unreachable manifest so retries do not accumulate phantom snapshots.
			dbErr := fmt.Errorf("failed to save snapshot to db: %w", err)
			if rmErr := os.Remove(path); rmErr != nil {
				return errors.Join(dbErr, fmt.Errorf("could not remove orphaned snapshot manifest %s: %w", path, rmErr))
			}
			return dbErr
		}

		fmt.Println("Snapshot saved:", id)
		if summaryText != "" {
			fmt.Println("AI Summary:", summaryText)
		}

		return nil
	},
}

func init() {
	saveCmd.Flags().StringVarP(&saveMessage, "message", "m", "snapshot", "log message describing the snapshot")
	saveCmd.Flags().BoolVarP(&saveAI, "ai", "a", false, "auto-generate AI summary of changes")
	saveCmd.Flags().StringVar(&saveAIProv, "provider", "auto", "AI provider for auto-generated summary (auto, heuristic, openai, gemini)")
	saveCmd.Flags().BoolVar(&saveWithEnv, "with-env", false, "capture environment variables (WARNING: this may include sensitive credentials)")
	saveCmd.Flags().BoolVar(&saveProgress, "progress", true, "show progress bar during save (default true when TTY)")
	rootCmd.AddCommand(saveCmd)
}
