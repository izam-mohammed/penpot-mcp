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

coordinates are absolute page pixels, even inside boards. text is sized with
the real font metrics, no browser required (that was the whole point).

## setup

needs go 1.26.

```sh
make build   # writes bin/penpot-mcp
```

make a token in penpot: *your account > integrations > personal access
tokens*. it's shown once, so copy it.

```sh
claude mcp add penpot-api -s user \
  -e PENPOT_ACCESS_TOKEN=<your token> \
  -- /absolute/path/to/bin/penpot-mcp
```

restart claude code, run `/mcp`, and `penpot-api` should be there.

| env var | needed | what |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN` | yes | your personal access token, never logged |
| `PENPOT_BASE_URL` | no | default `https://design.penpot.app`; self-hosted needs `enable-access-tokens` |

## develop

```sh
make check   # format, lint, build, tests, govulncheck, gitleaks
make live    # draws into a real file, needs the token
```

what's next lives in `PLAN.md`. standards in `docs/standards.md`.
