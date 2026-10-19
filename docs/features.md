# Penpot MCP: features

An MCP server that drives Penpot through its backend HTTP (RPC) API with a personal access token, instead of the plugin bridge used by the official server.

References:

- [penpot-api-reference.md](penpot-api-reference.md): the backend API (transport, auth, limits, `update-file`, change types, shape schema, command catalog).
- [existing-mcp-reference.md](existing-mcp-reference.md): the official Penpot MCP server and the local `penpot` skill, including their traps.

## Goals

- **No browser tab.** No plugin, heartbeat, tab sleep, `storage` loss or one-tab rule.
- **Any file, any page.** Every call names its target by id. There is no "active" file or page.
- **Full dashboard coverage.** Teams, projects and files can be created, renamed, moved and deleted, which the plugin API cannot do.
- **Small tool surface.** A few high-level tools plus a raw escape hatch, which is the shape that worked for the official server.
- **Safe writes.** Respect `revn`/`vern`, serialize writes per user, and never leave half-applied batches unreported.

## Non-goals (for v1)

- Running arbitrary JavaScript against the Plugin API (`execute_code`). There is no browser to run it in.
- Pixel-perfect text layout. The backend does not measure text, so sizes are estimates until the file is opened in Penpot.
- Live presence or cursors over the websocket.

## 1. Core: connection and config

| Feature | Notes |
|---|---|
| Config from env | `PENPOT_BASE_URL` (default `https://design.penpot.app`), `PENPOT_ACCESS_TOKEN`. Optional `PENPOT_EMAIL` / `PENPOT_PASSWORD` for the exporter. |
| HTTP client | `POST {base}/api/main/methods/{command}`. Always send `Authorization: Token <t>`, `Accept: application/json`, `Content-Type: application/json`, `X-Client: penpot-mcp/<version>`, `User-Agent`. |
| Token check on start | Call `get-profile`. A zero UUID id means the token was rejected; fail with a clear message. |
| Version detection | Read `info.version` from `/api/main/doc/openapi.json`. Gate newer commands (export jobs, `since 2.19` commands) on it. |
| Error mapping | Turn the Penpot error body `{type, code, hint, explain}` into MCP tool errors with the code and a fix hint (for example `revn-conflict`: re-fetch). |
| Rate-limit handling | On `429`, honour `Retry-After` and retry with backoff. Read `x-rate-limit-remaining`. |
| Transports | stdio first. Streamable HTTP later for remote use. |
| Multi-account | Optional named profiles (several tokens). Each token gets its own write slot. |

## 2. Navigation (read)

| Tool | Backed by | Returns |
|---|---|---|
| `whoami` | `get-profile` | id, name, email, default team and project |
| `list_teams` | `get-teams` | id, name, role, features |
| `list_projects` | `get-projects` / `get-all-projects` | id, name, team, pinned |
| `list_files` | `get-project-files`, `get-team-recent-files`, `get-team-shared-files` | id, name, project, modified, shared |
| `search_files` | `search-files` | matches by name |
| `get_file_summary` | `get-file` (cached) | pages, revn, vern, permissions, counts of colors, typographies, components |
| `list_pages` | `get-file` | page id, name, order |
| `get_page_tree` | `get-page` | compact shape tree: id, name, type, x, y, w, h, children. Depth and type filters. Never the raw multi-MB file. |
| `get_shape` | `get-page` with `objectId` | full attributes of one shape |
| `find_shapes` | cached page | by name (exact or substring), type, parent, bounds |

## 3. Dashboard management (write)

| Tool | Backed by |
|---|---|
| `create_project`, `rename_project`, `delete_project`, `move_project`, `pin_project` | `create-project`, `rename-project`, `delete-project`, `move-project`, `update-project-pin` |
| `create_file`, `rename_file`, `duplicate_file`, `move_files`, `delete_file` | `create-file`, `rename-file`, `duplicate-file`, `move-files`, `delete-file` |
| `set_file_shared` | `set-file-shared` (makes a file a shared library) |
| `link_library`, `unlink_library` | `link-file-to-library`, `unlink-file-from-library` |
| `create_team`, `invite_members` (later) | team commands |

Since 2.19 the server generates ids for projects and files; read them from the response.

## 4. Pages

| Tool | Change types |
|---|---|
| `add_page` | `add-page` with `{id, name}` |
| `rename_page` | `mod-page` |
| `delete_page` | `del-page` |
| `reorder_pages` | `mov-page` |

## 5. Drawing (the write engine)

All drawing goes through `update-file`. This is the core of the project.

**Engine:**

- **Per-file state cache:** `{revn, vern, features, pages}`. After a 200, set `revn = response.revn + 1`. On `409 vern-conflict` or `400 revn-conflict`, re-fetch and rebuild.
- **Per-user write queue:** Penpot allows one `update-file` in flight per profile (queue of 5, 60 s timeout). Serialize all writes, even across files.
- **Batching:** one tool call becomes one `update-file` with many changes, not many calls. Limit is 1000 burst, 500/min refill.
- **Geometry builder:** from `{x, y, width, height, rotation}` compute `selrect`, `points`, `transform`, `transformInverse`. The backend stores exactly what it receives and recomputes nothing.
- **Defaults per type:** merge sane defaults so the model only supplies meaningful attributes.
- **Client-side validation:** validate shapes against the OpenAPI shape schema before sending, so errors come back as readable messages instead of `500 data-validation`.
- **JSON pitfalls:** use `assign` (not `set`) in `mod-obj`. Do moves as `del-obj` + `add-obj` with the same id, or send `mov-objects` / `reorder-children` as transit.
- **Parent bookkeeping:** keep `parentId`, `frameId` and the parent's `shapes` list consistent. Run `reg-objects` for groups and bools.

**Tools:**

| Tool | Does |
|---|---|
| `create_board` | board (`frame`) with fill, radius, clip, optional flex or grid layout |
| `create_rect`, `create_ellipse` | rect, `circle` |
| `create_text` | text with content tree, font, size, weight, line height, color, alignment, grow type. Estimated width and height. |
| `create_path` | path from SVG `d` or commands |
| `create_image` | upload media, then a rect with a `fillImage` fill |
| `create_group`, `create_bool` | group or boolean op of existing shapes |
| `update_shape` | patch attributes (`assign`) with geometry kept consistent on move or resize |
| `move_shape`, `reparent_shape` | via delete and re-add |
| `delete_shapes` | `del-obj` |
| `draw` | one declarative tree (board with nested children) in a single batch. The main tool for screens. |
| `apply_changes` | raw escape hatch: send a `changes[]` array as-is |

**Styling covered:** fills (solid, gradient, image), strokes, radius, opacity, blend mode, shadows, blur, constraints, flex and grid layout props, layout child props.

## 6. Library assets

| Tool | Change types |
|---|---|
| `add_color`, `update_color`, `delete_color` | `add-color`, `mod-color`, `del-color` |
| `add_typography`, `update_typography`, `delete_typography` | `add-typography`, `mod-typography`, `del-typography` |
| `create_component` (from shapes), `instantiate_component`, `delete_component` | `add-component`, `mod-component`, `del-component` |
| `list_library` | colors, typographies, components of a file and its linked libraries |
| `sync_tokens` | design tokens: sets, themes, tokens (`set-token`, `set-token-set`, `set-token-theme`) |
| `import_palette` | idempotent bulk colors from a repo (hex list, CSS vars, JSON) |

## 7. Media and fonts

| Tool | Backed by |
|---|---|
| `upload_image` (local path) | `upload-file-media-object` (multipart). Returns media id, width, height. |
| `upload_image_from_url` | `create-file-media-object-from-url` (strict rate limit: 100, refill 50/5 min) |
| `list_fonts` | `get-font-variants` plus built-in Google fonts |
| `upload_font` (later) | `create-font-variant` |

## 8. Export

| Tool | Notes |
|---|---|
| `export_shape` | PNG, JPEG, WEBP, SVG or PDF of one or more shapes or boards, written straight to disk (never base64 into the context). Scale option. |
| `export_boards` | every top-level board of a page into a folder, with a name prefix filter |
| `export_file` | whole `.penpot` file via `export-binfile` |
| `import_file` | `.penpot` file via `import-binfile` |

Constraint: the exporter (`/api/export`) accepts only a session cookie and a transit body, not an access token. Options: log in with `PENPOT_EMAIL` / `PENPOT_PASSWORD` to get the cookie, or try the share-link route (untested). Retry on `ResourceRequest timed out` and `429 queue-full`.

## 9. Collaboration

| Tool | Backed by |
|---|---|
| `list_comments`, `add_comment`, `reply_comment`, `resolve_thread` | comment commands (30, refill 15/min) |
| `create_share_link`, `delete_share_link` | share links (10, refill 5/min) |
| `list_webhooks`, `create_webhook`, `delete_webhook` | webhooks |
| `list_snapshots`, `create_snapshot`, `restore_snapshot` | file snapshots (version history). Take one before large edits. |

## 10. Docs and agent helpers

| Tool / resource | Does |
|---|---|
| `overview` tool or prompt | how to use this server: ids, coordinates (absolute page px), the root frame zero UUID, batching, limits |
| `schema` tool | shape and change-type schema for a given type, like `penpot_api_info` |
| MCP resources | `penpot://file/{id}/summary`, `penpot://page/{id}/tree` for read-only context |

## 11. Reliability and safety

- Write queue per user, retries with backoff on `429`, re-fetch on conflicts.
- Never report success for a batch the server did not confirm.
- Optional auto snapshot before destructive tools (`delete_*`, `restore_snapshot`).
- Dry-run flag on write tools: return the `changes[]` without sending.
- Redact the token in every log and error.
- Idempotency: tools that create named things (`add_color`, `create_board` with `replace: true`) can replace an existing one by name.

## Known limits to design around

- **Writes per user:** one `update-file` at a time. Batch instead of parallelizing.
- **No server-side geometry or text measuring.** Flex and grid reflow happens only when the file opens in the editor [uncertain].
- **Exporter needs a session cookie,** not the access token.
- **Rate limits apply only when the instance enables `rpc-rlimit`.** Treat the numbers in the API reference as the worst case.
- **Self-hosted servers** need `enable-access-tokens`, and may require the `X-Client` header.

## Open questions to test first

1. Does `mov-objects` silently do nothing over JSON?
2. Does API-created text (no `positionData`) render in exports before the file is opened?
3. Does a share-link export work without a session cookie?
4. Rotation and `transform` convention.
5. Google font `fontId` format.
6. Do flex and grid layouts reflow without opening the editor?

## Suggested build order

1. Client, auth check, error mapping, rate-limit retry.
2. Navigation tools and dashboard CRUD.
3. Write engine (state cache, queue, geometry builder) with `create_rect` and `create_board`.
4. Text, images, `draw` tree tool.
5. Library assets and tokens.
6. Export (cookie login) and import.
7. Comments, share links, snapshots, webhooks.
