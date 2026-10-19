# penpot-mcp

An MCP server in Go that drives Penpot through its backend HTTP API, with
the official Penpot MCP as a fallback route for what the API cannot do.

Plan: `PLAN.md` (chain of `plans/NN-*.md`).
Standards: docs/standards.md
References: `docs/penpot-api-reference.md`, `docs/existing-mcp-reference.md`,
`docs/bridge.md`, `docs/features.md`, `docs/external/`.

Never read `.env` or print `PENPOT_ACCESS_TOKEN`, `PENPOT_MCP_URL` or
`PENPOT_PASSWORD`.
