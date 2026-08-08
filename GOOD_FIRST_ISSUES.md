# 🌟 Good First Issues for New Contributors

Welcome to the **Eko** open-source project! We love welcoming new contributors.

Below is a curated list of beginner-friendly features and enhancements labeled as **`good first issue`**. If you are looking to get started contributing to Go CLI tools, these are great tasks to pick up!

> 💡 **New to Open Source?** We are here to help! Feel free to ask questions in any issue thread, and our maintainers will guide you through writing your PR.

---

## 📌 Issue 1: Add `eko diff <id1> <id2>` terminal file diff command

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

## 📌 Issue 2: Generate shell completion scripts (`eko completion`)

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

## 📌 Issue 3: Add interactive terminal picker for `eko restore` and `eko summary`

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

## 📌 Issue 4: Support custom snapshot aliases and tags (`eko tag`)

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

## 📌 Issue 5: Slack and Discord Webhook notifications for AI summaries

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

## 📌 Issue 6: Add `eko config` command to manage global settings

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`, `cli`

### 🎯 Goal
Allow users to configure global Eko settings (e.g. default AI provider, default `--keep` count, webhook URL) via an `eko config` command that reads/writes a `~/.eko/config.json` file, so users don't need to set environment variables on every run.

### 💡 Expected Usage
```bash
# Set a default AI provider
eko config set ai.provider gemini

# Set a default keep count for eko clean
eko config set clean.keep 10

# View the current config
eko config get ai.provider
eko config list
```

### 🛠️ Implementation Steps
1. Create `internal/config/config.go` defining a `Config` struct with JSON marshaling.
2. Add `cmd/config.go` with Cobra subcommands `set`, `get`, and `list`.
3. Load config values in `cmd/root.go` (before flags are parsed) so individual flags still override config values.
4. Add unit tests in `internal/config/config_test.go`.

---

## 📌 Issue 7: Restore environment variables automatically after `eko restore`

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `enhancement`, `ux`

### 🎯 Goal
Currently `eko restore` generates a `.eko_env_restore.sh` shell script that the user must manually source. Automatically print a clear, actionable post-restore message (and optionally auto-source via eval) so the environment is restored seamlessly.

### 💡 Expected Behavior
```bash
$ eko restore 8c9d1a2f
Restored: 8c9d1a2f

✦ Environment variables from this snapshot have been written to .eko_env_restore.sh
  Run the following to restore your shell environment:

    source .eko_env_restore.sh

  Or add `--env` flag to apply them automatically in a subshell:
    eko restore 8c9d1a2f --env
```

### 🛠️ Implementation Steps
1. Update `cmd/restore.go` to print the actionable post-restore hint when `.eko_env_restore.sh` is generated.
2. Add an `--env` flag that uses `os/exec` to spawn a subshell with the environment applied.
3. Add unit tests for env hint rendering in `cmd/commands_test.go`.

---

## 📌 Issue 8: Show real-time progress bar during `eko save` and `eko restore`

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `ux`, `cli`

### 🎯 Goal
For large projects with thousands of files, `eko save` and `eko restore` give no visual feedback. Add an optional real-time progress bar (file count / total) printed to stderr so users can see the operation progressing.

### 💡 Expected Behavior
```bash
$ eko save
Saving snapshot... [████████████████░░░░] 1234/1500 files
Snapshot saved: 8c9d1a2f
```

### 🛠️ Implementation Steps
1. Add a `--progress` flag (default `true` when stdout is a TTY) to `cmd/save.go` and `cmd/restore.go`.
2. Use an atomic counter in `internal/util/fs.go` worker pool to track copied file count.
3. Print a carriage-return-based progress line to stderr using ANSI escape codes (no external dependency needed).
4. Add unit tests verifying the counter increments correctly.

---

## 📌 Issue 9: Add `eko size` command to report snapshot disk usage

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `cli`

### 🎯 Goal
Implement an `eko size` command that walks `.eko/snapshots/` and reports disk usage per snapshot (in KB/MB/GB) as well as a grand total, so developers can understand how much storage their history is consuming.

### 💡 Expected Output
```bash
$ eko size
ID         Created              Size
8c9d1a2f   2026-08-08 20:11    42.3 MB
3b7f2a1e   2026-08-07 14:32    40.1 MB
1f4a9b0c   2026-08-06 10:05    39.8 MB
──────────────────────────────────────
Total: 3 snapshots · 122.2 MB
```

### 🛠️ Implementation Steps
1. Create `cmd/size.go` with a Cobra command `sizeCmd`.
2. Walk `.eko/snapshots/<id>/` using `filepath.WalkDir` and sum file sizes.
3. Format sizes with human-readable units (KB, MB, GB).
4. Support `--json` flag for machine-readable output.
5. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 10: Add `eko copy` command to copy a snapshot into a new directory

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `cli`

### 🎯 Goal
Allow developers to extract a snapshot's contents into a separate destination directory without modifying the current workspace — useful for comparing project states side-by-side or bootstrapping a new environment from an old snapshot.

### 💡 Expected Usage
```bash
# Copy snapshot 8c9d1a2f into ./my-snapshot-copy/
eko copy 8c9d1a2f ./my-snapshot-copy

# Dry-run: print what would be copied
eko copy 8c9d1a2f ./my-snapshot-copy --dry-run
```

### 🛠️ Implementation Steps
1. Create `cmd/copy.go` with Cobra command `copyCmd` accepting `<snapshot-id>` and `<destination>` arguments.
2. Resolve snapshot path from SQLite DB using `internal/db/db.go`.
3. Reuse `internal/util/fs.go CopyDir()` to write to destination, failing if destination exists (unless `--force` flag is given).
4. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 11: Add `eko search` command to search file contents across snapshots

**Difficulty**: Medium (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `cli`

### 🎯 Goal
Implement an `eko search <pattern>` command that performs a text search across all files in a given snapshot (or all snapshots) and reports matching file paths and line numbers, similar to `grep -r` but scoped to Eko snapshots.

### 💡 Expected Usage
```bash
# Search for a string in the latest snapshot
eko search "TODO"

# Search in a specific snapshot
eko search "TODO" --snapshot 8c9d1a2f

# Search across all snapshots
eko search "deprecated" --all
```

### 🛠️ Implementation Steps
1. Create `cmd/search.go` with Cobra command `searchCmd`.
2. Walk `.eko/snapshots/<id>/` file tree and use `bufio.Scanner` to search each text file for the pattern (skip binaries using heuristics).
3. Print matching file path, line number, and surrounding context (like `grep -n`).
4. Support `--all` flag to iterate over all snapshot IDs from the DB.
5. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 12: Add `eko init --from <snapshot-id>` flag to bootstrap a new project from a snapshot

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `feature`, `cli`

### 🎯 Goal
Extend `eko init` with a `--from <snapshot-id>` flag that initializes a fresh `.eko/` directory **and** immediately restores the specified snapshot into the current empty directory — useful for onboarding onto a shared project checkpoint.

### 💡 Expected Usage
```bash
mkdir new-project && cd new-project
eko init --from 8c9d1a2f
# Eko initialized.
# Restored snapshot 8c9d1a2f into current directory.
```

### 🛠️ Implementation Steps
1. Add `--from` string flag to `cmd/init.go`.
2. After standard `eko init` logic runs, if `--from` is set, call `internal/snapshot.RestoreSnapshot()` with the resolved snapshot path.
3. Print a combined success message.
4. Add unit tests in `cmd/commands_test.go`.

---

## 📌 Issue 13: Write a GitHub Actions CI workflow example for projects using Eko

**Difficulty**: Easy (`good first issue`)  
**Labels**: `good first issue`, `help wanted`, `documentation`, `ci`

### 🎯 Goal
Add a `docs/ci-integration.md` guide showing developers how to integrate `eko save --ai` into a CI/CD pipeline (GitHub Actions) so snapshots are automatically captured on every push or merge, with AI summaries posted as PR comments.

### 💡 Expected Output
A new Markdown file at `docs/ci-integration.md` containing:
- A complete `.github/workflows/eko-snapshot.yml` example.
- Steps to store `GEMINI_API_KEY` or `OPENAI_API_KEY` as GitHub Secrets.
- Instructions to use `eko history --format md` to post the snapshot log as a PR comment via the GitHub CLI (`gh`).

### 🛠️ Implementation Steps
1. Create `docs/ci-integration.md` with a clear guide and annotated YAML workflow.
2. Include a working example of `eko save --ai` running on the `ubuntu-latest` runner after installing `eko` from source.
3. Show how to upload the `.eko/` directory as a GitHub Actions artifact for persistence between workflow runs.
4. Add a link to `docs/ci-integration.md` in the main `README.md` under the Contributing section.

---

## 🤝 How to Claim an Issue

1. Comment on the issue on GitHub: *"I'd like to work on this!"*
2. Fork the repository and create a feature branch (`git checkout -b feat/your-feature`).
3. Write clean Go code and run tests: `go test -v ./...`.
4. Submit a Pull Request targeting `main`.

Thank you for contributing to Eko! ✦
