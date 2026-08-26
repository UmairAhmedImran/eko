package util

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// makeTree creates a temporary directory tree for walker tests:
//
//	root/
//	├── a.txt
//	├── skip/          (matched by shouldIgnore)
//	│   └── secret.txt
//	├── sub/
//	│   ├── b.txt
//	│   └── deep/
//	│       └── c.txt
//	└── d.txt
func makeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := []string{
		"a.txt",
		filepath.Join("skip", "secret.txt"),
		filepath.Join("sub", "b.txt"),
		filepath.Join("sub", "deep", "c.txt"),
		"d.txt",
	}
	for _, f := range files {
		full := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(f), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestWalkFiles_BasicDiscovery(t *testing.T) {
	root := makeTree(t)

	ignore := func(name string, isDir bool) bool {
		return name == "skip"
	}

	got, err := WalkFiles(root, ignore)
	if err != nil {
		t.Fatalf("WalkFiles error: %v", err)
	}

	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	sort.Strings(paths)

	want := []string{"a.txt", "d.txt", "sub/b.txt", "sub/deep/c.txt"}
	if len(paths) != len(want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
	for i, p := range paths {
		if p != want[i] {
			t.Errorf("paths[%d] = %q, want %q", i, p, want[i])
		}
	}
}

func TestWalkFiles_IgnoreDirectory(t *testing.T) {
	root := makeTree(t)

	// Ignoring "sub" should exclude sub/b.txt and sub/deep/c.txt.
	ignore := func(name string, isDir bool) bool {
		return name == "sub" || name == "skip"
	}

	got, err := WalkFiles(root, ignore)
	if err != nil {
		t.Fatalf("WalkFiles error: %v", err)
	}

	var paths []string
	for _, f := range got {
		paths = append(paths, f.Path)
	}
	sort.Strings(paths)

	want := []string{"a.txt", "d.txt"}
	if len(paths) != len(want) {
		t.Fatalf("got %v, want %v", paths, want)
	}
}

func TestWalkFiles_EmptyDirectory(t *testing.T) {
	root := t.TempDir()
	got, err := WalkFiles(root, func(string, bool) bool { return false })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no files, got %v", got)
	}
}

func TestWalkFiles_InfoIsPopulated(t *testing.T) {
	root := makeTree(t)
	ignore := func(name string, isDir bool) bool { return name == "skip" }

	got, err := WalkFiles(root, ignore)
	if err != nil {
		t.Fatalf("WalkFiles error: %v", err)
	}
	for _, f := range got {
		if f.Info == nil {
			t.Errorf("FileEntry.Info is nil for %s", f.Path)
		}
		if f.Path == "" {
			t.Error("FileEntry.Path is empty")
		}
	}
}
