# Contributing to Eko ✦

Thank you for your interest in contributing to Eko! Eko is an AI-powered snapshot versioning CLI.

This guide will help you set up your development environment, understand our architecture, and walk you through adding new features.

---

## 1. Prerequisites

To develop Eko, you need:
- **Go 1.21+** ([Download](https://go.dev/dl/))

---

## 2. Project Architecture

Eko is structured as a lightweight CLI:

```
eko/
├── cmd/              # Cobra CLI commands (init, save, history, etc.)
├── internal/
│   ├── api/          # Internal diffing logic & structures
│   ├── db/           # Local SQLite database initialization
│   └── snapshot/     # Concurrent snapshot creation/restoration engine
└── main.go           # Entry point
```

---

## 3. Development Setup

1. **Fork and Clone the Repo:**
   ```bash
   git clone https://github.com/<your-username>/eko.git
   cd eko
   ```

2. **Building the CLI Locally:**
   ```bash
   go build -o eko .
   ```

---

## 4. How to Add a New Feature

Adding a new feature typically involves two steps: implementing engine logic and registering the Cobra CLI command.

### Step 1: Implement the Engine Logic
Write the core Go logic in the `internal/` package. Keep database operations inside `internal/db` and filesystem operations inside `internal/snapshot` or `internal/util`.

### Step 2: Register the Cobra CLI Command
Add a new command file in `cmd/` (e.g., `cmd/yourfeature.go`):
```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var yourFeatureCmd = &cobra.Command{
	Use:   "yourfeature",
	Short: "Description of your feature",
	Run: func(cmd *cobra.Command, args []string) {
		// Invoke internal engine logic here
		fmt.Println("Feature executed via CLI!")
	},
}

func init() {
	rootCmd.AddCommand(yourFeatureCmd)
}
```

---

## 5. Coding Standards & Conventions

### Go Guidelines
- **Paths:** Always resolve directories relative to the current working directory. Eko can be executed in any user folder.
- **SQLite Safety:** Database operations should handle errors gracefully.
- **Formatting:** Format Go files using `gofmt` and run `go vet ./...` before submitting a PR.

---

## 6. Submitting a Pull Request

1. Create a feature branch:
   ```bash
   git checkout -b feat/your-feature-name
   ```
2. Commit your modifications with a semantic commit message (e.g., `feat: add custom snapshot labels`, `fix: sqlite directory lock`).
3. Run automated tests to ensure everything is correct:
   ```bash
   go test ./...
   ```
4. Push to your fork and submit a PR to `kavix/eko:main`.
