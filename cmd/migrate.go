package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"eko/internal/db"
	"eko/internal/manifest"
	"eko/internal/objects"
	"eko/internal/util"

	"github.com/spf13/cobra"
)

var migrateDryRun bool

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate legacy snapshots to Content-Addressable Storage (CAS) format",
	Long: `Migrate converts old full-directory snapshots (.eko/snapshots/<id>/) to the new
CAS object store and manifest format, freeing disk space while preserving all snapshot history.`,
	Example: `  # Dry-run: check what can be migrated
  eko migrate --dry-run

  # Perform migration
  eko migrate`,
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		rows, err := database.Query("SELECT id, path, message, created_at FROM snapshots ORDER BY created_at ASC")
		if err != nil {
			return fmt.Errorf("failed to query snapshots: %w", err)
		}
		defer rows.Close()

		type item struct {
			id        string
			path      string
			message   string
			createdAt string
		}
		var legacyItems []item

		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.path, &it.message, &it.createdAt); err != nil {
				return err
			}
			// If no manifest exists and legacy dir exists, it's a candidate for migration
			if !manifest.Exists(ekoDir, it.id) {
				if info, err := os.Stat(it.path); err == nil && info.IsDir() {
					legacyItems = append(legacyItems, it)
				}
			}
		}
		rows.Close()

		if len(legacyItems) == 0 {
			fmt.Println("No legacy snapshots found to migrate. All snapshots are up to date.")
			return nil
		}

		if migrateDryRun {
			fmt.Printf("[DRY RUN] Found %d legacy snapshot(s) eligible for CAS migration:\n\n", len(legacyItems))
			for _, it := range legacyItems {
				fmt.Printf("  • %s  (%s) -> %s\n", it.id, it.createdAt, it.path)
			}
			fmt.Println("\nRun 'eko migrate' without --dry-run to convert them.")
			return nil
		}

		store, err := objects.New(ekoDir)
		if err != nil {
			return fmt.Errorf("failed to open object store: %w", err)
		}

		fmt.Printf("Migrating %d legacy snapshot(s) to CAS format...\n", len(legacyItems))
		for _, it := range legacyItems {
			tree := make(map[string]objects.FileEntry)
			err := filepath.Walk(it.path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if util.ShouldIgnore(filepath.Base(path), info.IsDir()) || info.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(it.path, path)
				relSlash := filepath.ToSlash(rel)

				hash, err := store.PutFile(path, "")
				if err != nil {
					return err
				}

				tree[relSlash] = objects.FileEntry{
					Hash: hash,
					Mode: info.Mode(),
					Size: info.Size(),
				}
				return nil
			})

			if err != nil {
				fmt.Printf("Warning: failed to process legacy snapshot %s: %v\n", it.id, err)
				continue
			}

			t, _ := time.Parse(time.RFC3339, it.createdAt)
			m := &manifest.Manifest{
				ID:        it.id,
				CreatedAt: t,
				Message:   it.message,
				Tree:      tree,
			}

			if err := manifest.Write(ekoDir, m); err != nil {
				return fmt.Errorf("failed to write manifest for %s: %w", it.id, err)
			}

			manifestPath := filepath.Join(ekoDir, "manifests", it.id+".json")
			_, _ = database.Exec("UPDATE snapshots SET path = ? WHERE id = ?", manifestPath, it.id)

			// Remove legacy full directory copy
			_ = os.RemoveAll(it.path)
			fmt.Printf("✔ Migrated snapshot %s\n", it.id)
		}

		fmt.Println("Migration complete! Legacy snapshots have been converted to CAS format.")
		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show what would be migrated without modifying files")
	rootCmd.AddCommand(migrateCmd)
}
