# asb-tui

A terminal UI for Azure Service Bus queue monitoring, built with Go, Bubble Tea, Bubbles and Lip Gloss.

## Requirements

- Go 1.22+
- Azure CLI installed (`az`)
- Access to Azure Service Bus namespace with read permissions

## Authentication

Typical local flow:
1. `az login`
2. set `ASB_NAMESPACE`
3. run app

No connection strings required.

## Configuration

Environment variables:

- `ASB_NAMESPACE` (required)
  Example: `my-namespace.servicebus.windows.net`
- `ASB_REFRESH_SECONDS` (optional, default `10`)
- `ASB_ACTIVE_WARN_THRESHOLD` (optional)
- `ASB_DLQ_WARN_THRESHOLD` (optional)

## Run

```bash
go run ./cmd/asb-tui
```

## Key Bindings

- `j/k` or `up/down` move selection
- `/` enter filter mode
- `esc` or `enter` exit filter mode
- `x` clear filter
- `tab` switch focus between list and detail pane
- `s` cycle sort
- `r` refresh selected queue
- `R` refresh all queues
- `?` help
- `q` quit


## Development

```bash
go test ./...
```

Suggested structure:

- `cmd/asb-tui/` application entrypoint
- `internal/config/` env parsing and validation
- `internal/asb/` Azure Service Bus read client
- `internal/ui/` Bubble Tea model/update/view
- `internal/ui/style/` Lip Gloss theme styles

