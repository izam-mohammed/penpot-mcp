GOLANGCI_LINT_VERSION := v2.14.0
GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOVULNCHECK := $(GOBIN)/govulncheck
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: check fmt lint build test vuln secrets tools live install-skill

## check: everything a change must pass (docs/standards.md)
check: fmt lint build test vuln secrets

fmt:
	$(GOLANGCI_LINT) fmt --diff

lint:
	$(GOLANGCI_LINT) run

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o bin/penpot-mcp ./cmd/penpot-mcp

test:
	go test -race ./...

vuln:
	$(GOVULNCHECK) ./...

secrets:
	gitleaks git --no-banner --redact

## tools: install the pinned check tools
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@latest

## live: draw into the "penpot-mcp test" file with a real token (not in CI)
live:
	@test -n "$$PENPOT_ACCESS_TOKEN" || { echo "set PENPOT_ACCESS_TOKEN (see .env.example)"; exit 2; }
	go test -tags live -run TestLive -count=1 -v ./internal/tools
