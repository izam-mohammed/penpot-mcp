# an agent exports boards to PNG, SVG or PDF on disk, and whole files to .penpot

## The ask

`docs/bridge.md` § 3 (`export_shape`) and § 10: the exporter on the API
route. It accepts only a session cookie and a transit body
(`docs/external/penpot-backend-api.md` § Authentication).

## Alternatives

### Decision 1: how is transit encoded? (two-way door)

Searched 2026-10-19: "transit-json go library russolsen transit golang";
`github.com/russolsen/transit` (last push 2020-01-30), the transit-format
spec.

| Must | Hand-written encoder (stdlib `encoding/json`) | russolsen/transit |
| --- | --- | --- |
| Still maintained | yes (ours) | no (no push since 2020) |
| **Survives** | **yes** | no |

**Chosen: a hand-written encoder** for the subset the exporter needs (maps,
keywords, UUIDs, vectors, booleans, numbers, strings), with a decoder for
its reply. Revisit if Penpot accepts JSON on `/api/export`.

### Decision 2: how does the exporter authenticate? (two-way door)

Searched 2026-10-19: "penpot exporter api export shapes access token share
link 2026"; Penpot exporter source (`exporter/src/app/http.cljs`), the API
reference § 12.

| Criterion | Weight | Login with email and password for a cookie | Share link route | MCP route `export_shape` |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 35 | 5 | 4 | 2 |
| Reliability | 35 | 4 | ? | 2 |
| Security | 30 | 3 | 3 | 4 |
| **Weighted total** | **100** | **4.05** | **?** | **2.60** |

- Cookie security 3: a password in the environment is a stronger secret
  than a token; it is used only to call `login-with-password`.
- Share link reliability `?`: untested; Phase 1's first item tries it.

**Chosen: cookie login**, the share link if Phase 1 shows it works without
a password, and the MCP route as the fallback when neither is set. Revisit
after that test.

## The decision

Tools: `export_shape` (PNG, JPEG, WEBP, SVG, PDF, scale; written to a path
on disk, never base64 into the reply), `export_boards` (every top-level
board of a page, name prefix filter), `export_file`, `import_file`
(`.penpot` via `export-binfile` and `import-binfile`). Retries on
`ResourceRequest timed out` and `429 queue-full`.

New secrets: `PENPOT_EMAIL`, `PENPOT_PASSWORD` (optional). Network host:
`PENPOT_BASE_URL` only. Dependencies: none new.

Standards: `docs/standards.md` (`make check`).

## Integrations

| Integration | Local docs | Proof script | Env vars |
| --- | --- | --- | --- |
| Penpot exporter | `docs/external/penpot-backend-api.md` | `scripts/prove/penpot-export.sh` | `PENPOT_BASE_URL`, `PENPOT_EMAIL`, `PENPOT_PASSWORD` |

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `scripts/prove/penpot-export.sh` | new | 1 | login, export one board, PASS on a PNG | fixed |
| `internal/transit/` | new | 2 | encoder and decoder | expected |
| `internal/penpot/exporter.go` | new | 2 | cookie session, export calls | expected |
| `internal/tools/export.go` | new | 2 | the export tools | expected |

*Reused, not rewritten:*

- `~/.claude/skills/penpot/scripts/check_boards.py`: its blank-PNG and
  truncation checks are ported into the export tool's result check.
- `internal/penpot/client.go` (plan 01) for `login-with-password` and
  `export-binfile`.

## Security

Adds a password secret and file writes to disk.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | the password or cookie lands in logs | read once from env, cookie held in memory only | `TestPasswordNeverLogged` |
| Tampering | the model writes an export over an important file | output only under the working directory, `.png/.svg/.pdf/.jpg/.webp/.penpot` extensions, no overwrite without `overwrite: true` | `TestExportPathGuard` |

Secrets: `PENPOT_PASSWORD`, set by the user in the MCP client config.
Checklist: OWASP API Security Top 10 2023 (API2 broken authentication).

## Declined

- russolsen/transit: unmaintained since 2020, fails the must.
- MCP route as the main export: needs a tab, slowest (2.60); kept only as
  the fallback.

## What the tests hold

- The transit encoder's output for a recorded export request matches the
  bytes the Penpot frontend sends.
- A blank or truncated PNG is reported as a failed export.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: prove the exporter (needs you: set PENPOT_EMAIL and PENPOT_PASSWORD)

- [x] `scripts/prove/penpot-export.sh` logs in, exports one board of the
  test file to PNG per `docs/external/penpot-backend-api.md` § What this
  project calls, prints `PASS` when the file starts with the PNG signature,
  and also tries the share-link route and prints its result; both env vars
  listed in `.env.example`.
  Landed 2026-10-20: written and syntax-checked, not run: it needs
  `PENPOT_EMAIL` and `PENPOT_PASSWORD`, which only the user holds. The
  password reaches curl on stdin, never its command line.
- [ ] Check: `sh scripts/prove/penpot-export.sh` exits 0 and prints `PASS`.

### Phase 2: the rough cut (needs you: try it)

- [x] The transit encoder, the exporter session and `export_shape` land
  (expected in `internal/transit/`, `internal/penpot/exporter.go`,
  `internal/tools/export.go`), pinned by `TestTransitRoundTrip`,
  `TestPasswordNeverLogged` and `TestExportPathGuard`.
  Landed 2026-10-20: the decoder sits beside the encoder in
  `internal/transit/decode.go` and follows the write cache. The
  exporter retries 429, 503 and timeouts three times and logs in again
  once after a 401. `Client.Download` sends the token only to the Penpot
  host (`TestDownloadKeepsTokenHome`).
- [ ] Check: `make check` passes, and `make live` writes
  `out/board.png` from the test file.
- [ ] You try it: ask Claude to export the board you drew to `design/` and
  open the PNG.

### Phase 3: boards, files and fallbacks

- [x] `export_boards`, `export_file`, `import_file`, the timeout retries and
  the MCP-route fallback land (expected in `internal/tools/export.go`),
  pinned by `TestBlankPngFails`; check:
  `make check` passes.
  Landed 2026-10-20: the MCP-route fallback is not built here: plan 06
  builds that route, and until then a missing login answers with
  `ErrNoExporter`, which names the two env vars. `export_file` and
  `import_file` read Penpot's server-sent events and use the access
  token. A flat or truncated PNG is refused before anything is written.

### Phase 4: open the PR

- [x] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.

Landed 2026-10-20: https://github.com/izam-mohammed/penpot-mcp/pull/5, stacked
on #4 (base `feat/library`).
