---
sidebar_position: 5
title: 🌟 Good First Issues
---

# 🌟 Good First Issues for New Contributors

Welcome to the **Eko** open-source community! We are excited to welcome new contributors.

Whether you are looking to write your first line of Go, expand CLI commands, improve documentation, or build AI integrations, we have curated a set of beginner-friendly tasks labeled as **`good first issue`**.

---

## 📌 Available Beginner Issues

### 1. `eko version` Command with Build Info
- **Goal**: Add `eko version` and `--version` flag printing version, Go runtime, target OS/Arch, and git commit hash.
- **Difficulty**: Easy (`good first issue`)

### 2. `eko clean` Snapshot Pruning Command
- **Goal**: Implement `eko clean --keep 5` to remove old snapshot directories and SQLite records.
- **Difficulty**: Medium (`good first issue`)

### 3. Custom `.ekoignore` Exclusions File
- **Goal**: Support reading project-level `.ekoignore` files to exclude glob patterns (e.g. `*.log`, `tmp/`).
- **Difficulty**: Medium (`good first issue`)

### 4. Export History in Markdown, HTML & CSV
- **Goal**: Extend `eko history` with `--format md`, `--format html`, and `--format csv` options.
- **Difficulty**: Easy (`good first issue`)

### 5. `eko status` Workspace Modification Inspector
- **Goal**: Compare local workspace files against the latest snapshot point and print colorized status.
- **Difficulty**: Easy (`good first issue`)

### 6. `eko diff` Terminal File Diff Command
- **Goal**: Render unified line diffs between snapshot states directly in the terminal interface.
- **Difficulty**: Medium (`good first issue`)

### 7. Shell Completion Scripts (`eko completion`)
- **Goal**: Auto-generate completion scripts for Bash, Zsh, Fish, and PowerShell using Cobra methods.
- **Difficulty**: Easy (`good first issue`)

### 8. Interactive Terminal Picker (TUI Mode)
- **Goal**: Prompt users with an interactive selection menu when `eko restore` or `eko summary` is run without arguments.
- **Difficulty**: Medium (`good first issue`)

### 9. Custom Snapshot Aliases & Tags (`eko tag`)
- **Goal**: Allow tagging snapshots with names like `pre-refactor` or `v1.0.0`.
- **Difficulty**: Medium (`good first issue`)

### 10. Slack & Discord Webhook Integration
- **Goal**: Post AI change summary payloads to `EKO_WEBHOOK_URL` whenever `eko save --ai` is executed.
- **Difficulty**: Medium (`good first issue`)

---

## 🤝 How to Get Started

1. Visit our [GitHub Repository](https://github.com/kavix/eko).
2. Browse full step-by-step implementation guides in [`GOOD_FIRST_ISSUES.md`](https://github.com/kavix/eko/blob/main/GOOD_FIRST_ISSUES.md).
3. Leave a comment on any issue: *"I'd like to work on this!"*
4. Submit a Pull Request targeting `main`. We'll review and merge your work quickly!

---

## ⭐️ Support Eko

If you find Eko useful, consider giving us a **⭐️ Star on GitHub**! It helps more developers discover the project.
