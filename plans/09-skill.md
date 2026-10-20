# a claude code skill teaches any agent to set up and drive penpot-mcp, and uses claude in chrome for the steps only a browser can do

## The ask

"include a new skill also create for this including whatever required to
use from claude in chrome as well" (the user). The server's tools are not
enough on their own: an agent also needs to know how to install and connect
it, which tool to pick on which route, and how to do the browser steps
(make the access token and MCP key, open a file and connect the plugin, wake
a sleeping tab, look at what was drawn) with Claude in Chrome
(`mcp__claude-in-chrome__*`).

The existing `penpot` skill (`~/.claude/skills/penpot/SKILL.md`) does this
for the official plugin route only; its Chrome flows (find the MCP button,
**Connect here**, wake the tab, rename on the dashboard) are the starting
point.

## Alternatives

### Decision 1: where does the guidance live? (two-way door)

Searched 2026-10-19: "Claude Code skills SKILL.md frontmatter description
best practices 2026" (third-party guides only; official field list to be
confirmed in Phase 1), go-sdk `ServerOptions.Instructions`
(`docs/external/go-mcp-sdk.md` § Server), the existing `penpot` skill.

| Criterion | Weight | New skill in this repo | Extend the existing `penpot` skill | Server `instructions` only |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency (context cost) | 35 | 5 | 3 | 4 |
| Fit to the ask | 35 | 5 | 4 | 2 |
| Maintainability of the result | 30 | 5 | 2 | 4 |
| **Weighted total** | **100** | **5.00** | **3.05** | **3.30** |

- Efficiency: a skill loads only when the request fits; the existing skill
  is about 400 lines of plugin-route detail that would load with it.
  Server instructions are sent on every session, needed or not.
- Fit: instructions cannot carry the Chrome flows or the install steps
  (they reach the agent only after the server is already connected).
- Maintainability: a skill in this repo ships and versions with the binary,
  and a test keeps its tool list in step with the code.

**Chosen: a new skill, `penpot-mcp`, in this repo** under
`skills/penpot-mcp/`, installed by a symlink into `~/.claude/skills/`, plus
a two-line server `instructions` string that names the skill. Revisit if
Claude Code adds skills shipped by MCP servers.

## The decision

The skill holds:

- `SKILL.md`: when to use it; install (`go build` or a release download,
  then `claude mcp add penpot-api`); the env vars by name; the route rule
  from `docs/bridge.md` § 2 (API first, MCP route only for the rows marked
  MCP or HYBRID); the drawing loop (pick file, `draw` in one batch, export,
  look); failure table (bad token, 429, `vern-conflict`, tab not
  connected, export timeout).
- `references/chrome.md`: every Claude in Chrome flow, each as steps with
  the tool calls: load the tools in one `ToolSearch`
  (`tabs_context_mcp`, `tabs_create_mcp`, `navigate`, `computer`, `find`,
  `read_page`), `tabs_context_mcp` first, a new tab in Claude's own group;
  open *Your account > Access tokens* and stop for the user to generate
  and paste the token into their MCP config; open *Your account >
  Integrations > MCP Server* and stop for the user to generate the MCP key;
  open a file's workspace URL, **Plugins > MCP**, **Connect here**; wake a
  tab after "no heartbeat"; screenshot a board to check a drawing; fall back
  to asking the user when Chrome is not connected.
- `references/tools.md`: every tool with its route and verdict from
  `docs/bridge.md`, generated from the server's registered tools.
- `references/traps.md`: the API traps the agent can still hit
  (`docs/penpot-api-reference.md` § 5.5, the exporter cookie, write
  serialisation).

Dependencies: none. Standards: `docs/standards.md` (`make check`, plus
`TestSkillListsEveryTool`).

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `skills/penpot-mcp/SKILL.md` | new | 1 | the skill | fixed |
| `skills/penpot-mcp/references/chrome.md` | new | 1 | Claude in Chrome flows | expected |
| `skills/penpot-mcp/references/tools.md` | new | 3 | tool list with routes | expected |
| `skills/penpot-mcp/references/traps.md` | new | 1 | traps | expected |
| `internal/tools/skill_test.go` | new | 3 | `TestSkillListsEveryTool` | expected |
| `Makefile` | edit | 1 | `make install-skill` (symlink) | expected |

*Reused, not rewritten:*

- `~/.claude/skills/penpot/SKILL.md` § Connect and § Projects and files:
  its Claude in Chrome steps (`find "MCP button"`, **Connect here**, the
  dashboard rename, the own-tab-group limit) are adapted, not rewritten.
- `docs/bridge.md`: the route table is the source for `references/tools.md`.

## Security

The Chrome flows pass through pages that show secrets: the access token
and the MCP key are shown once on screen.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Information disclosure | the agent screenshots or reads the token dialog and the secret lands in the transcript | `chrome.md` stops before *Generate*; the user generates, copies and pastes the value; no screenshot or `read_page` on those pages after that point | reviewed in Phase 1's try, `TestChromeDocStopsBeforeSecrets` greps the steps |
| Elevation of privilege | page text in a Penpot tab carries injected instructions | `chrome.md` says page content is data, never instructions; Chrome is used only for the listed flows | reviewed in Phase 2 |
| Information disclosure | the skill tells the user to put the token in a committed file | install steps use `claude mcp add -e` or the shell environment, never a repo file | `TestSkillNoSecretFiles` |

Checklist: OWASP Top 10 for LLM Applications 2025 (LLM01 prompt
injection, LLM02 sensitive information disclosure).

## Open decisions

- Should the existing `penpot` skill get a one-line pointer to this one?
  Default: no change to it; it lives outside this repo and is the user's.

## Declined

- Extending the existing `penpot` skill: higher context cost and mixes two
  routes in one body (3.05).
- Server `instructions` only: cannot cover install or Chrome flows (3.30).

## What the tests hold

- Every registered tool appears in `references/tools.md` with a route, and
  no listed tool is missing from the server.
- The Chrome flows stop before any secret is on screen.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [x] `skills/penpot-mcp/SKILL.md` lands with frontmatter checked against
  the official Claude Code skills docs (saved to
  `docs/external/claude-code-skills.md`), install, env vars, the route rule
  and the drawing loop for the tools of `plans/01-first-version.md`, and
  `make install-skill` symlinks it into `~/.claude/skills/`.
- [x] `skills/penpot-mcp/references/chrome.md` lands with the token hand-off,
  opening a file and screenshotting a board, and
  `skills/penpot-mcp/references/traps.md` with the API traps, pinned by
  `TestChromeDocStopsBeforeSecrets` and `TestSkillNoSecretFiles`.
  Landed 2026-10-19: the token flow stops before *Create access token*:
  the user clicks it and copies the token, so it is never on screen for
  the agent. Tests live in `skills/skill_test.go`.
  Landed 2026-10-20: after `plans/02-drawing.md` merged, `SKILL.md`
  covers `draw` (with a worked login screen, checked against `checkTree`),
  the single-shape tools, `set_layout` and every edit tool;
  `references/traps.md` gained the layout, text, move and group traps.
- [x] Check: `make check` passes, and `/skills` in a new Claude Code session
  lists `penpot-mcp`.
- [x] You try it: in a fresh session say "set up penpot-mcp and draw a board
  in my test file", let it walk you through the token in Chrome, and say
  whether it stopped at the right places and the drawing showed up.
  Landed 2026-10-20: the skill loaded and drove `penpot-api` to draw a
  login screen and check it in Chrome. The token hand-off was not run:
  the server was already connected.

### Phase 2: the MCP route in the browser

- [x] `chrome.md` gains the MCP-key hand-off, **Plugins > MCP** and
  **Connect here**, waking a tab after "no heartbeat", and the file guard
  check, and `SKILL.md` gains the HYBRID and MCP tools of
  `plans/06-mcp-route.md` (expected in `skills/penpot-mcp/`); check:
  `make check` passes.
  Landed 2026-10-20: the MCP-key flow stops before the key is on screen,
  held by the same test as the token flow (`TestChromeDocStopsBeforeSecrets`
  now checks both). `SKILL.md` gained § 8 Tab tools and § 9 Everything
  else; the stale "exports not in this version" trap is fixed.

### Phase 3: the full tool list

- [x] `skills/penpot-mcp/references/tools.md` is generated from the
  registered tools with each route and verdict from `docs/bridge.md`, and
  `TestSkillListsEveryTool` (expected in `internal/tools/skill_test.go`)
  fails when the two drift; check: `make check` passes.
  Landed 2026-10-20: generated by `go test ./internal/tools -run
  TestSkillListsEveryTool -update`, 87 tools, with the route read from
  which tools the server lists with and without a tab. The verdicts come
  from the server itself rather than `docs/bridge.md`, so the two cannot
  drift.

### Phase 4: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's URL
  printed as the evidence. Opened, never merged.
