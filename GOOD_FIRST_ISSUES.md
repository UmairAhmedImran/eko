# 🌟 Good First Issues for New Contributors

Welcome to the **Eko** open-source project! We love welcoming new contributors.

Below is a curated list of beginner-friendly features and enhancements labeled as **`good first issue`**. If you are looking to get started contributing to Go CLI tools, these are great tasks to pick up!

> 💡 **New to Open Source?** We are here to help! Feel free to ask questions in any issue thread, and our maintainers will guide you through writing your PR.

---

## 📌 Issue 1: Add `eko version` command with build commit info

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`, `cli`

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

## 📌 Issue 4: Export snapshot history in Markdown, HTML, and CSV formats

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`

### 🎯 Goal
Extend `eko history` to allow exporting snapshot logs in Markdown (`--format md`), HTML (`--format html`), or CSV (`--format csv`) format in addition to `--json`.

### 💡 Expected Usage
```bash
# Export history as Markdown table
eko history --format md

# Export history as CSV
eko history --format csv

# Export history as styled HTML page
eko history --format html > history.html
```

### 🛠️ Implementation Steps
1. Update `cmd/history.go` to add `--format` string flag (options: `text`, `json`, `md`, `csv`, `html`).
2. Format entries into Markdown table, CSV format using `encoding/csv`, or styled HTML template.
3. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 5: Add `eko status` command to inspect workspace changes

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`

### 🎯 Goal
Implement an `eko status` command that compares the current workspace files with the latest snapshot, showing added, modified, and deleted files (similar to `git status`).

### 💡 Expected Output
```bash
$ eko status
✦ Eko Workspace Status (vs Snapshot 8c9d1a2f)

Modified (2):
  - main.go
  - internal/ai/provider.go

Added (1):
  - config.yaml

Deleted (0):
  - None
```

### 🛠️ Implementation Steps
1. Create `cmd/status.go` with Cobra command `statusCmd`.
2. Re-use `internal/api/diff.go` to compute diffs between local directory state and latest snapshot.
3. Colorize output terminal tags (green for added, yellow for modified, red for deleted).
4. Add unit test in `cmd/commands_test.go`.

---

## 📌 Issue 6: Add `eko diff <id1> <id2>` terminal file diff command

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `cli`

### 🎯 Goal
Add an `eko diff` subcommand that displays line-by-line unified colored diffs between two snapshot points directly in the terminal interface.

### 💡 Expected Usage
```bash
# Compare latest snapshot with predecessor
eko diff

# Compare specific snapshot with current workspace
eko diff 8c9d1a2f

# Compare two specific snapshots
eko diff 3b7f2a1e 8c9d1a2f
```

### 🛠️ Implementation Steps
1. Create `cmd/diff.go` with Cobra command `diffCmd`.
2. Integrate a lightweight Go diff algorithm (e.g. `sergi/go-diff` or stdlib text comparison).
3. Print unified git-style `+` / `-` colored line diffs.
4. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 7: Generate shell completion scripts (`eko completion`)

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`

### 🎯 Goal
Add shell completion command support (`eko completion [bash|zsh|fish|powershell]`) so users can enjoy TAB completion for subcommands and flags.

### 💡 Expected Usage
```bash
# Generate completion script for Zsh
eko completion zsh > ~/.zsh/completion/_eko
```

### 🛠️ Implementation Steps
1. Create `cmd/completion.go`.
2. Utilize Cobra's built-in `GenBashCompletion`, `GenZshCompletion`, `GenFishCompletion`, and `GenPowerShellCompletion` methods.
3. Document shell installation instructions in the command `Long` description.
4. Add unit test in `cmd/commands_test.go`.

---

## 📌 Issue 8: Add interactive terminal picker for `eko restore` and `eko summary`

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `ux`, `tui`

### 🎯 Goal
When `eko restore` or `eko summary` is invoked without a snapshot ID in an interactive terminal, prompt the user with a sleek terminal selection menu listing past snapshots with timestamps and messages.

### 💡 Expected Behavior
```bash
$ eko restore
? Select a snapshot to restore:
  > 8c9d1a2f - Added AI change summary engine (2 minutes ago)
    3b7f2a1e - Initial workspace save (1 hour ago)
    1f4a9b0c - Setup SQLite schema (Yesterday)
```

### 🛠️ Implementation Steps
1. Check if standard input is a terminal (`isatty`).
2. Fetch recent snapshots from SQLite DB (`internal/db/db.go`).
3. Render selection list using standard Go prompt library or basic terminal input.
4. Add unit tests handling non-interactive fallbacks.

---

## 📌 Issue 9: Support custom snapshot aliases and tags (`eko tag`)

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`

### 🎯 Goal
Allow developers to assign human-readable tags/aliases (e.g. `v1.0.0`, `pre-refactor`) to snapshots so they can restore or view summaries using tags instead of 8-character hex hashes.

### 💡 Expected Usage
```bash
# Tag an existing snapshot
eko tag 8c9d1a2f pre-refactor

# Save a snapshot with a tag directly
eko save -m "Before database migration" --tag pre-migration

# Restore using tag alias
eko restore pre-refactor
```

### 🛠️ Implementation Steps
1. Update SQLite table schema to add an optional `tag TEXT UNIQUE` column in `.eko/db.sqlite`.
2. Add `cmd/tag.go` Cobra subcommand.
3. Update `internal/db/db.go` lookup logic so snapshot queries resolve both 8-character IDs and string tags.
4. Add unit test for snapshot tagging and resolution.

---

## 📌 Issue 10: Slack and Discord Webhook notifications for AI summaries

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `integrations`

### 🎯 Goal
Add optional webhook notifications so whenever `eko save --ai` generates a snapshot summary, it can post a formatted notification card to a Slack or Discord channel.

### 💡 Expected Behavior
When `EKO_WEBHOOK_URL` environment variable is set:
```bash
export EKO_WEBHOOK_URL="https://hooks.slack.com/services/..."
eko save --ai
# -> Posts snapshot summary & metrics payload to webhook URL asynchronously
```

### 🛠️ Implementation Steps
1. Add `internal/notify/webhook.go` helper to send JSON HTTP POST payloads.
2. Format message with snapshot ID, summary text, and timestamp.
3. Trigger notification in `cmd/save.go` when `--ai` is enabled and `EKO_WEBHOOK_URL` is set.
4. Add unit tests using standard `net/http/httptest` server mock.

---

## 🤝 How to Claim an Issue

1. Comment on the issue on GitHub: *"I'd like to work on this!"*
2. Fork the repository and create a feature branch (`git checkout -b feat/your-feature`).
3. Write clean Go code and run tests: `go test -v ./...`.
4. Submit a Pull Request targeting `main`.

Thank you for contributing to Eko! ✦
