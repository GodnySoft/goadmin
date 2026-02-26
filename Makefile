GO ?= $(shell if [ -x "$(CURDIR)/.local/go/bin/go" ]; then echo "$(CURDIR)/.local/go/bin/go"; else echo go; fi)
GOCACHE ?= $(CURDIR)/.cache/go-build
GOMODCACHE ?= $(CURDIR)/.cache/gomod
GOPROXY ?= https://proxy.golang.org
GOSUMDB ?= sum.golang.org
GOBIN ?= $(shell $(GO) env GOBIN 2>/dev/null || true)
ifeq ($(GOBIN),)
GOBIN := $(HOME)/go/bin
endif
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GOSEC ?= $(GOBIN)/gosec
MODULE := goadmin
BINARY := bin/goadmin
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LD_FLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
GO_TARBALL ?= https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
GO_TARBALL_LOCAL ?= /tmp/go.tgz
GO_DST ?= $(CURDIR)/.local/go
GO_BIN_DIR := $(dir $(GO))
ENV_VARS := PATH=$(GO_BIN_DIR):$(PATH) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) HTTP_PROXY= HTTPS_PROXY= ALL_PROXY= http_proxy= https_proxy= all_proxy= ftp_proxy= FTP_PROXY=
ENV_VARS := $(ENV_VARS) GOLANGCI_LINT_CACHE=$(CURDIR)/.cache/golangci-lint

.PHONY: all check fmt lint test race sec sbom tidy build run serve deps tools go-check clean go-install

all: check

check: go-check tidy fmt lint test race sec build sbom

fmt:
	$(ENV_VARS) $(GO) fmt ./...

lint: go-check tools
	$(ENV_VARS) $(GOLANGCI_LINT) run ./...

test: go-check
	$(ENV_VARS) $(GO) test ./...

race: go-check
	$(ENV_VARS) $(GO) test -race ./...

sec: go-check tools
	$(ENV_VARS) $(GOSEC) ./cmd/... ./internal/... ./pkg/...

sbom: go-check
	$(ENV_VARS) $(GO) run ./utils/sbom -o bin/sbom-gomod.cdx.json

build: go-check
	$(ENV_VARS) $(GO) build -trimpath -ldflags "$(LD_FLAGS)" -o $(BINARY) ./cmd/goadmin

run: build
	./$(BINARY) version

serve: build
	./$(BINARY) serve

tidy: go-check
	$(ENV_VARS) $(GO) mod tidy

deps: go-check tools
	echo "deps installed"

tools:
	@if ! command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		echo "installing golangci-lint"; $(ENV_VARS) $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2; \
	fi
	@if ! command -v $(GOSEC) >/dev/null 2>&1; then \
		echo "installing gosec"; $(ENV_VARS) $(GO) install github.com/securego/gosec/v2/cmd/gosec@v2.19.0; \
	fi

# Проверка наличия Go

go-check:
	@if ! command -v $(GO) >/dev/null 2>&1; then \
		echo "Go не установлен. Установите Go 1.22+"; exit 1; \
	fi

# Локальная установка Go в .local/go
go-install:
	@mkdir -p $(CURDIR)/.local
	@if [ -f "$(GO_TARBALL_LOCAL)" ]; then \
		echo "Использую локальный архив Go: $(GO_TARBALL_LOCAL)"; \
		tar -C $(CURDIR)/.local -xzf "$(GO_TARBALL_LOCAL)"; \
	else \
		echo "Скачиваю Go из $(GO_TARBALL)"; \
		env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY -u http_proxy -u https_proxy -u all_proxy -u ftp_proxy -u FTP_PROXY \
			curl --fail --location --retry 3 "$(GO_TARBALL)" -o /tmp/go.tgz && \
		tar -C $(CURDIR)/.local -xzf /tmp/go.tgz && \
		rm -f /tmp/go.tgz; \
	fi
	@$(CURDIR)/.local/go/bin/go version

clean:
	rm -rf bin
