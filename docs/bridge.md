# Bridge: API route plus MCP route

How the new Penpot MCP server offers every tool of the official Penpot MCP server, plus everything the backend API adds. Each tool runs on one of two routes:

- **API route:** direct calls to the Penpot backend (`POST {base}/api/main/methods/{command}`) with `Authorization: Token <access-token>`. No browser is needed. This is the default.
- **MCP route:** our server acts as an MCP client of the official Penpot MCP server (`https://design.penpot.app/mcp/stream?userToken=<mcp-key>`) and forwards the call. That server runs the code inside a Penpot browser tab through the plugin. Used only for what the API cannot do.

References: [penpot-api-reference.md](penpot-api-reference.md), [existing-mcp-reference.md](existing-mcp-reference.md), [features.md](features.md).

## Verdict legend

| Verdict | Meaning |
|---|---|
| **API** | Fully done through the backend API. No tab needed. |
| **API (own)** | No direct command; we build it from API data (client-side logic). No tab needed. |
| **HYBRID** | API does the work; the MCP route is used for one step (reflow, measuring) when a tab is connected, otherwise the API result stands with a documented gap. |
| **MCP** | Only possible through the official MCP route. Needs a connected Penpot tab. |
| **DROP** | Not needed without an editor. |

## 1. Architecture

```
MCP client (Claude Code, ...)
        |
        v
penpot-mcp (this server)
   |-- router: picks route per tool
   |
   |-- API route ---------> Penpot backend /api/main/methods/*   (access token)
   |                  \--> exporter /api/export                  (session cookie)
   |
   \-- MCP route ---------> official Penpot MCP /mcp/stream       (MCP key)
                                  |
                                  v
                           Penpot plugin in a browser tab
```

**Config:**

| Env var | Route | Required |
|---|---|---|
| `PENPOT_BASE_URL` | API | no, default `https://design.penpot.app` |
| `PENPOT_ACCESS_TOKEN` | API | yes |
| `PENPOT_EMAIL`, `PENPOT_PASSWORD` | API (exporter cookie) | only for exports |
| `PENPOT_MCP_URL` (with `userToken`) | MCP | only for MCP-route tools |

If `PENPOT_MCP_URL` is missing, MCP-route tools are hidden and HYBRID tools run API-only.

## 2. Routing rules

1. **API first.** Every tool that can run on the API runs there.
2. **MCP only when needed.** Use it for the steps the API cannot do: running Plugin API JavaScript, layout reflow, text measuring, markup and CSS generation, SVG import with the browser parser, component instance logic.
3. **File guard.** Before any MCP-route call on a file, run `return penpot.currentFile.id` through the MCP route. If it is not the target file id, fail with "open file <name> in Penpot and connect the MCP plugin" instead of editing the wrong file.
4. **Sync between routes.**
   - After API writes, the open tab updates live through the websocket. Wait for it before an MCP-route read: poll `penpot.currentFile` for the new shape ids, or call `penpot.waitForLayoutUpdate()`.
   - After MCP-route writes, re-fetch the file with `get-file` to refresh our `revn`/`vern` cache before the next API write.
5. **One writer at a time per file.** The API write queue and the MCP route share one lock per user, so the two routes never write at the same time.
6. **MCP errors.** The official server returns errors as text (`Tool execution failed: ...`), not JSON-RPC errors. Map "no heartbeat" and "no penpot instance connected" to a clear "tab not connected" error. Never retry blindly after its 120 s timeout: the work may already be applied, so re-read first.

## 3. Official MCP tools, carried over

All four tools of the official server exist in the new server.

| Official tool | New tool | Route | Verdict | How |
|---|---|---|---|---|
| `execute_code` | `execute_code` | MCP | **MCP** | Forward `{code}` unchanged. Add the file guard. Mark as the last resort in the tool description. |
| `high_level_overview` | `overview` | API (own) | **API (own)** | Our own text: ids, absolute page coordinates, root frame zero UUID, batching, limits, which tools need a tab. Append the official overview when the MCP route is on. |
| `penpot_api_info` | `api_info` | MCP + API | **HYBRID** | Plugin API types: forward to the official tool. Backend schemas (shape, change types, commands): read from `/api/main/doc/openapi.json`. Param `source: "plugin" \| "backend"`. |
| `export_shape` | `export_shape` | API | **API** | Exporter `POST /api/export` with the session cookie, transit body. PNG, JPEG, WEBP, SVG, PDF, scale. Writes to disk. Fallback to the MCP `export_shape` if the cookie login is not configured. |
| `import_image` (local mode only) | `upload_image` | API | **API** | `upload-file-media-object` (multipart) from a local path. Better than the official one: works against the hosted server too. |

## 4. Navigation and reading

| Tool | Route | Verdict | How |
|---|---|---|---|
| `whoami` | API | **API** | `get-profile`; zero UUID id means a bad token |
| `list_teams` | API | **API** | `get-teams` |
| `list_projects` | API | **API** | `get-projects`, `get-all-projects` |
| `list_files` | API | **API** | `get-project-files`, `get-team-recent-files`, `get-team-shared-files` |
| `search_files` | API | **API** | `search-files` |
| `get_file_summary` | API | **API** | `get-file` (cached), summarised |
| `list_pages` | API | **API** | `get-file` → `pages`, `pagesIndex` |
| `get_page_tree` | API | **API (own)** | `get-page`, compact tree built client-side |
| `get_shape` | API | **API** | `get-page` with `objectId` |
| `find_shapes` | API | **API (own)** | filter cached page objects by name, type, parent, bounds |
| `list_boards` (skill `boards`) | API | **API (own)** | top-level `frame` shapes whose parent is the root frame |
| `current_file` (skill `file`) | MCP | **MCP** | the file open in the tab: `penpot.currentFile`, `penpot.currentPage` |
| `get_selection` | MCP | **MCP** | `penpot.selection` |
| `audit` (skill `pd.audit`) | API | **API (own)** | colours, opacities, radii, font sizes, shapes outside boards, computed from page data |

## 5. Dashboard (API only, new)

The official MCP cannot do any of these; the skill used Chrome automation for them.

| Tool | Route | Verdict | How |
|---|---|---|---|
| `create_project`, `rename_project`, `delete_project`, `move_project`, `pin_project` | API | **API** | project commands |
| `create_file`, `rename_file`, `duplicate_file`, `move_files`, `delete_file` | API | **API** | file commands; server generates ids since 2.19 |
| `set_file_shared`, `link_library`, `unlink_library` | API | **API** | `set-file-shared`, `link-file-to-library`, `unlink-file-from-library` |
| `create_team`, `list_members`, `invite_members` | API | **API** | team commands |
| Chrome automation (create project/file, wake plugin) | n/a | **DROP** | replaced by the tools above |

## 6. Pages

| Tool | Route | Verdict | How |
|---|---|---|---|
| `add_page` | API | **API** | `add-page` with `{id, name}` |
| `rename_page` | API | **API** | `mod-page` |
| `delete_page` | API | **API** | `del-page` |
| `reorder_pages` | API | **API** | `mov-page` |
| `open_page` (Plugin `openPage`) | n/a | **DROP** | API writes target any `pageId`; no active page |

## 7. Drawing

| Tool | Route | Verdict | How |
|---|---|---|---|
| `create_board` | API | **API** | `add-obj` type `frame`, geometry built by us |
| `create_rect` | API | **API** | `add-obj` type `rect` |
| `create_ellipse` | API | **API** | `add-obj` type `circle` |
| `create_path` | API | **API** | `add-obj` type `path`, content from SVG `d` |
| `create_image` | API | **API** | upload media, then `rect` with a `fillImage` fill |
| `create_group`, `create_bool` | API | **API** | `add-obj` group/bool (JSON), then transit `mov-objects` of the shapes into it and `reg-objects`, which also computes a bool's content |
| `create_text` | API + MCP | **HYBRID** | API: `add-obj` text with estimated width and height. MCP (if tab connected): `waitForLayoutUpdate`, read the real size, write it back with `mod-obj`. |
| `import_svg` (Plugin `createShapeFromSvg`) | API + MCP | **HYBRID** | API: parse SVG ourselves into paths (or one `svg-raw` shape). MCP: forward to `createShapeFromSvg` when the tab is connected and the user wants the browser parser. Both drop gradients in some cases. |
| `update_shape` | API | **API** | `mod-obj` with `assign`; geometry recomputed by us on move or resize |
| `move_shape` | API | **API** | `mod-obj` `assign` of the shifted geometry for the shape and every descendant (path content shifted too), then `reg-objects` on groups around it |
| `reparent_shape` (also reorders, with `index`) | API | **API** | `mov-objects` sent as transit; JSON `mov-objects` answers 200 and moves nothing (checked 2026-10-19) |
| `delete_shapes` | API | **API** | `del-obj` |
| `draw` (declarative tree, replaces `builder.js` / `screens.js`) | API + MCP | **HYBRID** | API: whole tree in one `update-file` batch. MCP: optional reflow and text measuring pass at the end. |
| `apply_changes` (raw `changes[]`) | API | **API** | `update-file` as-is, the API-side escape hatch |
| `reflow` (skill `pd.reflow`, `waitForLayoutUpdate`) | MCP | **MCP** | forward `await penpot.waitForLayoutUpdate()` on the open file; without a tab, layout reflows when the file is next opened |
| `set_layout` (flex / grid) | API | **API** | API stores the layout attributes and we place the children ourselves (fixed sizes, top layer first, as Penpot does); opening the file does not reflow. MCP reflow only for fill or auto sizing. |

## 8. Library and tokens

| Tool | Route | Verdict | How |
|---|---|---|---|
| `add_color`, `update_color`, `delete_color` | API | **API** | `add-color`, `mod-color`, `del-color` |
| `add_typography`, `update_typography`, `delete_typography` | API | **API** | typography changes |
| `list_library` | API | **API** | `get-file` assets plus `get-file-libraries` |
| `import_palette` (skill `make_bootstrap.py`) | API | **API (own)** | parse the repo, then idempotent `add-color` by name |
| `create_component` | API | **API** | `add-component` from existing shapes |
| `instantiate_component` | API + MCP | **HYBRID** | API: copy the main shapes with new ids and component refs. MCP: `library.component.instance()` when the tab is connected, which handles nested copies correctly. |
| `detach`, `swap_component`, `reset_overrides`, `switch_variant`, `combine_as_variants` | MCP | **MCP** | Plugin API methods; the API has no high-level command and the sync rules are complex |
| `sync_tokens` (sets, themes, tokens) | API | **API** | `set-token`, `set-token-set`, `set-token-theme` |
| `apply_token` (bind to shape) | API + MCP | **HYBRID** | API: write the applied-tokens attribute and the resolved value. MCP: `shape.applyToken` when the tab is connected. |

## 9. Media and fonts

| Tool | Route | Verdict | How |
|---|---|---|---|
| `upload_image` | API | **API** | `upload-file-media-object` |
| `upload_image_from_url` | API | **API** | `create-file-media-object-from-url` (100, refill 50/5 min) |
| `list_fonts` | API | **API** | `get-font-variants` plus built-in fonts |
| `upload_font` | API | **API** | `create-font-variant` |

## 10. Export and code generation

| Tool | Route | Verdict | How |
|---|---|---|---|
| `export_shape` | API | **API** | exporter, cookie login, writes to disk, retries on `ResourceRequest timed out` and `429 queue-full` |
| `export_boards` (skill `export-all`) | API | **API (own)** | loop the exporter over top-level boards |
| `export_file`, `import_file` (`.penpot`) | API | **API** | `export-binfile`, `import-binfile` |
| `generate_markup` (HTML/SVG) | MCP | **MCP** | Plugin `penpot.generateMarkup`. Also the fast SVG fallback when the exporter times out. |
| `generate_style` (CSS) | MCP | **MCP** | Plugin `penpot.generateStyle` |
| `generate_font_faces` | MCP | **MCP** | Plugin `penpot.generateFontFaces` |

## 11. Collaboration and history

| Tool | Route | Verdict | How |
|---|---|---|---|
| `list_comments`, `add_comment`, `reply_comment`, `resolve_thread` | API | **API** | comment commands, any file |
| `create_share_link`, `delete_share_link` | API | **API** | share link commands |
| `list_webhooks`, `create_webhook`, `delete_webhook` | API | **API** | webhook commands |
| `list_snapshots`, `create_snapshot`, `restore_snapshot` | API | **API** | file snapshot commands |
| `plugin_data` get/set | API | **API** | `set-plugin-data` change, `get-file-plugin-data` |
| `active_users` | MCP | **MCP** | `penpot.currentFile` active users; presence is editor-only |
| Undo blocks (`undoBlockBegin/Finish`) | n/a | **DROP** | one `update-file` batch is atomic; use `create_snapshot` for rollback |
| Viewport, zoom, selection changes | MCP | **MCP** | Plugin `viewport` and `selection`, on request only |

## 12. Summary

| Verdict | Count | Tools |
|---|---|---|
| **API** | the rest | navigation, dashboard, pages, shapes, library, tokens, media, fonts, export, comments, share links, webhooks, snapshots |
| **API (own)** | 7 | `overview`, `get_page_tree`, `find_shapes`, `list_boards`, `audit`, `import_palette`, `export_boards` |
| **HYBRID** | 7 | `api_info`, `create_text`, `import_svg`, `draw`, `set_layout`, `instantiate_component`, `apply_token` |
| **MCP** | 14 | `execute_code`, `current_file`, `get_selection`, `reflow`, component ops (`detach`, `swap_component`, `reset_overrides`, variants), `generate_markup`, `generate_style`, `generate_font_faces`, `active_users`, viewport |
| **DROP** | 3 | Chrome automation, `open_page`, undo blocks |

**Gaps that stay without a tab:** real text sizes, flex and grid reflow, HTML and CSS generation, and complex component overrides. Everything else works with only an access token.
