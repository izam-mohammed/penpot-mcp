---
name: penpot-mcp
description: "Set up and drive the penpot-mcp server (the `penpot-api` MCP tools) to read and draw in Penpot through its backend API, with no browser tab: install and connect it, find teams, projects, files and pages, draw boards and rectangles, check the result. Uses Claude in Chrome only for what needs a browser: making the access token and looking at a drawing."
when_to_use: "connect penpot-mcp, add the penpot api mcp, penpot access token, list penpot projects or files, draw in penpot without the plugin, mcp__penpot-api tools, PENPOT_ACCESS_TOKEN was not accepted, penpot-mcp not connected"
---

# penpot-mcp

An MCP server (`penpot-api`) that drives Penpot through its backend HTTP
API with a personal access token. No plugin, no tab, any file the token can
reach. This skill covers installing it, using its tools, and the browser
steps Claude in Chrome does.

For the official Penpot MCP (plugin route, `execute_code`), use the
`penpot` skill instead.

## 1. Is it connected?

Look for `mcp__penpot-api__*` tools (load them with one `ToolSearch`,
`select:` all of them). Present: call `whoami` and go to section 3.

Absent: check `claude mcp list` for `penpot-api`.

- Listed but no tools: the server failed to start. Run the binary once by
  hand to see why: `PENPOT_ACCESS_TOKEN=... bin/penpot-mcp </dev/null`
  is the user's to run (never paste the token yourself). The common error
  is `PENPOT_ACCESS_TOKEN was not accepted`: make a new token (section 2).
- Not listed: install (section 2).

## 2. Install

1. Build: in the penpot-mcp repo, `make build` writes `bin/penpot-mcp`.
2. Token: the user needs a personal access token. With Claude in Chrome,
   follow [references/chrome.md](references/chrome.md) § Make an access
   token, which stops before the token is on screen. Without Chrome, tell
   the user: Penpot > *Your account* > *Integrations* > *Personal access
   tokens* > *Create new access token*.
3. Add it to Claude Code. The user runs this with their own token; you
   never type, echo or read it:

   ```sh
   claude mcp add penpot-api -s user \
     -e PENPOT_ACCESS_TOKEN=<token> -- /absolute/path/bin/penpot-mcp
   ```

   A self-hosted Penpot also needs `-e PENPOT_BASE_URL=https://...` and
   the `enable-access-tokens` flag on the instance.
4. Restart Claude Code, then `/mcp`: `penpot-api` lists its tools.

Never put the token in a file in a repository, a plan, a commit or a
chat message.

## 3. Find what to work on

| Need | Tool |
| --- | --- |
| who the token is | `whoami` |
| teams | `list_teams` |
| projects | `list_projects` (all teams) or with `team_id` |
| files in a project | `list_files` with `project_id` |
| pages of a file | `list_pages` with `file_id` |
| what is on a page | `get_page_tree` with `file_id`, optional `page_id`, `depth` |

`get_page_tree` is flat, top-most first, parents before their children:
each shape has `parent_id` and `depth`. `more_children` counts what the
depth cut off. Ask for a deeper `depth` rather than guessing.

Files are found by name through `list_projects` then `list_files`. When two
files share a name, ask which one.

## 4. Draw

| Tool | Does |
| --- | --- |
| `create_board` | a board at the page root or inside a board (`parent_id`) |
| `create_rect` | a rectangle at the root or inside a board |

Rules:

- **Coordinates are absolute page pixels**, also inside a board: a rect
  20 px into a board at (500, 0) sits at x = 520.
- `parent_id` must be a board (`type: frame` in the tree); a rect or group
  is refused.
- Fill is `#RRGGBB`; boards default to white, rectangles to grey.
- Every reply names the file and page it landed in. Check them before the
  next call.
- Writes to one account run one at a time (Penpot's limit). Do not call
  draw tools in parallel; send them one after another.
- Leave space: put new boards beside existing ones (read `get_page_tree`
  first, find the right-most `x + width`, add 100).

## 5. Check the result

Read it back: `get_page_tree` must show each new shape with the expected
`parent_id`. To see it, use Claude in Chrome:
[references/chrome.md](references/chrome.md) § Look at a drawing. Open
files show API edits live.

## 6. When it fails

| Error | Meaning | Do |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN was not accepted` | token wrong or expired | new token (section 2), user updates the MCP config |
| `this token cannot edit file` | view-only access | pick a file the account can edit |
| `HTTP 429 rate-limit` | Penpot's rate limit, already retried once | wait the seconds it says, then go on |
| `HTTP 409 validation/vern-conflict` | someone restored a version; already re-read once | call `get_page_tree` again, then redo |
| `parent ... is a rect` | parent is not a board | use the board's id |
| `HTTP 500 .../data-validation` | Penpot rejected a shape | report the `explain` text; do not retry blindly |

More traps: [references/traps.md](references/traps.md).
