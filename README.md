# penpot-mcp

An MCP server that drives [Penpot](https://penpot.app) through its backend
HTTP API. No browser tab, no plugin: it reads and writes any file your access
token can reach. One small Go binary, MCP over stdio.

Status: drawing and editing. See `PLAN.md` for what comes next and
`docs/bridge.md` for every planned tool.

## Tools

| Tool | Does |
| --- | --- |
| `whoami` | the account the token belongs to |
| `list_teams` | your teams |
| `list_projects` | projects, all teams or one |
| `list_files` | files in a project |
| `list_pages` | pages of a file |
| `get_page_tree` | a page's shapes, flat, top-most first: id, parent, depth, name, type, box |
| `create_board` | a board at the page root or inside another board |
| `create_rect` | a rectangle at the root or inside a board |
| `create_ellipse` | an ellipse filling a box |
| `create_text` | a text box, sized with the real font metrics; wraps when given a width |
| `create_path` | a vector path from SVG path data; every SVG command works |
| `create_image` | uploads a local PNG, JPEG, WebP, GIF or SVG and places it |
| `draw` | many shapes in one write (boards, rects, ellipses, text, paths, images, groups, bools, flex and grid boards): a whole screen per call |
| `set_layout` | gives a board a flex or grid layout and places its children |
| `update_shape` | name, position, size, rotation, fill, stroke, radius, opacity, visibility, or a text's text |
| `move_shape` | moves shapes by dx, dy with everything inside them |
| `reparent_shape` | moves shapes into another board, group or bool, or reorders them |
| `create_group`, `create_bool` | groups or combines shapes that share a parent |
| `delete_shapes` | deletes shapes and their children; never the root frame |
| `apply_changes` | last resort: raw `update-file` changes, checked first |

Coordinates are absolute page pixels, also for shapes inside a board.
Penpot stores layouts but only its editor reflows them, so `draw` and
`set_layout` place the children themselves, top layer first like Penpot.
Text is measured with the font's own metrics: Google fonts are fetched as
TTF from fonts.googleapis.com and Source Sans Pro from the Penpot instance,
then cached under the user cache directory (`penpot-mcp/fonts`).

## Build

Needs Go 1.26 (the `toolchain` line in `go.mod` fetches the pinned patch).

```sh
make build        # writes bin/penpot-mcp
```

## Configure

1. In Penpot, open *Your account > Integrations > Personal access tokens*
   and create a token. It is shown once.
2. Add the server to Claude Code, with the token in its environment:

   ```sh
   claude mcp add penpot-api -s user \
     -e PENPOT_ACCESS_TOKEN=<your token> \
     -- /absolute/path/to/bin/penpot-mcp
   ```

3. Restart Claude Code and run `/mcp`: `penpot-api` lists its tools.

| Env var | Needed | Meaning |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN` | yes | your personal access token |
| `PENPOT_BASE_URL` | no | default `https://design.penpot.app`; self-hosted: your `PENPOT_PUBLIC_URI` (needs `enable-access-tokens`) |

The binary checks the token at start and exits with
`PENPOT_ACCESS_TOKEN was not accepted` if Penpot rejects it. The token is
never logged.

## Develop

```sh
make tools   # golangci-lint and govulncheck
make check   # format, lint, build, race tests, govulncheck, gitleaks
make live    # draws into a file named "penpot-mcp test" (needs the token)
```

Standards: `docs/standards.md`. Proof scripts for each integration:
`scripts/prove/`.
