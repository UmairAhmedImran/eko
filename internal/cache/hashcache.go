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
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"eko/internal/telemetry"
)

// HashCache wraps the db.sqlite hash_cache table with prepared statement caching
// and thread-safe serializability.
type HashCache struct {
	db         *sql.DB
	stmtLookup *sql.Stmt
	stmtStore  *sql.Stmt
	mu         sync.Mutex // serializes DB writes to prevent SQLite lock contention
}

// New initialises the hash_cache table and pre-compiles prepared SQL statements.
func New(db *sql.DB) (*HashCache, error) {
	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordSQLite(
			context.Background(),
			"hash_cache_init",
			start,
			success,
		)
	}()

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

	stmtLookup, err := db.Prepare("SELECT sha256 FROM hash_cache WHERE path=? AND mtime_ns=? AND size=?")
	if err != nil {
		return nil, fmt.Errorf("hash_cache: prepare lookup: %w", err)
	}

	stmtStore, err := db.Prepare("INSERT OR REPLACE INTO hash_cache (path, mtime_ns, size, sha256) VALUES (?, ?, ?, ?)")
	if err != nil {
		stmtLookup.Close()
		return nil, fmt.Errorf("hash_cache: prepare store: %w", err)
	}

	success = true

	return &HashCache{
		db:         db,
		stmtLookup: stmtLookup,
		stmtStore:  stmtStore,
	}, nil
}

// Close releases pre-compiled SQL statements.
func (c *HashCache) Close() {
	if c.stmtLookup != nil {
		c.stmtLookup.Close()
	}
	if c.stmtStore != nil {
		c.stmtStore.Close()
	}
}

// Lookup returns the cached sha256 for a file using pre-compiled prepared statements.
func (c *HashCache) Lookup(path string, info os.FileInfo) (hash string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.stmtLookup.QueryRow(path, info.ModTime().UnixNano(), info.Size()).Scan(&hash)
	return hash, err == nil
}

// Store saves or updates the cache entry using pre-compiled prepared statements.
func (c *HashCache) Store(path string, info os.FileInfo, hash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordSQLite(
			context.Background(),
			"hash_cache_store",
			start,
			success,
		)
	}()

	_, err := c.stmtStore.Exec(
		path,
		info.ModTime().UnixNano(),
		info.Size(),
		hash,
	)
	if err != nil {
		return fmt.Errorf("hash_cache: store %s: %w", path, err)
	}

	success = true
	return nil
}

// Purge removes stale entries for paths that no longer exist.
func (c *HashCache) Purge(existingPaths map[string]bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordSQLite(
			context.Background(),
			"hash_cache_purge",
			start,
			success,
		)
	}()

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

	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range stale {
		if _, err := c.db.Exec(
			"DELETE FROM hash_cache WHERE path=?",
			p,
		); err != nil {
			return err
		}
	}

	success = true
	return nil
}
