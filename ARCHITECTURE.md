# 🏛️ Eko Architecture & Systems Design

This document is a comprehensive, ground-truth architectural reference for **Eko** — derived directly from the source code. It covers every layer: CLI, concurrency engines, AI abstraction, storage schema, CI/CD, data flows, and error handling contracts.

## Table of Contents

1. [High-Level System Overview](#1-high-level-system-overview)
2. [Package Dependency Graph](#2-package-dependency-graph)
3. [CLI Command Layer](#3-cli-command-layer)
4. [Worker-Pool Concurrency Engine](#4-worker-pool-concurrency-engine)
5. [Lock-Free Atomic Restore (CAS)](#5-lock-free-atomic-restore-cas)
6. [Incremental Hash Cache & Concurrency Control](#6-incremental-hash-cache--concurrency-control)
7. [AI Provider Abstraction & Fallback Chain](#7-ai-provider-abstraction--fallback-chain)
8. [Diff & ChangeSet Engine](#8-diff--changeset-engine)
9. [Storage Layer & SQLite Schema](#9-storage-layer--sqlite-schema)
10. [Filesystem Layout](#10-filesystem-layout)
11. [End-to-End Sequence: `eko save --ai`](#11-end-to-end-sequence-eko-save---ai)
12. [End-to-End Sequence: `eko restore`](#12-end-to-end-sequence-eko-restore)
13. [End-to-End Sequence: `eko clean` & Garbage Collection](#13-end-to-end-sequence-eko-clean--garbage-collection)
14. [Error Handling & Fallback Strategy](#14-error-handling--fallback-strategy)
15. [CI/CD Pipeline](#15-cicd-pipeline)
16. [Environment Variable Lifecycle](#16-environment-variable-lifecycle)
17. [High-Performance Storage & Zero-Copy Reflinks](#17-high-performance-storage--zero-copy-reflinks)

---

## 1. High-Level System Overview

Eko is structured into four decoupled layers that communicate only through well-defined interfaces.

```mermaid
graph TD
    subgraph USER["👤 Developer"]
        Terminal["Terminal / Shell"]
    end

    subgraph CLI["1️⃣  CLI Layer  (cmd/)"]
        Root["root.go\nCobra Engine"]
        Init["init.go"]
        Save["save.go\n--ai --provider --message"]
        Restore["restore.go"]
        History["history.go\n--json --verbose --format"]
        Summary["summary.go\n--provider --json --save"]
        Clean["clean.go\n--keep --dry-run"]
        Migrate["migrate.go\n--dry-run"]
        Tag["tag.go\n&lt;id&gt; &lt;tag-name&gt;"]
    end

    subgraph Core["2️⃣  Core Engines  (internal/)"]
        SnapEng["snapshot.go\nCreateSnapshot()\nRestoreSnapshot()"]
        CASEng["objects/store.go\nPut() / Get()\nExtractTo() / RestoreTree()"]
        ManifestEng["manifest/manifest.go\nWrite() / Read()\nAllHashes()"]
        CacheEng["cache/hashcache.go\nLookup() / Store()"]
        FSEng["util/fs.go\nCopyDir()\nShouldIgnore()"]
        DiffEng["api/diff.go\nBuildDiff()"]
        AIEng["ai/summary.go\nGenerateSnapshotSummary()"]
        ProviderAbs["ai/provider.go\nProvider interface\nGetProvider()"]
    end

    subgraph Providers["3️⃣  AI Providers  (internal/ai/)"]
        Gemini["GeminiProvider\nGEMINI_API_KEY"]
        OpenAI["OpenAIProvider\nOPENAI_API_KEY\nEKO_AI_ENDPOINT\nEKO_AI_MODEL"]
        Heuristic["HeuristicProvider\n(offline / no key)"]
    end

    subgraph Storage["4️⃣  Persistence Layer  (.eko/)"]
        DB[("db.sqlite\nSQLite WAL + hash_cache")]
        ObjStore[".eko/objects/&lt;prefix&gt;/&lt;hash&gt;.gz\nContent-Addressable Storage"]
        ManifestDir[".eko/manifests/&lt;id&gt;.json\nLightweight tree manifest"]
        EnvFile[".eko_env_vars.json\nShell environment dump"]
        RestoreSh[".eko_env_restore.sh\nGenerated on restore"]
    end

    Terminal -->|"eko &lt;command&gt;"| Root
    Root --> Init & Save & Restore & History & Summary & Clean & Migrate

    Save --> SnapEng --> CacheEng --> DB
    SnapEng --> CASEng --> ObjStore
    SnapEng --> ManifestEng --> ManifestDir
    Save --> AIEng --> ProviderAbs
    ProviderAbs --> Gemini & OpenAI & Heuristic
    Save -->|"INSERT"| DB

    Restore --> SnapEng
    SnapEng -->|"Phase 1: RemoveAll"| FSEng
    SnapEng -->|"Phase 2: RestoreTree"| CASEng
    SnapEng --> RestoreSh

    History & Summary -->|"SELECT"| DB
    Summary --> DiffEng --> AIEng
    Clean -->|"DELETE manifests + Mark&Sweep GC"| DB & ManifestDir & ObjStore
    Migrate -->|"Convert legacy dirs to CAS"| CASEng & ManifestEng
```

---

## 2. Package Dependency Graph

Shows which packages import which — no circular dependencies.

```mermaid
graph LR
    subgraph cmd
        root["cmd/root.go"]
        save["cmd/save.go"]
        restore["cmd/restore.go"]
        history["cmd/history.go"]
        summary["cmd/summary.go"]
        clean["cmd/clean.go"]
        initcmd["cmd/init.go"]
    end

    subgraph internal
        snap["internal/snapshot"]
        fsutil["internal/util"]
        dbpkg["internal/db"]
        aipkg["internal/ai"]
        apipkg["internal/api"]
    end

    subgraph stdlib
        osexec["os / filepath"]
        sync["sync / sync/atomic"]
        database["database/sql"]
        net["net/http"]
        crypto["crypto/rand"]
        json["encoding/json"]
    end

    subgraph ext["External"]
        cobra["github.com/spf13/cobra"]
        sqlite3["github.com/mattn/go-sqlite3"]
    end

    root --> cobra
    save --> snap & dbpkg & aipkg & cobra
    restore --> snap & dbpkg & cobra
    history --> dbpkg & apipkg & cobra
    summary --> dbpkg & aipkg & apipkg & cobra
    clean --> sqlite3 & cobra
    initcmd --> dbpkg & cobra

    snap --> fsutil & json & crypto & osexec & sync
    fsutil --> osexec
    dbpkg --> database & sqlite3
    aipkg --> apipkg & net & json & osexec
    apipkg --> fsutil & osexec
```

---

## 3. CLI Command Layer

Every subcommand, its flags, and what internal function it calls.

```mermaid
flowchart TD
    Root["eko (root)\ncmd/root.go"]

    Root --> Init["eko init\n─────────────\ndb.InitDB()\ndb.MigrateDB()"]
    Root --> Save["eko save\n─────────────\n-m / --message string\n-a / --ai bool\n--provider string\n[auto|heuristic|openai|gemini]"]
    Root --> Restore["eko restore &lt;id&gt;\n─────────────\nsnapshot.RestoreSnapshot(path)"]
    Root --> History["eko history\n─────────────\n--json bool\n--verbose bool\n--format string\n[text|json|md|csv|html]"]
    Root --> Summary["eko summary [id1] [id2]\n─────────────\n--provider string\n--json bool\n--save bool"]
    Root --> Clean["eko clean\n─────────────\n--keep int (default 10)\n--dry-run bool"]

    Save -->|"1. CreateSnapshot()"| SnapEng["snapshot engine"]
    Save -->|"2. GenerateSnapshotSummary()"| AIEng["AI engine"]
    Save -->|"3. INSERT INTO snapshots"| DB[("db.sqlite")]

    Restore -->|"SELECT path WHERE id=?"| DB
    Restore -->|"RestoreSnapshot(path)"| SnapEng

    History -->|"SELECT * FROM snapshots\nORDER BY created_at DESC"| DB
    Summary -->|"SELECT 2 snapshot paths"| DB
    Summary -->|"BuildDiff(from, to)"| DiffEng["diff engine"]
    Summary -->|"GenerateSummary(changeset)"| AIEng

    Clean -->|"SELECT id, path ORDER BY created_at DESC"| DB
    Clean -->|"Validate path prefix"| Guard{{"Safety guard:\npath must start with\n.eko/snapshots/"}}
    Guard -->|"os.RemoveAll(dir)"| Disk["Filesystem"]
    Guard -->|"DELETE FROM snapshots WHERE id=?"| DB
```

---

## 4. Worker-Pool Concurrency Engine

`internal/util/fs.go` — `CopyDir()` uses a **serial directory walk + parallel file copy** pattern.

```mermaid
graph TD
    subgraph Main["Main Goroutine — Serial Tree Walk"]
        Walk["filepath.Walk(src)"]
        Filter{"ShouldIgnore(name, isDir)?"}
        MkDir["os.MkdirAll(target, 0755)\n⚡ Synchronous — prevents mkdir races"]
        Symlink["os.Symlink()\n⚡ Synchronous"]
        EarlyBail{"select: errs channel\nnon-blocking drain"}
        Enqueue["tasks ← copyTask{src, dst, mode}"]
    end

    subgraph Chan["Buffered Channels"]
        TaskChan["chan copyTask\nbuffer = runtime.NumCPU() × 2"]
        ErrChan["chan error\nbuffer = runtime.NumCPU()"]
    end

    subgraph Pool["Worker Pool — runtime.NumCPU() goroutines"]
        W1["Worker 1\nos.Open → io.Copy → os.Chmod"]
        W2["Worker 2\nos.Open → io.Copy → os.Chmod"]
        WN["Worker N\nos.Open → io.Copy → os.Chmod"]
    end

    subgraph Drain["Cleanup"]
        CloseTask["close(tasks)"]
        WGWait["wg.Wait()"]
        CloseErr["close(errs)"]
        RetErr["return ←errs\n(nil if empty)"]
    end

    Walk --> Filter
    Filter -->|"isDir=true, not ignored"| MkDir --> Enqueue
    Filter -->|"isSymlink"| Symlink
    Filter -->|"ignored"| Skip["filepath.SkipDir / continue"]
    Filter -->|"isFile"| EarlyBail
    EarlyBail -->|"no error"| Enqueue
    EarlyBail -->|"error already stored"| AbortWalk["return error → abort walk"]

    Enqueue --> TaskChan
    TaskChan --> W1 & W2 & WN
    W1 & W2 & WN -->|"on failure"| ErrChan

    Walk -->|"done"| CloseTask --> WGWait --> CloseErr --> RetErr
```

**Key design decisions:**
| Decision | Reason |
|----------|--------|
| Serial `filepath.Walk` | Parent dirs must exist before workers write into them |
| Buffered task channel `NumCPU×2` | Keeps workers fed without blocking the walker |
| Non-blocking error check in walk | Fails fast — stops enqueuing if a worker already died |
| `close(errs)` drain | `←errs` returns `nil` on empty closed channel safely |

---

## 5. Lock-Free Atomic Restore (CAS)

`internal/snapshot/snapshot.go` — `RestoreSnapshot()` uses `atomic.Pointer[error]` for lock-free first-error capture during parallel deletion.

```mermaid
sequenceDiagram
    autonumber
    participant Main as RestoreSnapshot()
    participant WG as sync.WaitGroup
    participant G1 as Goroutine: RemoveAll("main.go")
    participant G2 as Goroutine: RemoveAll("internal/")
    participant G3 as Goroutine: RemoveAll("go.mod")
    participant CAS as atomic.Pointer[error]
    participant Phase2 as Phase 2: CopyDir()

    Main->>Main: os.ReadDir(".") → entries
    Main->>Main: Filter: skip .eko, skip ShouldIgnore()
    Main->>WG: wg.Add(N)
    Main->>G1: go RemoveAll("main.go")
    Main->>G2: go RemoveAll("internal/")
    Main->>G3: go RemoveAll("go.mod")

    Note over CAS: initially nil

    alt G2 hits permission error
        G2->>CAS: CompareAndSwap(nil, &permErr) → SUCCESS
        Note over CAS: stores &permErr
    end

    alt G3 hits disk error
        G3->>CAS: CompareAndSwap(nil, &diskErr) → FAILS (already set)
        Note over G3: diskErr silently discarded
    end

    G1-->>WG: Done()
    G2-->>WG: Done()
    G3-->>WG: Done()
    WG-->>Main: Wait() returns

    Main->>CAS: Load()
    alt error stored
        CAS-->>Main: return *err  (Phase 2 never runs)
    else nil
        CAS-->>Main: nil → proceed
        Main->>Phase2: util.CopyDir(snapshotPath, ".")
        Phase2-->>Main: success
        Main->>Main: restoreEnvVars() → .eko_env_restore.sh
## 6. Incremental Hash Cache & Concurrency Control

`internal/cache/hashcache.go` & `internal/objects/store.go` — optimizes speed, memory, and thread safety.

```mermaid
flowchart LR
    WalkFile["Walk file: main.go"] --> MutexLock["sync.Mutex Lock\n(Thread-Safe Serializability)"]
    MutexLock --> Lookup{"stmtLookup.QueryRow()\nPre-compiled sql.Stmt"}
    Lookup -->|"Hit"| UseCached["Use cached SHA-256 hash\n(⚡ <1.0ms query)"]
    Lookup -->|"Miss"| ReadFile["Read file using 32KB sync.Pool buffer"]
    ReadFile --> StoreCache["stmtStore.Exec()\nPre-compiled sql.Stmt"]
    StoreCache --> StoreBlob["Store blob in CAS\ngzip.BestCompression (0444)"]
```

### Key Performance Innovations:
1. **Pre-Compiled SQL Statements (`sql.Stmt`)**: Compiles SQL lookup and store queries once per session, cutting warm save latency to **<1.0 ms**.
2. **In-Memory Manifest LRU Cache (`internal/manifest/cache.go`)**: Caches parsed snapshot manifests in a thread-safe LRU cache (capacity: 50 manifests), reducing repeated `manifest.Read()` operations down to **~24.26 ns/op** RAM access.
3. **Mutex Serialization (`sync.Mutex`)**: Prevents SQLite lock contention during multi-threaded worker pool operations.
4. **`sync.Pool` Buffer Reuse**: Reuses 32KB byte read buffers to eliminate Go heap memory allocations and GC pause jitter.
5. **Maximal Compression (`gzip.BestCompression`)**: Compresses all stored object blobs for maximum disk space savings.

---

## 7. AI Provider Abstraction & Fallback Chain

`internal/ai/provider.go` — `GetProvider()` and all three concrete providers.

```mermaid
flowchart TD
    Client["eko save --ai\neko summary --provider X"] -->|"providerName string"| GetProvider

    GetProvider{"GetProvider(providerName)"}

    GetProvider -->|"'gemini'"| GeminiP
    GetProvider -->|"'openai'"| OpenAIP
    GetProvider -->|"'heuristic' / 'offline' / 'mock'"| HeuristicP
    GetProvider -->|"'auto' / empty"| AutoCheck

    AutoCheck{"Env key check"}
    AutoCheck -->|"GEMINI_API_KEY set"| GeminiP["GeminiProvider\n─────────────────\nPOST generativelanguage.googleapis.com\nModel: gemini-1.5-flash\nFormatPatchSnippet(cs, 4000)\nJSON response parse"]
    AutoCheck -->|"OPENAI_API_KEY or EKO_AI_API_KEY set"| OpenAIP["OpenAIProvider\n─────────────────\nPOST EKO_AI_ENDPOINT/chat/completions\nModel: EKO_AI_MODEL (gpt-4o-mini)\nBearer auth header\nHTTP timeout: 15s"]
    AutoCheck -->|"no keys found"| HeuristicP["HeuristicProvider\n─────────────────\nPure local computation\nNo network / no API key\nCounts added/modified/deleted files\ncategorizeFiles() by extension"]

    GeminiP -->|"HTTP error / non-200"| FallH["→ HeuristicProvider fallback"]
    OpenAIP -->|"no key / HTTP error / non-200"| FallH

    GeminiP & OpenAIP & HeuristicP --> Result["Summary string\nStored in DB via db.SaveSummary()"]

    subgraph ChangeSet["ChangeSet struct (ai/summary.go)"]
        CS["Diffs []DiffFile\nAddedFiles []string\nModifiedFiles []string\nDeletedFiles []string\nTotalInsertions int\nTotalDeletions int"]
    end

    Result -.->|"built from"| CS
```

---

## 8. Diff & ChangeSet Engine

`internal/api/diff.go` + `internal/ai/summary.go` — how file diffs are computed and passed to the AI layer.

```mermaid
flowchart LR
    subgraph Input
        FromDir[".eko/snapshots/&lt;prev-id&gt;/\n(or empty string for first snapshot)"]
        ToDir[".eko/snapshots/&lt;new-id&gt;/"]
    end

    subgraph BuildDiff["api.BuildDiff(fromDir, toDir)"]
        WalkFrom["filepath.Walk(fromDir)\ncollect relative paths → seen map"]
        WalkTo["filepath.Walk(toDir)\ncollect relative paths → seen map"]
        Compare["For each rel path in seen:\nreadFileSafe(from/rel)\nreadFileSafe(to/rel)\nif orig == mod → skip"]
        Output["[]DiffFile{\n  Name: rel path\n  Original: file content\n  Modified: file content\n}"]
    end

    subgraph BuildChangeSet["ai.BuildChangeSet(diffs)"]
        Classify{"For each DiffFile"}
        Added["Original='' && Modified≠''\n→ AddedFiles"]
        Deleted["Original≠'' && Modified=''\n→ DeletedFiles"]
        Modified["Both non-empty\n→ ModifiedFiles\nCount +/- lines"]
        CountLines["TotalInsertions\nTotalDeletions\ncounted from line diffs"]
    end

    subgraph FormatPatch["FormatPatchSnippet(cs, maxLen)"]
        Patch["Builds unified diff text\ntruncated to maxLen chars\nfor LLM prompt"]
    end

    FromDir & ToDir --> BuildDiff
    WalkFrom & WalkTo --> Compare --> Output

    Output --> BuildChangeSet
    Classify --> Added & Deleted & Modified --> CountLines

    BuildChangeSet --> FormatPatch --> LLM["AI Provider\nPrompt + ChangeSet → Summary string"]
```

---

## 9. Storage Layer & SQLite Schema

`internal/db/db.go` — schema, migrations, and all query patterns used across commands.

```mermaid
erDiagram
    SNAPSHOTS {
        TEXT    id          PK  "8-char random hex (crypto/rand)"
        TEXT    message         "User log message (-m flag)"
        TEXT    path            ".eko/snapshots/&lt;id&gt;"
        DATETIME created_at     "DEFAULT CURRENT_TIMESTAMP"
        TEXT    summary         "AI-generated summary (nullable)"
    }
```

**Query patterns by command:**

```mermaid
flowchart LR
    subgraph Reads
        HQ["history:\nSELECT id,message,created_at,summary\nFROM snapshots\nORDER BY created_at DESC"]
        SQ["summary:\nSELECT path FROM snapshots\nORDER BY created_at DESC LIMIT 2"]
        RQ["restore:\nSELECT path FROM snapshots\nWHERE id = ?"]
        CQ["clean:\nSELECT id,path,created_at\nFROM snapshots\nORDER BY created_at DESC"]
        PQ["save (prev):\nSELECT path FROM snapshots\nORDER BY created_at DESC, rowid DESC\nLIMIT 1"]
    end

    subgraph Writes
        IQ["save:\nINSERT INTO snapshots\n(id,message,path,summary)\nVALUES (?,?,?,?)"]
        UQ["summary --save:\nUPDATE snapshots\nSET summary=?\nWHERE id=?"]
        DQ["clean:\nDELETE FROM snapshots\nWHERE id=?"]
    end

    subgraph Schema
        MI["MigrateDB():\nCREATE TABLE IF NOT EXISTS\nALTER TABLE ADD COLUMN summary\n(idempotent — safe on upgrade)"]
    end
```

**SQLite connection modes used by `eko clean`:**

```mermaid
flowchart LR
    Normal["Normal run\nfile:.eko/db.sqlite?mode=rw\nFull read-write\nNo auto-create"]
    DryRun["--dry-run\nfile:.eko/db.sqlite?mode=ro&_query_only=true\nRead-only at connection level\nZero bytes written"]

    Clean["eko clean"] -->|"--dry-run=false"| Normal
    Clean -->|"--dry-run=true"| DryRun
```

---

## 10. Filesystem Layout

Complete map of every file Eko reads, writes, or manages.

```mermaid
graph TD
    Root["project-root/"]

    Root --> EkoDir[".eko/"]
    Root --> WorkDir["... project files ..."]
    Root --> RestoreSh[".eko_env_restore.sh\n(written on eko restore)\nChmod 0755"]

    EkoDir --> DBFile["db.sqlite\nSQLite WAL database\nAll snapshot metadata"]
    EkoDir --> SnapsDir["snapshots/"]

    SnapsDir --> S1["&lt;8-char-id-1&gt;/\n── full project tree copy\n── .eko_env_vars.json"]
    SnapsDir --> S2["&lt;8-char-id-2&gt;/\n── full project tree copy\n── .eko_env_vars.json"]
    SnapsDir --> SN["&lt;8-char-id-N&gt;/ ..."]

    subgraph EnvFiles["Environment files (per snapshot)"]
        EV[".eko_env_vars.json\n{ KEY: VALUE, ... }\nJSON, chmod 0600\nWritten by captureEnvVars()"]
    end

    S1 --> EV

    subgraph Ignored["Always ignored by CopyDir / ShouldIgnore()"]
        IG1[".eko/"]
        IG2[".git/"]
        IG3["node_modules/"]
        IG4["*.exe, *.dll, *.so, *.dylib"]
        IG5["*.zip, *.tar, *.gz, *.rar"]
        IG6[".DS_Store, Thumbs.db"]
        IG7["__pycache__/, *.pyc"]
    end
```

---

## 11. End-to-End Sequence: `eko save --ai`

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Developer
    participant CLI as cmd/save.go
    participant DB  as internal/db
    participant Snap as internal/snapshot
    participant FS  as internal/util/fs.go
    participant Diff as internal/api/diff.go
    participant AI  as internal/ai

    Dev->>CLI: eko save --ai -m "added auth module"

    CLI->>DB: InitDB() → open .eko/db.sqlite, MigrateDB()
    CLI->>DB: SELECT path FROM snapshots ORDER BY created_at DESC LIMIT 1
    DB-->>CLI: prevPath = ".eko/snapshots/3b7f2a1e"

    CLI->>Snap: CreateSnapshot()
    Snap->>Snap: generateID() → crypto/rand → "8c9d1a2f"
    Snap->>FS: CopyDir(".", ".eko/snapshots/8c9d1a2f")

    Note over FS: Serial walk + NumCPU workers
    FS-->>Snap: ✓ all files copied

    Snap->>Snap: captureEnvVars(".eko/snapshots/8c9d1a2f")
    Note over Snap: os.Environ() → JSON → .eko_env_vars.json (chmod 0600)
    Snap-->>CLI: id="8c9d1a2f", path=".eko/snapshots/8c9d1a2f"

    CLI->>Diff: BuildDiff(".eko/snapshots/3b7f2a1e", ".eko/snapshots/8c9d1a2f")
    Diff-->>CLI: []DiffFile (changed file pairs)

    CLI->>AI: GenerateSnapshotSummary(ctx, prevPath, newPath, "auto")
    AI->>AI: BuildChangeSet(diffs) → ChangeSet{Added, Modified, Deleted, +/- lines}
    AI->>AI: GetProvider("auto") → check GEMINI_API_KEY / OPENAI_API_KEY
    AI->>AI: FormatPatchSnippet(cs, 4000) → truncated diff text

    alt Gemini key present
        AI->>AI: GeminiProvider.GenerateSummary(ctx, cs)
        Note over AI: POST generativelanguage.googleapis.com
    else OpenAI key present
        AI->>AI: OpenAIProvider.GenerateSummary(ctx, cs)
        Note over AI: POST EKO_AI_ENDPOINT/chat/completions
    else No keys
        AI->>AI: HeuristicProvider.GenerateSummary(ctx, cs)
        Note over AI: Local rule-based (no network)
    end

    AI-->>CLI: summaryText = "Added auth module with JWT..."

    CLI->>DB: INSERT INTO snapshots(id, message, path, summary) VALUES(...)
    DB-->>CLI: ✓ row inserted

    CLI-->>Dev: Snapshot saved: 8c9d1a2f\nAI Summary: Added auth module with JWT...
```

---

## 12. End-to-End Sequence: `eko restore`

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Developer
    participant CLI as cmd/restore.go
    participant DB  as internal/db
    participant Snap as internal/snapshot
    participant CAS as internal/objects
    participant FS  as Working Directory

    Dev->>CLI: eko restore 8c9d1a2f

    CLI->>DB: InitDB()
    CLI->>DB: SELECT path FROM snapshots WHERE id = '8c9d1a2f'
        Note over Snap: Read .eko_env_vars.json → write .eko_env_restore.sh
        Snap-->>CLI: nil

        CLI-->>Dev: Restored: 8c9d1a2f
    end
```

---

## 13. End-to-End Sequence: `eko clean`

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Developer
    participant CLI as cmd/clean.go
    participant DB  as SQLite
    participant Guard as Safety Validator
    participant FS  as os.RemoveAll

    Dev->>CLI: eko clean --keep 5 --dry-run

    CLI->>DB: openCleanDB(readOnly=true)\nDSN: mode=ro&_query_only=true

    CLI->>DB: SELECT id, path, created_at\nFROM snapshots\nORDER BY created_at DESC

    Note over CLI: First 5 rows = KEEP\nRemaining rows = candidates

    loop For each candidate row
        CLI->>Guard: Validate path prefix
        Note over Guard: path MUST start with ".eko/snapshots/"
        Guard->>Guard: filepath.Abs(path)\nstrings.HasPrefix check

        alt path is suspicious (not in .eko/snapshots/)
            Guard-->>CLI: abort — refuse to delete
        else path missing on disk
            Note over CLI: mark missing=true\nskip RemoveAll\nstill delete DB row
        else valid path on disk
            alt dry-run = true
                CLI-->>Dev: [DRY RUN] would delete: &lt;id&gt; (&lt;date&gt;)
            else dry-run = false
                CLI->>FS: os.RemoveAll(".eko/snapshots/&lt;id&gt;")
                FS-->>CLI: ✓
                CLI->>DB: DELETE FROM snapshots WHERE id = ?
                DB-->>CLI: ✓
            end
        end
    end

    CLI-->>Dev: Removed N snapshot(s). Kept 5.
```

---

## 14. Error Handling & Fallback Strategy

```mermaid
flowchart TD
    subgraph AI["AI Layer Fallbacks"]
        Req["API Request"] --> NetOK{"HTTP OK?"}
        NetOK -->|"200"| Parse["Parse JSON response"]
        NetOK -->|"non-200 / timeout / no key"| Fallback["→ HeuristicProvider\n(always succeeds offline)"]
        Parse -->|"malformed"| Fallback
    end

    subgraph FS["Filesystem Layer"]
        Copy["CopyDir()"] --> WErr{"Walk error?"}
        WErr -->|"yes"| RetWalk["return walkErr (priority)"]
        WErr -->|"no"| WrkErr{"Worker error?"}
        WrkErr -->|"yes"| RetWork["return first worker err"]
        WrkErr -->|"no"| RetNil["return nil"]
    end

    subgraph Restore["Restore Layer"]
        Phase1["Phase 1: Parallel Delete"] --> CASErr{"firstErr.Load()?"}
        CASErr -->|"non-nil"| AbortP2["return error\nPhase 2 never runs\n(workspace partially deleted\nbut .eko intact)"]
        CASErr -->|"nil"| Phase2["Phase 2: CopyDir()"]
    end

    subgraph Clean["Clean Safety Guards"]
        PathCheck{"path starts with\n.eko/snapshots/?"} -->|"no"| HardAbort["return error\nzero deletions"]
        PathCheck -->|"yes"| DirCheck{"dir exists?"}
        DirCheck -->|"missing"| StaleRow["delete DB row only\nskip RemoveAll"]
        DirCheck -->|"exists"| SafeDel["RemoveAll + DELETE"]
    end

    subgraph DB["DB Layer"]
        Init["InitDB()"] --> Open["sql.Open()"]
        Open --> Migrate["MigrateDB()\nCREATE TABLE IF NOT EXISTS\nALTER TABLE ADD COLUMN\n(idempotent)"]
        Migrate -->|"error"| Warn["log.Printf warning\ndo NOT fatal — allow degraded mode"]
    end
```

---

## 15. CI/CD Pipeline

```mermaid
flowchart TD
    subgraph Triggers["Triggers"]
        Push["git push → any branch ≠ main"]
        PushMain["git push → main"]
        PR["Pull Request → main"]
        IssueEvent["Issue opened/closed/labeled"]
        TagPush["git tag v*"]
        Manual["workflow_dispatch\n(manual trigger)"]
    end

    subgraph AutoPR[".github/workflows/auto-pr.yml"]
        APR1["Checkout"]
        APR2["Derive PR title\nfrom branch name\nfeat/foo → feat: foo"]
        APR3{"PR already\nexists?"}
        APR4["gh pr create\n--base main\n--title ...\n--body checklist"]
        APR5["Skip"]
    end

    subgraph CI[".github/workflows/ci.yml"]
        CI1["actions/checkout@v4"]
        CI2["actions/setup-go@v5\ngo-version: 1.26"]
        CI3["go build -v ./..."]
        CI4["go test -v ./..."]
    end

    subgraph Badge[".github/workflows/update-badge.yml"]
        B1["gh issue list\n--label 'good first issue'\n--state open\n--jq length"]
        B2{"Badge count\nchanged?"}
        B3["sed -i replace count\nin README.md"]
        B4["git commit [skip ci]\ngit push"]
        B5["No-op"]
    end

    subgraph Release[".github/workflows/release.yml"]
        R1["goreleaser\nCross-platform binaries\nlinux/darwin/windows\namd64/arm64"]
        R2["GitHub Release\nwith checksums"]
    end

    Push --> AutoPR
    APR1 --> APR2 --> APR3
    APR3 -->|"no"| APR4
    APR3 -->|"yes"| APR5

    PushMain & PR --> CI
    CI1 --> CI2 --> CI3 --> CI4

    IssueEvent & Manual --> Badge
    B1 --> B2 -->|"changed"| B3 --> B4
    B2 -->|"same"| B5

    TagPush --> Release
    R1 --> R2
```

---

## 16. Environment Variable Lifecycle

How shell environment is captured, stored, and restored across `eko save` / `eko restore`.

```mermaid
sequenceDiagram
    autonumber
    participant Shell as Developer Shell
    participant Save  as eko save
    participant JSON  as .eko_env_vars.json
    participant Restore as eko restore
    participant Script as .eko_env_restore.sh

    Note over Shell: export DB_URL=postgres://...\nexport API_KEY=secret123

    Shell->>Save: eko save

    Save->>Save: os.Environ() → []string{"DB_URL=postgres://...", ...}
    Save->>Save: Parse KEY=VALUE pairs\nbuild map[string]string

    Save->>JSON: json.MarshalIndent(envMap)\nos.WriteFile(..., 0600)
    Note over JSON: {\n  "DB_URL": "postgres://...",\n  "API_KEY": "secret123"\n}

    Note over Shell: ... time passes, env changes ...

    Shell->>Restore: eko restore 8c9d1a2f

    Restore->>JSON: os.ReadFile(.eko_env_vars.json)
    JSON-->>Restore: envMap

    Restore->>Script: OpenFile(".eko_env_restore.sh", 0755)
    Note over Script: #!/bin/sh\n# Eko Shell Environment Restore Script\nexport DB_URL='postgres://...'\nexport API_KEY='secret123'
    Note over Script: Values escaped: ' → '\''

    Restore-->>Shell: Restored: 8c9d1a2f

    Shell->>Script: source .eko_env_restore.sh
    Note over Shell: Environment fully restored ✓
```

---

*Architecture document generated from live source code — `internal/snapshot/snapshot.go`, `internal/util/fs.go`, `internal/db/db.go`, `internal/ai/provider.go`, `internal/api/diff.go`, `cmd/*.go`.*

---

## 17. High-Performance Storage & Zero-Copy Reflinks

Eko utilizes high-throughput storage engines and platform-specific system calls to minimize both I/O latency and disk usage.

```mermaid
flowchart TD
    subgraph Save["eko save (Compression Engine)"]
        File["File input"] --> ExtCheck{"Is already binary\nor compressed?"}
        ExtCheck -->|"Yes (.png, .zip, etc.)"| RawStore["Store Raw (.raw)\nBypass compression CPU cycles"]
        ExtCheck -->|"No (text/code)"| ZstdStore["ZSTD Compression (.zst)\n3x faster decompression than Gzip"]
    end

    subgraph Restore["eko restore (Extraction Engine)"]
        ObjFile["Stored Object"] --> ObjType{"Stored as .raw\nor .zst?"}
        ObjType -->|".zst"| Decompress["Decompress normally\nRecycle decoders via sync.Pool"]
        ObjType -->|".raw"| Reflink{"CoW Reflink supported\nby platform/volume?"}
        Reflink -->|"Yes (APFS / btrfs / xfs)"| Clone["OS Reflink Clone\nconstant-time ~0.12 ms"]
        Reflink -->|"No"| RawCopy["Standard CopyFile\nFast block-by-block copy"]
    end
```

### Key Performance Architectures:
1. **Adaptive ZSTD Compression**: Integrates `github.com/klauspost/compress/zstd` to achieve higher compression ratios and faster decompression speeds. Small files are constrained to single-threaded modes, and larger files utilize multi-threaded decompression.
2. **sync.Pool Decoder Recycling**: Reuses `*zstd.Decoder` states in a global `sync.Pool`, eliminating expensive buffer allocation overhead on every single read.
3. **OS-Level Copy-on-Write (CoW) Reflinks**: Uses the **Strategy Pattern** to select file cloning capabilities at runtime:
   - macOS (APFS): `clonefile(2)`
   - Linux (btrfs, xfs): `ioctl(FICLONE)`
   - Fallback: standard optimized byte copy.
   Restores uncompressed `.raw` files in **~128 microseconds** (70.8x faster than standard copies).
