# penpot-mcp

An MCP server in Go that drives Penpot through its backend HTTP API, with
the official Penpot MCP as a fallback route for what the API cannot do.

Plan: `PLAN.md` (chain of `plans/NN-*.md`).
Skills: `skills/penpot-mcp*/`, embedded in the binary and installed with
`penpot-mcp skills install`. A tool added or renamed means regenerating
`skills/penpot-mcp/references/tools.md` (`go test ./internal/tools -run
TestSkillListsEveryTool -update`) and naming it in the job skill it
belongs to; `go test ./skills` fails until both are done. It also means
the book's pages: pick its group in `docGroups`
(`internal/tools/docs_test.go`) and run `go test ./internal/tools -run
TestDocsListEveryTool -update`.

Standards: docs/standards.md
References: `docs/penpot-api-reference.md`, `docs/existing-mcp-reference.md`,
`docs/bridge.md`, `docs/features.md`, `docs/external/`.

Never read `.env` or print `PENPOT_ACCESS_TOKEN`, `PENPOT_MCP_URL` or
`PENPOT_PASSWORD`.
