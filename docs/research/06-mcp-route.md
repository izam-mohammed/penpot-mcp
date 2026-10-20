# MCP-route tools against the threat table

Reviewed 2026-10-20, `plans/06-mcp-route.md` phase 4. Every tool that
`internal/tools/plugin.go` registers, read against the plan's threat table
and OWASP Top 10 for LLM Applications 2025 (LLM01 prompt injection, LLM05
improper output handling, LLM06 excessive agency).

| Tool | Writes | Hints | Guard | Finding | Fixed |
| --- | --- | --- | --- | --- | --- |
| `execute_code` | yes | destructive, closed world | file guard, writer lock, 100 kB cap | worded as last resort; any Plugin API call is possible by design | as built |
| `current_file` | no | read-only | none needed | none | - |
| `get_selection` | no | read-only | file guard | none | - |
| `reflow` | layout only | had none | file guard, lock | default hints read as destructive | set not destructive, idempotent |
| `generate_markup`, `generate_style`, `generate_font_faces` | no | read-only | file guard, UUID ids | none | - |
| `detach`, `reset_overrides` | yes, overrides lost | had none | file guard, lock, UUID ids | lose work without saying so | destructive hint, descriptions say what is lost |
| `swap_component`, `switch_variant`, `combine_as_variants` | yes | had none | file guard, lock, UUID ids | hints missing | set, not destructive |
| `active_users` | no | read-only | file guard | shows collaborators' names: needed for the tool's purpose | - |
| `viewport` | view only | none | file guard | view state only | - |
| `api_info` | no | read-only | none needed | none | - |
| `import_svg` | yes | had none | file guard, lock, file reads only under the working directory | hints missing | set, not destructive |

Across tools:

- **Ids and strings reach the tab as data.** Every id is checked as a UUID
  first, and every value is inserted with JSON encoding (`js()`), so a
  shape name or id cannot become code (`TestPluginIDsAreData`).
- **Text from files is data** (LLM01): names, text and console logs come
  back from the tab and could hold instructions. The server instructions
  now say so; nothing stronger is possible without hiding the content the
  tools exist to return.
- **No retry after a timeout** (`TestNoRetryAfterTimeout`), so a slow tab
  cannot double a write.
- **The MCP URL never leaves the route** (`TestMcpUrlNeverLogged`).

Live check, 2026-10-20: three `claude -p` sessions against the real
account, with every tab tool listed and `execute_code` blocked so an attempt
would show. Asked for a page's text, a file's library and a count of shapes
by type, none of them called `execute_code`. The text question exported the
whole file and unzipped it, because `get_page_tree` returned no text; it now
returns each text shape's text (capped at 500 characters), and the same
question is answered with one `get_page_tree` call.
