<!-- Generated from the flag sets in cmd/penpot-mcp/main.go by `go test ./cmd/penpot-mcp -run TestCLIDoc -update`. Do not edit. -->

# Command line

`penpot-mcp` has two forms. Everything else is set with environment variables, see [configuration](../configuration.md).

## `penpot-mcp`

Starts the server. With no flag it speaks MCP over stdio, which is what `claude mcp add` runs.

```sh
penpot-mcp [--http HOST:PORT] [--version]
```

| Flag | Default | Does |
| --- | --- | --- |
| `--http` | none | serve MCP over streamable HTTP on host:port, such as 127.0.0.1:8787, instead of stdio |
| `--version` | none | print the version and exit |

## `penpot-mcp skills install`

Writes the agent skills embedded in the binary into the agent's skills directory. It needs no token. It refuses a skill directory it did not write unless `--force` is given.

```sh
penpot-mcp skills install [--agent claude|codex] [--dir PATH] [--force]
```

| Flag | Default | Does |
| --- | --- | --- |
| `--agent` | `claude` | where to install: claude (Claude Code) or codex |
| `--dir` | none | install into this skills directory instead |
| `--force` | none | replace skill directories penpot-mcp did not write, symlinks included |

With `--agent claude` the directory is `$CLAUDE_CONFIG_DIR/skills`, or `~/.claude/skills` when that is not set. With `--agent codex` it is `~/.agents/skills`.
