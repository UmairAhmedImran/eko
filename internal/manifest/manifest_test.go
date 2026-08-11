package manifest_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"eko/internal/manifest"
	"eko/internal/objects"
)

func TestLRUCache(t *testing.T) {
	cache := manifest.NewCache(2)

	m1 := &manifest.Manifest{ID: "snap-1", Message: "First"}
	m2 := &manifest.Manifest{ID: "snap-2", Message: "Second"}
	m3 := &manifest.Manifest{ID: "snap-3", Message: "Third"}

	cache.Put(m1)
	cache.Put(m2)

	if cache.Len() != 2 {
		t.Fatalf("expected cache length 2, got %d", cache.Len())
	}

	// Access m1 to make it MRU
	if _, ok := cache.Get("snap-1"); !ok {
		t.Fatalf("expected snap-1 in cache")
	}

	// Insert m3, which should evict m2 (LRU)
	cache.Put(m3)

	if _, ok := cache.Get("snap-2"); ok {
		t.Fatalf("expected snap-2 to be evicted")
	}
	if _, ok := cache.Get("snap-1"); !ok {
		t.Fatalf("expected snap-1 to still be in cache")
	}
	if _, ok := cache.Get("snap-3"); !ok {
		t.Fatalf("expected snap-3 to be in cache")
	}

	// Invalidate snap-1
	cache.Invalidate("snap-1")
	if _, ok := cache.Get("snap-1"); ok {
		t.Fatalf("expected snap-1 to be invalidated")
	}
	if cache.Len() != 1 {
		t.Fatalf("expected cache length 1, got %d", cache.Len())
	}
}

func TestManifestReadWriteDeleteCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eko-manifest-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := &manifest.Manifest{
		ID:        "test-snap-100",
		CreatedAt: time.Now(),
		Message:   "Cache test snapshot",
		Tree: map[string]objects.FileEntry{
			"main.go": {Hash: "abc123hash", Size: 100, Mode: 0644},
		},
	}

	// Write should populate the cache
	if err := manifest.Write(tmpDir, m); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Delete file directly on disk (without calling manifest.Delete) to ensure Read returns from cache
	path := filepath.Join(tmpDir, "manifests", m.ID+".json")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file error: %v", err)
	}

	readM, err := manifest.Read(tmpDir, m.ID)
	if err != nil {
		t.Fatalf("read error (should hit cache): %v", err)
	}
	if readM.Message != m.Message {
		t.Fatalf("expected message %q, got %q", m.Message, readM.Message)
	}

	// Call Delete which invalidates cache
	// Re-create dummy file first so os.Remove doesn't fail
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("write dummy file error: %v", err)
	}
	if err := manifest.Delete(tmpDir, m.ID); err != nil {
		t.Fatalf("delete error: %v", err)
	}

	// Now Read should fail since cache was invalidated and file was removed
	if _, err := manifest.Read(tmpDir, m.ID); err == nil {
		t.Fatalf("expected read error after delete, got nil")
	}
}

func BenchmarkManifestReadWithCache(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "eko-manifest-bench-*")
	if err != nil {
		b.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	m := &manifest.Manifest{
		ID:        "bench-snap",
		CreatedAt: time.Now(),
		Message:   "Benchmark snapshot",
		Tree: make(map[string]objects.FileEntry),
	}
	for i := 0; i < 500; i++ {
		m.Tree[fmt.Sprintf("file_%d.txt", i)] = objects.FileEntry{Hash: "hash", Size: 1234, Mode: 0644}
	}
	if err := manifest.Write(tmpDir, m); err != nil {
		b.Fatalf("write error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := manifest.Read(tmpDir, m.ID); err != nil {
			b.Fatalf("read error: %v", err)
		}
	}
}
