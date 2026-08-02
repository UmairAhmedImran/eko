---
sidebar_position: 4
---

# CLI Reference

Comprehensive reference guide for the `eko` command line interface.

## Commands Overview

### `eko init`
Initialize an Eko project in the current directory.

```bash
eko init
```

### `eko save`
Capture and save a snapshot of the current workspace state.

```bash
# Save snapshot with default message
eko save

# Save with custom log description
eko save -m "fixed database lock"

# Save with auto-generated AI change summary
eko save --ai
```

### `eko summary` / `eko summarize`
Generate AI-powered change summaries of snapshot differences.

```bash
eko summary
eko summary <snapshot-id>
eko summary <snapshot-id1> <snapshot-id2>
eko summary --json --provider gemini
```

### `eko history`
List all recorded snapshots and metadata.

```bash
eko history
eko history --verbose
eko history --json
```

### `eko restore <snapshot-id>`
Revert current workspace state to a past snapshot.

```bash
eko restore 3b7f2a1e
```
