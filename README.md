# penpot-mcp

the official penpot mcp needs a browser tab open, a plugin awake, and a
little prayer. i got tired of babysittin' it, so this one skips the browser
and talks to penpot's backend api directly ~ one small go binary, mcp over
stdio.

## what it does

- reads: teams, projects, files, pages, a page's shape tree
- dashboard: projects, files and pages: create, rename, move, duplicate,
  delete (with `confirm: true`) and restore, plus libraries and members
- draws: boards, rects, ellipses, text, svg paths, images, groups, bools,
  flex and grid layouts
- `draw` builds a whole screen in one write
- edits: update, move, reparent, delete, plus `apply_changes` for when
  nothing else fits
- library: colours, type styles, components and their copies, design
  tokens, and a palette pulled straight from your repo's css, swift,
  kotlin, markdown or json
- exports: boards to png, jpeg, webp, svg or pdf on disk, whole files to
  and from `.penpot`
- people: comments and replies, share links, webhooks, saved versions
  and restores (which save a version first, because trust issues)
- with a penpot tab open (optional): plugin code, reflow, real text
  sizes, html and css, component swaps and variants, through penpot's
  own mcp. the old one, demoted to sidekick

coordinates are absolute page pixels, even inside boards. text is sized with
the real font metrics, no browser required (that was the whole point).

## install

grab a binary from the [releases](https://github.com/izam-mohammed/penpot-mcp/releases)
page (macos, linux, windows; amd64 and arm64), check it against
`checksums.txt`, and put `penpot-mcp` on your path. or, on a mac:

```sh
brew install --cask izam-mohammed/tap/penpot-mcp
```

or build it, with go 1.26:

```sh
make build   # writes bin/penpot-mcp
```

## setup

make a token in penpot: *your account > integrations > personal access
tokens*. it's shown once, so copy it.

```sh
claude mcp add penpot-api -s user \
  -e PENPOT_ACCESS_TOKEN=<your token> \
  -- /absolute/path/to/bin/penpot-mcp
```

restart claude code, run `/mcp`, and `penpot-api` should be there.

### over http

for a client on another machine, or one that only speaks http:

```sh
penpot-mcp --http 127.0.0.1:8787
claude mcp add --transport http penpot-api http://127.0.0.1:8787/mcp
```

it listens on loopback only, unless you set `PENPOT_MCP_HTTP_TOKEN` (16+
characters) and send it as `Authorization: Bearer <token>`. a public port
without a token would hand your penpot account to whoever finds it, so
it refuses to start.

### several accounts

point `PENPOT_PROFILES` at a json file listing the accounts, by the names
of the env vars that hold their secrets (never the secrets themselves):

```json
[
  {"name": "work", "token_env": "PENPOT_WORK_TOKEN"},
  {"name": "home", "token_env": "PENPOT_HOME_TOKEN", "base_url": "https://penpot.example.com"}
]
```

every tool then takes a `profile` argument. each account gets its own
write queue, so one slow account never holds up another.

| env var | needed | what |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN` | yes | your personal access token, never logged |
| `PENPOT_BASE_URL` | no | default `https://design.penpot.app`; self-hosted needs `enable-access-tokens` |
| `PENPOT_MCP_URL` | for tab tools | *your account > integrations > mcp server*. the whole url, key included |
| `PENPOT_EMAIL`, `PENPOT_PASSWORD` | for board exports | penpot's exporter only takes a login session, not a token. read once, never logged |
| `PENPOT_PROFILES` | for several accounts | path to the profiles json above; profiles may also name `email_env`, `password_env` and `mcp_url_env` |
| `PENPOT_MCP_HTTP_TOKEN` | for `--http` off loopback | the bearer token http clients must send |

## develop

```sh
make check   # format, lint, build, tests, govulncheck, gitleaks
make live    # draws into a real file, needs the token
make live-mcp  # the tab tools, with the test file open and the plugin connected
goreleaser check  # before tagging a release; a v* tag runs .github/workflows/release.yml
```

what's next lives in `PLAN.md`. standards in `docs/standards.md`.
