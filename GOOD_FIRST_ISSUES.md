# 🌟 Good First Issues for New Contributors

Welcome to the **Eko** open-source project! We love welcoming new contributors.

Below is a curated list of beginner-friendly features and enhancements labeled as **`good first issue`**. If you are looking to get started contributing to Go CLI tools, these are great tasks to pick up!

---

## 📌 Issue 1: Add `eko version` command with build commit info

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`

### 🎯 Goal
Implement an `eko version` subcommand (and `-v` / `--version` flag on the root command) that prints version details, Go runtime version, target OS/Arch, and Git commit hash.

### 💡 Expected Output
```bash
$ eko version
eko version 1.1.0 (darwin/arm64)
Go version: go1.22.0
Git commit: 8c9d1a2f
Built: 2026-08-02
```

### 🛠️ Implementation Steps
1. Create `cmd/version.go` defining Cobra command `versionCmd`.
2. Add global variables `Version`, `Commit`, `BuildDate` set via `-ldflags` during build.
3. Wire `-v / --version` flag into `cmd/root.go`.
4. Add unit test in `cmd/commands_test.go`.

---

## 📌 Issue 2: Add `eko clean` command to prune old snapshots

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`

### 🎯 Goal
Provide a command to purge old snapshots to free disk space while keeping a specified number of recent snapshots.

### 💡 Expected Usage
```bash
# Keep last 5 snapshots and remove older ones
eko clean --keep 5

# Dry-run to see what would be deleted without deleting
eko clean --keep 5 --dry-run
```

### 🛠️ Implementation Steps
1. Create `cmd/clean.go` with flags `--keep` (default: 10) and `--dry-run` (default: `false`).
2. Query `snapshots` table in SQLite DB sorted by `created_at DESC`.
3. For snapshots exceeding the keep count, delete directory `.eko/snapshots/<id>` using `os.RemoveAll` and execute `DELETE FROM snapshots WHERE id = ?`.
4. Add unit test verifying directory cleanup and DB record deletion.

---

## 📌 Issue 3: Support custom `.ekoignore` project configuration file

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`

### 🎯 Goal
Allow developers to place a `.ekoignore` file in their project root to specify custom files and folder glob patterns to exclude from snapshots (e.g. `*.log`, `tmp/`, `vendor/`).

### 💡 Expected Behavior
If `.ekoignore` exists in the project root:
```text
# .ekoignore
*.log
tmp/
coverage/
```
`eko save` should automatically skip files matching patterns in `.ekoignore` in addition to default ignored paths (`.eko`, `.git`, `node_modules`).

### 🛠️ Implementation Steps
1. Update `internal/util/fs.go` function `ShouldIgnore(name string, isDir bool)` to load and parse `.ekoignore` if present.
2. Use `path/filepath.Match` or standard glob patterns to test filenames.
3. Add unit tests in `internal/util/fs_test.go`.

---

## 📌 Issue 4: Export snapshot history in Markdown and CSV formats

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`

### 🎯 Goal
Extend `eko history` to allow exporting snapshot logs in Markdown (`--format md`) or CSV (`--format csv`) format in addition to `--json`.

### 💡 Expected Usage
```bash
# Export history as Markdown table
eko history --format md

# Export history as CSV
eko history --format csv
```

### 🛠️ Implementation Steps
1. Update `cmd/history.go` to add `--format` string flag (options: `text`, `json`, `md`, `csv`).
2. Format entries into Markdown table or CSV format using Go standard library `encoding/csv`.
3. Add unit tests in `cmd/commands_test.go`.

---

## 🤝 How to Claim an Issue

1. Comment on the issue on GitHub: *"I'd like to work on this!"*
2. Fork the repository and create a feature branch (`git checkout -b feat/your-feature`).
3. Write clean Go code and run tests: `go test -v ./...`.
4. Submit a Pull Request targeting `main`.

Thank you for contributing to Eko! ✦
