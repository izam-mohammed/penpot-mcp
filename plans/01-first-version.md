# an agent lists your penpot files and draws a board with a rectangle in one, with no browser open

## The ask

A Penpot MCP server built on Penpot's own APIs, written in Go, and kept
lightweight and efficient. `docs/features.md` lists the features and `docs/bridge.md`
maps every official MCP tool to the API route or the MCP route.

This first plan is the roughest version someone can use: a Go binary that
Claude Code starts over stdio, which reads teams, projects, files and pages
through the backend API and draws a board and a rectangle into a chosen file
with `update-file`. Everything else in `docs/bridge.md` deepens it in later
plans (`PLAN.md`).

## Constraints

- Lightweight and efficient, which decided the language.
- Written in Go.
- Each piece of work is committed properly and pushed as it lands.

## Alternatives

### Decision 1: which language is the server written in? (one-way door)

Everything else is built on it; a rewrite later is the whole project.

Searched 2026-10-19: "MCP server memory footprint Go vs Rust vs Node
benchmark", "modelcontextprotocol go-sdk releases 2026", "mark3labs mcp-go
vs official go-sdk"; the go-sdk v1.8.0 source and releases page, a
community Go/Rust/Java MCP benchmark (armantik on Medium), the qubittool
Node vs Go write-up (whose own revision withdraws its numbers).

| Must | Go (user's pick) | Rust | TypeScript (Node) |
| --- | --- | --- | --- |
| Maintained official MCP SDK | yes (go-sdk v1.8.0, 2026-09-14) | yes (`rmcp`) | yes (reference SDK) |
| Single local process Claude Code can start over stdio | yes | yes | yes |
| **Survives** | **yes** | **yes** | **yes** |

| Criterion | Weight | Go (user's pick) | Rust | TypeScript (Node) |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 40 | 4 | 5 | 2 |
| Running cost and operability | 20 | 5 | 5 | 2 |
| Reliability | 15 | 4 | 5 | 3 |
| Security | 15 | 4 | 4 | 3 |
| Fit to the ask | 10 | 5 | 3 | 5 |
| **Weighted total** | **100** | **4.30** | **4.65** | **2.60** |

- Go efficiency 4, Rust 5: about 12 MB against 10 MB RSS in the one
  community MCP benchmark found (armantik, 2026); the workload here is
  waiting on Penpot's HTTP API, so the gap does not show per call.
- Node efficiency 2: ~50 MB idle runtime plus `node_modules`; no
  reproducible MCP number found (the qubittool figures were withdrawn).
- Operability: Go and Rust ship one static binary; Node needs a runtime.
- Reliability: Rust's compile-time checks rank highest; Node has no static
  types without a build step.
- Security: Go and Rust have small dependency trees here; npm's is larger.
- Fit to the ask, Rust 3: no maintained Go-style transit library either, so
  the exporter's transit encoder is hand-written in any case; Rust's MCP
  client side for the MCP route is less complete than go-sdk's
  `StreamableClientTransport` (not verified in depth: `?` would settle on
  reading `rmcp`'s client docs). Node 5: the official MCP is TypeScript.

Sensitivity: Rust wins by 0.35 on raw numbers. Moving 10 points from
performance to fit to the ask gives Go 4.40, Rust 4.45: Rust still wins on
the table. The user was asked with these totals on 2026-10-19 and kept Go.
Risk: Go's GC adds a few MB and short pauses; harmless for an I/O-bound
tool server. Limit: none needed.

**Chosen: Go (the user's pick, kept 2026-10-19).** Rust scored higher (4.65
against 4.30, on raw memory and reliability); asked with those totals, the
user kept Go: the server is I/O bound and the memory gap is a few MB. Revisit if the binary's idle RSS passes 30 MB.

### Decision 2: which MCP library? (two-way door)

Searched 2026-10-19: "mark3labs mcp-go vs official modelcontextprotocol
go-sdk comparison", go-sdk v1.8.0 release notes, mcp-go v1.3.0 release
(2026-10-14), the go-sdk design document.

| Criterion | Weight | Official go-sdk | mark3labs/mcp-go | Hand-written JSON-RPC |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 35 | 4 | 4 | 5 |
| Fit to the ask | 30 | 5 | 4 | 2 |
| Compatibility (protocol revisions) | 20 | 5 | 4 | 2 |
| Security | 15 | 5 | 4 | 3 |
| **Weighted total** | **100** | **4.65** | **4.00** | **3.20** |

- go-sdk fit 5: server (`AddTool` with typed input and schema inference),
  stdio, streamable HTTP and a streamable HTTP client in one module
  (`docs/external/go-mcp-sdk.md`).
- Compatibility: go-sdk v1.7.0 added protocol 2026-07-28 and kept
  2025-11-25; hand-written would track every revision by hand.
- Hand-written efficiency 5: no dependencies, but the saving is a few
  hundred KB of binary.

**Chosen: official go-sdk v1.8.0.** Typed tools, both transports and the
client the MCP route needs, kept in step with the spec. Revisit if its
dependency set grows past what `govulncheck` keeps clean.

### Decision 3: how the server talks to Penpot (two-way door)

`net/http` from the standard library with JSON bodies and
`Accept: application/json` is the only sensible approach: a wrapper library
(resty and similar) adds a dependency for nothing the client needs.

Searched 2026-10-19: "go http client library vs net/http"; the
`net/http` docs. Nothing better for a single-host JSON RPC client.

**Chosen: `net/http`.** Revisit if HTTP/2 multiplexing to Penpot is needed
(it already is, by default, over TLS).

### Decision 4: the project's checks (two-way door)

Searched 2026-10-19: "golangci-lint v2 latest release 2026 gofumpt",
"golangci-lint v2 vs staticcheck speed 2026 govulncheck gitleaks";
golangci-lint changelog (v2.14.0, 2026-09-24, bundles staticcheck 0.7 and
gofumpt 0.12), govulncheck docs.

| Criterion | Weight | golangci-lint v2 + govulncheck | gofmt + go vet + staticcheck + osv-scanner |
| --- | ---: | ---: | ---: |
| Performance efficiency | 40 | 4 | 4 |
| Fit to the ask | 35 | 5 | 4 |
| Maintainability of the result | 25 | 5 | 3 |
| **Weighted total** | **100** | **4.60** | **3.75** |

- Efficiency: no 2026 benchmark found; both run the same analyzers.
- golangci-lint fit 5: format, vet, staticcheck, errcheck and gosec in one
  pass; govulncheck reports only reachable vulnerabilities.

**Chosen: golangci-lint v2 + govulncheck + gitleaks**, run by `make check`
(`docs/standards.md`). Revisit if `make check` passes 60 s.

## The decision

A Go 1.26 module (decision 1) using the official go-sdk v1.8.0 over stdio
(decision 2), talking to Penpot with `net/http` (decision 3). One binary,
`penpot-mcp`, reads `PENPOT_BASE_URL` and `PENPOT_ACCESS_TOKEN`, checks the
token with `get-profile` at start, and exposes eight tools: `whoami`,
`list_teams`, `list_projects`, `list_files`, `list_pages`,
`get_page_tree`, `create_board`, `create_rect`. Writes go through one
in-process queue per profile with a per-file `{revn, vern}` cache, because
Penpot allows one `update-file` in flight per profile
(`docs/external/penpot-backend-api.md` § Limits). Geometry (`selrect`,
`points`, `transform`) is computed in Go for unrotated shapes only.

New dependencies: `github.com/modelcontextprotocol/go-sdk` v1.8.0 and its
transitive set (`docs/external/go-mcp-sdk.md` § Gotchas). Network host:
`PENPOT_BASE_URL` only. Secrets: `PENPOT_ACCESS_TOKEN`. No background
service, no other permission.

Standards: `docs/standards.md` (format, lint, build, race tests,
govulncheck and gitleaks through `make check`).

## Integrations

| Integration | Local docs | Proof script | Env vars |
| --- | --- | --- | --- |
| Penpot backend RPC API | `docs/external/penpot-backend-api.md` | `scripts/prove/penpot-api.sh` | `PENPOT_BASE_URL`, `PENPOT_ACCESS_TOKEN` |
| Go MCP SDK | `docs/external/go-mcp-sdk.md` | `scripts/prove/go-mcp-sdk.sh` | none |

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `scripts/prove/penpot-api.sh` | new | 1 | `get-profile` with the token, PASS on a non-zero id | fixed |
| `scripts/prove/go-mcp-sdk.sh` | new | 1 | builds a throwaway echo server, lists its tool over stdio | fixed |
| `.env.example` | new | 1 | the env var names, empty | fixed |
| `go.mod`, `go.sum` | new | 2 | module `github.com/izam-mohammed/penpot-mcp`, go 1.26 | fixed |
| `Makefile`, `.golangci.yml`, `.github/workflows/check.yml` | new | 2 | `make check` and CI | expected |
| `cmd/penpot-mcp/main.go` | new | 2 | config, token check, server start | expected |
| `internal/penpot/client.go` | new | 2 | RPC client, headers, error type | expected |
| `internal/penpot/geometry.go` | new | 2 | `selrect`, `points`, `transform` for a box | expected |
| `internal/penpot/writer.go` | new | 2 | per-profile write queue, revn/vern cache | expected |
| `internal/tools/*.go` | new | 2 | the eight tools | expected |
| `README.md` | new | 2 | install, config for Claude Code, the tools | expected |

*Reused, not rewritten:*

- The repository holds only docs; searched for any Go code, scripts or
  config to extend and found none.
- `docs/penpot-api-reference.md` § 7.8 minimal shapes: the rect and frame
  bodies the geometry builder fills, used as test fixtures.

## Security

What is protected: the user's Penpot account (every team, project and file
the token reaches, with full edit rights) and the access token itself.

Who can reach it: only the local MCP client that starts the binary over
stdio (Claude Code). Its input is model-generated, so every tool argument
is untrusted input. Outbound calls go only to `PENPOT_BASE_URL` over TLS.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | the token lands in a log, error or tool result | token read once from env, never formatted; errors built from Penpot's `{type, code, hint}` only | `TestTokenNeverLogged` captures logs and tool errors of a failing call |
| Tampering | a model sends a malformed shape or a huge payload | typed tool inputs (schema from go-sdk), bounds on width/height/count, name length caps | `TestCreateRectRejectsBadInput` |
| Elevation of privilege | a prompt-injected model edits a file it was not asked to | no delete or move tools in this plan; every write names its file id; the result echoes the file name | `TestWriteEchoesTarget` |
| Denial of service | parallel writes trip Penpot's per-profile limit (429) | one write queue per profile; `Retry-After` honoured once | `TestWriterSerialises`, `TestRetryAfter` |
| Spoofing | `PENPOT_BASE_URL` points at a look-alike host | `https://` required unless the host is `localhost`; TLS verified | `TestBaseURLMustBeHTTPS` |

Data: design files and profile name and email pass through memory only;
nothing is stored on disk. Secrets: `PENPOT_ACCESS_TOKEN`, set by the user
in the MCP client config or the shell, never committed. Logged: tool names,
command names, HTTP status and Penpot error codes; never the token, never
request bodies. Checklist: OWASP Top 10 for LLM Applications 2025 (LLM01
prompt injection, LLM06 excessive agency) and OWASP API Security Top 10
2023 for the outbound client.

## Edge cases

- Bad or expired token: `get-profile` returns the zero UUID; the binary
  exits at start with "PENPOT_ACCESS_TOKEN was not accepted".
- File the token cannot edit (`permissions.canEdit` false): the write tool
  refuses before sending.
- Stale `revn` after someone edits in the browser: a lower revn is accepted
  by Penpot; `409 vern-conflict` re-fetches once, then fails clearly.
- Rotated shapes: not supported yet; `rotation` is not an input in this
  plan.

## Declined

- Rust: scored higher (4.65 against 4.30) on raw memory and reliability;
  declined by the user, who chose Go since the server is I/O bound.
- TypeScript (Node): lowest on efficiency and operability (2.60).
- Using `ancrz/penpot-mcp-server` (Python, 68 tools, last push 2026-02-22)
  instead of building: it reads Penpot's Postgres directly, so it fails the
  must of working against hosted design.penpot.app with only a token.
- mark3labs/mcp-go: lower on compatibility with new protocol revisions and
  lacks nothing we need over go-sdk (4.00 against 4.65).
- Hand-written JSON-RPC: lowest fit and compatibility (3.20).
- Separate gofmt, go vet, staticcheck, osv-scanner: same analyzers, more
  configs to keep in step (3.75).

## What the tests hold

- The client sends `Authorization: Token`, `Accept: application/json` and
  `X-Client`, and decodes Penpot's error body into a typed error.
- The geometry builder's output for a box matches the fixture shapes from
  `docs/penpot-api-reference.md` § 7.8.
- Writes to one profile never overlap, and the cached `revn` advances to
  `response.revn + 1`.
- The token never appears in logs or tool results.
- Bad tool input is refused before any HTTP call.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: prove the integrations (needs you: set PENPOT_ACCESS_TOKEN)

- [x] `scripts/prove/penpot-api.sh` calls `get-profile` with
  `Authorization: Token $PENPOT_ACCESS_TOKEN` per
  `docs/external/penpot-backend-api.md` § Authentication and prints `PASS`
  when the returned id is not the zero UUID, with `PENPOT_BASE_URL` and
  `PENPOT_ACCESS_TOKEN` listed in `.env.example`.
- [x] `scripts/prove/go-mcp-sdk.sh` builds a throwaway one-tool server in a
  temp dir against go-sdk v1.8.0 per `docs/external/go-mcp-sdk.md` § Server,
  pipes `initialize` and `tools/list` over stdio, and prints `PASS` when the
  tool is listed.
- [x] Check: `sh scripts/prove/penpot-api.sh` and
  `sh scripts/prove/go-mcp-sdk.sh` both exit 0 and print `PASS`.

### Phase 2: the rough cut (needs you: try it)

- [x] The Go module, `Makefile`, `.golangci.yml` and the CI workflow land
  with `make check` running format, lint, build, race tests, govulncheck
  and gitleaks per `docs/standards.md`.
  Landed 2026-10-19: `go.mod` pins `toolchain go1.26.9`; go1.26.0's
  standard library failed govulncheck with 25 reachable advisories.
- [x] The RPC client (expected in `internal/penpot/client.go`) posts to
  `/api/main/methods/{command}` with the headers from
  `docs/external/penpot-backend-api.md` § What this project calls, maps the
  error body to a typed error and honours one `Retry-After`, pinned by
  `TestClientHeaders`, `TestErrorBody`, `TestRetryAfter`,
  `TestTokenNeverLogged` and `TestBaseURLMustBeHTTPS`.
- [x] The binary checks the token with `get-profile` at start and serves
  `whoami`, `list_teams`, `list_projects`, `list_files`, `list_pages` and
  `get_page_tree` (a compact tree: id, name, type, x, y, width, height,
  children), pinned by `httptest` replays in `internal/tools`.
- [x] `create_board` and `create_rect` add a `frame` and a `rect` to a
  given file and page through `update-file` with computed `selrect`,
  `points` and `transform` (expected in `internal/penpot/geometry.go`) and
  a per-profile write queue with a `{revn, vern}` cache (expected in
  `internal/penpot/writer.go`), pinned by `TestBoxGeometry`,
  `TestWriterSerialises`, `TestCreateRectRejectsBadInput` and
  `TestWriteEchoesTarget`.
  Landed 2026-10-19: `get_page_tree` returns a flat list with `parent_id`
  and `depth`, since a nested node type gives a recursive output schema
  that go-sdk rejects.
- [x] `README.md` says how to build the binary and add it to Claude Code
  with `claude mcp add penpot-api -e PENPOT_ACCESS_TOKEN=... -- <path>`.
- [x] Check: `make check` passes, and `make live` (needs the env vars)
  lists the user's teams and draws a 400 x 300 board with a 100 x 100 rect
  in a file named `penpot-mcp test`.
- [ ] You try it: add the binary to Claude Code, ask it to list your
  projects and draw a board with a red rectangle in a test file, then open
  that file in Penpot and say whether the shapes look right. What you want
  changed goes into the later plans in `PLAN.md`.

### Phase 3: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.
