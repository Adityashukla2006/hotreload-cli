# hotreload

A production-grade CLI tool that watches a project folder for code changes and automatically rebuilds and restarts your server.

## Quick Start

```bash
# Build the tool
go build -o ./bin/hotreload .

# Run the demo (edit testserver/main.go to see hot reload)
./bin/hotreload --root ./testserver --build "go build -o ./bin/testserver ./testserver" --exec "./bin/testserver"
```

On Windows, use backslashes for the exec path:
```bash
./bin/hotreload.exe --root ./testserver --build "go build -buildvcs=false -o ./bin/testserver.exe ./testserver" --exec ".\bin\testserver.exe"
```

## Usage

```bash
hotreload --root <project-folder> --build "<build-command>" --exec "<run-command>"
```

### Flags

| Flag | Description |
|------|-------------|
| `--root`, `-r` | Directory to watch (recursive, default: `.`) |
| `--build`, `-b` | Command to build the project when changes are detected |
| `--exec`, `-e` | Command to run the server after a successful build |

## Features

- **Immediate first build** — builds and starts your server on startup
- **Debounced watching** — rapid saves (e.g. vim atomic writes) collapse into a single rebuild
- **Recursive watching** — monitors all subdirectories, including newly created ones
- **File filtering** — ignores `.git/`, `node_modules/`, `dist/`, `build/`, `vendor/`, editor temp files
- **Build cancellation** — if changes occur during a build, the build is discarded and only the latest state is rebuilt
- **Build-fail resilience** — if the build fails, the previous server keeps running
- **Process group killing** — kills the entire process tree (Windows: `taskkill /T`, Unix: process group)
- **Graceful shutdown** — SIGTERM first, then force kill after 5 seconds
- **Crash loop detection** — if the server exits within 2 seconds, no auto-restart (3 second cooldown)
- **Real-time logs** — server stdout/stderr stream directly to the terminal
- **Watcher limit handling** — helpful message when OS inotify limits are reached

## Architecture

```
main.go              → entry point
cmd/root.go          → Cobra CLI, flags, validation
internal/
  watcher.go         → recursive filesystem watcher (fsnotify)
  debounce.go        → change batching (300ms)
  builder.go         → build command execution (cancellable)
  process.go         → server process lifecycle
  process_windows.go  → taskkill /T for process tree
  process_unix.go    → process group + SIGTERM/SIGKILL
  supervisor.go     → orchestrates: watcher → debounce → build → process
  logger.go          → real-time log streaming
  shell.go           → cross-platform shell args
```

Pipeline: **watcher event → debounce → build → restart server**

## Running Tests

```bash
go test ./...
```
