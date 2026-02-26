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
GO_DST ?= $(CURDIR)/.local/go
ENV_VARS := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) HTTP_PROXY= HTTPS_PROXY= ALL_PROXY= http_proxy= https_proxy= all_proxy= ftp_proxy= FTP_PROXY=

.PHONY: all check fmt lint test race sec tidy build run serve deps tools go-check clean go-install

all: check

check: go-check tidy fmt lint test race sec build

fmt:
	$(GO) fmt ./...

lint: go-check tools
	$(ENV_VARS) $(GOLANGCI_LINT) run ./...

test: go-check
	$(ENV_VARS) $(GO) test ./...

race: go-check
	$(ENV_VARS) $(GO) test -race ./...

sec: go-check tools
	$(ENV_VARS) $(GOSEC) ./...

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
	@mkdir -p $(GO_DST)
	@echo "Скачиваю Go в $(GO_DST)"
	curl -L $(GO_TARBALL) -o /tmp/go.tgz
	tar -C $(CURDIR)/.local -xzf /tmp/go.tgz
	rm -f /tmp/go.tgz

clean:
	rm -rf bin
