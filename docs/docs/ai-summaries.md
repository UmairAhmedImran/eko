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
