# penpot-mcp

penpot-mcp is an MCP server that drives [Penpot](https://penpot.app)
through its backend API. No browser tab, no plugin that has to stay awake,
any file your access token can reach.

the [official penpot mcp](https://github.com/penpot/penpot/tree/develop/mcp)
wants a tab open, a plugin awake and the right file in front of it. this one
talks to the api directly ~ and still calls the official one for the few
things only a tab can do.

## what is in here

- [Getting started](getting-started.md): install, make a token, add it to
  your agent, draw something.
- Configuration, routes and guides: how each setting changes what works,
  and one guide per job.
- The reference: every tool with its parameters, and the command line.
  These pages are written from the server itself, so they match the
  version you run.

## who it is for

anyone who wants an agent (Claude Code, Codex, or any client that speaks
MCP) to create files, draw screens, fill a library, export boards and read
comments in Penpot, without babysitting a browser.

The source and releases are on
[GitHub](https://github.com/izam-mohammed/penpot-mcp).
