# penpot-mcp has a documentation site with a page for every tool, generated from the server

## The ask

A tool that needs detailed documentation gets it, planned in its own
repository.

Built so far: `docs/built/BUILT-2026-10-22-1826.md`.

ragrank is the model: a Sphinx site on Read the Docs with getting started,
guides and an API reference (`ragrank/docs/docs/`, `ragrank/docs/api_reference/`,
`.github/workflows/build-docs.yml`, which pings two Read the Docs webhooks on
a release).

What it means here: penpot-mcp is a public MCP server people install with
Homebrew or a release binary. It registers 87 tools
(`skills/penpot-mcp/references/tools.md`), reads seven environment
variables (`PENPOT_ACCESS_TOKEN`, `PENPOT_BASE_URL`, `PENPOT_MCP_URL`,
`PENPOT_EMAIL`, `PENPOT_PASSWORD`, `PENPOT_PROFILES`,
`PENPOT_MCP_HTTP_TOKEN`), has an HTTP mode (`--http`), a `skills install`
command with three flags, four routes (API, login or tab, hybrid, tab) and
self-hosted Penpot support. The README covers install and lists the env
vars in one table; the per-tool detail (parameters, route, what each
returns) lives only in the agent skills and the code. A person setting it
up, or choosing whether a tool needs a tab, has nowhere to read it.
Detailed docs are required: an MCP server other people install, with many
tools and config keys.

The repository already generates one reference from the server:
`TestSkillListsEveryTool` (`internal/tools/skill_test.go`) lists every
registered tool and its route into
`skills/penpot-mcp/references/tools.md`, and fails when the file differs.
`docs/` holds developer references (`penpot-api-reference.md`,
`bridge.md`, `features.md`, `docs/external/`); user docs must not mix with
them.

`plans/12-site.md` makes `site/index.html`, a one-page landing page on
GitHub Pages. The site is the landing page; this plan's docs are the
reference, served from the same Pages deploy under `/docs/`.

## Alternatives

### Decision 1: what builds the docs? (two-way door)

The pages are Markdown; moving to another generator later means a new
config and theme, not rewritten content. The docs need a sidebar, search
across 87 tool pages, light and dark themes, a way to include generated
files, and static output Pages can serve.

Searched 2026-10-22: "MkDocs Material vs Starlight vs Sphinx vs mdBook vs
Docusaurus 2026", "Material for MkDocs maintenance mode Zensical 2026",
"Zensical release status", "mdBook 0.5 release", "Hugo Hextra theme
FlexSearch hugo module", "mdBook vs Starlight vs Hugo Hextra page weight";
the Material for MkDocs announcement of maintenance mode (9.7.0,
11/11/2025, critical fixes for at least 12 months), Zensical's own docs
(alpha, 0.0.x versions), the mdBook 0.5 changelog, the Hextra theme page,
and ragrank's own Sphinx setup. No published like-for-like benchmark of
page weight or build time across these was found.

| Must | mdBook | Hugo with the Hextra theme | Starlight (Astro) | Sphinx with MyST (ragrank's setup) | Docusaurus | MkDocs Material | Zensical | pkg.go.dev |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Static output Pages can serve | yes | yes | yes | yes | yes | yes | yes | no (hosted by Google) |
| Documents tools and setup for a user, not Go packages | yes | yes | yes | yes | yes | yes | yes | no (`internal/` packages only) |
| Still maintained, with a stable release | yes | yes | yes | yes | yes | no (maintenance mode since 11/2025) | no (alpha) | yes |
| **Survives** | **yes** | **yes** | **yes** | **yes** | **yes** | no | no | no |

| Criterion | Weight | mdBook | Hugo with Hextra | Starlight | Sphinx with MyST (ragrank's setup) | Docusaurus |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Performance efficiency | 35 | 4 | 4 | 4 | 3 | 2 |
| Fit to the ask | 25 | 4 | 5 | 5 | 4 | 5 |
| Security | 15 | 5 | 3 | 2 | 3 | 2 |
| Maintainability of the result | 15 | 5 | 3 | 3 | 3 | 2 |
| Compatibility | 10 | 4 | 5 | 2 | 2 | 2 |
| **Weighted total** | **100** | **4.30** | **4.05** | **3.60** | **3.15** | **2.75** |

- Efficiency: mdBook, Hugo and Starlight all write static HTML from one
  build with no runtime; each page still loads a search script (mdBook's
  built-in index, Hextra's FlexSearch, Starlight's Pagefind), so none is
  scored 5. Sphinx 3: a Python build and a theme's scripts. Docusaurus 2:
  a React app that hydrates on every page. No byte counts were found to
  separate the first three; Phase 1 records the built size of the first
  page as the baseline, and Phase 2 the search index size with 87 tool
  pages.
- Fit: Hextra, Starlight and Docusaurus have richer components (tabs,
  callouts); mdBook has the sidebar, search, themes and `{{#include}}` for
  generated files, which is all the ask needs (4). Sphinx's strength is
  Python autodoc, which a Go server cannot use (4).
- Security: mdBook is one release binary pinned by version and checksum,
  with no package tree (5). Hugo adds a third-party theme fetched as a
  module (3). Starlight and Docusaurus add Node and an npm tree (2). Sphinx
  adds a pip tree (3).
- Maintainability: mdBook is `book.toml` and `SUMMARY.md` (5); the others
  add a theme pin, a lockfile or a `conf.py` (3, Docusaurus 2).
- Compatibility: Hugo is a Go binary like golangci-lint in the Makefile's
  `tools` target (5); mdBook is a pinned binary like gitleaks (4); Node or
  Python in a Go repository (2).

**Chosen: mdBook.** It meets the ask with the fewest moving parts and the
smallest dependency surface. Sphinx, ragrank's setup, scored lower here
because its main gain (Python API autodoc) does not apply to a Go server;
see `## Open decisions`. Revisit if search over the tool pages is slow on a
phone in the Phase 2 try, or versioned docs per release are wanted.

### Decision 2: where are the docs served? (two-way door)

They have to be reachable at a URL, built on `main`, with no person
deploying them, next to the landing page from `plans/12-site.md`.

Searched 2026-10-22: "Read the Docs vs GitHub Pages hosting project
documentation private repository free plan 2026", "GitHub Pages
availability private repositories GitHub Free"; the Read the Docs pricing
page (Community is free for open source and ad-supported), GitHub's Pages
limits page (free for public repositories; penpot-mcp is public), and
`plans/12-site.md` decision 1.

| Must | Baseline: GitHub Pages, the `pages.yml` from `plans/12-site.md`, under `/docs/` | Read the Docs Community (ragrank's setup) | Cloudflare Pages | Vercel |
| --- | --- | --- | --- | --- |
| No new account, service or secret | yes | no (a Read the Docs account; ragrank keeps webhook secrets) | no (account and API token) | no (account and token) |
| No ads on the docs | yes | no (Community is ad-supported) | yes | yes |
| **Survives** | **yes** | no | no | no |

Only the baseline survives, so there is nothing to score against it.

**Chosen: GitHub Pages, from the same `pages.yml`, at
`https://izam-mohammed.github.io/penpot-mcp/docs/`.** The landing page
stays at the root and links to `/docs/`; one deploy publishes both. Revisit
if the docs need versions per release or pull request previews.

### Decision 3: how do the reference pages stay true to the server? (two-way door)

87 tool pages, the CLI flags and the env vars must not drift from the
binary.

Searched 2026-10-22: "generate MCP tool reference markdown from tools/list
input schema", "go flag PrintDefaults generate docs"; the repository's own
`TestSkillListsEveryTool` and its `-update` flag, and the MCP Go SDK's
`ListTools` reply (name, description, input schema) as used there.

| Criterion | Baseline: hand-written pages | Extend the existing test to write the docs from `ListTools` | A script calls `tools/list` on the built binary at docs build time |
| --- | --- | --- | --- |
| Performance efficiency | nothing to run | in `go test`, against the in-process session it already starts | a build and a server start before every docs build |
| Fit to the ask | drifts | cannot drift; `make check` fails | cannot drift, but no route marks (they come from the test's maps) |
| Compatibility | none | reuses `renderToolsDoc`, `session`, `sessionWith` | a second generator beside the first |

**Chosen: extend the existing test.** `TestDocsListEveryTool` (expected in
`internal/tools/docs_test.go`) uses the same sessions as
`TestSkillListsEveryTool` to write one page per tool group with each
tool's description, route and a parameter table from its input schema;
`-update` writes, otherwise it fails on a difference. The CLI page is
written the same way from the flag sets in `cmd/penpot-mcp/main.go`, and a
test fails when an env var name the code reads is missing from the
configuration page. Revisit if the tool count makes grouped pages too long
to scan; then one page per tool.

### Decision 4: what checks the built docs? (two-way door)

A broken link or a missing chapter should fail `make check`, offline.

Searched 2026-10-22: "lychee link checker offline mode local html vs
mdbook-linkcheck mdbook 0.5 compatibility"; lychee's docs (`--offline`
checks local files and HTML), mdBook's `build.create-missing` option. No
statement was found on whether `mdbook-linkcheck` supports mdBook 0.5.

| Criterion | Baseline: `mdbook build` only | `mdbook-linkcheck` backend | lychee `--offline` on the built HTML |
| --- | --- | --- | --- |
| Performance efficiency | fastest | runs inside the build | one extra pass over local files, no network |
| Fit to the ask | missing chapters only | links and anchors | links, local files; anchors to be confirmed in Phase 4 |
| Reliability | misses broken links | `?` on mdBook 0.5 | independent of the generator |

**Chosen: `mdbook build` with `create-missing = false`, any warning
treated as a failure, then lychee `--offline` on the output.** Revisit if
lychee misses broken anchors in Phase 4's planted-link test.

## The decision

The docs are an mdBook book in `docs/book/` (`book.toml`,
`src/SUMMARY.md`, decision 1), apart from the developer references already
in `docs/`, built into `site/docs/` by the Pages workflow from
`plans/12-site.md` and published at `/docs/` beside the landing page
(decision 2). The tool reference, the CLI page and the env var check are
generated or held by tests in `internal/tools/` and `cmd/penpot-mcp/`
(decision 3). `make docs` builds the book and runs lychee offline;
`make check` runs it (decision 4). The getting-started text keeps the
README's voice; reference pages are plain.

Chapters: getting started (install, the token, `claude mcp add`, skills,
first drawing), configuration (every env var, profiles, self-hosted
Penpot, HTTP mode and its token), routes (API, login or tab, hybrid, tab,
and what needs a tab), guides by job (files and projects, drawing whole
screens, library and tokens, export, comments and sharing, several
accounts), the generated tool and CLI reference, and troubleshooting from
the errors the server returns.

New dependencies: `mdbook` and `lychee`, each a release binary pinned by
version and SHA-256 in the Makefile's `tools` target and in CI. No new
secret or permission; the Pages permissions are `plans/12-site.md`'s.

Standards: `docs/standards.md` (format, lint, build, tests, govulncheck
and gitleaks through `make check`); this plan adds `make docs` to it.

## Where it lands

| Path | New or edit | Phase | What changes | Firm? |
| --- | --- | --- | --- | --- |
| `docs/book/book.toml`, `docs/book/src/SUMMARY.md` | new | 1 | the book | expected |
| `docs/book/src/getting-started.md` | new | 1 | install to first drawing | expected |
| `internal/tools/docs_test.go` | new | 2 | `TestDocsListEveryTool` | expected |
| `docs/book/src/reference/tools/*.md` | new | 2 | generated tool pages | fixed |
| `cmd/penpot-mcp/docs_test.go` | new | 2 | the CLI page and the env var check | expected |
| `docs/book/src/configuration.md`, `routes.md`, `guides/*.md` | new | 3 | configuration, routes and guides | expected |
| `Makefile` | edit | 1, 4 | `make docs`, tools, in `make check` | reuse |
| `.github/workflows/pages.yml` | edit | 4 | builds the book into `site/docs/` (the file `plans/12-site.md` lands) | expected |
| `site/index.html`, `README.md` | edit | 4 | link to `/docs/` | expected |
| `docs/standards.md`, `CLAUDE.md` | edit | 4 | the docs check; regenerate docs with the skill | reuse |

*Reused, not rewritten:*

- `internal/tools/skill_test.go` `renderToolsDoc`, `session`,
  `sessionWith`, `hybrid`, `exporterTools`: the tool list and route marks
  the docs pages are written from.
- `README.md`'s install steps and extras table, and the job skills under
  `skills/penpot-mcp*/`: the source text of getting started and the
  guides, adapted for a person, not copied for an agent.
- `.github/workflows/pages.yml` from `plans/12-site.md`: one deploy for
  both.

## Security

The docs are static and public. The risks are the two new binaries in the
build, the deploy path, and a credential shown in an example.

| Threat | Here | Handled by | Proved by |
| --- | --- | --- | --- |
| Tampering: a replaced `mdbook` or `lychee` download | the `tools` target and CI | pinned version and SHA-256, the build stops on a mismatch | a wrong checksum in a scratch copy fails `make tools` |
| Elevation of privilege: a PR deploys the docs | `pages.yml` | `plans/12-site.md`'s triggers and job permissions, unchanged | the PR shows no Pages run |
| Information disclosure: a token or MCP URL key in an example | configuration and getting-started pages | placeholders only (`<your token>`, `<mcp url>`); generated pages carry descriptions and schemas, never values from the environment; gitleaks in `make check` | gitleaks clean; the generator test runs with no `PENPOT_` variable set |
| Information disclosure: a third-party request from a page | theme assets | mdBook's theme is served from the same origin; no analytics | the network panel in the try shows only the Pages host |

No data collected, no cookies. The HTTP-mode page repeats the server's
rule that it refuses to listen off loopback without
`PENPOT_MCP_HTTP_TOKEN`. The checklist followed: GitHub's Pages security
notes for custom workflows and the shared gate's workflow rules.

## Open decisions

- ragrank's stack (Sphinx on Read the Docs) was the model named in the
  ask and lost here (decisions 1 and 2): default, mdBook on GitHub Pages,
  unless the user wants ragrank's stack kept for consistency.

## Declined

- MkDocs Material: failed the must (maintenance mode since 11/2025).
- Zensical: failed the must (alpha).
- pkg.go.dev: failed the musts (Google-hosted; documents packages, not
  tools).
- Hugo with Hextra: lost on security and maintainability (a third-party
  theme fetched each build).
- Starlight, Docusaurus: lost on security and compatibility (Node and an
  npm tree in a Go repository); Docusaurus also on efficiency.
- Sphinx with MyST: lost on efficiency and compatibility; its autodoc
  does not apply to Go.
- Read the Docs, Cloudflare Pages, Vercel: failed the musts (a new
  account; Read the Docs Community shows ads).
- Hand-written tool pages, and a second generator calling the built
  binary: drift, or a second copy of the route marks.
- `mdbook-linkcheck`: support for mdBook 0.5 unconfirmed.

## What the tests hold

`TestDocsListEveryTool` holds that every registered tool has a page with
its route and parameters as the server reports them. The CLI test holds
the flags; the env var test holds that every `PENPOT_` name the code reads
is on the configuration page. `make docs` holds that the book builds
without warnings and every local link resolves. The guides' prose is
checked by reading in the tries.

## Repository knowledge

Read root `KNOWLEDGE.md` when present before each phase. Follow the global and
repository instructions for recording useful development findings.

## Phases

### Phase 1: the rough cut: the book builds locally (needs you: try it)

- [ ] `docs/book/` with `book.toml` (`create-missing = false`),
  `SUMMARY.md`, an introduction and `getting-started.md` (install, the
  token, `claude mcp add`, `skills install`, the first `draw`), from the
  README's install steps.
- [ ] `make docs` builds the book with a pinned `mdbook`, failing on any
  warning, and `make tools` installs it by version and checksum; the
  built size of `getting-started.html` and its assets recorded in the
  phase's commit message as the baseline.
- [ ] Check: `make check` passes, and `make docs` writes
  `docs/book/book/index.html`.
- [ ] You try it: run `make docs` and `mdbook serve docs/book`, open the
  getting-started page in light and dark, follow it on a clean shell up to
  `/mcp` listing `penpot-api`, and say whether this is the shape you want
  before the rest is written.

### Phase 2: the tool and CLI reference, generated

- [ ] `TestDocsListEveryTool` (expected in `internal/tools/docs_test.go`)
  writes `docs/book/src/reference/tools/<group>.md` from the registered
  tools with `-update`, each tool with its description, route and a
  parameter table from its input schema, and fails when a page differs;
  `SUMMARY.md` lists the groups.
- [ ] A test in `cmd/penpot-mcp/` writes `reference/cli.md` from the main
  and `skills install` flag sets, and fails when it differs.
- [ ] `CLAUDE.md` says a tool added or renamed also means
  `go test ./internal/tools -run TestDocsListEveryTool -update`.
- [ ] Check: `make check` passes, and a renamed tool in a scratch copy
  fails `go test ./internal/tools` until `-update` is run; the search
  index size recorded in the commit message.

### Phase 3: configuration, routes and guides

- [ ] `configuration.md` with every env var, profiles, self-hosted Penpot
  and HTTP mode, held by a test that fails when a `PENPOT_` name the code
  reads is missing from it.
- [ ] `routes.md` (API, login or tab, hybrid, tab, and connecting the
  official MCP), and one guide per job skill: files and projects, drawing,
  library and tokens, export, collaboration and several accounts, each
  with a worked example from a real run.
- [ ] A troubleshooting page from the errors the server returns, each with
  what to do.
- [ ] Check: `make check` passes, and `make docs` builds with no warning.
- [ ] You try it: read the routes page and one guide, and say whether a
  person new to penpot-mcp could set up exports from it.

### Phase 4: the docs check and publishing

- [ ] `make docs` also runs a pinned lychee `--offline` on the built book,
  and `make check` runs `make docs`; a planted broken link and a planted
  broken anchor each fail it.
- [ ] `pages.yml` builds the book into `site/docs/` before uploading
  `site/`, so one deploy serves both; `site/index.html` and `README.md`
  link `/docs/`.
- [ ] `docs/standards.md` names `make docs`, mdBook and lychee under
  checks and decided.
- [ ] Check: `make check` passes, and after the workflow runs on `main`,
  `https://izam-mohammed.github.io/penpot-mcp/docs/` answers 200.

### Phase 5: open the PR

- [ ] The branch pushed and its PR opened against the branch it was cut from
  (or, if one is already open for this branch, bring the open PR's body up
  to date) with a Conventional Commit title for the whole change, a body
  naming this plan, what changed and why, and the checks run with their
  result. The PR's URL printed as the evidence. Opened, never merged.
