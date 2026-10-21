GOLANGCI_LINT_VERSION := v2.14.0
GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOVULNCHECK := $(GOBIN)/govulncheck
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: check fmt lint build test vuln secrets tools live live-mcp install-skill

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

## live-mcp: with the "penpot-mcp test" file open in a tab and the MCP plugin
## connected, read the open file and reflow it (not in CI)
live-mcp:
	@test -n "$$PENPOT_MCP_URL" || { echo "set PENPOT_MCP_URL (see .env.example)"; exit 2; }
	go test -tags live -run TestTabLive -count=1 -v ./internal/tools

## install-skill: link every skill into the personal skills directory, to
## work on them from this checkout (users run `penpot-mcp skills install`)
install-skill:
	@base="$${CLAUDE_CONFIG_DIR:-$$HOME/.claude}/skills"; mkdir -p "$$base"; \
	for src in $(CURDIR)/skills/penpot-mcp*/; do \
		src="$${src%/}"; dest="$$base/$$(basename "$$src")"; \
		if [ -e "$$dest" ] && [ ! -L "$$dest" ]; then echo "$$dest exists and is not a link; leaving it"; continue; fi; \
		ln -sfn "$$src" "$$dest" && echo "linked $$dest"; \
	done
