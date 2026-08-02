package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"eko/internal/db"
)

func TestSummaryCommand_NoSnapshots(t *testing.T) {
	_ = setupTestDir(t)
	_ = initCmd.RunE(initCmd, []string{})

	err := summaryCmd.RunE(summaryCmd, []string{})
	if err == nil {
		t.Error("expected error when running summary with no snapshots, got nil")
	}
}

func TestSummaryCommand_SingleSnapshot(t *testing.T) {
	dir := setupTestDir(t)
	_ = initCmd.RunE(initCmd, []string{})

	os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	_ = saveCmd.RunE(saveCmd, []string{})

	// Ensure provider is set to heuristic
	summaryProvider = "heuristic"
	summaryJSON = false

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := summaryCmd.RunE(summaryCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error running summary: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "Snapshot Change Summary") {
		t.Errorf("expected summary header, got: %q", output)
	}
	if !strings.Contains(output, "app.go") {
		t.Errorf("expected summary to mention app.go, got: %q", output)
	}
}

func TestSummaryCommand_TwoSnapshots_JSON(t *testing.T) {
	dir := setupTestDir(t)
	_ = initCmd.RunE(initCmd, []string{})

	// Snapshot 1
	os.WriteFile(filepath.Join(dir, "v1.txt"), []byte("version 1"), 0644)
	_ = saveCmd.RunE(saveCmd, []string{})

	// Snapshot 2
	os.WriteFile(filepath.Join(dir, "v2.txt"), []byte("version 2"), 0644)
	_ = saveCmd.RunE(saveCmd, []string{})

	summaryProvider = "heuristic"
	summaryJSON = true
	summarySave = true
	defer func() {
		summaryJSON = false
		summarySave = false
	}()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := summaryCmd.RunE(summaryCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error running summary: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var res map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("expected valid JSON output, got %q: %v", buf.String(), err)
	}

	if res["summary"] == nil || res["summary"] == "" {
		t.Error("expected non-empty summary field in JSON")
	}

	// Verify that --save updated the snapshot summary in SQLite DB
	database := db.InitDB()
	defer database.Close()

	var dbSummary string
	err = database.QueryRow("SELECT summary FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&dbSummary)
	if err != nil {
		t.Fatalf("failed to query saved summary from db: %v", err)
	}
	if dbSummary == "" {
		t.Error("expected saved summary in DB to be non-empty")
	}
}

func TestSaveCommand_AI(t *testing.T) {
	dir := setupTestDir(t)
	_ = initCmd.RunE(initCmd, []string{})

	os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package feature\n"), 0644)

	saveAI = true
	saveAIProv = "heuristic"
	defer func() {
		saveAI = false
		saveAIProv = "auto"
	}()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := saveCmd.RunE(saveCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error on save --ai: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "AI Summary:") {
		t.Errorf("expected stdout to contain AI Summary, got: %q", output)
	}

	database := db.InitDB()
	defer database.Close()

	var summary string
	err = database.QueryRow("SELECT summary FROM snapshots LIMIT 1").Scan(&summary)
	if err != nil {
		t.Fatalf("failed to read summary from db: %v", err)
	}
	if summary == "" {
		t.Error("expected summary column in DB to be populated by save --ai")
	}
}
