package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDiff(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	// 1. Write identical files
	if err := os.WriteFile(filepath.Join(dir1, "same.txt"), []byte("same content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "same.txt"), []byte("same content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Write modified files
	if err := os.WriteFile(filepath.Join(dir1, "diff.txt"), []byte("original content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "diff.txt"), []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Write file only in dir1 (deleted in dir2)
	if err := os.WriteFile(filepath.Join(dir1, "deleted.txt"), []byte("deleted content"), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Write file only in dir2 (added in dir2)
	if err := os.WriteFile(filepath.Join(dir2, "added.txt"), []byte("added content"), 0644); err != nil {
		t.Fatal(err)
	}

	diffs, err := BuildDiff(dir1, dir2)
	if err != nil {
		t.Fatalf("BuildDiff error: %v", err)
	}

	expectedDiffs := map[string]struct {
		original string
		modified string
	}{
		"diff.txt":    {original: "original content", modified: "modified content"},
		"deleted.txt": {original: "deleted content", modified: ""},
		"added.txt":   {original: "", modified: "added content"},
	}

	if len(diffs) != len(expectedDiffs) {
		t.Errorf("expected %d diffs, got %d", len(expectedDiffs), len(diffs))
	}

	for _, d := range diffs {
		exp, ok := expectedDiffs[d.Name]
		if !ok {
			t.Errorf("unexpected diff file: %s", d.Name)
			continue
		}
		if d.Original != exp.original {
			t.Errorf("%s Original: got %q, want %q", d.Name, d.Original, exp.original)
		}
		if d.Modified != exp.modified {
			t.Errorf("%s Modified: got %q, want %q", d.Name, d.Modified, exp.modified)
		}
	}
}
