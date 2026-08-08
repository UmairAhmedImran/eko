---
sidebar_position: 4
title: 🏛️ Architecture & Design
---

# 🏛️ Eko System Architecture & Design

This document details **Eko's** underlying software architecture, component cell diagrams, concurrency model, and data storage workflows.

---

## 1. High-Level Architecture Overview

Eko is composed of three core architectural layers:

1. **CLI Command Layer (`cmd/`)**: Built on the Go Cobra framework (`init`, `save`, `restore`, `history`, `summary`, `clean`, `migrate`).
2. **Engine & Utility Layer (`internal/`)**:
   - `snapshot`: Orchestrates snapshot creation, manifest writing, env serialization, and atomic directory restores.
   - `objects`: Content-Addressable Storage (CAS) engine (`.eko/objects/<prefix>/<hash>.gz`), gzip compression, atomic writes, and mark-and-sweep garbage collection.
   - `manifest`: Lightweight JSON snapshot manifests (`.eko/manifests/<id>.json`).
   - `cache`: Incremental SQLite hash cache (`hash_cache` table in `db.sqlite`) to skip reading unchanged files.
   - `util`: Worker-pool directory copy engine and thread-safe error reporting.
   - `api`: File diff and workspace change calculators.
   - `ai`: Multi-provider AI change summary generator (Gemini, OpenAI, Heuristic fallback).
3. **Persistence Layer (`.eko/`)**: Local SQLite database (`db.sqlite`), CAS object store (`objects/`), and snapshot tree manifests (`manifests/`).

```mermaid
graph TD
    subgraph CLI ["1. CLI Command Layer (cmd/)"]
        RootCmd["root.go (Cobra Engine)"]
        InitCmd["init.go (eko init)"]
        SaveCmd["save.go (eko save --ai)"]
        RestoreCmd["restore.go (eko restore)"]
        HistoryCmd["history.go (eko history)"]
        SummaryCmd["summary.go (eko summary)"]
        CleanCmd["clean.go (eko clean)"]
        MigrateCmd["migrate.go (eko migrate)"]
    end

    subgraph Core ["2. Core Engines & Utilities (internal/)"]
        SnapshotEng["Snapshot Engine\n(internal/snapshot/)"]
        CASEngine["CAS Object Store\n(internal/objects/)"]
        ManifestEngine["Manifest Engine\n(internal/manifest/)"]
        CacheEngine["Hash Cache Engine\n(internal/cache/)"]
        FSEngine["Worker-Pool FS Engine\n(internal/util/fs.go)"]
        DiffEngine["Diff & Comparison Engine\n(internal/api/diff.go)"]
        AIEngine["AI Summary Engine\n(internal/ai/)"]
    end

    subgraph Storage ["3. Persistence Layer (.eko/)"]
        SQLiteDB[("SQLite Database\n.eko/db.sqlite")]
        CASObjects["CAS Object Store\n.eko/objects/<prefix>/<hash>.gz"]
        Manifests["Tree Manifests\n.eko/manifests/<id>.json"]
        EnvState[".eko_env_vars.json"]
    end

    RootCmd --> InitCmd & SaveCmd & RestoreCmd & HistoryCmd & SummaryCmd & CleanCmd & MigrateCmd

    InitCmd -->|Initialize Schema| SQLiteDB
    SaveCmd -->|Check Hash Cache| CacheEngine
    SaveCmd -->|Store Blobs| CASEngine --> CASObjects
    SaveCmd -->|Write Manifest| ManifestEngine --> Manifests
    SaveCmd -->|Generate Summary| AIEngine
    RestoreCmd -->|Extract Tree| CASEngine
    CleanCmd -->|Garbage Collect Blobs| CASEngine
    MigrateCmd -->|Convert Legacy Dirs| CASEngine & ManifestEngine
    SummaryCmd -->|Compute Diff| DiffEngine
```

---

## 2. Concurrency Worker Pool Cell Diagram

Eko uses a hybrid **Serial Walk + Worker Pool** model:
- **Serial Walk**: Walks source directory tree serially to synchronously create parent target directories (`os.MkdirAll`) before parallel workers begin writing files, preventing directory creation race conditions.
- **Worker Pool**: Spawns `runtime.NumCPU()` worker goroutines to process copy tasks concurrently.

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

## 3. Lock-Free Atomic CAS Restore Sequence

During workspace restoration, existing non-ignored workspace items are deleted in parallel using **`atomic.Pointer[error]`** Compare-And-Swap (CAS) to capture the first error without mutex lock overhead:

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

## 4. AI Provider Strategy Engine

Eko abstracts LLM services behind a clean `Provider` interface:

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
