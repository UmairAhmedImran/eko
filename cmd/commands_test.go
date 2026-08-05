package cmd

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"eko/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDir creates a temp directory, changes to it, and registers a cleanup.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInitCommand(t *testing.T) {
	dir := setupTestDir(t)

	// Run the init command
	_ = initCmd.RunE(initCmd, []string{})

	// Check that .eko/snapshots was created
	snapDir := filepath.Join(dir, ".eko", "snapshots")
	if info, err := os.Stat(snapDir); err != nil || !info.IsDir() {
		t.Fatalf("expected .eko/snapshots to be a directory, error: %v", err)
	}

	// Check that the database file was created
	dbFile := filepath.Join(dir, ".eko", "db.sqlite")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("expected .eko/db.sqlite to exist, error: %v", err)
	}

	// Open database and verify snapshots table exists
	database, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	var name string
	err = database.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='snapshots'").Scan(&name)
	if err != nil {
		t.Fatalf("expected snapshots table to exist: %v", err)
	}
}

func TestSaveCommand(t *testing.T) {
	dir := setupTestDir(t)

	// First initialize the project
	_ = initCmd.RunE(initCmd, []string{})

	// Create a dummy file to snapshot
	testFile := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run the save command
	if err := saveCmd.RunE(saveCmd, []string{}); err != nil {
		t.Fatal(err)
	}

	// Verify database record
	database := db.InitDB()
	defer database.Close()

	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM snapshots").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 snapshot record, got %d", count)
	}

	var id, message, path string
	err = database.QueryRow("SELECT id, message, path FROM snapshots LIMIT 1").Scan(&id, &message, &path)
	if err != nil {
		t.Fatal(err)
	}

	if id == "" {
		t.Error("expected non-empty snapshot ID")
	}
	if message != "snapshot" {
		t.Errorf("expected message 'snapshot', got %q", message)
	}

	// Verify files exist in the snapshot path
	snapFilePath := filepath.Join(dir, path, "hello.txt")
	if content, err := os.ReadFile(snapFilePath); err != nil || string(content) != "hello world" {
		t.Errorf("expected snapshot to contain 'hello world', got err=%v, content=%s", err, string(content))
	}
}

func TestHistoryCommand(t *testing.T) {
	dir := setupTestDir(t)

	// Initialize and create dummy file
	_ = initCmd.RunE(initCmd, []string{})
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save snapshot
	_ = saveCmd.RunE(saveCmd, []string{})

	// Setup capture of stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = historyCmd.RunE(historyCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("expected history output to contain snapshot entries, got empty string")
	}
}

func TestHistoryCommand_jsonOutput(t *testing.T) {
	dir := setupTestDir(t)

	_ = initCmd.RunE(initCmd, []string{})
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = saveCmd.RunE(saveCmd, []string{})

	if err := historyCmd.Flags().Set("json", "true"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		jsonOutput = false
		if err := historyCmd.Flags().Set("json", "false"); err != nil {
			t.Fatal(err)
		}
	}()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = historyCmd.RunE(historyCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var entries []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("expected valid JSON output, got %q: %v", buf.String(), err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if entries[0]["id"] == "" {
		t.Error("expected JSON entry to include id")
	}
	if entries[0]["created_at"] == "" {
		t.Error("expected JSON entry to include created_at")
	}
}

func TestRestoreCommand(t *testing.T) {
	dir := setupTestDir(t)

	// Initialize and save initial snapshot with hello.txt
	_ = initCmd.RunE(initCmd, []string{})
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello version 1"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = saveCmd.RunE(saveCmd, []string{})

	// Get snapshot ID
	database := db.InitDB()
	var id string
	err := database.QueryRow("SELECT id FROM snapshots LIMIT 1").Scan(&id)
	database.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Modify file
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello version 2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run restore
	_ = restoreCmd.RunE(restoreCmd, []string{id})

	// Check file restored to version 1
	content, err := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello version 1" {
		t.Errorf("expected content to be restored to 'hello version 1', got %q", string(content))
	}
}

func TestInitCommand_gitWarning(t *testing.T) {
	_ = setupTestDir(t)
	if err := os.Mkdir(".git", 0755); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	_ = initCmd.RunE(initCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Tip: A Git repository was detected") {
		t.Errorf("expected output to contain Git tip warning, got: %q", output)
	}
}

func TestSaveCommand_customMessage(t *testing.T) {
	dir := setupTestDir(t)
	_ = initCmd.RunE(initCmd, []string{})

	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set custom save message
	saveMessage = "custom test description"
	defer func() { saveMessage = "snapshot" }() // reset to default

	_ = saveCmd.RunE(saveCmd, []string{})

	database := db.InitDB()
	defer database.Close()

	var message string
	err := database.QueryRow("SELECT message FROM snapshots LIMIT 1").Scan(&message)
	if err != nil {
		t.Fatal(err)
	}
	if message != "custom test description" {
		t.Errorf("expected message 'custom test description', got %q", message)
	}
}

// --- clean command helpers ---

// withCleanFlags sets the clean flags for one test and restores the defaults.
func withCleanFlags(t *testing.T, keep int, dryRun bool) {
	t.Helper()
	cleanKeep, cleanDryRun = keep, dryRun
	t.Cleanup(func() { cleanKeep, cleanDryRun = 10, false })
}

// newLegacyProject creates .eko/snapshots and a database using the pre-summary
// schema, so clean is exercised against a database it must not migrate.
func newLegacyProject(t *testing.T) string {
	t.Helper()
	dir := setupTestDir(t)
	if err := os.MkdirAll(filepath.Join(".eko", "snapshots"), 0755); err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite3", filepath.Join(".eko", "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(`CREATE TABLE snapshots (
		id TEXT PRIMARY KEY,
		message TEXT,
		path TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
	return dir
}

// addSnapshotRow inserts a snapshot row with an explicit recorded path.
func addSnapshotRow(t *testing.T, id, path, createdAt string) {
	t.Helper()
	database, err := sql.Open("sqlite3", filepath.Join(".eko", "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.Exec(
		"INSERT INTO snapshots(id, message, path, created_at) VALUES (?, ?, ?, ?)",
		id, "snapshot", path, createdAt,
	); err != nil {
		t.Fatal(err)
	}
}

// addSnapshot inserts a well-formed snapshot row and creates its directory.
func addSnapshot(t *testing.T, id, createdAt string) {
	t.Helper()
	addSnapshotRow(t, id, ".eko/snapshots/"+id, createdAt)
	dir := filepath.Join(".eko", "snapshots", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte(id), 0644); err != nil {
		t.Fatal(err)
	}
}

// snapshotIDs returns the snapshot ids still recorded, newest first.
func snapshotIDs(t *testing.T) []string {
	t.Helper()
	database, err := sql.Open("sqlite3", filepath.Join(".eko", "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	rows, err := database.Query("SELECT id FROM snapshots ORDER BY created_at DESC, rowid DESC")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

// snapshotColumns returns the column names of the snapshots table.
func snapshotColumns(t *testing.T) []string {
	t.Helper()
	database, err := sql.Open("sqlite3", filepath.Join(".eko", "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	rows, err := database.Query("PRAGMA table_info(snapshots)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	cols := []string{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return cols
}

func hasColumn(cols []string, name string) bool {
	for _, c := range cols {
		if c == name {
			return true
		}
	}
	return false
}

// hashFile returns the SHA-256 of a file, used to prove byte-for-byte immutability.
func hashFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

// assertNotExist fails when path exists.
func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected %s not to exist, stat error was %v", path, err)
	}
}

// assertDirExists fails when path is not an existing directory.
func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", path)
	}
}

// --- clean command tests ---

func TestCleanCommand_flagDefaults(t *testing.T) {
	if got := cleanCmd.Flags().Lookup("keep").DefValue; got != "10" {
		t.Errorf("expected --keep to default to 10, got %q", got)
	}
	if got := cleanCmd.Flags().Lookup("dry-run").DefValue; got != "false" {
		t.Errorf("expected --dry-run to default to false, got %q", got)
	}
}

// A missing database must produce an explicit error. Neither open path may
// create the database or its WAL/SHM sidecars before validation.
func TestCleanCommand_missingDatabaseCreatesNothing(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		name := "normal"
		if dryRun {
			name = "dry-run"
		}
		t.Run(name, func(t *testing.T) {
			setupTestDir(t)
			// Marker-only project: .eko exists, the database does not.
			if err := os.MkdirAll(filepath.Join(".eko", "snapshots"), 0755); err != nil {
				t.Fatal(err)
			}
			withCleanFlags(t, 1, dryRun)

			err := cleanCmd.RunE(cleanCmd, []string{})
			if err == nil {
				t.Fatal("expected an explicit missing-database error, got nil")
			}
			if !strings.Contains(err.Error(), "no eko database found") {
				t.Fatalf("expected a missing-database error, got %v", err)
			}

			assertNotExist(t, filepath.Join(".eko", "db.sqlite"))
			assertNotExist(t, filepath.Join(".eko", "db.sqlite-wal"))
			assertNotExist(t, filepath.Join(".eko", "db.sqlite-shm"))
		})
	}
}

// A dry run must not change one byte of the database, even on a legacy schema
// that InitDB would have migrated on open.
func TestCleanCommand_dryRunLeavesLegacyDatabaseByteIdentical(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-01 00:00:00")
	addSnapshot(t, "bbbbbbbb", "2026-01-02 00:00:00")
	addSnapshot(t, "cccccccc", "2026-01-03 00:00:00")

	dbPath := filepath.Join(".eko", "db.sqlite")
	before := hashFile(t, dbPath)

	withCleanFlags(t, 1, true)
	if err := cleanCmd.RunE(cleanCmd, []string{}); err != nil {
		t.Fatal(err)
	}

	if after := hashFile(t, dbPath); after != before {
		t.Errorf("dry run changed the database: %s -> %s", before, after)
	}
	assertNotExist(t, filepath.Join(".eko", "db.sqlite-wal"))
	assertNotExist(t, filepath.Join(".eko", "db.sqlite-shm"))

	for _, id := range []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"} {
		assertDirExists(t, filepath.Join(".eko", "snapshots", id))
	}
	if got := strings.Join(snapshotIDs(t), ","); got != "cccccccc,bbbbbbbb,aaaaaaaa" {
		t.Errorf("dry run changed the snapshot rows, got %v", got)
	}
	if cols := snapshotColumns(t); hasColumn(cols, "summary") {
		t.Errorf("dry run migrated the legacy schema: %v", cols)
	}
}

// A normal run with nothing to remove must also leave a legacy schema alone.
func TestCleanCommand_normalRunDoesNotMigrateLegacySchema(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-01 00:00:00")

	dbPath := filepath.Join(".eko", "db.sqlite")
	before := hashFile(t, dbPath)

	withCleanFlags(t, 10, false)
	if err := cleanCmd.RunE(cleanCmd, []string{}); err != nil {
		t.Fatal(err)
	}

	if after := hashFile(t, dbPath); after != before {
		t.Errorf("a run with no candidates changed the database: %s -> %s", before, after)
	}
	cols := snapshotColumns(t)
	if hasColumn(cols, "summary") {
		t.Errorf("clean added the summary column to a legacy database: %v", cols)
	}
	if strings.Join(cols, ",") != "id,message,path,created_at" {
		t.Errorf("expected the legacy schema to be preserved, got %v", cols)
	}
	assertDirExists(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
}

// Snapshots sharing a created_at must still be ordered deterministically, so
// the same keep count always removes the same snapshots.
func TestCleanCommand_deterministicOrder(t *testing.T) {
	newLegacyProject(t)
	for _, id := range []string{"aaaaaaaa", "bbbbbbbb", "cccccccc", "dddddddd"} {
		addSnapshot(t, id, "2026-01-01 00:00:00")
	}

	withCleanFlags(t, 2, false)
	if err := cleanCmd.RunE(cleanCmd, []string{}); err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(snapshotIDs(t), ","); got != "dddddddd,cccccccc" {
		t.Errorf("expected the two newest rows to survive, got %v", got)
	}
	assertDirExists(t, filepath.Join(".eko", "snapshots", "cccccccc"))
	assertDirExists(t, filepath.Join(".eko", "snapshots", "dddddddd"))
	assertNotExist(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	assertNotExist(t, filepath.Join(".eko", "snapshots", "bbbbbbbb"))
}

func TestCleanCommand_keepZeroRemovesEverything(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-01 00:00:00")
	addSnapshot(t, "bbbbbbbb", "2026-01-02 00:00:00")

	withCleanFlags(t, 0, false)
	if err := cleanCmd.RunE(cleanCmd, []string{}); err != nil {
		t.Fatal(err)
	}

	if got := snapshotIDs(t); len(got) != 0 {
		t.Errorf("expected every snapshot row to be removed, got %v", got)
	}
	assertNotExist(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	assertNotExist(t, filepath.Join(".eko", "snapshots", "bbbbbbbb"))
	assertDirExists(t, filepath.Join(".eko", "snapshots"))
}

func TestCleanCommand_rejectsNegativeKeep(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-01 00:00:00")

	withCleanFlags(t, -1, false)
	err := cleanCmd.RunE(cleanCmd, []string{})
	if err == nil {
		t.Fatal("expected a negative --keep to be rejected")
	}
	if !strings.Contains(err.Error(), "--keep must be zero or greater") {
		t.Fatalf("expected a --keep validation error, got %v", err)
	}
	assertDirExists(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	if got := len(snapshotIDs(t)); got != 1 {
		t.Errorf("expected the snapshot row to survive, got %d rows", got)
	}
}

// Every candidate is validated before any of them is removed, so one bad row
// stops the whole run with nothing deleted.
func TestCleanCommand_pathIDMismatchAbortsBeforeAnyDeletion(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-02 00:00:00")
	// Older, so it is validated after the well-formed candidate: its recorded
	// path belongs to a different snapshot id.
	addSnapshotRow(t, "bbbbbbbb", ".eko/snapshots/aaaaaaaa", "2026-01-01 00:00:00")

	withCleanFlags(t, 0, false)
	err := cleanCmd.RunE(cleanCmd, []string{})
	if err == nil {
		t.Fatal("expected the mismatched path to abort the run")
	}
	if !strings.Contains(err.Error(), "is not") {
		t.Fatalf("expected a recorded-path mismatch error, got %v", err)
	}

	assertDirExists(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	if got := len(snapshotIDs(t)); got != 2 {
		t.Errorf("expected both snapshot rows to survive, got %d rows", got)
	}
}

func TestCleanCommand_rejectsMissingSnapshotDirectory(t *testing.T) {
	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-02 00:00:00")
	addSnapshotRow(t, "bbbbbbbb", ".eko/snapshots/bbbbbbbb", "2026-01-01 00:00:00")

	withCleanFlags(t, 0, false)
	err := cleanCmd.RunE(cleanCmd, []string{})
	if err == nil {
		t.Fatal("expected a missing snapshot directory to abort the run")
	}
	if !strings.Contains(err.Error(), "cannot resolve") {
		t.Fatalf("expected an unresolvable-path error, got %v", err)
	}
	assertDirExists(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	if got := len(snapshotIDs(t)); got != 2 {
		t.Errorf("expected both snapshot rows to survive, got %d rows", got)
	}
}

func TestCleanCommand_rejectsNonDirectorySnapshot(t *testing.T) {
	newLegacyProject(t)
	addSnapshotRow(t, "aaaaaaaa", ".eko/snapshots/aaaaaaaa", "2026-01-01 00:00:00")
	if err := os.WriteFile(filepath.Join(".eko", "snapshots", "aaaaaaaa"), []byte("not a directory"), 0644); err != nil {
		t.Fatal(err)
	}

	withCleanFlags(t, 0, false)
	err := cleanCmd.RunE(cleanCmd, []string{})
	if err == nil {
		t.Fatal("expected a non-directory snapshot path to abort the run")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("expected a non-directory error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(".eko", "snapshots", "aaaaaaaa")); statErr != nil {
		t.Errorf("expected the file to survive: %v", statErr)
	}
}

// A snapshot directory that is a symlink out of .eko/snapshots must never be
// followed, and a symlink to the snapshots directory itself must be refused.
func TestCleanCommand_rejectsSymlinkedSnapshotPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs elevated privileges on Windows")
	}

	t.Run("escapes the snapshots directory", func(t *testing.T) {
		dir := newLegacyProject(t)
		outside := filepath.Join(dir, "outside")
		if err := os.MkdirAll(outside, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outside, "keep.txt"), []byte("keep"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(".eko", "snapshots", "aaaaaaaa")); err != nil {
			t.Fatal(err)
		}
		addSnapshotRow(t, "aaaaaaaa", ".eko/snapshots/aaaaaaaa", "2026-01-01 00:00:00")

		withCleanFlags(t, 0, false)
		err := cleanCmd.RunE(cleanCmd, []string{})
		if err == nil {
			t.Fatal("expected an escaping symlink to abort the run")
		}
		if !strings.Contains(err.Error(), "resolves outside") {
			t.Fatalf("expected a containment error, got %v", err)
		}
		assertDirExists(t, outside)
		if _, statErr := os.Stat(filepath.Join(outside, "keep.txt")); statErr != nil {
			t.Errorf("expected the outside file to survive: %v", statErr)
		}
	})

	t.Run("targets the snapshots directory", func(t *testing.T) {
		dir := newLegacyProject(t)
		root := filepath.Join(dir, ".eko", "snapshots")
		if err := os.Symlink(root, filepath.Join(".eko", "snapshots", "aaaaaaaa")); err != nil {
			t.Fatal(err)
		}
		addSnapshotRow(t, "aaaaaaaa", ".eko/snapshots/aaaaaaaa", "2026-01-01 00:00:00")

		withCleanFlags(t, 0, false)
		err := cleanCmd.RunE(cleanCmd, []string{})
		if err == nil {
			t.Fatal("expected a snapshots-root symlink to abort the run")
		}
		if !strings.Contains(err.Error(), "snapshots directory itself") {
			t.Fatalf("expected a snapshots-root error, got %v", err)
		}
		assertDirExists(t, root)
	})

	// An alias that stays inside the snapshots directory passes every
	// containment check above: it resolves to a real direct child of the root.
	// Only the row id ties a candidate to the directory that is about to be
	// removed. Here the alias points at the one snapshot --keep preserves, so
	// without that check clean deletes a kept snapshot off disk and leaves its
	// row behind pointing at nothing.
	t.Run("aliases another snapshot inside the snapshots directory", func(t *testing.T) {
		dir := newLegacyProject(t)
		addSnapshot(t, "cccccccc", "2026-01-03 00:00:00") // newest, kept by --keep 1
		alias := filepath.Join(".eko", "snapshots", "aaaaaaaa")
		if err := os.Symlink(filepath.Join(dir, ".eko", "snapshots", "cccccccc"), alias); err != nil {
			t.Fatal(err)
		}
		// Oldest, so it is the only candidate, and it resolves to the kept one.
		addSnapshotRow(t, "aaaaaaaa", ".eko/snapshots/aaaaaaaa", "2026-01-01 00:00:00")

		dbPath := filepath.Join(".eko", "db.sqlite")
		before := hashFile(t, dbPath)

		withCleanFlags(t, 1, false)
		err := cleanCmd.RunE(cleanCmd, []string{})
		if err == nil {
			t.Fatal("expected an in-root snapshot alias to abort the run")
		}
		if !strings.Contains(err.Error(), "resolves to a different snapshot") {
			t.Fatalf("expected an alias error, got %v", err)
		}

		// Validation completes before the first deletion, so the kept snapshot,
		// the alias itself and every row must be untouched.
		assertDirExists(t, filepath.Join(".eko", "snapshots", "cccccccc"))
		content, readErr := os.ReadFile(filepath.Join(".eko", "snapshots", "cccccccc", "hello.txt"))
		if readErr != nil || string(content) != "cccccccc" {
			t.Errorf("expected the kept snapshot to be intact, got err=%v content=%q", readErr, string(content))
		}
		if _, lstatErr := os.Lstat(alias); lstatErr != nil {
			t.Errorf("expected the alias itself to survive: %v", lstatErr)
		}
		if after := hashFile(t, dbPath); after != before {
			t.Errorf("the rejected run changed the database: %s -> %s", before, after)
		}
		if got := strings.Join(snapshotIDs(t), ","); got != "cccccccc,aaaaaaaa" {
			t.Errorf("expected both snapshot rows to survive, got %v", got)
		}
	})
}

// Deletion is not atomic: the run stops at the first failure, keeps what it
// already removed, and reports exactly how far it got.
func TestCleanCommand_nonAtomicFailureReportsProgress(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not block removal on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	newLegacyProject(t)
	addSnapshot(t, "aaaaaaaa", "2026-01-01 00:00:00") // oldest, removed last
	addSnapshot(t, "bbbbbbbb", "2026-01-02 00:00:00")
	addSnapshot(t, "cccccccc", "2026-01-03 00:00:00") // newest, kept

	// RemoveAll cannot unlink a file inside a directory it may not write to.
	locked := filepath.Join(".eko", "snapshots", "aaaaaaaa", "locked")
	if err := os.MkdirAll(locked, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "pinned.txt"), []byte("pinned"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0755) })

	withCleanFlags(t, 1, false)
	err := cleanCmd.RunE(cleanCmd, []string{})
	if err == nil {
		t.Fatal("expected the blocked removal to fail the run")
	}
	if !strings.Contains(err.Error(), "removed 1 of 2") {
		t.Fatalf("expected the error to report partial progress, got %v", err)
	}

	assertNotExist(t, filepath.Join(".eko", "snapshots", "bbbbbbbb"))
	assertDirExists(t, filepath.Join(".eko", "snapshots", "aaaaaaaa"))
	if got := strings.Join(snapshotIDs(t), ","); got != "cccccccc,aaaaaaaa" {
		t.Errorf("expected only the removed snapshot's row to be gone, got %v", got)
	}
}
