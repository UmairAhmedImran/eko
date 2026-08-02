package db

import (
	"database/sql"
	"fmt"
	"log"

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

// MigrateDB creates the schema if needed and adds any missing columns (e.g. summary).
func MigrateDB(database *sql.DB) error {
	_, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			message TEXT,
			path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			summary TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating snapshots table: %w", err)
	}

	// Safely ensure summary column exists for pre-existing databases
	_, _ = database.Exec("ALTER TABLE snapshots ADD COLUMN summary TEXT")
	return nil
}

// SaveSummary updates the AI-generated summary for a given snapshot ID.
func SaveSummary(database *sql.DB, id, summary string) error {
	_, err := database.Exec("UPDATE snapshots SET summary = ? WHERE id = ?", summary, id)
	return err
}
