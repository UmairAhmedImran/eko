// Package objects implements a content-addressable storage (CAS) engine for Eko.
//
// Every file blob is stored exactly once, identified by its SHA-256 hash.
// Objects are compressed with zstd when beneficial, while already-compressed
// and small files are stored raw. Legacy gzip objects remain readable.
//
// Layout inside .eko/objects/:
//
//	<2-char prefix>/
//	    <remaining 62 chars>.zst   <- zstd-compressed raw file bytes
//	    <remaining 62 chars>.raw   <- uncompressed raw file bytes
//	    <remaining 62 chars>.gz    <- legacy gzip object
//
// All objects are stored read-only (0444). Writes are atomic: the blob is
// written to a .tmp file first, then renamed into place, so a crashed write
// never leaves a corrupt object.
package objects

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"eko/internal/telemetry"
	"eko/internal/util"

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

	return &Store{
		baseDir: base,
	}, nil
}

// objectPath returns the on-disk path for a given SHA-256 hex hash.
//
// Existing objects are preferred in the following order:
//
//	.zst -> .raw -> .gz
//
// The .gz format is retained for backwards compatibility with older Eko
// snapshots.
func (s *Store) objectPath(hash string) string {
	prefix := filepath.Join(
		s.baseDir,
		hash[:2],
		hash[2:],
	)

	for _, ext := range []string{
		".zst",
		".raw",
		".gz",
	} {
		path := prefix + ext

		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default path for a missing object.
	return prefix + ".gz"
}

// Exists reports whether a blob with this hash is already stored.
func (s *Store) Exists(hash string) bool {
	if len(hash) < 2 {
		return false
	}

	prefix := filepath.Join(
		s.baseDir,
		hash[:2],
		hash[2:],
	)

	for _, ext := range []string{
		".zst",
		".raw",
		".gz",
	} {
		if _, err := os.Stat(prefix + ext); err == nil {
			return true
		}
	}

	return false
}

// Put stores data by its SHA-256 hash.
//
// The storage format is selected automatically:
//   - zstd for compressible data
//   - raw for small/already-compressed data
//
// If a blob with the same hash already exists, the call is a no-op.
func (s *Store) Put(data []byte) (hash string, err error) {
	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordCAS(
			context.Background(),
			"put",
			start,
			success,
		)
	}()

	hash, err = s.putWithCompression(
		data,
		shouldCompress(data),
	)

	if err == nil {
		success = true
	}

	return hash, err
}

// putWithCompression stores data either compressed with zstd or uncompressed.
//
// The hash is always calculated from the original uncompressed bytes.
func (s *Store) putWithCompression(
	data []byte,
	compress bool,
) (string, error) {
	hash := hashBytes(data)

	// Fast path: already stored.
	if s.Exists(hash) {
		return hash, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring the lock because another goroutine may
	// have stored the object while we were waiting.
	if s.Exists(hash) {
		return hash, nil
	}

	ext := ".raw"
	if compress {
		ext = ".zst"
	}

	path := filepath.Join(
		s.baseDir,
		hash[:2],
		hash[2:]+ext,
	)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf(
			"objects: mkdir prefix: %w",
			err,
		)
	}

	// Atomic write:
	//
	//	<hash>.zst.tmp
	//	      |
	//	      v
	//	<hash>.zst
	//
	// This prevents partially written objects from appearing as valid objects.
	tmp := path + ".tmp"

	f, err := os.OpenFile(
		tmp,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0644,
	)
	if err != nil {
		return "", fmt.Errorf(
			"objects: create tmp: %w",
			err,
		)
	}

	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}

	if compress {
		var opts []zstd.EOption

		// Small files benefit from avoiding unnecessary encoder parallelism.
		if len(data) < 1024*1024 {
			opts = append(
				opts,
				zstd.WithEncoderConcurrency(1),
			)
		}

		encoder, err := zstd.NewWriter(f, opts...)
		if err != nil {
			cleanup()
			return "", fmt.Errorf(
				"objects: create zstd encoder: %w",
				err,
			)
		}

		if _, err := encoder.Write(data); err != nil {
			_ = encoder.Close()
			cleanup()

			return "", fmt.Errorf(
				"objects: compress: %w",
				err,
			)
		}

		if err := encoder.Close(); err != nil {
			cleanup()

			return "", fmt.Errorf(
				"objects: close zstd encoder: %w",
				err,
			)
		}
	} else {
		if _, err := f.Write(data); err != nil {
			cleanup()

			return "", fmt.Errorf(
				"objects: write raw: %w",
				err,
			)
		}
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)

		return "", fmt.Errorf(
			"objects: close tmp: %w",
			err,
		)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)

		return "", fmt.Errorf(
			"objects: rename: %w",
			err,
		)
	}

	// Objects are immutable after creation.
	_ = os.Chmod(path, 0444)

	return hash, nil
}

// PutFile reads filePath, stores it in the object store, and returns its hash.
//
// If cachedHash is non-empty and the corresponding object exists, the file
// does not need to be read again.
func (s *Store) PutFile(
	filePath string,
	cachedHash string,
) (string, error) {
	if cachedHash != "" && s.Exists(cachedHash) {
		return cachedHash, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf(
			"objects: read %s: %w",
			filePath,
			err,
		)
	}

	compress := shouldCompress(data) &&
		!hasCompressedExtension(filePath)

	return s.putWithCompression(
		data,
		compress,
	)
}

// shouldCompress determines whether the data should be compressed.
//
// Small files are kept raw because compression overhead is usually not worth
// it. Common already-compressed formats are also kept raw.
func shouldCompress(data []byte) bool {
	if len(data) < 1024 {
		return false
	}

	if len(data) >= 4 {
		// Gzip.
		if data[0] == 0x1f &&
			data[1] == 0x8b {
			return false
		}

		// ZIP.
		if data[0] == 0x50 &&
			data[1] == 0x4b &&
			data[2] == 0x03 &&
			data[3] == 0x04 {
			return false
		}

		// PNG.
		if data[0] == 0x89 &&
			data[1] == 0x50 &&
			data[2] == 0x4e &&
			data[3] == 0x47 {
			return false
		}

		// PDF.
		if data[0] == 0x25 &&
			data[1] == 0x50 &&
			data[2] == 0x44 &&
			data[3] == 0x46 {
			return false
		}

		// JPEG.
		if data[0] == 0xff &&
			data[1] == 0xd8 &&
			data[2] == 0xff {
			return false
		}

		// Zstd.
		if data[0] == 0x28 &&
			data[1] == 0xb5 &&
			data[2] == 0x2f &&
			data[3] == 0xfd {
			return false
		}
	}

	return true
}

// hasCompressedExtension avoids recompressing files that are conventionally
// already compressed.
func hasCompressedExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".zip",
		".tar",
		".gz",
		".zst",
		".tgz",
		".png",
		".jpg",
		".jpeg",
		".gif",
		".pdf",
		".mp4",
		".mp3",
		".dmg",
		".exe",
		".dll",
		".so",
		".dylib",
		".rar",
		".7z":
		return true
	}

	return false
}

// Get decompresses and returns the raw bytes for a stored hash.
//
// Reading order:
//
//  1. zstd
//  2. raw
//  3. legacy gzip
func (s *Store) Get(
	hash string,
) (data []byte, err error) {
	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordCAS(
			context.Background(),
			"get",
			start,
			success,
		)
	}()

	if len(hash) < 2 {
		return nil, fmt.Errorf(
			"objects: invalid hash %q",
			hash,
		)
	}

	prefix := filepath.Join(
		s.baseDir,
		hash[:2],
		hash[2:],
	)

	// 1. Zstd.
	if f, openErr := os.Open(prefix + ".zst"); openErr == nil {
		defer f.Close()

		decoder, decoderErr := zstd.NewReader(f)
		if decoderErr != nil {
			return nil, fmt.Errorf(
				"objects: zstd open %s: %w",
				hash[:8],
				decoderErr,
			)
		}
		defer decoder.Close()

		data, err = io.ReadAll(decoder)
		if err != nil {
			return nil, fmt.Errorf(
				"objects: zstd decompress %s: %w",
				hash[:8],
				err,
			)
		}

		success = true
		return data, nil
	}

	// 2. Raw.
	if f, openErr := os.Open(prefix + ".raw"); openErr == nil {
		defer f.Close()

		data, err = io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf(
				"objects: read raw %s: %w",
				hash[:8],
				err,
			)
		}

		success = true
		return data, nil
	}

	// 3. Legacy gzip.
	if f, openErr := os.Open(prefix + ".gz"); openErr == nil {
		defer f.Close()

		gz, gzipErr := gzip.NewReader(f)
		if gzipErr != nil {
			return nil, fmt.Errorf(
				"objects: gzip open %s: %w",
				hash[:8],
				gzipErr,
			)
		}
		defer gz.Close()

		data, err = io.ReadAll(gz)
		if err != nil {
			return nil, fmt.Errorf(
				"objects: gzip decompress %s: %w",
				hash[:8],
				err,
			)
		}

		success = true
		return data, nil
	}

	return nil, fmt.Errorf(
		"objects: open %s: file not found",
		hash[:8],
	)
}

// ExtractTo writes the content of hash to dstPath with the given mode.
//
// Raw objects use CopyOrCloneFile when possible, avoiding an unnecessary
// read/write cycle. Compressed objects are decompressed normally.
func (s *Store) ExtractTo(
	hash string,
	dstPath string,
	mode os.FileMode,
) error {
	if len(hash) < 2 {
		return fmt.Errorf(
			"objects: invalid hash %q",
			hash,
		)
	}

	prefix := filepath.Join(
		s.baseDir,
		hash[:2],
		hash[2:],
	)

	rawPath := prefix + ".raw"

	// Raw objects can be copied or cloned directly.
	if _, err := os.Stat(rawPath); err == nil {
		if err := os.MkdirAll(
			filepath.Dir(dstPath),
			0755,
		); err != nil {
			return err
		}

		if err := util.CopyOrCloneFile(
			rawPath,
			dstPath,
		); err != nil {
			return err
		}

		return os.Chmod(
			dstPath,
			mode.Perm(),
		)
	}

	// Compressed objects need to be decompressed.
	data, err := s.Get(hash)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(
		filepath.Dir(dstPath),
		0755,
	); err != nil {
		return err
	}

	return os.WriteFile(
		dstPath,
		data,
		mode,
	)
}

// RestoreTree extracts all files described by tree into dstDir using a
// parallel worker pool.
//
// The optional onProgress callback is invoked after each successfully
// extracted file.
func (s *Store) RestoreTree(
	tree map[string]FileEntry,
	dstDir string,
	onProgress func(),
) error {
	type job struct {
		rel  string
		hash string
		mode os.FileMode
	}

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	jobs := make(chan job, numWorkers*2)
	errs := make(chan error, numWorkers)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range jobs {
				dst := filepath.Join(
					dstDir,
					filepath.FromSlash(j.rel),
				)

				if err := s.ExtractTo(
					j.hash,
					dst,
					j.mode,
				); err != nil {
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
		jobs <- job{
			rel:  rel,
			hash: entry.Hash,
			mode: entry.Mode,
		}
	}

	close(jobs)

	wg.Wait()
	close(errs)

	for err := range errs {
		return err
	}

	return nil
}

// AllHashes returns every hash currently stored.
//
// Both current formats (.zst/.raw) and legacy gzip (.gz) objects are included.
func (s *Store) AllHashes() ([]string, error) {
	var hashes []string

	err := filepath.Walk(
		s.baseDir,
		func(
			path string,
			info os.FileInfo,
			err error,
		) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			dir := filepath.Base(
				filepath.Dir(path),
			)
			name := info.Name()

			if len(name) > 4 &&
				(name[len(name)-4:] == ".zst" ||
					name[len(name)-4:] == ".raw") {
				hashes = append(
					hashes,
					dir+name[:len(name)-4],
				)

				return nil
			}

			if len(name) > 3 &&
				name[len(name)-3:] == ".gz" {
				hashes = append(
					hashes,
					dir+name[:len(name)-3],
				)
			}

			return nil
		},
	)

	return hashes, err
}

// GarbageCollect deletes objects that are not referenced by any hash in keep.
//
// It returns:
//
//	(bytes freed, objects removed, error)
//
// If dryRun is true, objects are not actually deleted.
func (s *Store) GarbageCollect(
	keep map[string]bool,
	dryRun bool,
) (int64, int, error) {
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

		if dryRun {
			continue
		}

		// Objects are normally read-only.
		_ = os.Chmod(path, 0644)

		if err := os.Remove(path); err != nil {
			continue
		}
	}

	return freed, count, nil
}

// FileEntry is a reference to a stored blob.
//
// Used by RestoreTree and manifests.
type FileEntry struct {
	Hash string      `json:"hash"`
	Mode os.FileMode `json:"mode"`
	Size int64       `json:"size"`
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
