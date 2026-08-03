# 🏛️ Eko Architecture & Systems Design

This document provides a deep, comprehensive visual tour of **Eko's** system architecture, concurrency engine, storage schemas, and execution workflows.

---

## 1. High-Level System Architecture

Eko is structured into three decoupled layers: **CLI Command Layer**, **Core Systems & AI Engines**, and **Storage & Persistence Layer**.

```mermaid
graph TD
    subgraph CLI ["1. CLI Command Layer (cmd/)"]
        RootCmd["root.go (Cobra Engine)"]
        InitCmd["init.go (eko init)"]
        SaveCmd["save.go (eko save --ai)"]
        RestoreCmd["restore.go (eko restore)"]
        HistoryCmd["history.go (eko history)"]
        SummaryCmd["summary.go (eko summary)"]
    end

    subgraph Core ["2. Core Engines & Utilities (internal/)"]
        SnapshotEng["Snapshot Engine\n(internal/snapshot/snapshot.go)"]
        FSEngine["Worker-Pool FS Engine\n(internal/util/fs.go)"]
        DiffEngine["Diff & Comparison Engine\n(internal/api/diff.go)"]
        AIEngine["AI Summary Engine\n(internal/ai/)"]
    end

    subgraph Storage ["3. Persistence Layer (.eko/)"]
        SQLiteDB[("SQLite Database\n.eko/db.sqlite")]
        SnapshotDir["Snapshot Storage\n.eko/snapshots/<id>/"]
        EnvState[".eko_env_vars.json"]
    end

    %% Wiring
    RootCmd --> InitCmd
    RootCmd --> SaveCmd
    RootCmd --> RestoreCmd
    RootCmd --> HistoryCmd
    RootCmd --> SummaryCmd

    InitCmd -->|Initialize Schema| SQLiteDB
    SaveCmd -->|Create State| SnapshotEng
    SaveCmd -->|Generate Summary| AIEngine
    RestoreCmd -->|Restore State| SnapshotEng
    HistoryCmd -->|Query Log| SQLiteDB
    SummaryCmd -->|Compute Diff| DiffEngine
    SummaryCmd -->|Synthesize Summary| AIEngine

    SnapshotEng -->|Parallel Copy| FSEngine
    SnapshotEng -->|Record Metadata| SQLiteDB
    FSEngine -->|Write Files| SnapshotDir
    SnapshotEng -->|Serialize Env| EnvState
    AIEngine -->|Read File Pair Diffs| DiffEngine
    AIEngine -->|Persist Summary| SQLiteDB
```

---

## 2. Component Cell Diagrams

### Cell Diagram 1: Worker-Pool Concurrency Engine (`internal/util/fs.go`)

Eko processes directory copying using a hybrid **Serial Directory Tree Walk + Parallel Worker Pool** design.
- **Why Serial Walk?** Walking the directory structure serially guarantees that parent directories are created on disk *before* parallel workers try to write files into them, preventing filesystem race conditions.
- **Why Worker Pool?** A fixed pool of `runtime.NumCPU()` worker goroutines saturates both CPU cores and disk I/O bandwith.

```mermaid
graph LR
    subgraph TreeWalker ["1. Serial Tree Walk (Main Goroutine)"]
        Walk["filepath.Walk(src)"]
        Filter{"ShouldIgnore()?"}
        Mkdir["os.MkdirAll(target, 0755)\n(Synchronous)"]
    end

    subgraph Queue ["2. Task Channel"]
        TaskChan["chan copyTask\n(Buffer: NumCPU * 2)"]
    end

    subgraph WorkerPool ["3. Worker Pool (NumCPU Workers)"]
        W1["Worker Goroutine 1"]
        W2["Worker Goroutine 2"]
        W3["Worker Goroutine N"]
    end

    subgraph ErrorBus ["4. Thread-Safe Error Channel"]
        ErrChan["chan error\n(Buffer: NumCPU)"]
    end

    Walk --> Filter
    Filter -->|No| Mkdir
    Mkdir -->|Enqueue File Copy| TaskChan
    Filter -->|Yes| Skip["Skip Dir/File"]

    TaskChan --> W1
    TaskChan --> W2
    TaskChan --> W3

    W1 -->|Copy Failure| ErrChan
    W2 -->|Copy Failure| ErrChan
    W3 -->|Copy Failure| ErrChan

    ErrChan -->|Bail & Abort Walk| Walk
```

---

### Cell Diagram 2: Lock-Free Atomic Compare-And-Swap (CAS) Restore (`internal/snapshot/snapshot.go`)

When `eko restore` is executed, existing workspace entries must be removed before the snapshot files are copied back.
To perform parallel deletion safely without mutex lock contention, Eko utilizes Go 1.19+ **`atomic.Pointer[error]`** lock-free Compare-And-Swap (CAS).

```mermaid
sequenceDiagram
    autonumber
    participant Main as Restore Main Goroutine
    participant WG as sync.WaitGroup
    participant G1 as Worker Goroutine 1 (file A)
    participant G2 as Worker Goroutine 2 (file B)
    participant CAS as atomic.Pointer[error]

    Main->>Main: Read top-level workspace entries (excluding .eko)
    Main->>WG: Add(N) goroutines
    Main->>G1: Spawn os.RemoveAll("fileA")
    Main->>G2: Spawn os.RemoveAll("fileB")

    alt G1 encounters permission error
        G1->>CAS: CompareAndSwap(nil, &err1) -> SUCCESS (Stores err1)
    end

    alt G2 encounters disk error later
        G2->>CAS: CompareAndSwap(nil, &err2) -> FAILS (err1 is already stored)
    end

    G1->>WG: Done()
    G2->>WG: Done()
    WG->>Main: Wait() finishes
    Main->>CAS: Load()
    Note over Main: Returns first error (err1). Short-circuits restore phase!
```

---

### Cell Diagram 3: AI Provider Abstraction Engine (`internal/ai/`)

Eko supports multi-provider LLM summary generation with automatic fallback:
- **`Google Gemini Provider`** (`GEMINI_API_KEY`)
- **`OpenAI / Custom LLM Provider`** (`OPENAI_API_KEY` or `EKO_AI_API_KEY`)
- **`Local Heuristic Provider`** (Offline fallback using diff metrics)

```mermaid
graph TD
    Client["eko summary / eko save --ai"] -->|Request Summary| Engine["GenerateSnapshotSummary()"]
    Engine --> ProviderSelect{"Select Provider?"}

    ProviderSelect -->|--provider gemini| Gemini["GeminiProvider\n(Google Gemini API)"]
    ProviderSelect -->|--provider openai| OpenAI["OpenAIProvider\n(OpenAI / Custom LLM)"]
    ProviderSelect -->|--provider heuristic| Local["HeuristicProvider\n(Offline Rule Engine)"]
    ProviderSelect -->|Auto (Default)| AutoCheck{"API Keys Present?"}

    AutoCheck -->|GEMINI_API_KEY set| Gemini
    AutoCheck -->|OPENAI_API_KEY set| OpenAI
    AutoCheck -->|No API keys| Local

    Gemini -->|Prompt Engineering & JSON Format| Response["Summary Result Struct"]
    OpenAI -->|Prompt Engineering & JSON Format| Response
    Local -->|Extract Added/Deleted Metrics| Response

    Response -->|Update Database| DB[(".eko/db.sqlite")]
```

---

## 3. Storage & Database Schema Cell Diagram

All snapshot records and metadata are persisted locally in `.eko/db.sqlite`:

```mermaid
erDiagram
    SNAPSHOTS {
        TEXT id PK "Random 8-char Hex (e.g. 8c9d1a2f)"
        TEXT message "User Log Message"
        TEXT path "Storage Path (.eko/snapshots/<id>)"
        DATETIME created_at "Creation Timestamp"
        TEXT summary "AI-Generated Change Summary"
    }

    FILESYSTEM {
        DIRECTORY eko_dir ".eko/"
        FILE db_sqlite ".eko/db.sqlite"
        DIRECTORY snapshots ".eko/snapshots/<id>/"
        FILE env_vars ".eko/snapshots/<id>/.eko_env_vars.json"
    }

    SNAPSHOTS ||--|| FILESYSTEM : "References Storage Path"
```

---

## 4. End-to-End Sequence Diagrams

### Sequence 1: `eko save --ai` Execution Flow

```mermaid
sequenceDiagram
    autonumber
    actor Developer
    participant CLI as cmd/save.go
    participant Snap as internal/snapshot
    participant FS as internal/util (Worker Pool)
    participant DB as internal/db (SQLite)
    participant AI as internal/ai

    Developer->>CLI: eko save --ai -m "Added auth feature"
    CLI->>Snap: CreateSnapshot()
    Snap->>Snap: generateID() -> "8c9d1a2f"
    Snap->>FS: CopyDir(".", ".eko/snapshots/8c9d1a2f")
    FS-->>Snap: Success (Snapshot files copied)
    Snap->>Snap: captureEnvVars() -> .eko_env_vars.json
    Snap-->>CLI: Returns ID & Path

    CLI->>DB: INSERT INTO snapshots (id, message, path, summary)
    
    opt --ai flag active
        CLI->>AI: GenerateSnapshotSummary(db, "8c9d1a2f", provider)
        AI->>AI: Build diff pairs vs predecessor snapshot
        AI->>AI: Invoke LLM API / Heuristic Engine
        AI->>DB: SaveSummary(db, "8c9d1a2f", summaryText)
    end

    CLI-->>Developer: "Snapshot saved: 8c9d1a2f \n AI Summary: ..."
```

---

### Sequence 2: `eko restore <snapshot-id>` Execution Flow

```mermaid
sequenceDiagram
    autonumber
    actor Developer
    participant CLI as cmd/restore.go
    participant DB as internal/db (SQLite)
    participant Snap as internal/snapshot
    participant FS as internal/util

    Developer->>CLI: eko restore 8c9d1a2f
    CLI->>DB: SELECT path FROM snapshots WHERE id = '8c9d1a2f'
    DB-->>CLI: Returns path ".eko/snapshots/8c9d1a2f"

    CLI->>Snap: RestoreSnapshot(".eko/snapshots/8c9d1a2f")
    
    Note over Snap: Phase 1: Parallel Delete Workspace
    Snap->>FS: RemoveAll() top-level entries (except .eko)
    FS-->>Snap: Verify no deletion errors via atomic CAS

    Note over Snap: Phase 2: Parallel Copy Snapshot
    Snap->>FS: CopyDir(".eko/snapshots/8c9d1a2f", ".")
    FS-->>Snap: Success

    Note over Snap: Phase 3: Restore Shell Environment
    Snap->>Snap: restoreEnvVars() -> Generate .eko_env_restore.sh

    Snap-->>CLI: Restoration Complete
    CLI-->>Developer: "Restored: 8c9d1a2f"
```
