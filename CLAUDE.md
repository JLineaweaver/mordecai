# Mordecai - Development Guide

## Build & Run

```bash
go build -o mordecai ./cmd/mordecai    # Build
./mordecai --config config.yaml         # Run
go test ./...                           # Test all
```

## Architecture

- **Module interface** (`internal/module/module.go`): Each module implements `Name()` and `Fetch(ctx, cfg)`, returning markdown content.
- **Delivery interface** (`internal/delivery/delivery.go`): Each channel implements `Name()` and `Send(ctx, digest)`.
- **Orchestrator** (`internal/digest/digest.go`): Runs all enabled modules in parallel, assembles results into a digest, sends via all enabled delivery channels.
- **Config** (`internal/config/config.go`): YAML with `${ENV_VAR}` substitution. Each module has an `enabled` flag + arbitrary settings passed as `map[string]interface{}`.

## Conventions

- Modules live in `internal/module/<name>/<name>.go`
- Delivery channels live in `internal/delivery/<name>/<name>.go`
- New modules must be registered in `cmd/mordecai/main.go` and have config support in `internal/config/config.go`
- Modules return markdown-formatted content; the orchestrator and delivery layer handle formatting
- No API keys required for Phase 1 modules; keep free-tier/keyless sources where possible
- Config uses `map[string]interface{}` for module settings to keep things flexible

## Module Status

- **news**: Implemented (RSS feeds)
- **sports**: Stub (Phase 2 - ESPN API)
- **stocks**: Stub (Phase 2 - Yahoo Finance)
- **weather**: Stub (Phase 3 - Open-Meteo)
