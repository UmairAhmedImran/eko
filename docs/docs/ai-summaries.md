---
sidebar_position: 3
---

# AI-Powered Change Summaries

Eko provides intelligent, automated change summaries for your snapshot points using LLM providers (Google Gemini, OpenAI, or local heuristic engines).

---

## 🔑 Setting Up API Keys

To enable AI change summaries, export your preferred API key in your terminal environment.

### 1. Google Gemini (Recommended)

Get an API key from [Google AI Studio](https://aistudio.google.com/).

**macOS / Linux (Bash & Zsh)**:
```bash
export GEMINI_API_KEY="your-gemini-api-key-here"
```

**Windows (PowerShell)**:
```powershell
$env:GEMINI_API_KEY="your-gemini-api-key-here"
```

**Windows (Command Prompt)**:
```cmd
set GEMINI_API_KEY="your-gemini-api-key-here"
```

---

### 2. OpenAI / Custom LLM (vLLM / Ollama)

Get an API key from [OpenAI Platform](https://platform.openai.com/).

**macOS / Linux**:
```bash
export OPENAI_API_KEY="your-openai-api-key-here"

# (Optional) Custom base URL for local LLMs like Ollama or vLLM:
export EKO_AI_ENDPOINT="http://localhost:11434/v1"
```

---

### 3. Permanent Setup (Shell Profile)

To keep your API key active across terminal restarts, add it to your shell profile (`~/.zshrc` or `~/.bashrc`):

```bash
echo 'export GEMINI_API_KEY="your-gemini-api-key-here"' >> ~/.zshrc
source ~/.zshrc
```

---

## 🚀 Generating Summaries

### Auto-Generate Summary When Saving a Snapshot
```bash
eko save --ai
```
*Output:*
```text
Snapshot saved: 8c9d1a2f
AI Summary: Added user authentication endpoint and updated SQLite database migration logic.
```

### Summarize Latest Snapshot
```bash
eko summary
```

### Summarize Specific Snapshot
```bash
eko summary 3b7f2a1e
```

### Summarize Differences Between Two Snapshots
```bash
eko summary 3b7f2a1e 8c9d1a2f
```

### Force a Specific Provider
```bash
eko summary 3b7f2a1e --provider gemini
eko summary 3b7f2a1e --provider openai
eko summary 3b7f2a1e --provider heuristic
```

---

## ⚙️ AI Provider Options

| Provider | Flag | Description | Credentials Required |
| -------- | ---- | ----------- | --------------------- |
| **Auto** | `-p auto` | Auto-detects Gemini or OpenAI key; falls back to Heuristic engine | None (fallback) |
| **Gemini** | `-p gemini` | Google Gemini 1.5 Flash model | `GEMINI_API_KEY` |
| **OpenAI** | `-p openai` | OpenAI GPT-4o-mini / custom LLM | `OPENAI_API_KEY` or `EKO_AI_API_KEY` |
| **Heuristic** | `-p heuristic` | Local rule-based fast analysis | None (offline) |

### Configuration Environment Variables

| Environment Variable | Default Value | Description |
| -------------------- | ------------- | ----------- |
| `GEMINI_API_KEY` | *(None)* | Google Gemini API key |
| `OPENAI_API_KEY` | *(None)* | OpenAI API key |
| `EKO_AI_API_KEY` | *(None)* | Alternative key for AI provider |
| `EKO_AI_ENDPOINT` | `https://api.openai.com/v1` | Custom OpenAI-compatible REST endpoint |
| `EKO_AI_MODEL` | `gpt-4o-mini` / `gemini-1.5-flash` | Custom model name override |

---

## 🧠 Eko GitMind AI Reasoning Suite

Eko introduces **GitMind**, an architecture-aware AI reasoning engine that analyzes your snapshot histories, code architecture, and workspace diffs to provide developer intelligence.

### 1. `eko ai status` (Intent & Role Analysis)
Analyzes current workspace changes and determines the developer's high-level intent, listing file roles and recommended next steps.

```bash
eko ai status
```

**Example Output:**
```text
🤖 AI Workspace Status
──────────────────────────────────────────────────
🎯 Intent: Refactored database transaction logic and corrected connection pool locks.

Files:
  ✓ internal/db/db.go           (Go database pool & statement caching)
  ✓ internal/db/db_test.go      (Added concurrent transaction tests)

💡 Suggested Next Step:
  → Run: go test -v ./internal/db/...
```

### 2. `eko ai review` (Automated Code Review)
Performs a pre-commit code review of the workspace modifications, checking for edge cases, performance bugs, and calculating a Risk Score (0-100).

```bash
eko ai review
```

**Example Output:**
```text
🤖 AI Code Review
──────────────────────────────────────────────────
📊 Commit Risk Score: 35/100 (Low-Medium)

Findings:
  ⚠ internal/db/db.go:L124
    - Issue: database connection is closed inside a loop on retry failures.
    - Suggestion: Defer closing the connection outside the loop or utilize connection pool methods.
```

### 3. `eko ai semdiff` (Behavioral Semantic Diff)
Traditional diffs show lines of code. `eko ai semdiff` explains the **behavioral change** introduced by the modifications.

```bash
eko ai semdiff
```

**Example Output:**
```text
🤖 Behavioral Change Analysis:
  - Before: SQL statement locks were held until the transaction fully committed.
  - After: SQL statement locks are released immediately after query completion.
  - Potential Impact: Significantly reduces database contention, but may lead to dirty reads if non-transactional statements are run concurrently.
```

### 4. `eko ai risk` (5-Area Risk Matrix)
Evaluates risks across five critical areas: Database, Authentication, API, Tests, and Configurations.

```bash
eko ai risk
```

**Example Output:**
```text
Commit Risk Analysis
Overall: MEDIUM
┌──────────────────────┬────────┐
│ Area                 │ Risk   │
├──────────────────────┼────────┤
│ Database             │ HIGH   │
│ Authentication       │ LOW    │
│ API                  │ MEDIUM │
│ Tests                │ LOW    │
│ Configuration        │ LOW    │
└──────────────────────┴────────┘
Reasons:
  ⚠ Database connection pool sizes were updated without a resource limits check.
  ⚠ Added raw SQL queries bypass ORM validation layers.
```

### 5. `eko ai security` (Credential Leak Scanner)
Scans modified files in the workspace for hardcoded API keys, secrets, certificates, or credentials.

```bash
eko ai security
```

**Example Output:**
```text
🤖 AI Security Scan
──────────────────────────────────────────────────
🔴 1 Vulnerability / Secret Found:
  ✗ internal/db/db.go:L14
    - Hardcoded secret found: "postgres://admin:secretPass123@localhost:5432/eko"
    - Action: Move credentials to environment variables or use a secret manager.
```

### 6. `eko ai pr` (GitHub Pull Request Generator)
Generates an optimized, beautifully formatted GitHub Pull Request description in Markdown based on the workspace changeset.

```bash
eko ai pr
```

**Example Output:**
```markdown
# 🚀 Pull Request: Refactored DB Transaction Pool

## 📝 Description
This PR optimizes database connection pool utilization and introduces connection caching helpers.

## 🛠️ Changes
- **internal/db/db.go**: Added pre-compiled SQL statement cache.
- **internal/db/db_test.go**: Added concurrent thread test cases.

## 📊 Impact & Risk
- Database contention reduced.
- Risk: Low-Medium (database pool sizes updated).
```
