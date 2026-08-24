package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	pgvector "github.com/pgvector/pgvector-go"
)

// PGVectorStore is a PostgreSQL + pgvector backed store for semantic search over
// code ASTs, diffs, and repository memory embeddings.
type PGVectorStore struct {
	db *sql.DB
}

// NewPGVectorStore opens a connection to the PostgreSQL database at dsn and
// ensures the pgvector extension and embeddings table exist.
//
// dsn format: "host=<h> port=5432 user=<u> password=<p> dbname=<db> sslmode=disable"
func NewPGVectorStore(dsn string) (*PGVectorStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("pgvector: open connection: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pgvector: ping: %w", err)
	}
	s := &PGVectorStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// migrate creates the pgvector extension and the embeddings table if they don't
// already exist. The table uses an HNSW index for sub-millisecond ANN queries.
func (s *PGVectorStore) migrate() error {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE TABLE IF NOT EXISTS embeddings (
			id          TEXT PRIMARY KEY,
			snapshot_id TEXT NOT NULL,
			kind        TEXT NOT NULL,
			content     TEXT NOT NULL,
			embedding   vector(1536)
		)`,
		// HNSW index for cosine similarity — best trade-off between recall and
		// query latency for 1536-dim OpenAI-compatible embeddings.
		`CREATE INDEX IF NOT EXISTS embeddings_hnsw_idx
			ON embeddings USING hnsw (embedding vector_cosine_ops)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("pgvector: migrate: %w", err)
		}
	}
	return nil
}

// Embedding holds a vector embedding for a piece of eko content.
type Embedding struct {
	// ID is a stable content-derived identifier (e.g. SHA256 of content).
	ID string
	// SnapshotID is the eko snapshot this embedding belongs to.
	SnapshotID string
	// Kind classifies the content: "ast", "diff", or "memory".
	Kind string
	// Content is the raw text that was embedded.
	Content string
	// Vector is the 1536-dimensional float32 embedding.
	Vector []float32
}

// Upsert stores an embedding, replacing any existing row with the same ID.
func (s *PGVectorStore) Upsert(ctx context.Context, e Embedding) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO embeddings (id, snapshot_id, kind, content, embedding)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE
		   SET snapshot_id = EXCLUDED.snapshot_id,
		       kind        = EXCLUDED.kind,
		       content     = EXCLUDED.content,
		       embedding   = EXCLUDED.embedding`,
		e.ID, e.SnapshotID, e.Kind, e.Content,
		pgvector.NewVector(e.Vector),
	)
	if err != nil {
		return fmt.Errorf("pgvector: upsert %s: %w", e.ID, err)
	}
	return nil
}

// SearchResult is a single result from a semantic similarity search.
type SearchResult struct {
	Embedding
	// Distance is the cosine distance from the query vector (lower = more similar).
	Distance float64
}

// Search returns the top-k embeddings closest to query by cosine similarity.
// Pass kind="" to search across all content types, or "ast"/"diff"/"memory" to
// restrict the search.
func (s *PGVectorStore) Search(ctx context.Context, query []float32, kind string, topK int) ([]SearchResult, error) {
	var (
		rows *sql.Rows
		err  error
	)

	qvec := pgvector.NewVector(query)

	if kind == "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, snapshot_id, kind, content,
			        embedding <=> $1 AS distance
			 FROM   embeddings
			 ORDER  BY distance
			 LIMIT  $2`,
			qvec, topK,
		)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, snapshot_id, kind, content,
			        embedding <=> $1 AS distance
			 FROM   embeddings
			 WHERE  kind = $2
			 ORDER  BY distance
			 LIMIT  $3`,
			qvec, kind, topK,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("pgvector: search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.SnapshotID, &r.Kind, &r.Content, &r.Distance); err != nil {
			return nil, fmt.Errorf("pgvector: scan row: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// DeleteBySnapshot removes all embeddings that belong to the given snapshot ID.
func (s *PGVectorStore) DeleteBySnapshot(ctx context.Context, snapshotID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM embeddings WHERE snapshot_id = $1`, snapshotID,
	)
	if err != nil {
		return fmt.Errorf("pgvector: delete snapshot %s: %w", snapshotID, err)
	}
	return nil
}

// Close releases the database connection pool.
func (s *PGVectorStore) Close() error {
	return s.db.Close()
}
