# Official Penpot MCP server (the MCP route)

Fetched: 19/10/2026 from the live server `https://design.penpot.app/mcp/stream`
(JSON-RPC `initialize`, `tools/list`, `high_level_overview`,
`penpot_api_info`) and the source in `penpot/penpot` `mcp/` on `develop`.
Version: server reports `{name: "penpot", version: "1.0.0"}`; Penpot 2.18.4.

The full reference is `../existing-mcp-reference.md`.

## Authentication

The per-user MCP key in the URL: `.../mcp/stream?userToken=<key>`, generated
in Penpot under *Your account > Integrations > MCP Server*. Env var:
`PENPOT_MCP_URL` (the whole URL, a secret). A `User-Agent` header is
required, or the server answers `403`.

## What this project calls

| Tool | Input | Used for |
|---|---|---|
| `execute_code` | `{code: string}` (function body, `await` allowed, `return` gives the result) | every MCP-route step |
| `penpot_api_info` | `{type: string, member?: string}` | `api_info` with `source: "plugin"` |
| `high_level_overview` | `{}` | appended to our `overview` |
| `export_shape` | `{shapeId, format: png\|svg, mode}` | export fallback |

Reply to `execute_code`: text holding `JSON.stringify({result, log})`.

## Errors

Failures come back as text results starting `Tool execution failed:`, not
JSON-RPC errors. "no heartbeat" and "no penpot instance connected" mean the
plugin tab is not connected.

## Limits

- 120 s per call (`PENPOT_MCP_TOOL_TIMEOUT_S`); a timeout does not undo
  applied work.
- Reply capped at about 15,000,000 characters on the hosted server.
- One connected tab per user; only the file and page open in it can be
  edited.

## Gotchas

- Check `penpot.currentFile.id` before every write: the connected file can
  change under the agent.
- No resources or prompts (`-32601`).
