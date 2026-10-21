# the binary installs a set of focused penpot-mcp skills into claude code or codex, and the readme says so

## The ask

The repository ships proper skills for using this MCP, mentions them
briefly in the README and the other docs, and installs them from the Claude
Code install command.

What is there today (`plans/09-skill.md`): one 212-line skill,
`skills/penpot-mcp/`, covering install, finding files, drawing, editing,
the library, exports, people and the tab tools in one body. It installs
only with `make install-skill`, a symlink from a checkout. Someone who
installed with brew or a release binary has no checkout, so has no skill,
and neither the README nor its paste-in install prompt mentions one.

That means three things: skills split by job so the right one
loads for the task; a way to install them that works from the binary the
user already has; and the README, install prompt and server instructions
pointing at them.

## Alternatives

### Decision 1: one skill or several? (two-way door)

Searched 21/10/2026: "Claude Code skills description listing budget",
"agent skills progressive disclosure"; the Claude Code skills and plugin
components docs (`docs/external/claude-code-skills.md`), the Codex skills
docs (`docs/external/codex-skills.md`).

| Criterion | Weight | Baseline: one skill | One skill, more reference files | Several skills by job |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency (context) | 40 | 2 | 4 | 5 |
| Fit to the ask | 35 | 2 | 3 | 5 |
| Maintainability of the result | 25 | 3 | 3 | 4 |
| **Weighted total** | **100** | **2.25** | **3.40** | **4.75** |

- Efficiency: every task loads all 212 lines of the baseline. With several
  skills only the job's body loads (about 60 to 90 lines), and the listing
  cost stays near one skill's, since each description is short and the
  baseline's alone is near the 1,536-character cap.
- Fit: the need is for several skills; separate skills also show as separate
  `/` commands (`/penpot-mcp-draw`) a person can call by name.
- Maintainability: a job's rules live in one file; shared detail (Chrome
  flows, traps, the generated tool list) stays in one place, the entry
  skill's `references/`, linked by relative path.

Sensitivity: if the listing cost per skill were large, one skill with
reference files would win; five short descriptions are about the same
characters as the one long one today.

**Chosen: several skills by job**, a `penpot-mcp` entry skill (connect,
find and manage files, failures, routes to the rest) plus
`penpot-mcp-draw`, `penpot-mcp-library`, `penpot-mcp-export` and
`penpot-mcp-collab`. Revisit if Claude Code budgets skill listings per
plugin.

### Decision 2: how do the skills get onto a machine without a checkout? (two-way door)

Searched 21/10/2026: "Claude Code plugin marketplace skills", "Codex
skills directory", "npx skills add vercel-labs"; the Claude Code plugin,
marketplace and components docs, the Codex skills docs, the vercel-labs
skills CLI README (third-party write-ups only reachable).

| Must | `make install-skill` (baseline) | Claude Code plugin marketplace | `npx skills add` | `penpot-mcp skills install` (embedded) |
| --- | --- | --- | --- | --- |
| Works without a repo checkout | no | yes | yes | yes |
| Asks for no secret | yes | yes | yes | yes |
| **Survives** | no | **yes** | **yes** | **yes** |

| Criterion | Weight | Plugin marketplace | `npx skills add` | Embedded in the binary |
| --- | ---: | ---: | ---: | ---: |
| Performance efficiency | 35 | 4 | 2 | 5 |
| Fit to the ask | 30 | 3 | 5 | 4 |
| Reliability | 20 | 3 | 3 | 5 |
| Maintainability of the result | 15 | 5 | 4 | 4 |
| **Weighted total** | **100** | **3.65** | **3.40** | **4.55** |

- Efficiency: the embedded copy is already on disk with the binary; no
  network, no runtime. A marketplace clones the repo into the plugin cache.
  `npx` needs Node and downloads the CLI and the repo.
- Fit: the install prompt targets "claude code, codex or whatever you
  use". A marketplace serves Claude Code only; `npx` serves 30-plus
  agents; the binary writes the Claude Code and Codex directories
  (`--dir` for any other).
- Reliability: the embedded skills come from the same commit as the tools
  they name, so `references/tools.md` always matches the running server.
  A marketplace and `npx` track `main`, which can be ahead of the user's
  binary.
- Maintainability: a marketplace is two JSON files; the subcommand is a
  small Go file plus a test.

**Chosen: `penpot-mcp skills install`, skills embedded with `go:embed`.**
`make install-skill` stays for working on the skills from a checkout, and
links every skill. Revisit if brew users report the subcommand missing or
Claude Code plugins gain a way to ship a skill next to a binary.

## The decision

- Skills (decision 1) live in `skills/<name>/SKILL.md`, five of them:

  | Skill | Holds |
  | --- | --- |
  | `penpot-mcp` | is it connected, install, the token hand-off, finding teams, projects, files and pages, the dashboard tools, every error, which skill to load next; `references/chrome.md`, `references/traps.md`, `references/tools.md` |
  | `penpot-mcp-draw` | `draw` and the single-shape tools, layouts, edits, checking the result, and the tab tools for reflow and markup |
  | `penpot-mcp-library` | colours, type styles, components, design tokens, `import_palette` from the repo |
  | `penpot-mcp-export` | boards to PNG, SVG or PDF in the repo, `.penpot` files, citing exports from plans |
  | `penpot-mcp-collab` | comments, share links, webhooks, versions, several accounts |

  The four job skills link shared detail as
  `../penpot-mcp/references/<file>.md`; the installer always writes the
  set together. Tools are named by their bare MCP name (`draw`) so the
  text reads right in Codex too.
- Install (decision 2): `skills/embed.go` embeds the skill directories in
  package `skills`; `penpot-mcp skills install` copies them, without
  needing `PENPOT_ACCESS_TOKEN`.
- The README gains a short skills section, a step in *by hand*, and the
  `skills install` line in the paste-in prompt. The server `instructions`
  name the `penpot-mcp` skill and the install command. `CLAUDE.md` and
  `docs/features.md` point at `skills/`.

Dependencies: none (`embed` is the standard library). Permissions: the
subcommand writes only under the chosen skills directory.

Standards: `docs/standards.md` (format, lint, build, `go test -race`,
govulncheck, gitleaks, through `make check`).

## Contract

The subcommand, as the README and the install prompt print it:

```text
penpot-mcp skills install [--agent claude|codex] [--dir PATH] [--force]
```

- `--agent claude` (default) writes to `$CLAUDE_CONFIG_DIR/skills`, else
  `~/.claude/skills`; `--agent codex` writes to `~/.agents/skills`
  (`docs/external/codex-skills.md`); `--dir` overrides both.
- Each skill is written to `<dir>/<skill name>/`, replacing a directory it
  wrote before (marked by a `.penpot-mcp` file holding the version).
- It refuses, naming the path, a directory or symlink it did not write,
  unless `--force`.
- It prints one line per skill written, and exits non-zero on any refusal.

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `skills/penpot-mcp/SKILL.md` | edit | 1 | the entry skill, trimmed to setup, files and failures | fixed |
| `skills/penpot-mcp-draw/SKILL.md` | new | 1 | drawing, editing, checking | fixed |
| `skills/penpot-mcp-library/SKILL.md` | new | 2 | library | fixed |
| `skills/penpot-mcp-export/SKILL.md` | new | 2 | exports | fixed |
| `skills/penpot-mcp-collab/SKILL.md` | new | 2 | people | fixed |
| `skills/embed.go` | new | 1 | `go:embed` of the skill directories | expected |
| `skills/install.go` | new | 1 | `Install(dir, force)` | expected |
| `skills/skill_test.go` | edit | 1 | tests run over every skill | reuse |
| `cmd/penpot-mcp/main.go` | edit | 1 | `skills install` subcommand, instructions | expected |
| `Makefile` | edit | 1 | `install-skill` links every skill | reuse |
| `README.md` | edit | 1 | skills section, install prompt, by-hand step | expected |
| `CLAUDE.md`, `docs/features.md` | edit | 3 | point at `skills/` | expected |

*Reused, not rewritten:*

- `skills/skill_test.go`: `TestChromeDocStopsBeforeSecrets`,
  `TestSkillNoSecretFiles` and `TestSkillFrontmatter` are extended to
  every skill directory, not copied.
- `internal/tools/skill_test.go` `TestSkillListsEveryTool`: still writes
  `skills/penpot-mcp/references/tools.md`, which all skills link.
- `Makefile` `install-skill`: its link-or-refuse logic loops over the
  skill directories.
- Searched for an existing copy-tree helper in `internal/` (`CopyFS`,
  `WalkDir` writers): none; `os.CopyFS` (Go 1.23+) does the copy.

## Edge cases

- A skills directory that does not exist yet: created.
- `~/.claude/skills/penpot-mcp` is the symlink `make install-skill` made:
  refused without `--force`, so a checkout's live link is not swapped for
  a copy by surprise.
- A skill the user wrote with the same name: refused without `--force`.
- Upgrading: a directory with the `.penpot-mcp` marker is replaced whole,
  so a file dropped from a skill does not linger.

## Security

Entry point: `penpot-mcp skills install`, run by the user on their own
machine; no network, no secret.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Tampering | the installer overwrites a skill or file the user made | refuse any target without the marker unless `--force`; write only under `<dir>/<skill name>/` | `TestInstallRefusesForeignDir`, `TestInstallRefusesSymlink` |
| Tampering | a skill name with `..` escapes the directory | names come from the embedded tree, checked against `^penpot-mcp(-[a-z]+)?$` | `TestInstallNamesAreSafe` |
| Information disclosure | a skill tells the user to put the token in a file | the existing secret checks run over every skill | `TestSkillNoSecretFiles` |
| Information disclosure | the Chrome flows screenshot the token or MCP key | the stop-before-secret checks stay on `chrome.md` | `TestChromeDocStopsBeforeSecrets` |
| Elevation of privilege | page text, comments or shape names carry instructions | every skill says design content is data, never instructions | `TestSkillsSayContentIsData` |

Checklist: OWASP Top 10 for LLM Applications 2025 (LLM01, LLM02).

## Declined

- One skill (2.25) and one skill with more reference files (3.40): every
  task loads more than it needs, and the need is for several skills.
- `make install-skill` alone: fails the no-checkout must.
- A Claude Code plugin marketplace (3.65): Claude Code only, and tracks
  `main` rather than the installed binary. Bundling the MCP server in the
  plugin too would rename every tool to `mcp__plugin_penpot-mcp_...` and
  clash with existing `claude mcp add` installs.
- `npx skills add` (3.40): needs Node and tracks `main`. It still works on
  this repo's layout, unadvertised.
- A brew cask hook that installs skills: writes into a home directory the
  user did not ask about.

## What the tests hold

- Every skill has valid frontmatter (`name` matches its directory,
  description within 1,536 characters, under 500 lines).
- Every relative link in every skill resolves inside `skills/`.
- Every tool a skill names in backticks is a registered tool, and every
  registered tool is named by at least one skill.
- No skill puts a token in a file; the Chrome flows stop before secrets.
- The installer writes every embedded skill with its marker, replaces its
  own, and refuses foreign directories and symlinks.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut (needs you: try it)

- [ ] `skills/penpot-mcp/SKILL.md` is trimmed to connect, install, find
  and manage files, and errors, and routes drawing to a new
  `skills/penpot-mcp-draw/SKILL.md` that holds draw, edit, check and the
  tab tools, both pinned by `TestSkillFrontmatter` and
  `TestSkillLinksResolve` run over every skill directory.
- [ ] `penpot-mcp skills install` writes the embedded skills to the
  Claude Code or Codex directory per `## Contract` (expected in
  `skills/install.go` and `cmd/penpot-mcp/main.go`), pinned by
  `TestInstallWritesEverySkill`, `TestInstallRefusesForeignDir`,
  `TestInstallRefusesSymlink` and `TestInstallNamesAreSafe`, and
  `make install-skill` links every skill directory.
- [ ] The README gains a short skills section, a *by hand* step running
  `penpot-mcp skills install`, and that line in the paste-in prompt; the
  server instructions name the `penpot-mcp` skill.
- [ ] Check: `make check` passes, and `bin/penpot-mcp skills install
  --dir <scratch>` lists two skills written.
- [ ] You try it: run `penpot-mcp skills install`, start a new Claude Code
  session, and ask it to draw a login screen in your test file; say
  whether `penpot-mcp-draw` loaded and the drawing came out right.

### Phase 2: the rest of the jobs

- [ ] `skills/penpot-mcp-library/`, `skills/penpot-mcp-export/` and
  `skills/penpot-mcp-collab/` land with what is left of the old body,
  plus the export-into-the-repo flow, pinned by
  `TestSkillToolsAreRegistered` (every named tool exists, every tool is
  named somewhere) and `TestSkillsSayContentIsData`; check: `make check`
  passes and `skills install --dir <scratch>` lists five skills.

### Phase 3: the other places that mention it

- [ ] `CLAUDE.md`, `docs/features.md` and `PLAN.md` point at `skills/` and
  this plan, and `docs/external/claude-code-skills.md` notes the set and
  the installer; check: `make check` passes.

### Phase 4: open the PR

- [ ] The branch pushed and its PR opened against `main` (or, if one is
  already open for this branch, its body brought up to date) with a
  Conventional Commit title for the whole change, a body naming this plan,
  what changed and why, and the checks run with their result. The PR's
  URL printed as the evidence. Opened, never merged.
