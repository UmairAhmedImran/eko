package snapshot

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"eko/internal/db"
	"eko/internal/manifest"
)

// chdir changes the working directory for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
}

// setupProject creates a temp project directory with a .eko subtree
// and some source files, then chdirs into it.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".eko"), 0755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# project"), 0644)
	chdir(t, dir)
	return dir
}

func TestGenerateID_length(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID error: %v", err)
	}
	if len(id) != 8 {
		t.Errorf("expected 8-char hex id, got %q (len=%d)", id, len(id))
	}
}

func TestGenerateID_hex(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID error: %v", err)
	}
	for _, c := range id {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("expected hex char, got %c", c)
		}
	}
}

func TestCreateSnapshot_CAS(t *testing.T) {
	setupProject(t)
	database := db.InitDB()
	defer database.Close()

	id, path, err := CreateSnapshot(database, false, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if id == "" {
		t.Errorf("expected non-empty id")
	}

	if !manifest.Exists(".eko", id) {
		t.Errorf("expected manifest to exist for id %s at %s", id, path)
	}

	m, err := manifest.Read(".eko", id)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}

	if _, ok := m.Tree["main.go"]; !ok {
		t.Errorf("expected main.go in manifest tree")
	}
}

func TestRestoreSnapshot_CAS(t *testing.T) {
	dir := setupProject(t)
	database := db.InitDB()
	defer database.Close()

	_, path, err := CreateSnapshot(database, false, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	// Mutate working directory
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main // modified"), 0644)
	os.WriteFile(filepath.Join(dir, "newfile.go"), []byte("package newfile"), 0644)

	// Restore
	if err := RestoreSnapshot(path, nil); err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}

	// Verify main.go reverted and newfile.go deleted
	content, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if string(content) != "package main" {
		t.Errorf("expected 'package main', got %q", string(content))
	}

	if _, err := os.Stat(filepath.Join(dir, "newfile.go")); !os.IsNotExist(err) {
		t.Errorf("expected newfile.go to be removed after restore")
	}
}

func TestCreateSnapshot_EnvOptIn(t *testing.T) {
	dir := setupProject(t)
	database := db.InitDB()
	defer database.Close()

	// Set a custom environment variable
	t.Setenv("TEST_EKO_ENV_VAR", "eko_test_val")

	// 1. Save WITHOUT env (default)
	idNoEnv, pathNoEnv, err := CreateSnapshot(database, false, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	mNoEnv, err := manifest.Read(".eko", idNoEnv)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if mNoEnv.EnvHash != "" {
		t.Errorf("expected empty EnvHash when calling CreateSnapshot without env capture")
	}

	// Verify restore does NOT create .eko_env_restore.sh
	if err := RestoreSnapshot(pathNoEnv, nil); err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".eko_env_restore.sh")); !os.IsNotExist(err) {
		t.Errorf("expected .eko_env_restore.sh not to be created when env capture is disabled")
	}

	// 2. Save WITH env
	idWithEnv, pathWithEnv, err := CreateSnapshot(database, true, nil)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	mWithEnv, err := manifest.Read(".eko", idWithEnv)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if mWithEnv.EnvHash == "" {
		t.Errorf("expected non-empty EnvHash when calling CreateSnapshot with env capture")
	}

	// Verify restore DOES create .eko_env_restore.sh with 0600 permissions
	if err := RestoreSnapshot(pathWithEnv, nil); err != nil {
		t.Fatalf("RestoreSnapshot failed: %v", err)
	}
	restoreScriptPath := filepath.Join(dir, ".eko_env_restore.sh")
	info, err := os.Stat(restoreScriptPath)
	if err != nil {
		t.Fatalf(".eko_env_restore.sh was not created: %v", err)
	}

	// Check permissions on Unix/macOS
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("expected file permissions 0600, got %o", perm)
		}
	}

	// Check content
	content, err := os.ReadFile(restoreScriptPath)
	if err != nil {
		t.Fatalf("failed to read .eko_env_restore.sh: %v", err)
	}
	if !strings.Contains(string(content), "export TEST_EKO_ENV_VAR='eko_test_val'") {
		t.Errorf("expected env restore script to contain TEST_EKO_ENV_VAR, got:\n%s", string(content))
	}
}
