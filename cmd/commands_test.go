package cmd

import (
	"bytes"
	"database/sql"
	"eko/internal/db"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	initCmd.Run(initCmd, []string{})

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
	initCmd.Run(initCmd, []string{})

	// Create a dummy file to snapshot
	testFile := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run the save command
	saveCmd.Run(saveCmd, []string{})

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
	initCmd.Run(initCmd, []string{})
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save snapshot
	saveCmd.Run(saveCmd, []string{})

	// Setup capture of stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	historyCmd.Run(historyCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if len(output) == 0 {
		t.Error("expected history output to contain snapshot entries, got empty string")
	}
}

func TestRestoreCommand(t *testing.T) {
	dir := setupTestDir(t)

	// Initialize and save initial snapshot with hello.txt
	initCmd.Run(initCmd, []string{})
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello version 1"), 0644); err != nil {
		t.Fatal(err)
	}
	saveCmd.Run(saveCmd, []string{})

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
	restoreCmd.Run(restoreCmd, []string{id})

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

	initCmd.Run(initCmd, []string{})

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
	initCmd.Run(initCmd, []string{})

	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Set custom save message
	saveMessage = "custom test description"
	defer func() { saveMessage = "snapshot" }() // reset to default

	saveCmd.Run(saveCmd, []string{})

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
