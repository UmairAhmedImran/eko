package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"eko/internal/telemetry"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB opens the local SQLite database and ensures migrations are applied.
func InitDB() *sql.DB {
	database, err := sql.Open("sqlite3", ".eko/db.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	if err := MigrateDB(database); err != nil {
		log.Printf("warning: db migration error: %v", err)
	}
	return database
}

// MigrateDB creates the schema if needed and adds any missing columns (e.g. summary, tag).
func MigrateDB(database *sql.DB) error {
	_, err := execWithTelemetry(
		database,
		"create_schema",
		`
		CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			message TEXT,
			path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			summary TEXT,
			tag TEXT UNIQUE
		)
		`,
	)
	if err != nil {
		return fmt.Errorf("error creating snapshots table: %w", err)
	}

	// Safely ensure summary and tag columns exist for pre-existing databases
	_, _ = execWithTelemetry(
		database,
		"migration",
		"ALTER TABLE snapshots ADD COLUMN summary TEXT",
	)

	_, _ = execWithTelemetry(
		database,
		"migration",
		"ALTER TABLE snapshots ADD COLUMN tag TEXT UNIQUE",
	)
	return nil
}

// SaveSummary updates the AI-generated summary for a given snapshot ID.
func SaveSummary(database *sql.DB, id, summary string) error {
	_, err := execWithTelemetry(
		database,
		"save_summary",
		"UPDATE snapshots SET summary = ? WHERE id = ?",
		summary,
		id,
	)
	return err
}

// SaveTag assigns a human-readable tag/alias to a given snapshot ID.
func SaveTag(database *sql.DB, id, tag string) error {
	_, err := execWithTelemetry(
		database,
		"save_tag",
		"UPDATE snapshots SET tag = ? WHERE id = ?",
		tag,
		id,
	)
	return err
}

// ResolveSnapshot resolves an 8-character snapshot ID or a human-readable tag to its manifest/snapshot path.
func ResolveSnapshot(database *sql.DB, target string) (id, path string, err error) {
	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordSQLite(
			context.Background(),
			"resolve_snapshot",
			start,
			success,
		)
	}()

	err = database.QueryRow(
		"SELECT id, path FROM snapshots WHERE id = ? OR tag = ?",
		target,
		target,
	).Scan(&id, &path)

	if err != nil {
		return "", "", fmt.Errorf(
			"snapshot or tag %q not found: %w",
			target,
			err,
		)
	}

	success = true

	return id, path, nil
}

func execWithTelemetry(
	database *sql.DB,
	operation string,
	query string,
	args ...any,
) (sql.Result, error) {
	start := time.Now()
	success := false

	defer func() {
		telemetry.RecordSQLite(
			context.Background(),
			operation,
			start,
			success,
		)
	}()

	result, err := database.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	success = true

	return result, nil
}
