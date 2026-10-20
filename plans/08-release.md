# anyone installs penpot-mcp as one download, runs it locally or as an http server, with one or several accounts

## The ask

`docs/features.md` § 1: streamable HTTP for remote use, named profiles for
several accounts, and released binaries.

## Alternatives

### Decision 1: how are binaries released? (two-way door)

Searched 2026-10-19: "goreleaser v2 2026 release go binary homebrew
alternative"; goreleaser install and v2 blog pages (v2.18 listed as
current), GitHub Releases docs.

| Criterion | Weight | GoReleaser v2 in CI | `make release` + `gh release` |
| --- | ---: | ---: | ---: |
| Performance efficiency | 30 | 4 | 4 |
| Fit to the ask | 40 | 5 | 3 |
| Security | 30 | 5 | 3 |
| **Weighted total** | **100** | **4.70** | **3.30** |

- GoReleaser fit 5: darwin, linux, windows on amd64 and arm64, checksums,
  a Homebrew cask, in one config.
- Security 5: checksums and SBOMs per release; the make route would need
  both written by hand.

**Chosen: GoReleaser v2** in a tag-triggered workflow. Revisit if the
release needs signing GoReleaser cannot do.

## The decision

- Streamable HTTP via go-sdk `NewStreamableHTTPHandler` in stateless mode
  (`docs/external/go-mcp-sdk.md` § Streamable HTTP server), behind
  `--http :8787`, off by default.
- Named profiles: `PENPOT_PROFILES` points to a config file of
  `{name, base_url, token_env}`; each profile has its own write queue.
- Releases: GoReleaser (decision 1).

New dependency: GoReleaser (CI only, not in the binary). Network: the HTTP
listener, off unless asked. Secrets: per-profile token env var names.

Standards: `docs/standards.md` (`make check`; a release adds
`goreleaser check`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `cmd/penpot-mcp/main.go` | edit | 1 | `--http` flag | expected |
| `internal/config/profiles.go` | new | 2 | named profiles | expected |
| `.goreleaser.yaml`, `.github/workflows/release.yml` | new | 3 | releases | expected |

*Reused, not rewritten:*

- `internal/penpot/writer.go` (plan 01): one instance per profile.

## Security

An HTTP listener exposes the account to the network.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Spoofing | anyone on the network calls the HTTP server | binds `127.0.0.1` by default; a non-loopback bind needs `PENPOT_MCP_HTTP_TOKEN` as a bearer token | `TestHttpNeedsTokenOffLoopback` |
| Denial of service | huge requests | `MaxRequestBodyBytes` set, per-request timeout | `TestHttpBodyCap` |
| Tampering | a tampered release binary | checksums and SBOM published with each release | `goreleaser check` in CI |

Secrets: `PENPOT_MCP_HTTP_TOKEN`. Checklist: OWASP API Security Top 10 2023
(API2, API4 unrestricted resource consumption).

## Declined

- `make release` + `gh release`: lower on fit and security (3.30).

## What the tests hold

- Off-loopback HTTP without a token refuses to start.
- Two profiles write in parallel; one profile's writes stay serial.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] `--http 127.0.0.1:8787` serves the same tools over streamable HTTP,
  pinned by `TestHttpNeedsTokenOffLoopback` and `TestHttpBodyCap`.
  Landed 2026-10-20: in `internal/httpserve/`, stateless, 4 MB body cap,
  bearer token compared in constant time, served at `/mcp`.
- [ ] Check: `make check` passes, and `claude mcp add --transport http
  penpot-api http://127.0.0.1:8787/mcp` lists the tools.
- [ ] You try it: run the server with `--http`, add it to Claude Code over
  HTTP, and list your projects.

### Phase 2: several accounts

- [x] Named profiles land (expected in `internal/config/profiles.go`) with
  a `profile` argument on every tool, pinned by `TestProfilesParallel`;
  check: `make check` passes.
  Landed 2026-10-20: the `profile` argument is taken out of each call by
  a server middleware (`internal/tools/profiles.go`), which also adds it
  to every tool's schema in `tools/list`, so no tool's input changed.
  Profiles can also name `email_env`, `password_env` and `mcp_url_env`;
  without `PENPOT_ACCESS_TOKEN` the first profile is the default.

### Phase 3: releases

- [x] `.goreleaser.yaml` and a tag-triggered release workflow land, and
  `README.md` gains the download and Homebrew steps; check: `make check` and
  `goreleaser check` pass.
  Landed 2026-10-20: `goreleaser check` passes, and a local
  `goreleaser release --snapshot --skip=publish,sbom` built all six
  archives. The Homebrew cask needs an `izam-mohammed/homebrew-tap` repo
  and a `HOMEBREW_TAP_TOKEN` secret; without them the release still goes
  out and the cask is skipped.

### Phase 4: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.
