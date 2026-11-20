# Routes

A tool reaches Penpot by one of four routes. The route decides what you
have to set up before the tool works, and each tool's page in the
reference says which one it takes.

| Route | Needs | Tools |
| --- | --- | --- |
| API | `PENPOT_ACCESS_TOKEN` | Most of them: files, projects, pages, drawing, the library, tokens, comments, versions, sharing, `.penpot` files. |
| login or tab | `PENPOT_EMAIL` and `PENPOT_PASSWORD`, or a connected tab | `export_shape`, `export_boards`. |
| hybrid | the token; a tab is optional | `draw`, `create_text`, `set_layout`, `instantiate_component`, `apply_token`. |
| tab | `PENPOT_MCP_URL` and a connected tab with the file open | `current_file`, `get_selection`, `active_users`, `viewport`, `reflow`, `import_svg`, `generate_markup`, `generate_style`, `generate_font_faces`, `execute_code`, `detach`, `reset_overrides`, `swap_component`, `switch_variant`, `combine_as_variants`. |

`api_info` is an API tool that answers more when `PENPOT_MCP_URL` is set:
it documents a backend method from Penpot's OpenAPI document, and a Plugin
API type or member from the official MCP server.

## API

The default. The server calls Penpot's backend with your access token. No
browser, no plugin, any file the token can reach. If a token cannot edit a
file, the call fails with `this token cannot edit file`.

Penpot runs one write per account at a time, so write tools are called one
after another, not in parallel.

## Login or tab

Penpot's exporter takes a login session, not an access token. So board
exports have two ways in:

- **A login.** Set `PENPOT_EMAIL` and `PENPOT_PASSWORD`. Exports then run
  with no browser, in every format.
- **A tab.** With no login, the connected tab renders the export. The
  shape's page must be open there, and there is no WEBP.

A Google or SSO account has no password, so the tab is its only way.

With neither, the tool answers `exports need PENPOT_EMAIL and
PENPOT_PASSWORD ... or PENPOT_MCP_URL`.

## Hybrid

These tools do their work on the API and add a step in the tab when one is
connected. Without a tab they still work. With one, `draw`, `create_text`
and `set_layout` also return a `tab` field with the real text sizes.

## Tab

The tab tools go through the official Penpot MCP, which talks to a plugin
running in a Penpot browser tab. They appear only with `PENPOT_MCP_URL` set.

To connect a tab:

1. In Penpot, open *your account > integrations > mcp server* and copy the
   MCP URL. The key is part of it and is shown once.
2. Add it to the server: `-e PENPOT_MCP_URL=<mcp url>` on the
   `claude mcp add` line, then restart the agent.
3. Open the file in Penpot, open the MCP plugin from the toolbar and click
   *Connect*. A grey dot means it is not connected.
4. Call `current_file`. It must name the file you mean. Every tab tool
   repeats that check and refuses when the tab has another file open.

Leave the tab open: closing it disconnects the plugin. A tab asleep in the
background answers `no Penpot tab is connected`; bring it to the front and
reconnect.

One tab serves one account and edits only the file open in it. With
[profiles](configuration.md#several-accounts), each profile can have its own
`mcp_url_env`.

`execute_code` runs Plugin API JavaScript in the tab. It is the last
resort, for what no other tool does. If it answers `the code may have run
anyway`, the call timed out after it was sent: read the file back before
trying again.

## Connecting the official MCP

penpot-mcp calls the official Penpot MCP for you through `PENPOT_MCP_URL`;
you do not need to add the official server to your agent as well.
