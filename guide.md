<p align="center">
  <img src="assets/eko-banner.png" alt="Eko Logo" width="550" />
</p>

# Eko User Guide ✦

Eko is an AI snapshot versioning tool designed to capture, inspect, diff, and restore directory states. It can be run either as a lightweight command-line interface (CLI) or as a rich native desktop application.

---

## 1. Installation

Depending on your use case, you can compile Eko in one of three ways:

### Option A: Standalone CLI + UI Binary (Recommended)
This compiles a single binary `eko` containing both the CLI commands and the native visual UI.

1. Build the React/Next.js frontend assets:
   ```bash
   (cd ui && npm run build)
   ```
2. Build the Go binary (requires Go, but does not require the Wails CLI):
   - **On macOS**:
     ```bash
     CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags desktop,production -o eko .
     ```
   - **On Windows / Linux**:
     ```bash
     go build -tags desktop,production -o eko .
     ```
   *You can now run `./eko ui` or any other CLI command directly from this single binary.*

### Option B: Native macOS App Bundle
This packages Eko as a standard double-clickable macOS application bundle.

1. Ensure you have the Wails CLI installed:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```
2. Build the application bundle:
   ```bash
   wails build
   ```
   *Note: This creates a native app bundle at `build/bin/eko.app`. You can copy the executable inside `/build/bin/eko.app/Contents/MacOS/eko` to your system path (e.g., `/usr/local/bin/eko`) to use the `eko` command globally.*

### Option C: Lightweight CLI Only
This compiles only the command-line interface, skipping all GUI/Wails dependencies. Ideal for servers, headless environments, or if you do not want graphical components compiled in.

1. Compile the binary with the `no_gui` build tag:
   ```bash
   go build -tags no_gui -o eko .
   ```
2. Copy the resulting `eko` binary to a folder in your `$PATH` (e.g., `/usr/local/bin/`).

---

## 2. Basic CLI Usage

To use Eko inside any project or directory:

### Step 1: Initialize Eko
Create the hidden SQLite database and backups folder (`.eko/`) in the target directory:
```bash
cd /path/to/your-project
eko init
```
*Output:* `Eko initialized.`

### Step 2: Save a Snapshot
Capture the current state of all files in the directory (excluding the `.eko` folder itself):
```bash
eko save
```
*Output:* `Snapshot saved: <id>` (where `<id>` is a unique 8-character hexadecimal identifier).

You can also auto-generate an AI change summary when saving:
```bash
eko save --ai
```

### Step 3: Generate AI Change Summaries
Inspect AI-generated summaries of changes between snapshots:
```bash
# Summarize changes in the latest snapshot vs predecessor
eko summary

# Summarize changes in a specific snapshot
eko summary 3b7f2a1e

# Summarize changes between two specific snapshots
eko summary 3b7f2a1e 8c9d1a2f

# Output summary as formatted JSON
eko summary --json

# Force a specific provider (auto, heuristic, openai, gemini) and save result to DB
eko summary 3b7f2a1e --provider gemini --save
```

### Step 4: AI Provider Configuration
Eko supports multiple AI summary engines:
- **Auto (default)**: Automatically uses Gemini or OpenAI if credentials are found, otherwise falls back to local heuristic mode.
- **Heuristic / Offline (`-p heuristic`)**: Fast, structured local rule-based analysis (no API key or internet required).
- **OpenAI (`-p openai`)**: Connects to OpenAI API or custom local LLMs (vLLM / Ollama).
- **Gemini (`-p gemini`)**: Connects to Google Gemini API.

#### Environment Variables:
- `GEMINI_API_KEY`: API key for Google Gemini provider.
- `OPENAI_API_KEY` / `EKO_AI_API_KEY`: API key for OpenAI provider.
- `EKO_AI_ENDPOINT`: Custom base URL for OpenAI-compatible LLMs (default: `https://api.openai.com/v1`).
- `EKO_AI_MODEL`: Custom LLM model (default: `gpt-4o-mini` for OpenAI, `gemini-1.5-flash` for Gemini).

### Step 5: View Snapshot History
List all saved snapshots with their creation timestamps and summaries:
```bash
eko history

# Show verbose history with detailed AI summaries
eko history --verbose

# Output history list in JSON format
eko history --json
```

### Step 6: Restore a Previous State
Revert all files in your directory concurrently to the exact state captured in a given snapshot:
```bash
eko restore <snapshot-id>
```
*Example:* `eko restore 3b7f2a1e`
*Output:* `Restored: 3b7f2a1e`

---

## 3. Graphical UI Usage (Wails Only)

If you compiled Eko using **Option A (Desktop App)**, you can launch the native visual memory timeline interface directly from your command line:

```bash
eko ui
```

### Visual UI Features:
- **Interactive Timeline**: Scroll through snapshots in chronological order.
- **Changed Files List**: Inspect exactly how many files and which specific paths were added, modified, or deleted in each snapshot.
- **Monaco Diff Viewer**: Click on any changed file to view side-by-side split diffs with full syntax highlighting.
- **Graphical Restore**: Click the **Restore** button on any snapshot in the timeline or details panel to revert your workspace instantly.
