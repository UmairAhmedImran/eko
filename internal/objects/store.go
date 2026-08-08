// Package objects implements a content-addressable storage (CAS) engine for Eko.
//
// Every file blob is stored exactly once, identified by its SHA-256 hash and
// compressed with gzip. Identical file content across any number of snapshots
// occupies only one entry in the object store, giving Git-like deduplication
// without the complexity of delta encoding.
//
// Layout inside .eko/objects/:
//
//	<2-char prefix>/
//	    <remaining 62 chars>.gz   ← gzip-compressed raw file bytes
//
// All objects are stored read-only (0444). Writes are atomic: the blob is
// written to a .tmp file first, then renamed into place — so a crashed write
// never leaves a corrupt object.
package objects

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const objectsSubdir = "objects"

// bufferPool reuses 32KB byte slices to minimize GC allocations during blob storage.
var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

// Store is a thread-safe content-addressable blob store.
type Store struct {
	baseDir string
	mu      sync.Mutex // guards concurrent Puts of the same hash
}

// New creates (or opens) the object store under ekoDir/objects.
func New(ekoDir string) (*Store, error) {
	base := filepath.Join(ekoDir, objectsSubdir)
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("objects: mkdir %s: %w", base, err)
	}
	return &Store{baseDir: base}, nil
}

// objectPath returns the on-disk path for a given SHA-256 hex hash.
func (s *Store) objectPath(hash string) string {
	return filepath.Join(s.baseDir, hash[:2], hash[2:]+".gz")
}

// Exists reports whether a blob with this hash is already stored.
func (s *Store) Exists(hash string) bool {
	_, err := os.Stat(s.objectPath(hash))
	return err == nil
}

// Put compresses and stores data by its SHA-256 hash using gzip.BestCompression.
// If a blob with that hash already exists the call is a no-op (pure dedup).
// Returns the hex-encoded SHA-256 hash.
func (s *Store) Put(data []byte) (string, error) {
	hash := hashBytes(data)

	// Fast path: already stored — dedup hit, no I/O needed.
	if s.Exists(hash) {
		return hash, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring lock (another goroutine may have stored it).
	if s.Exists(hash) {
		return hash, nil
	}

	path := s.objectPath(hash)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("objects: mkdir prefix: %w", err)
	}

	// Atomic write: write to .tmp then rename.
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("objects: create tmp: %w", err)
	}

	// Use BestCompression for maximum disk space savings
	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		f.Close()
		os.Remove(tmp)
		return "", fmt.Errorf("objects: compress: %w", err)
	}
	if err := gz.Close(); err != nil {
		f.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("objects: rename: %w", err)
	}

	// Make the blob immutable so it can never be accidentally overwritten.
	_ = os.Chmod(path, 0444)
	return hash, nil
}

// PutFile reads filePath, stores it in the object store, and returns its hash.
// If cachedHash is non-empty it is used directly (hash-cache hit: no file read).
func (s *Store) PutFile(filePath, cachedHash string) (string, error) {
	if cachedHash != "" && s.Exists(cachedHash) {
		return cachedHash, nil // full cache hit: nothing to do
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("objects: read %s: %w", filePath, err)
	}
	return s.Put(data)
}

// Get decompresses and returns the raw bytes for a stored hash.
func (s *Store) Get(hash string) ([]byte, error) {
	f, err := os.Open(s.objectPath(hash))
	if err != nil {
		return nil, fmt.Errorf("objects: open %s: %w", hash[:8], err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("objects: gzip open %s: %w", hash[:8], err)
	}
	defer gz.Close()

	data, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("objects: decompress %s: %w", hash[:8], err)
	}
	return data, nil
}

// ExtractTo writes the decompressed content of hash to dstPath with the given mode.
func (s *Store) ExtractTo(hash, dstPath string, mode os.FileMode) error {
	data, err := s.Get(hash)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, mode)
}

// RestoreTree extracts all files described by tree (path → FileEntry) into dstDir
// using a parallel worker pool for maximum throughput.
func (s *Store) RestoreTree(tree map[string]FileEntry, dstDir string) error {
	type job struct {
		rel  string
		hash string
		mode os.FileMode
	}

	numWorkers := runtime.NumCPU()
	jobs := make(chan job, numWorkers*2)
	errs := make(chan error, numWorkers)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				dst := filepath.Join(dstDir, filepath.FromSlash(j.rel))
				if err := s.ExtractTo(j.hash, dst, j.mode); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	for rel, entry := range tree {
		jobs <- job{rel: rel, hash: entry.Hash, mode: entry.Mode}
	}
	close(jobs)
	wg.Wait()
	close(errs)

	return <-errs
}

// AllHashes returns every hash currently stored (used by GC).
func (s *Store) AllHashes() ([]string, error) {
	var hashes []string
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		// filename = <62-char-suffix>.gz → full hash = prefix dir + suffix
		dir := filepath.Base(filepath.Dir(path))
		name := info.Name()
		if len(name) > 3 && name[len(name)-3:] == ".gz" {
			hashes = append(hashes, dir+name[:len(name)-3])
		}
		return nil
	})
	return hashes, err
}

// GarbageCollect deletes objects that are not referenced by any hash in keep.
// Returns the number of bytes freed.
func (s *Store) GarbageCollect(keep map[string]bool, dryRun bool) (int64, int, error) {
	all, err := s.AllHashes()
	if err != nil {
		return 0, 0, err
	}

	var freed int64
	var count int
	for _, h := range all {
		if keep[h] {
			continue
		}
		path := s.objectPath(h)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		freed += info.Size()
		count++
		if !dryRun {
			// Make writable before removing (objects stored 0444)
			_ = os.Chmod(path, 0644)
			_ = os.Remove(path)
		}
	}
	return freed, count, nil
}

// FileEntry is a reference to a stored blob. Used by RestoreTree and manifests.
type FileEntry struct {
	Hash string      `json:"hash"`
	Mode os.FileMode `json:"mode"`
	Size int64       `json:"size"`
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
