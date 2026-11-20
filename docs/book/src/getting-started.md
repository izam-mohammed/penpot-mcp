# getting started

from nothing to a drawing in Penpot, in six steps. you need Penpot
(the hosted one or your own), an agent that speaks MCP, and a terminal.

## 1. install it

with Homebrew:

```sh
brew install --cask izam-mohammed/tap/penpot-mcp
```

no brew? grab a binary from
[releases](https://github.com/izam-mohammed/penpot-mcp/releases) (macos,
linux, windows; amd64 and arm64) and put it on your path.

check it landed:

```sh
penpot-mcp --version
```

## 2. make a token

in Penpot: *your account > integrations > personal access tokens*. it is
shown once, so copy it.

on a self-hosted Penpot the instance needs the `enable-access-tokens`
flag, or the integrations page has no tokens in it. see
[configuration](configuration.md#self-hosted-penpot).

treat the token like a password, because it is one. never put it in a
file in a repository, a commit or a chat message.

## 3. add it to your agent

for Claude Code:

```sh
claude mcp add penpot -s user \
  -e PENPOT_ACCESS_TOKEN=<your token> \
  -- "$(command -v penpot-mcp)"
```

`<your token>` is yours to paste; the server reads it from the
environment and never logs it. other settings go on the same line as more
`-e NAME=value`, see [configuration](configuration.md).

another client? the server is the `penpot-mcp` binary on stdio, with
`PENPOT_ACCESS_TOKEN` in its environment. a client that only speaks HTTP
can use [HTTP mode](configuration.md#http-mode).

## 4. give your agent the manual

```sh
penpot-mcp skills install                # claude code
penpot-mcp skills install --agent codex  # codex
```

this writes five skills that say which of the tools to call and what the
errors mean. they come from the same release as the tools. see the
[command line](reference/cli.md) for `--dir` and `--force`.

## 5. restart and check

restart the agent. `/mcp` should list `penpot`, and `/skills` the
`penpot-mcp` ones.

if `penpot` is listed but has no tools, the server failed to start.
run `penpot-mcp </dev/null` in your own shell with the token set to see
why. the usual one is `PENPOT_ACCESS_TOKEN was not accepted`: make a new
token. more in [troubleshooting](troubleshooting.md).

## 6. draw something

ask your agent:

> in my "scratch" penpot file, draw a 390 by 844 login screen with a
> title, an email field and a blue sign in button.

it will find the file with `list_projects` and `list_files`, read the page
with `get_page_tree`, then send the whole screen as one `draw` call. a
call looks like this:

```json
{"file_id": "<file id>", "shapes": [
  {"ref": "screen", "type": "board", "name": "Login", "x": 0, "y": 0, "width": 390, "height": 844,
   "layout": "flex", "direction": "column", "gap": 16, "padding": 24},
  {"parent": "screen", "type": "text", "text": "Welcome back", "font_family": "Inter", "font_size": 28, "font_weight": 700},
  {"parent": "screen", "type": "rect", "name": "Email", "width": 342, "height": 48, "radius": 8, "fill": "#FFFFFF", "stroke": "#D0D5DD"},
  {"parent": "screen", "type": "rect", "name": "Sign in", "width": 342, "height": 48, "radius": 8, "fill": "#2563EB"}
]}
```

open the file in Penpot and the board is there. an open file shows API
edits live.

## where next

- [Configuration](configuration.md): the environment variables, several
  accounts, a self-hosted instance.
- [Routes](routes.md): which tools need only the token and which need a
  tab or a login.
- the guides, one per job: [files and projects](guides/files.md),
  [drawing](guides/drawing.md), [library and tokens](guides/library.md),
  [export](guides/export.md),
  [collaboration](guides/collaboration.md).
