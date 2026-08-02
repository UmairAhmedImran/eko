---
sidebar_position: 1
---

# Introduction to Eko ✦

**Eko** is an AI-powered, high-performance snapshot versioning command-line interface (CLI) written in Go.

It allows developers to capture, compare, and restore local directory states concurrently. Think of Eko as a lightning-fast local "Time Machine" for your codebase.

## Key Features

- **AI-Powered Change Summaries:** Automatically analyze code diffs and generate intelligent, human-readable snapshot change summaries using Gemini, OpenAI, or local heuristic engines.
- **Concurrent Snapshots:** Instantly capture filesystem states using a worker-pool concurrency model.
- **Atomic Restores:** Revert workspaces cleanly with safety guards and thread-safe error recovery.
- **Metadata Management:** Structured storage of snapshots and logs in a local, self-contained SQLite database.
- **Diff Comparison:** Efficiently compare changes between snapshot points.
