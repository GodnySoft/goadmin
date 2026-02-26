# goadmin

Инфраструктурный агент для Ubuntu 24.04 LTS. Архитектура: модульный монолит (Ports & Adapters), Zero-Trust, Go 1.22+.

## CLI

```bash
go build -trimpath -ldflags "-s -w -X main.version=0.1.0" -o bin/goadmin ./cmd/goadmin
./bin/goadmin version
./bin/goadmin host status
```

## Качество

- `golangci-lint run ./...`
- `go test ./...`
- `go test -race ./...`
- `gosec ./...`
