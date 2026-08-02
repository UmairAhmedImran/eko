---
sidebar_position: 2
---

# Getting Started

Get up and running with Eko in minutes.

## Prerequisites

- **Go**: Version 1.21+ (if building from source)

## Installation

### 1. Homebrew (macOS)
```bash
brew tap kavix/tap
brew install eko
```

### 2. Build From Source
```bash
git clone https://github.com/kavix/eko.git
cd eko
go build -o eko .
```

---

## Quick Start Guide

### 1. Initialize Eko
```bash
eko init
```
This creates the hidden SQLite database and storage directory inside `.eko/`.

### 2. Save a Snapshot
```bash
eko save

# Auto-generate an AI summary of changes when saving:
eko save --ai
```

### 3. Generate AI Summaries
```bash
eko summary
```

### 4. View Snapshot History
```bash
eko history
```

### 5. Restore a Snapshot State
```bash
eko restore <snapshot-id>
```
