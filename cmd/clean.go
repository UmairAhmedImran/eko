package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

const (
	// cleanSnapshotsDir is the only directory clean is ever allowed to delete from.
	cleanSnapshotsDir = ekoDir + "/snapshots"

	// cleanDBPath is the project database, relative to the project root.
	cleanDBPath = ekoDir + "/db.sqlite"

	// cleanNormalDSN opens the existing database read-write. mode=rw is
	// deliberate: SQLite's default rwc mode would create a database when one is
	// missing, so running clean from the wrong directory would leave a stray
	// db.sqlite behind instead of failing.
	cleanNormalDSN = "file:" + cleanDBPath + "?mode=rw"

	// cleanDryRunDSN opens the database read-only and rejects writes at the
	// connection level, so a dry run cannot change a single byte.
	cleanDryRunDSN = "file:" + cleanDBPath + "?mode=ro&_query_only=true"
)

var (
	cleanKeep   int
	cleanDryRun bool
)

// cleanCandidate is one snapshot selected for deletion. dir is only set once
// the recorded path has been validated against the snapshots directory.
//
// missing marks a row whose directory is already gone. That happens when a
// previous run removed the directory and then failed before deleting the row,
// so the row is stale rather than suspicious: there is nothing left to delete
// on disk and only the database still has to be cleaned up.
type cleanCandidate struct {
	id        string
	path      string
	createdAt string
	dir       string
	missing   bool
}

// openCleanDB opens the project database directly, without running migrations.
// clean must not rewrite an older schema: a dry run has to be byte-for-byte
// inert, and a normal run must validate every deletion candidate before it is
// allowed to touch anything.
func openCleanDB(readOnly bool) (*sql.DB, error) {
	dsn := cleanNormalDSN
	if readOnly {
		dsn = cleanDryRunDSN
	}

	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", cleanDBPath, err)
	}

	// sql.Open is lazy, so Ping is what actually surfaces a missing database.
	if err := database.Ping(); err != nil {
		database.Close()
		if _, statErr := os.Stat(cleanDBPath); errors.Is(statErr, os.ErrNotExist) {
			return nil, fmt.Errorf("no eko database found at %s\nrun 'eko init' to initialize eko in this directory", cleanDBPath)
		}
		return nil, fmt.Errorf("failed to open %s: %w", cleanDBPath, err)
	}

	return database, nil
}

// loadCleanCandidates returns the snapshots older than the newest keep ones,
// plus the total number of snapshots recorded.
//
// Only id, path and created_at are read, so clean behaves identically against
// databases written before the summary column existed.
func loadCleanCandidates(database *sql.DB, keep int) ([]cleanCandidate, int, error) {
	rows, err := database.Query("SELECT id, path, created_at FROM snapshots ORDER BY created_at DESC, rowid DESC")
	if err != nil {
		return nil, 0, fmt.Errorf("error reading snapshots: %w", err)
	}
	defer rows.Close()

	var candidates []cleanCandidate
	total := 0
	for rows.Next() {
		var id string
		var path, createdAt sql.NullString
		if err := rows.Scan(&id, &path, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("error reading snapshots: %w", err)
		}
		total++
		if total <= keep {
			continue
		}
		candidates = append(candidates, cleanCandidate{
			id:        id,
			path:      path.String,
			createdAt: createdAt.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating snapshot rows: %w", err)
	}

	return candidates, total, nil
}

// resolveCleanTargets validates every candidate before any of them is removed.
// One bad row aborts the whole run: a partially trusted delete list is how a
// clean command turns into an accidental rm -rf.
func resolveCleanTargets(candidates []cleanCandidate) ([]cleanCandidate, error) {
	root, err := filepath.Abs(cleanSnapshotsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", cleanSnapshotsDir, err)
	}
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", cleanSnapshotsDir, err)
	}

	targets := make([]cleanCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.id == "" {
			return nil, errors.New("refusing to clean: found a snapshot row with an empty id")
		}

		// The recorded path must be exactly the directory this id owns.
		want := filepath.Join(cleanSnapshotsDir, c.id)
		if c.path == "" || filepath.Clean(c.path) != want {
			return nil, fmt.Errorf("refusing to clean snapshot %s: recorded path %q is not %q", c.id, c.path, want)
		}

		abs, err := filepath.Abs(c.path)
		if err != nil {
			return nil, fmt.Errorf("refusing to clean snapshot %s: %w", c.id, err)
		}

		// A row whose directory no longer exists is the one recoverable case.
		// The path has already been checked to be exactly the directory this id
		// owns, so nothing here can widen what clean is allowed to delete: there
		// is simply nothing on disk left to delete. Recovering instead of
		// aborting is what lets a run that failed between the disk delete and
		// the row delete be finished by the next run.
		//
		// Lstat, not Stat, so a dangling symlink is still treated as an
		// unexpected entry and aborts the run below.
		if _, statErr := os.Lstat(abs); errors.Is(statErr, os.ErrNotExist) {
			c.missing = true
			targets = append(targets, c)
			continue
		}

		// EvalSymlinks resolves every link in the path, so an alias cannot make
		// a row point at a directory clean is not allowed to delete.
		resolved, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("refusing to clean snapshot %s: cannot resolve %s: %w", c.id, c.path, err)
		}

		info, err := os.Stat(resolved)
		if err != nil {
			return nil, fmt.Errorf("refusing to clean snapshot %s: cannot inspect %s: %w", c.id, c.path, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("refusing to clean snapshot %s: %s is not a directory", c.id, c.path)
		}
		if resolved == rootReal {
			return nil, fmt.Errorf("refusing to clean snapshot %s: %s resolves to the snapshots directory itself", c.id, c.path)
		}
		// Direct child only. Any other resolved parent means the row points
		// somewhere clean is not allowed to delete from.
		if filepath.Dir(resolved) != rootReal {
			return nil, fmt.Errorf("refusing to clean snapshot %s: %s resolves outside %s", c.id, c.path, cleanSnapshotsDir)
		}
		// The resolved directory must be the one this id owns. An alias that
		// stays inside the snapshots directory satisfies every check above
		// while pointing at a different snapshot, so the name is what ties the
		// row to the directory that is actually about to be deleted.
		if filepath.Base(resolved) != c.id {
			return nil, fmt.Errorf("refusing to clean snapshot %s: %s resolves to a different snapshot %q", c.id, c.path, filepath.Base(resolved))
		}

		c.dir = resolved
		targets = append(targets, c)
	}

	return targets, nil
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old snapshots",
	Long: `Clean removes old snapshots and frees the disk space they use.

Snapshots are ordered newest first. The newest --keep snapshots are kept and
every older one is removed from .eko/snapshots and from the database. Every
snapshot to be removed is validated first, and a single unexpected path aborts
the run before anything is deleted.

Use --dry-run to see exactly what would be removed; a dry run opens the
database read-only and changes nothing.`,
	Example: `  # Keep the 5 newest snapshots and remove the rest
  eko clean --keep 5

  # Show what would be removed, without removing anything
  eko clean --keep 5 --dry-run`,
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cleanKeep < 0 {
			return fmt.Errorf("--keep must be zero or greater, got %d", cleanKeep)
		}

		database, err := openCleanDB(cleanDryRun)
		if err != nil {
			return err
		}
		defer database.Close()

		candidates, total, err := loadCleanCandidates(database, cleanKeep)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			fmt.Printf("Nothing to clean: %d snapshot(s), keeping up to %d.\n", total, cleanKeep)
			return nil
		}

		targets, err := resolveCleanTargets(candidates)
		if err != nil {
			return err
		}

		if cleanDryRun {
			fmt.Printf("Dry run: %d of %d snapshot(s) would be removed, nothing was changed.\n", len(targets), total)
			for _, t := range targets {
				if t.missing {
					fmt.Printf("  %s %s %s (already gone; only the database row would be removed)\n", t.id, t.createdAt, t.path)
					continue
				}
				fmt.Printf("  %s %s %s\n", t.id, t.createdAt, t.path)
			}
			return nil
		}

		// Deletion is intentionally not atomic. Each snapshot is removed from
		// disk and then from the database, and the first failure stops the run;
		// snapshots already removed stay removed, so the error reports exactly
		// how far the run got and the next run can continue from there.
		//
		// Continuing works because a row whose directory is already gone is
		// resolved as missing rather than treated as an error, so a run that
		// failed between the two deletes leaves a state the next run can finish.
		for i, t := range targets {
			// A missing directory has nothing to remove; only the stale row is
			// left, and deleting it below is what completes the earlier run.
			if !t.missing {
				if err := os.RemoveAll(t.dir); err != nil {
					return fmt.Errorf("removed %d of %d snapshot(s); failed to remove %s: %w", i, len(targets), t.path, err)
				}
			}
			if _, err := database.Exec("DELETE FROM snapshots WHERE id = ?", t.id); err != nil {
				// The snapshot at i is already off disk at this point, so the
				// count has to include it.
				return fmt.Errorf("removed %d of %d snapshot(s); removed %s from disk but could not delete its database row: %w", i+1, len(targets), t.path, err)
			}
			fmt.Println("Removed:", t.id)
		}

		fmt.Printf("Removed %d snapshot(s), kept %d.\n", len(targets), total-len(targets))
		return nil
	},
}

func init() {
	cleanCmd.Flags().IntVar(&cleanKeep, "keep", 10, "number of most recent snapshots to keep")
	cleanCmd.Flags().BoolVar(&cleanDryRun, "dry-run", false, "show what would be removed without removing anything")
	rootCmd.AddCommand(cleanCmd)
}
