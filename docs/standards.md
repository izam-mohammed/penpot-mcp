# penpot-mcp: what every change passes

Run everything: `make check`. A change is done when it passes, locally and
in CI.

## Toolchain

| What | Version | Pinned in |
| --- | --- | --- |
| Go | 1.26 | the `go` line in `go.mod` |
| Modules | Go modules | `go.sum`, committed |
| golangci-lint | v2.14.x | `Makefile` (`GOLANGCI_LINT_VERSION`), installed with `go install` |
| govulncheck | latest `golang.org/x/vuln` release | `Makefile` |
| gitleaks | v8 | `Makefile` |

## Checks

| Check | Tool | Config | Command | Passing means |
| --- | --- | --- | --- | --- |
| Format | golangci-lint fmt (gofumpt, goimports) | `.golangci.yml` | `golangci-lint fmt --diff` | no diff |
| Lint | golangci-lint (govet, staticcheck, errcheck, gosec, revive) | `.golangci.yml` | `golangci-lint run` | no issues |
| Types | the Go compiler | `go.mod` | `go build ./...` | builds |
| Unit tests | `go test` | none | `go test -race ./...` | all pass |
| Proof scripts | POSIX `sh` + `curl` | none | `sh scripts/prove/<name>.sh` | prints `PASS` (needs the env vars; not in CI) |
| Dependency audit | govulncheck | none | `govulncheck ./...` | no reachable vulnerability |
| Secret scan | gitleaks | `.gitleaks.toml` when added | `gitleaks git --no-banner` | no findings |
| Build | `go build` | none | `go build -trimpath -ldflags="-s -w" -o bin/penpot-mcp ./cmd/penpot-mcp` | builds |

`make check` runs format, lint, build, tests, govulncheck and gitleaks in
that order and stops on the first failure.

## Tests

- Unit tests sit beside the code: `internal/<pkg>/<file>_test.go`.
- Penpot is never called from unit tests. The API client is tested against
  `net/http/httptest` servers that replay recorded replies from
  `internal/<pkg>/testdata/*.json`.
- Live checks against a real Penpot account are the proof scripts and the
  `make live` target, run by hand with the env vars set. They are not in CI.
- One test: `go test -race -run TestName ./internal/<pkg>`.

## Constraints

- A small, lightweight binary: no CGO (`CGO_ENABLED=0`), static, stripped.
  A new dependency needs its line in the plan that adds it.
- Secrets (`PENPOT_ACCESS_TOKEN`, `PENPOT_MCP_URL`, `PENPOT_PASSWORD`) are
  read from the environment only, never logged, and never written to test
  data. A test asserts their absence from captured logs.

## Conventions

- Conventional Commits, lowercase subject, no dashes other than `-`, no
  attribution lines (enforced by the commit-msg hook).
- Branches: `feat/<name>`, `fix/<name>`, `plan/<name>`.
- Layout: `cmd/penpot-mcp/` (main), `internal/` (all packages), `scripts/`,
  `docs/`, `plans/`.

## CI

`.github/workflows/check.yml` runs `make check` on every push and pull
request. A red check blocks merge.

## Decided

- golangci-lint v2 over separate `gofmt` + `go vet` + `staticcheck`: one
  pass, one config, bundles all three (`plans/01-first-version.md`,
  decision 4).
- govulncheck over osv-scanner: reachability-filtered, fewer false alarms
  for Go (`plans/01-first-version.md`, decision 4).
- gitleaks: already installed, scans history.
