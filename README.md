# Mordecai - Your Personal Game Guide

A daily digest CLI tool that gathers personalized information from various sources and delivers a formatted summary to Discord (or other channels). Designed to run on a schedule via cron, Docker, or GitHub Actions.

## Quick Start

```bash
# Clone and build
git clone https://github.com/jlineaweaver/mordecai.git
cd mordecai
go build -o mordecai ./cmd/mordecai

# Configure
cp config.example.yaml config.yaml
# Edit config.yaml with your preferences

# Run
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
./mordecai --config config.yaml
```

## Docker

```bash
cp config.example.yaml config.yaml
# Edit config.yaml

export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
docker-compose up --build
```

## Configuration

Copy `config.example.yaml` to `config.yaml`. Environment variables can be referenced with `${VAR_NAME}` syntax.

### Modules

| Module | Status | Description |
|--------|--------|-------------|
| News | Available | RSS feed headlines |
| Sports | Planned | Scores and upcoming games (ESPN) |
| Stocks | Planned | Stock prices (Yahoo Finance) |
| Weather | Planned | Forecast (Open-Meteo) |

### Delivery Channels

| Channel | Status | Description |
|---------|--------|-------------|
| Discord | Available | Webhook-based delivery |

## Adding a New Module

1. Create a new directory under `internal/module/yourmodule/`
2. Implement the `module.Module` interface:

```go
type Module interface {
    Name() string
    Fetch(ctx context.Context, cfg map[string]interface{}) (*Result, error)
}
```

3. Register it in `cmd/mordecai/main.go`
4. Add config support in `internal/config/config.go`

## GitHub Actions

Add `DISCORD_WEBHOOK_URL` as a repository secret, then the included workflow will run your digest daily at 7:00 AM ET.

## License

MIT
