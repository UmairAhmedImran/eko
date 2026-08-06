package cmd

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"eko/internal/db"

	"github.com/spf13/cobra"
)

// Values accepted by --format.
const (
	historyFormatText = "text"
	historyFormatJSON = "json"
	historyFormatMD   = "md"
	historyFormatCSV  = "csv"
)

var (
	jsonOutput    bool
	verboseOutput bool
	historyFormat string
)

type historyEntry struct {
	ID        string `json:"id"`
	Message   string `json:"message,omitempty"`
	CreatedAt string `json:"created_at"`
	Summary   string `json:"summary,omitempty"`
}

// resolveHistoryFormat reduces --format and the older --json flag to the single
// format history should render.
//
// An unsupported format is rejected before the conflict check, so a typo is
// always reported as a typo rather than as a flag conflict. --json is kept as a
// shortcut for --format json; it only conflicts when --format was passed
// explicitly asking for something else, because silently printing JSON to
// someone who wrote --format md is worse than refusing.
func resolveHistoryFormat(format string, formatChanged, legacyJSON bool) (string, error) {
	switch format {
	case historyFormatText, historyFormatJSON, historyFormatMD, historyFormatCSV:
	default:
		return "", fmt.Errorf("unsupported --format %q: use one of text, json, md, csv", format)
	}

	if legacyJSON {
		if formatChanged && format != historyFormatJSON {
			return "", fmt.Errorf("--json conflicts with --format %s: pass only one of them", format)
		}
		return historyFormatJSON, nil
	}

	return format, nil
}

// escapeMarkdownCell makes one field safe to place in a table cell. An embedded
// newline would end the row early and a bare pipe would open a new column, so a
// message containing either would otherwise corrupt the whole table.
func escapeMarkdownCell(value string) string {
	replacer := strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "|", "\\|")
	return replacer.Replace(value)
}

// renderHistoryMarkdown writes a table with a fixed column order. With no
// entries it still writes the header, so the output is always a valid table.
func renderHistoryMarkdown(entries []historyEntry) {
	fmt.Println("| ID | Created At | Message | Summary |")
	fmt.Println("| --- | --- | --- | --- |")
	for _, entry := range entries {
		fmt.Printf("| %s | %s | %s | %s |\n",
			escapeMarkdownCell(entry.ID),
			escapeMarkdownCell(entry.CreatedAt),
			escapeMarkdownCell(entry.Message),
			escapeMarkdownCell(entry.Summary),
		)
	}
}

// renderHistoryCSV writes RFC 4180 output. encoding/csv handles quoting, so
// commas, quotes and newlines inside a message survive a round trip instead of
// being stripped the way the Markdown renderer has to strip them.
func renderHistoryCSV(entries []historyEntry) error {
	writer := csv.NewWriter(os.Stdout)
	if err := writer.Write([]string{"id", "created_at", "message", "summary"}); err != nil {
		return fmt.Errorf("error writing history CSV: %w", err)
	}
	for _, entry := range entries {
		if err := writer.Write([]string{entry.ID, entry.CreatedAt, entry.Message, entry.Summary}); err != nil {
			return fmt.Errorf("error writing history CSV: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error writing history CSV: %w", err)
	}
	return nil
}

var historyCmd = &cobra.Command{
	Use:     "history",
	Short:   "Show snapshots",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolved before the database is touched, so an unusable flag
		// combination fails without opening or querying anything.
		format, err := resolveHistoryFormat(historyFormat, cmd.Flags().Changed("format"), jsonOutput)
		if err != nil {
			return err
		}

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

		switch format {
		case historyFormatJSON:
			data, err := json.Marshal(entries)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		case historyFormatMD:
			renderHistoryMarkdown(entries)
			return nil
		case historyFormatCSV:
			return renderHistoryCSV(entries)
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
	historyCmd.Flags().BoolVar(&jsonOutput, "json", false, "output history as JSON (shortcut for --format json)")
	historyCmd.Flags().StringVar(&historyFormat, "format", historyFormatText, "output format: text, json, md, or csv")
	historyCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "show verbose history with detailed AI summaries")
	rootCmd.AddCommand(historyCmd)
}
