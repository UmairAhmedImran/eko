package objects

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestStore_Zstd(t *testing.T) {
	tempDir := t.TempDir()
	store, err := New(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Large text content to trigger compression
	content := bytes.Repeat([]byte("This is some compressible text content. "), 100)
	hash, err := store.Put(content)
	if err != nil {
		t.Fatalf("failed to put data: %v", err)
	}

	// Verify that it is stored as .zst
	path := filepath.Join(tempDir, "objects", hash[:2], hash[2:]+".zst")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected zstd file to exist at %s, error: %v", path, err)
	}

	// Read and verify content
	retrieved, err := store.Get(hash)
	if err != nil {
		t.Fatalf("failed to get data: %v", err)
	}
	if !bytes.Equal(content, retrieved) {
		t.Errorf("content mismatch: got %q, want %q", retrieved, content)
	}
}

func TestStore_RawBinary(t *testing.T) {
	tempDir := t.TempDir()
	store, err := New(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// PNG Magic bytes to trigger raw storage
	pngContent := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x01, 0x02, 0x03}
	hash, err := store.Put(pngContent)
	if err != nil {
		t.Fatalf("failed to put raw data: %v", err)
	}

	// Verify that it is stored as .raw
	path := filepath.Join(tempDir, "objects", hash[:2], hash[2:]+".raw")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected raw file to exist at %s, error: %v", path, err)
	}

	// Read and verify content
	retrieved, err := store.Get(hash)
	if err != nil {
		t.Fatalf("failed to get raw data: %v", err)
	}
	if !bytes.Equal(pngContent, retrieved) {
		t.Errorf("content mismatch for raw data")
	}
}

func TestStore_LegacyGzipFallback(t *testing.T) {
	tempDir := t.TempDir()
	store, err := New(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("legacy gzip data content")
	hash := hashBytes(content)

	// Manually write a legacy .gz object
	path := filepath.Join(tempDir, "objects", hash[:2], hash[2:]+".gz")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if _, err := gz.Write(content); err != nil {
		t.Fatal(err)
	}
	gz.Close()
	f.Close()

	// Verify Exist works for legacy
	if !store.Exists(hash) {
		t.Error("expected Exists to return true for legacy .gz object")
	}

	// Verify Get works for legacy
	retrieved, err := store.Get(hash)
	if err != nil {
		t.Fatalf("failed to read legacy object: %v", err)
	}
	if !bytes.Equal(content, retrieved) {
		t.Errorf("legacy content mismatch: got %q", retrieved)
	}
}

func TestStore_AllHashes(t *testing.T) {
	tempDir := t.TempDir()
	store, err := New(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	// Put 1 zstd object
	zstdContent := bytes.Repeat([]byte("compress me please! "), 100)
	hash1, _ := store.Put(zstdContent)

	// Put 1 raw object
	rawContent := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00} // gzip header
	hash2, _ := store.Put(rawContent)

	hashes, err := store.AllHashes()
	if err != nil {
		t.Fatal(err)
	}

	if len(hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(hashes))
	}

	found1 := false
	found2 := false
	for _, h := range hashes {
		if h == hash1 {
			found1 = true
		}
		if h == hash2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("missing hashes in AllHashes: found1=%v, found2=%v", found1, found2)
	}
}
