---
sidebar_position: 3
---

# AI-Powered Change Summaries

Eko provides intelligent, automated change summaries for your snapshot points.

## Generating Summaries

### Summarize Latest Snapshot
```bash
eko summary
```

### Summarize Specific Snapshot
```bash
eko summary <snapshot-id>
```

### Summarize Differences Between Two Snapshots
```bash
eko summary <snapshot-id1> <snapshot-id2>
```

---

## AI Provider Configuration

Eko supports multiple AI summary engines:

- **Auto (default)**: Uses Gemini or OpenAI if API keys are set, falling back to local heuristic mode.
- **Heuristic / Offline (`-p heuristic`)**: Fast, structured local rule-based analysis (no network or API key required).
- **OpenAI (`-p openai`)**: Connects to OpenAI API or custom local LLMs (vLLM / Ollama).
- **Gemini (`-p gemini`)**: Connects to Google Gemini API.

### Environment Variables

| Environment Variable | Description |
| -------------------- | ----------- |
| `GEMINI_API_KEY` | API key for Google Gemini provider |
| `OPENAI_API_KEY` / `EKO_AI_API_KEY` | API key for OpenAI provider |
| `EKO_AI_ENDPOINT` | Custom base URL for OpenAI-compatible REST endpoints |
| `EKO_AI_MODEL` | Custom LLM model name |
