---
sidebar_position: 4
---

# CLI Reference Manual

Exhaustive reference guide for all commands, flags, options, and aliases in the **Eko** command line interface.

---

## 📋 Command Matrix

| Command | Aliases | Description | Key Flags |
| ------- | ------- | ----------- | --------- |
| [`eko init`](#1-eko-init) | None | Initialize Eko project & SQLite database | None |
| [`eko save`](#2-eko-save) | None | Capture project snapshot | `-m`, `-a/--ai`, `--provider` |
| [`eko summary`](#3-eko-summary-alias-eko-summarize) | `summarize` | Generate AI-powered change summary | `-j/--json`, `-p/--provider`, `-s/--save` |
| [`eko history`](#4-eko-history) | None | List snapshot history | `-j/--json`, `-v/--verbose` |
| [`eko restore`](#5-eko-restore-snapshot-id) | None | Revert project to a past snapshot state | `<snapshot-id>` |

---

## 1. `eko init`

Initializes a new Eko project in the current working directory. Creates the hidden `.eko/` folder containing the `snapshots/` directory and local SQLite database (`db.sqlite`).

```bash
eko init
```

**Behavior & Safety Guards**:
- Checks if the project is already initialized.
- Detects if a `.git` repository exists and displays a tip (Eko operates independently of Git and automatically ignores `.git`).

---

## 2. `eko save`

Captures the current filesystem state (excluding `.eko`, `.git`, `node_modules`, build artifacts) and stores it as a new 8-hex-character snapshot ID.

```bash
# Save snapshot with default message ("snapshot")
eko save

# Save with custom log description
eko save -m "fixed SQLite concurrency bug"

# Auto-generate AI change summary when saving
eko save --ai

# Auto-generate AI summary using a specific AI provider
eko save --ai --provider gemini
```

### Flags

| Flag | Short | Type | Default | Description |
| ---- | ----- | ---- | ------- | ----------- |
| `--message` | `-m` | String | `"snapshot"` | Log message describing the snapshot |
| `--ai` | `-a` | Bool | `false` | Auto-generate AI change summary using LLM/heuristic provider |
| `--provider` | | String | `"auto"` | AI provider for auto-summary (`auto`, `heuristic`, `openai`, `gemini`) |

---

## 3. `eko summary` (Alias: `eko summarize`)

Calculates file diffs (insertions, deletions, modifications) between snapshots and generates an AI-powered summary.

```bash
# Summarize changes in the latest snapshot vs predecessor
eko summary

# Summarize changes introduced in snapshot <id>
eko summary 3b7f2a1e

# Summarize changes between two specific snapshots
eko summary 3b7f2a1e 8c9d1a2f

# Output summary in JSON format
eko summary --json

# Force Gemini AI provider and save generated summary to SQLite DB
eko summary 3b7f2a1e --provider gemini --save
```

### Flags

| Flag | Short | Type | Default | Description |
| ---- | ----- | ---- | ------- | ----------- |
| `--json` | `-j` | Bool | `false` | Output change stats and summary in structured JSON format |
| `--provider` | `-p` | String | `"auto"` | AI provider engine (`auto`, `heuristic`, `openai`, `gemini`) |
| `--save` | `-s` | Bool | `false` | Save/update the generated summary in the SQLite database record |

---

## 4. `eko history`

Lists all recorded snapshots in reverse chronological order with creation timestamps, log messages, and AI summaries.

```bash
# Standard history view
eko history

# Verbose view with detailed AI summaries
eko history --verbose

# Programmatic JSON output
eko history --json
```

### Flags

| Flag | Short | Type | Default | Description |
| ---- | ----- | ---- | ------- | ----------- |
| `--json` | | Bool | `false` | Output history list as JSON array |
| `--verbose` | `-v` | Bool | `false` | Show verbose history with detailed AI summaries |

---

## 5. `eko restore <snapshot-id>`

Reverts the working directory to the exact state captured in snapshot `<snapshot-id>`.

```bash
eko restore 3b7f2a1e
```

**Restoration Engine Details**:
1. **Parallel Delete Phase**: Removes top-level workspace files concurrently using goroutines and lock-free Compare-And-Swap (CAS) error handling.
2. **Parallel Copy Phase**: Re-populates workspace files from `.eko/snapshots/<id>/`.
3. **Environment Restoration**: Generates `.eko_env_restore.sh` to restore captured environment variables.

---

## 🌍 Global Environment Variables

| Variable | Default Value | Usage |
| -------- | ------------- | ----- |
| `GEMINI_API_KEY` | *(None)* | API Key for Google Gemini LLM provider |
| `OPENAI_API_KEY` | *(None)* | API Key for OpenAI LLM provider |
| `EKO_AI_API_KEY` | *(None)* | General API Key for AI provider |
| `EKO_AI_ENDPOINT` | `https://api.openai.com/v1` | Custom endpoint for OpenAI-compatible LLMs (vLLM, Ollama) |
| `EKO_AI_MODEL` | `gpt-4o-mini` / `gemini-1.5-flash` | Model override for AI provider |
