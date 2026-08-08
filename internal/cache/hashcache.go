// Package cache implements an incremental hash cache for Eko.
//
// On every eko save, the engine needs a SHA-256 hash for every project file.
// Re-reading and re-hashing unchanged files is expensive for large codebases.
// The hash cache stores a (path, mtime_ns, size) → sha256 mapping so that
// files whose metadata hasn't changed can skip the read entirely.
//
// The cache is backed by a hash_cache table in the same db.sqlite used for
// snapshot metadata. It is lazily populated: the first save after a cold start
// hashes everything; subsequent saves only hash changed files.
//
// Thread safety: HashCache is safe for concurrent use. Database writes use
// INSERT OR REPLACE so concurrent saves on the same file converge correctly.
package cache

import (
	"database/sql"
	"fmt"
	"os"
)

// HashCache wraps the db.sqlite hash_cache table.
type HashCache struct {
	db *sql.DB
}

// New initialises the hash_cache table (idempotent) and returns a HashCache.
func New(db *sql.DB) (*HashCache, error) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS hash_cache (
			path     TEXT    NOT NULL,
			mtime_ns INTEGER NOT NULL,
			size     INTEGER NOT NULL,
			sha256   TEXT    NOT NULL,
			PRIMARY KEY (path, mtime_ns, size)
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("hash_cache: init table: %w", err)
	}
	return &HashCache{db: db}, nil
}

// Lookup returns the cached sha256 for a file, or ("", false) on miss.
// A cache hit means mtime_ns AND size both match the stored entry for path.
func (c *HashCache) Lookup(path string, info os.FileInfo) (hash string, ok bool) {
	err := c.db.QueryRow(
		"SELECT sha256 FROM hash_cache WHERE path=? AND mtime_ns=? AND size=?",
		path, info.ModTime().UnixNano(), info.Size(),
	).Scan(&hash)
	return hash, err == nil
}

// Store saves or updates the cache entry for path.
func (c *HashCache) Store(path string, info os.FileInfo, hash string) error {
	_, err := c.db.Exec(
		`INSERT OR REPLACE INTO hash_cache (path, mtime_ns, size, sha256)
		 VALUES (?, ?, ?, ?)`,
		path, info.ModTime().UnixNano(), info.Size(), hash,
	)
	if err != nil {
		return fmt.Errorf("hash_cache: store %s: %w", path, err)
	}
	return nil
}

// Purge removes stale entries for paths that no longer exist.
// Called periodically to keep the cache table lean.
func (c *HashCache) Purge(existingPaths map[string]bool) error {
	rows, err := c.db.Query("SELECT DISTINCT path FROM hash_cache")
	if err != nil {
		return err
	}
	defer rows.Close()

	var stale []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err == nil && !existingPaths[p] {
			stale = append(stale, p)
		}
	}
	rows.Close()

	for _, p := range stale {
		if _, err := c.db.Exec("DELETE FROM hash_cache WHERE path=?", p); err != nil {
			return err
		}
	}
	return nil
}
