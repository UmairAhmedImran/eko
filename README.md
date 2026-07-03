# Eko ✦

**Eko** is an AI-powered snapshot versioning CLI designed to help you capture and restore the evolution of your projects. Think of it as a "Time Machine" for your local development environment.

## Features

- **Snapshots:** Instantly save the state of your project.
- **Restore:** Revert your project to any previous snapshot with a single command.
- **Database Driven:** Efficiently tracks snapshots and metadata.

## Getting Started

### Prerequisites

- Go 1.26+

### Installation

#### Homebrew (macOS)

You can install Eko via Homebrew:

```bash
brew tap kavix/tap
brew install eko
```

#### From Source

```bash
go build -o eko main.go
```

### Usage

1. **Initialize Eko in your project:**
   ```bash
   eko init
   ```

2. **Save a snapshot:**
   ```bash
   eko save
   ```

3. **View history:**
   ```bash
   eko history
   ```

4. **Restore a snapshot:**
   ```bash
   eko restore <snapshot-id>
   ```

## Development

```bash
go build -o eko main.go
```

## License
MIT
