# when a penpot tab is connected, the agent also gets everything only the browser can do: run plugin code, reflow layouts, measure text, generate css

## The ask

"have all of the tools in mcp in the new mcp ... if not possible to achieve
through api, use their mcp route as well" (the user). `docs/bridge.md` § 1
to § 3 and every MCP and HYBRID row.

## Alternatives

### Decision 1: how does the server reach the plugin? (one-way door)

Searched 2026-10-19: the official MCP source (`penpot/penpot` `mcp/`), go-sdk
`StreamableClientTransport` docs, the plugin's WebSocket bridge (port 4402).

| Must | Be an MCP client of the official server | Run our own plugin and WebSocket bridge |
| --- | --- | --- |
| Works with hosted design.penpot.app as-is | yes | no (needs a plugin published or loaded by hand) |
| **Survives** | **yes** | no |

| Criterion | Weight | MCP client of the official server |
| --- | ---: | ---: |
| Performance efficiency | 40 | 3 |
| Reliability | 35 | 3 |
| Compatibility | 25 | 5 |
| **Weighted total** | **100** | **3.50** |

- Efficiency 3: an extra hop (our server, Penpot's MCP, the tab) per call.
- Reliability 3: tab sleep and heartbeat loss stay
  (`docs/external/penpot-official-mcp.md` § Limits).

Sensitivity: the only survivor; nothing flips it.
Risk: Penpot changes its MCP tool names; limited by a start-up
`tools/list` check that hides MCP-route tools when the names are missing.

**Chosen: MCP client of the official server** with go-sdk's
`StreamableClientTransport`. Revisit if Penpot publishes the plugin
WebSocket protocol as stable.

## The decision

With `PENPOT_MCP_URL` set, the server connects to the official MCP at start
and adds: `execute_code` (forwarded, with the file guard), `current_file`,
`get_selection`, `reflow`, `generate_markup`, `generate_style`,
`generate_font_faces`, `detach`, `swap_component`, `reset_overrides`,
`switch_variant`, `combine_as_variants`, `active_users`, `viewport`. HYBRID
tools gain their MCP step: `create_text` and `draw` read the real text size
after `waitForLayoutUpdate`, `set_layout` reflows, `import_svg` can use
`createShapeFromSvg`, `api_info` answers `source: "plugin"`. The routing
rules are `docs/bridge.md` § 2: file guard, one lock shared with the API
writer, re-fetch `revn` after MCP writes, no blind retry after a timeout.

New secret: `PENPOT_MCP_URL`. Network host: `PENPOT_BASE_URL`'s `/mcp`.
Dependencies: none new (go-sdk already has the client).

Standards: `docs/standards.md` (`make check`).

## Integrations

| Integration | Local docs | Proof script | Env vars |
| --- | --- | --- | --- |
| Official Penpot MCP | `docs/external/penpot-official-mcp.md` | `scripts/prove/penpot-mcp.sh` | `PENPOT_MCP_URL` |

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `scripts/prove/penpot-mcp.sh` | new | 1 | `initialize` and `tools/list` against the official MCP | fixed |
| `internal/mcproute/` | new | 2 | client, file guard, error text mapping | expected |
| `internal/tools/plugin.go` | new | 2 | the MCP-only tools | expected |
| `internal/tools/draw.go`, `internal/penpot/text.go` | edit | 3 | hybrid steps | expected |

*Reused, not rewritten:*

- `~/.claude/skills/penpot/scripts/penpot_mcp.py`: its `User-Agent`,
  session re-init on 400/404 and heartbeat error handling are ported.
- `internal/penpot/writer.go` (plan 01): its per-profile lock is shared, not
  duplicated.

## Security

`execute_code` runs model-written JavaScript in the user's Penpot tab with
full edit rights.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Elevation of privilege | injected text makes the model run code that edits another file | file guard: `penpot.currentFile.id` must equal the named `fileId`, else refuse | `TestFileGuardRefuses` |
| Information disclosure | the MCP URL (with its key) leaks in logs or errors | URL never formatted; errors carry only the official server's text | `TestMcpUrlNeverLogged` |
| Tampering | a timed-out call is retried and duplicates shapes | no automatic retry after a timeout; the tool reports "may have applied, re-read" | `TestNoRetryAfterTimeout` |
| Elevation of privilege | `execute_code` called when an API tool would do | tool description marks it last resort; `destructiveHint` set | reviewed in Phase 4 |

Checklist: OWASP Top 10 for LLM Applications 2025 (LLM01, LLM05 improper
output handling, LLM06).

## Declined

- Our own plugin and WebSocket bridge: fails the hosted must.

## What the tests hold

- Without `PENPOT_MCP_URL`, no MCP-route tool is listed and HYBRID tools
  run API-only.
- The file guard refuses a mismatched file before any code runs.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: prove the official MCP (needs you: set PENPOT_MCP_URL)

- [x] `scripts/prove/penpot-mcp.sh` sends `initialize` and `tools/list` with
  a `User-Agent` per `docs/external/penpot-official-mcp.md` § What this
  project calls and prints `PASS` when `execute_code` is listed;
  `PENPOT_MCP_URL` listed in `.env.example`.
  Landed 2026-10-20: written and syntax-checked, not run: it needs the
  user's `PENPOT_MCP_URL`. The URL reaches curl through `-K -` on stdin
  and is never printed.
- [x] Check: `sh scripts/prove/penpot-mcp.sh` exits 0 and prints `PASS`.
  Landed 2026-10-20: PASS against design.penpot.app, server 1.0.0.

### Phase 2: the rough cut (needs you: try it)

- [x] The MCP-route client with the file guard and error mapping, and the
  `execute_code`, `current_file` and `reflow` tools land (expected in
  `internal/mcproute/`, `internal/tools/plugin.go`), pinned by
  `TestFileGuardRefuses`, `TestMcpUrlNeverLogged` and
  `TestNoRetryAfterTimeout`.
  Landed 2026-10-20: the server connects on first use, not at start, so
  a sleeping tab never blocks start-up; the `tools/list` check runs then.
  MCP-route writes hold the API writer's lock (`Writer.Exclusive`) and
  drop the file's cached revision after.
- [x] Check: `make check` passes, and with the test file open in a tab,
  `make live-mcp` reads `current_file` and runs a `reflow`.
  Landed 2026-10-20: passed; with another file open in the tab it
  refused first, as the guard should.
- [ ] You try it: open the test file in Penpot with the MCP plugin
  connected, ask Claude which file is open and to reflow a board, and say
  whether it matched.

### Phase 3: hybrid steps and the rest

- [x] Text measuring and reflow after `create_text`, `draw` and
  `set_layout`, `import_svg` through `createShapeFromSvg`, `api_info`
  plugin source, and the remaining MCP-only tools land (expected in
  `internal/tools/draw.go`, `internal/tools/plugin.go`), pinned by
  `TestHybridFallsBackWithoutTab`; check: `make check` passes.
  Landed 2026-10-20: `draw`, `create_text` and `set_layout` add a `tab`
  field with the reflow and the laid-out text sizes when the tab has the
  file open, and a note otherwise; the API write always stands. Every
  Plugin API method is checked with `typeof` first, so an older Penpot
  says which method it lacks. Not built: `api_info`'s backend source
  (OpenAPI) and hybrid steps for `instantiate_component` and
  `apply_token`.

### Phase 4: review the agency

- [ ] Every MCP-route tool's description and hints (expected in
  `internal/tools/plugin.go`) are read against the threat table, findings
  fixed and listed in `docs/research/06-mcp-route.md`, and
  `execute_code`'s last-resort wording is confirmed with a live session;
  check: `make check` passes.
  Landed 2026-10-20: reviewed in `docs/research/06-mcp-route.md`; missing
  hints fixed and the server instructions now call file content data.
  Left open: the live-session check of `execute_code`'s wording.

### Phase 5: open the PR

- [x] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.

Landed 2026-10-20: https://github.com/izam-mohammed/penpot-mcp/pull/6, stacked
on #5 (base `feat/export`).

### Phase 6: the leftovers

- [x] `api_info` answers backend methods from Penpot's OpenAPI document
  (`/api/main/doc/openapi.json`, fetched once), with or without a tab,
  pinned by `TestAPIInfoBackend`; check: `make check` passes.
  Landed 2026-10-20: in `internal/tools/apidoc.go`; a schema too large to
  show whole is outlined one level deep, `member` opens one part.
  `TestLiveAPIInfo` passed against design.penpot.app.
- [x] `apply_token` runs in the tab for what the API path refuses (a token
  that refers to another, a text's colour) and `instantiate_component`
  gets the after-write reflow, pinned by `TestApplyTokenAliasNeedsTab` and
  `TestInstanceTabStep`; check: `make live-mcp` applies a referring token.
  Landed 2026-10-20: the first live run found `sync_tokens` leaves sets
  inactive, which resolves nothing; it now marks the sets it writes
  active unless the document has its own `$metadata`.

### Phase 7: open the PR

- [x] The branch pushed and its PR opened against `main` with a
  Conventional Commit title, a body naming this plan, what changed and
  why, and the checks run with their result. The PR's URL printed as the
  evidence.
  Landed 2026-10-20: https://github.com/izam-mohammed/penpot-mcp/pull/12
