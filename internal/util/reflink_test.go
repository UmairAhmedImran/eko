package util

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyOrCloneFile(t *testing.T) {
	tempDir := t.TempDir()

	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "destination.txt")

	content := []byte("Hello, reflink and standard copying!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Trigger Clone / Copy
	err := CopyOrCloneFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("CopyOrCloneFile failed: %v", err)
	}

	// Verify destination exists and has identical content
	retrieved, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if !bytes.Equal(content, retrieved) {
		t.Errorf("content mismatch: got %q, want %q", retrieved, content)
	}
}
