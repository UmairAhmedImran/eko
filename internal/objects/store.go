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
	"eko/internal/util"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/klauspost/compress/zstd"
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
	// Compatibility check: check if .zst or .raw exists, otherwise default to .gz
	prefix := filepath.Join(s.baseDir, hash[:2], hash[2:])
	for _, ext := range []string{".zst", ".raw"} {
		path := prefix + ext
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return prefix + ".gz"
}

// Exists reports whether a blob with this hash is already stored.
func (s *Store) Exists(hash string) bool {
	prefix := filepath.Join(s.baseDir, hash[:2], hash[2:])
	for _, ext := range []string{".zst", ".raw", ".gz"} {
		if _, err := os.Stat(prefix + ext); err == nil {
			return true
		}
	}
	return false
}

// Put compresses and stores data by its SHA-256 hash using zstd.
// If a blob with that hash already exists the call is a no-op (pure dedup).
// Returns the hex-encoded SHA-256 hash.
func (s *Store) Put(data []byte) (string, error) {
	return s.putWithCompression(data, shouldCompress(data))
}

func (s *Store) putWithCompression(data []byte, compress bool) (string, error) {
	hash := hashBytes(data)

	// Fast path: already stored — dedup hit, no I/O needed.
	if s.Exists(hash) {
		return hash, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring lock.
	if s.Exists(hash) {
		return hash, nil
	}

	ext := ".raw"
	if compress {
		ext = ".zst"
	}

	path := filepath.Join(s.baseDir, hash[:2], hash[2:]+ext)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("objects: mkdir prefix: %w", err)
	}

	// Atomic write: write to .tmp then rename.
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("objects: create tmp: %w", err)
	}

	if compress {
		var opts []zstd.EOption
		if len(data) < 1024*1024 {
			// Single-threaded encoding for files under 1MB to minimize overhead
			opts = append(opts, zstd.WithEncoderConcurrency(1))
		}
		encoder, err := zstd.NewWriter(f, opts...)
		if err != nil {
			f.Close()
			os.Remove(tmp)
			return "", err
		}
		if _, err := encoder.Write(data); err != nil {
			encoder.Close()
			f.Close()
			os.Remove(tmp)
			return "", fmt.Errorf("objects: compress: %w", err)
		}
		if err := encoder.Close(); err != nil {
			f.Close()
			os.Remove(tmp)
			return "", err
		}
	} else {
		if _, err := f.Write(data); err != nil {
			f.Close()
			os.Remove(tmp)
			return "", fmt.Errorf("objects: write raw: %w", err)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("objects: rename: %w", err)
	}

	// Make the blob immutable.
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
	compress := shouldCompress(data) && !hasCompressedExtension(filePath)
	return s.putWithCompression(data, compress)
}

func shouldCompress(data []byte) bool {
	if len(data) < 1024 {
		return false // skip compressing very small files
	}
	if len(data) >= 4 {
		// Gzip
		if data[0] == 0x1f && data[1] == 0x8b {
			return false
		}
		// Zip
		if data[0] == 0x50 && data[1] == 0x4b && data[2] == 0x03 && data[3] == 0x04 {
			return false
		}
		// PNG
		if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4e && data[3] == 0x47 {
			return false
		}
		// PDF
		if data[0] == 0x25 && data[1] == 0x50 && data[2] == 0x44 && data[3] == 0x46 {
			return false
		}
		// JPEG
		if data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
			return false
		}
		// Zstd
		if data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
			return false
		}
	}
	return true
}

func hasCompressedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".zip", ".tar", ".gz", ".zst", ".tgz", ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".mp4", ".mp3", ".dmg", ".exe", ".dll", ".so", ".dylib", ".rar", ".7z":
		return true
	}
	return false
}

// Get decompresses and returns the raw bytes for a stored hash.
func (s *Store) Get(hash string) ([]byte, error) {
	prefix := filepath.Join(s.baseDir, hash[:2], hash[2:])

	// 1. Try Zstd
	if f, err := os.Open(prefix + ".zst"); err == nil {
		defer f.Close()
		decoder, err := zstd.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("objects: zstd open %s: %w", hash[:8], err)
		}
		defer decoder.Close()
		return io.ReadAll(decoder)
	}

	// 2. Try Raw (uncompressed)
	if f, err := os.Open(prefix + ".raw"); err == nil {
		defer f.Close()
		return io.ReadAll(f)
	}

	// 3. Try Gzip (legacy fallback)
	if f, err := os.Open(prefix + ".gz"); err == nil {
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, fmt.Errorf("objects: gzip open %s: %w", hash[:8], err)
		}
		defer gz.Close()
		return io.ReadAll(gz)
	}

	return nil, fmt.Errorf("objects: open %s: file not found", hash[:8])
}

// ExtractTo writes the decompressed content of hash to dstPath with the given mode.
func (s *Store) ExtractTo(hash string, dstPath string, mode os.FileMode) error {
	prefix := filepath.Join(s.baseDir, hash[:2], hash[2:])

	// If the file is stored as a raw uncompressed file (.raw), we can use reflink/zero-copy copy!
	rawPath := prefix + ".raw"
	if _, err := os.Stat(rawPath); err == nil {
		err := util.CopyOrCloneFile(rawPath, dstPath)
		if err != nil {
			return err
		}
		return os.Chmod(dstPath, mode.Perm())
	}

	// Otherwise, fallback to reading and decompressing normally
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
// The optional onProgress callback is invoked after each file is extracted.
func (s *Store) RestoreTree(tree map[string]FileEntry, dstDir string, onProgress func()) error {
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
				if onProgress != nil {
					onProgress()
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
		dir := filepath.Base(filepath.Dir(path))
		name := info.Name()
		if len(name) > 4 && (name[len(name)-4:] == ".zst" || name[len(name)-4:] == ".raw") {
			hashes = append(hashes, dir+name[:len(name)-4])
		} else if len(name) > 3 && name[len(name)-3:] == ".gz" {
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
