# Penpot Backend HTTP (RPC) API Reference

Reference for building a Penpot MCP server that talks to the Penpot backend directly with a personal access token.

**Sources and their versions (checked 2026-10-19):**

- The live OpenAPI document at `https://design.penpot.app/api/main/doc/openapi.json`. It reports `"version": "2.18"` and lists 149 commands. All parameter tables below come from it.
- The Penpot source at `github.com/penpot/penpot` (`main`, shallow clone made on 2026-10-19). This code is newer than the live server: it has 163 commands, and some of them are marked `since 2.19` or `since 2.20`. These newer commands are flagged in this document.
- The official integration guide at `help.penpot.app/technical-guide/integration/` (it is also `docs/technical-guide/integration.md` in the repo).

**Legend:**

- **[VERIFIED]**: tested against `design.penpot.app` with curl.
- **[SOURCE]**: read from the backend or common source code.
- **[UNCERTAIN]**: inferred, not tested. Check it against your target instance before you depend on it.

---

## 1. Transport

### 1.1 Base URL and endpoints

| Item | Value |
|---|---|
| SaaS base | `https://design.penpot.app` |
| Self-hosted base | `PENPOT_PUBLIC_URI`, for example `https://penpot.example.com` |
| **Current RPC endpoint** | `POST {base}/api/main/methods/{command}` [SOURCE] [VERIFIED] |
| Legacy RPC endpoint (still routed, same handler) | `POST {base}/api/rpc/command/{command}` [SOURCE] [VERIFIED] |
| Interactive docs (HTML) | `GET {base}/api/main/doc` (`/api/_doc` and `/api/doc` redirect here with a 308) |
| OpenAPI JSON | `GET {base}/api/main/doc/openapi.json` (about 1.1 MB; it holds the full schemas, including every change type and the shape schema) |
| OpenAPI (UI) | `GET {base}/api/main/doc/openapi` |
| Management API (internal, shared key) | `POST {base}/api/management/methods/{command}`. It needs `X-Shared-Key`, so it is not for clients. |
| Exporter (render to PNG/JPEG/WEBP/SVG/PDF) | `POST {base}/api/export`, plus `POST /api/export/jobs`, `GET/DELETE /api/export/jobs/{id}` (the jobs API is newer and may not be on the live server). See section 12. |
| Assets download | `GET {base}/assets/by-id/{storage-object-id}`, `GET {base}/assets/by-file-media-id/{id}`, `GET {base}/assets/by-file-media-id/{id}/thumbnail`. These accept an access token. |
| WebSocket notifications | `{base}/ws/notifications` (used by the frontend for real-time presence and changes; not needed for an MCP server) |

**Methods:**

- Any command can be called with `POST`.
- Only commands whose names start with `get-` can also be called with `GET` or `HEAD`, with the params in the query string.
- Any other command called with GET returns `405`, with the body `{"type":"restriction","code":"method-not-allowed"}` [SOURCE].
- Use `POST` with a JSON body for everything.

### 1.2 Content types

**Request body** (the `content-type` header):

- `application/json`: object keys may be camelCase (`teamId`) or kebab-case (`team-id`). The server converts all keys to kebab-case keywords (`json/read-kebab-key`) [SOURCE].
- `application/transit+json`: this is what the Penpot frontend uses. See section 1.4.
- `multipart/form-data`: needed for file uploads (`upload-file-media-object`, `upload-chunk`, `update-profile-photo`, `update-team-photo`, `import-binfile`, `create-font-variant`). For `upload-file-media-object`, `upload-chunk` and both photo commands, upload maps are only accepted when they arrive as multipart. Otherwise the call fails with `params-validation`, hint "upload requires multipart" [SOURCE].

**Response body** (chosen by the `Accept` header, or by the `?_fmt=json` query param):

- `Accept: application/json` gives a JSON response with **camelCase** keys [VERIFIED].
- `application/transit+json` gives transit.
- If `Accept` is missing or `*/*`, **the response is transit** [VERIFIED]. Always send `Accept: application/json`.
- `?transit_verbose` gives verbose transit.
- A `nil` result gives `204 No Content`.

**Value decoding:** after parsing, request params are validated and decoded against each command's malli schema, using a JSON transformer:

- UUID strings become UUIDs.
- Enum strings become keywords.
- Points (`{x,y}` or `"x,y"`), matrices (`{a,b,c,d,e,f}` or `"a,b,c,d,e,f"`) and rects (`{x,y,width,height,x1,y1,x2,y2}`) become records.

This works because each schema declares how its values decode. **Fields typed `any` are NOT decoded** (see the JSON pitfalls in section 5.5).

### 1.3 Headers

| Header | Notes |
|---|---|
| `Authorization: Token <access-token>` | Access token authentication [SOURCE]. Use the word `Token`, not `Bearer`: `Bearer` is treated as a *session* token and an access token sent that way fails. |
| `Content-Type: application/json` | |
| `Accept: application/json` | Without it you get transit. |
| `X-Client: <name>` | Required only when the instance enables the flag `client-header-check-middleware` (CSRF hardening). Without it the server returns `403` with an empty body. design.penpot.app does not require it [VERIFIED]. Send it anyway, for example `X-Client: penpot-mcp/0.1`. |
| `X-Session-Id: <uuid>` | Optional. Ties a request to a client session (used for websocket echo suppression). |
| `If-None-Match: <etag>` | `get-file` supports ETags (conditional middleware). |

### 1.4 Transit quick notes (only needed for the exporter and for some change payloads)

Transit-JSON is a JSON array or map with tagged strings:

- Map: `["^ ", k1, v1, k2, v2, ...]`
- Keyword: `"~:name"`
- UUID: `"~u<uuid>"`
- Set: `{"~#set":[...]}`
- Instant: `"~m<epoch-ms>"`
- Penpot custom tags: `"~#shape"`, `"~#point"`, `"~#matrix"`, `"~#rect"`, `"~#penpot/path-data"`

In Node, use the `transit-js` package. The Penpot exporter accepts **only** transit bodies (section 12).

---

## 2. Authentication

### 2.1 Personal access tokens

- **Where to create one:** in the web UI, open *Your account > Access tokens > Generate new token*. You can also call `create-access-token`, but that needs an existing session.
  - Expiration options: never, 30, 60, 90 or 180 days.
  - The token string is shown once.
- **How to send it:** `Authorization: Token <token>` [SOURCE: `app.http.middleware/wrap-auth`, regex `(?i)^(Token|Bearer)\s+(.*)`].
- **Self-hosted servers:** the flag `access-tokens` must be enabled, as in `PENPOT_FLAGS="... enable-access-tokens"`. Without it the token middleware is not installed and every call is anonymous.
- **What the server checks:** the token is a signed JWE whose `iss` is `access-token` (MCP-type tokens use `urn:penpot:mcp-token`). The token row must exist and must not be expired.
- **What the token can do:** it acts as its profile, with full permissions on everything that profile can reach. A few commands also need *token perms*. The only example is `get-error-reports`, which needs `error-reports:read`; that perm can only be granted with SQL.
- **Bad, unknown or missing token:** commands that need auth return `401 {"type":"authentication","code":"authentication-required","hint":"authentication required for this endpoint"}` [VERIFIED].
- **Anonymous `get-profile`:** returns `200` with `{"id":"00000000-0000-0000-0000-000000000000","fullname":"Anonymous User"}` [VERIFIED]. To check that a token is valid, call `get-profile` and make sure `id` is not the zero UUID.

### 2.2 Session cookie (only needed for the exporter)

- `login-with-password` with `{email, password}` sets the cookie `auth-token=<session-jwt>`. The cookie name is configurable and defaults to `auth-token`.
- The session token is also accepted as `Authorization: Bearer <session-token>`.
- The exporter (`/api/export`) authenticates **only through this cookie**. Access tokens do not work there (section 12).

### 2.3 Commands that need no auth

`login-with-password`, `login-with-ldap`, `logout`, `register-profile`, `prepare-register-profile`, `recover-profile`, `request-profile-recovery`, `verify-token`, `get-sso-provider`, `create-demo-profile`, `get-profile`, `get-team-info`, `get-file-info`, `get-file-fragment`, `get-view-only-bundle`, `get-environment-data`, `get-enabled-flags`.

---

## 3. Errors, status codes, limits

### 3.1 Error body

```json
{"type":"validation","code":"params-validation","hint":"...","explain":"<human readable malli explain>"}
```

Errors are JSON when `Accept: application/json` is sent, and transit otherwise. Internal fields (`state`, `path`, `context`) are removed.

| HTTP | `type` | Typical `code`s |
|---|---|---|
| 400 | `validation` | `params-validation`, `spec-validation`, `schema-validation`, `data-validation`, `malformed-json`, `revn-conflict`, `invalid-image` |
| 400 | `restriction` | `feature-not-supported`, `file-feature-mismatch`, `file-in-components-v1`, quota errors (`max-quote-reached`) |
| 401 | `authentication` | `authentication-required`, `nitrate-sso-required` |
| 403 | `authorization` | `token-auth-required`, `missing-perms`. Plain permission denials also appear as `not-found` with `object-not-found`. |
| 403 | (empty body) | Missing `X-Client` header, when the instance enforces it |
| 404 | `not-found` | `object-not-found`, or an unknown command name |
| 405 | `restriction` | `method-not-allowed` (GET on a command that is not a `get-` command) |
| 409 | `validation` | `vern-conflict` (`update-file` against a file whose version was restored) |
| 413 | `validation` | `request-body-too-large` |
| 429 | `rate-limit` | Body `{type, code, hint, ttl}` plus a `Retry-After` header (seconds). Also returned for concurrency-limit overflow, with no body. |
| 500 | `server-error` | `assertion` (the `data-validation` variant includes `explain`, and is the usual result of an invalid shape in `update-file`), `unexpected`, `unhandled`, `database-error`, `io-exception` |
| 503 | `nitrate-unavailable` / `nitrate-not-configured` | |
| 504 | `server-error` | `statement-timeout`, `idle-in-transaction-timeout` |

### 3.2 Rate limits (`backend/resources/rlimit.edn`) [SOURCE]

- Rate limits are active only when the instance has the flag `rpc-rlimit`. With `soft-rpc-rlimit` they are logged but not enforced.
- Limits are keyed per profile, falling back to IP.
- **Bucket format:** `capacity/refill/interval`, so `"1000/500/1m"` means a burst of 1000 that refills 500 per minute.
- Headers: `x-rate-limit-remaining` and `x-rate-limit-reset`.

| Commands | Limit |
|---|---|
| default (anything unlisted) | window 200000/h |
| `update-file` | bucket 1000/500/1m |
| `get-file` | 180/90/1m |
| `get-page` | 150/75/30s |
| `get-projects` | 120/60/30s |
| `get-project`, `get-project-files` | 80/40/30s |
| `get-teams` | 5000/2500/30s |
| `create-file`, `rename-file`, `duplicate-file`, `move-files` | 60/30/1m |
| `delete-file` | 80/40/1m |
| `create-project` | 100/50/1m |
| other project mutations | 40/20/1m |
| `upload-file-media-object` | 200/100/1m |
| `create-file-media-object-from-url` | 100/50/5m |
| `search-files` | 60/30/1m |
| comments (create/update/delete) | 30/15/1m |
| comment thread mutations | 80/40/1m |
| `create-webhook`, `update-webhook` | 20/10/5m |
| `get-webhooks`, `delete-webhook` | 20/10/1m |
| `create-share-link`, `delete-share-link` | 10/5/1m |
| `export-binfile`, `import-binfile`, `clone-template` | 80/40/1m |
| `create-access-token`, `delete-access-token` | 60/30/1m |
| `login-with-password` | 100/50/1m |
| `get-team-members` | 4000/2000/30s |
| `get-profile` | 500/250/30s |
| `get-comment-threads` | 500/250/30s |
| `get-font-variants` | 250/125/30s |
| `get-file-libraries` | 200/100/30s |
| `get-team-recent-files`, `get-unread-comment-threads` | 120/60/30s |
| `get-team-shared-files`, `get-team-info`, `get-team-users`, `get-team-invitations`, `get-team-deleted-files` | 60/30/30s |
| `get-comments`, `get-file-snapshots`, `get-library-usage`, `has-file-libraries` | 300/150/30s |
| `get-comment-thread`, `get-library-file-references` | 60/30/30s |
| `get-file-info`, `get-view-only-bundle`, `get-all-projects`, `get-file-summary`, `get-file-stats`, `get-file-fragment` | 60/30/30s |
| `get-access-tokens` | 150/75/30s |
| `set-file-shared`, library link/unlink, file snapshot create/restore/update/delete/lock | 80/40/1m |
| team mutations (create, update, delete, member role, leave, ...) | 60/30/1m |
| profile mutations | 30/15/1m |
| font ops (`create-font-variant`, `delete-font`, ...) | 100/50/1m |
| upload sessions (`create-upload-session`, `upload-chunk`, `assemble-file-media-object`) | 100/50/1m |
| `create-file-object-thumbnail` and other thumbnail ops | 5000/3000/1m |
| `create-team-invitations`, `send-user-feedback` | 30/15/5m |

Source: `backend/resources/rlimit.edn` on `develop`, checked 2026-10-19. The file reloads every 30 s (`:refresh "30s"`). The hosted instance can run different numbers. [UNCERTAIN]

### 3.3 Concurrency limits (`climit.edn`) [SOURCE]

These matter more than the rate limits for an MCP server.

| Limiter | Per profile | Global | Timeout |
|---|---|---|---|
| `update-file` | 1 permit, queue 5 | 20 | 60 s |
| root (every RPC call) | 10 permits, queue 30 | 40 | 30 s |
| process-image (media upload) | 1 | 8 | 60 s |
| process-font | 1 | 4 | 60 s |
| file-thumbnail-ops | 2 | 20 | 60 s |
| create-file-snapshot / restore-file-snapshot | 1, queue 2 | 3 | 60 s |
| import-binfile / create-import-binfile-job | 1, queue 2 | 4 | 60 s |
| auth | n/a | 8 | 30 s |

- When a queue is full or the timeout passes, the server answers `429` with no body.

**What this means for an MCP server:**

- **Reads can run in parallel** across files, pages and projects (up to 10 in flight per profile).
- **Writes cannot.** One profile can run only **one `update-file` at a time, even across different files**. Serialize writes per user in a single queue.
- To write many shapes, send them as many changes in **one** `update-file` call (bucket 1000, refill 500/min), not as many parallel calls.
- Several accounts or tokens each get their own write slot.

### 3.4 Quotas

Instances can cap the number of projects per team, files per project, access tokens per profile, and so on. When a cap is hit, the server returns `restriction` with `max-quote-reached`.

---

## 4. Core concepts and data model

```
Profile ──< TeamProfileRel >── Team ──< Project ──< File ──data──> FileData
                                  └── Webhooks, Invitations, Fonts (team fonts)
FileData = { pages: [pageId...], pagesIndex: {pageId: Page}, colors: {id: Color},
             typographies: {id: Typography}, components: {id: Component},
             media: {id: FileMedia}, tokensLib, options: {componentsV2: true, baseFontSize} }
Page     = { id, name, objects: {shapeId: Shape}, background?, guides?, flows?, defaultGrids? }
```

**IDs and the shape tree:**

- Every page has a **root frame** whose id is `00000000-0000-0000-0000-000000000000` (the "zero" UUID). Its `parentId` and `frameId` are both the zero UUID.
- Top-level shapes have `parentId = frameId = 00000000-...` and are listed in the root frame's `shapes` array.
- `frameId` is the nearest containing **board** (frame), or the root frame.
- `parentId` is the direct container, which can be a group, a bool or a board.
- Clients generate every id themselves (UUID v4 is fine): pages, shapes, colors, typographies and components.
- Since 2.19, the backend *rejects* client ids on `create-project`, `create-file` and the media-creation calls. For these, the server generates the id; read it from the response.

**`revn` and `vern`:**

- `revn` is an integer revision counter. Every `update-file` raises it by 1.
- `vern` is the version number. It changes only when a snapshot is restored.
- Both appear in the `get-file` response.

**Features:**

- Each file and team has a set of feature strings. The defaults are `components/v2`, `styles/v2`, `layout/grid`, `fdata/shape-data-type`, `fdata/path-data`, `plugins/runtime`, `design-tokens/v1`, `tokens/numeric-input` and `variants/v1`.
- The backend may also add `fdata/pointer-map` and `fdata/objects-map`.
- When `features` is omitted from a call, the client feature check is skipped. **Do not send `features` unless you know what you are doing.**
- If you do send it, send exactly the file's `features` from `get-file`.
- If you send `fdata/pointer-map` in `get-file`, the server returns *unresolved* pointers. If you omit `features`, it resolves the pointers and returns the full data.

**Permissions:** `get-file` returns `permissions: {type, isOwner, isAdmin, canEdit, canRead, isLogged}`. A user can call `update-file` only when `canEdit` is true.

**Team roles:** `owner`, `admin`, `editor`, `viewer`.

**Units:** coordinates are absolute page coordinates in px, with no nesting offsets: a shape inside a board at (100,100) still uses page coordinates.

---
## 5. `update-file`: the only write path for file content

Pages, shapes, colors, typographies, components, guides, flows and tokens are all edited by sending a list of **changes** to `update-file`.

**What happens on the server** [SOURCE `backend/src/app/rpc/commands/files_update.clj`]:

1. The file is locked (`xact-lock`).
2. The file is migrated if it needs it.
3. The changes are applied in order with `app.common.files.changes/process-changes`.
4. **Every added or modified shape is hard-validated against the shape schema**, whatever flags are set.
5. Optional whole-file validation runs if the `file-validation` / `file-schema-validation` flags are set.
6. The file is persisted, and `revn` goes up by 1.
7. A `file_change` row is written. It is kept for about 1 h and is used for "lagged" changes.
8. Notifications go out to websocket subscribers (open editors refresh live) and to webhooks. Webhooks are batched per profile and file over 2 minutes.

### 5.1 Params

| param | type | req | notes |
|---|---|---|---|
| `id` | uuid | yes | The file id. |
| `sessionId` | uuid | yes | Any UUID. Generate one per MCP server session and reuse it. It ties changes to a client session, so that editor tabs can tell their own echoes apart. |
| `revn` | int ≥ 0 | yes | The revision your changes are based on. **It must be ≤ the stored revn.** If it is greater, the server returns `400 revn-conflict`. A lower (stale) revn is accepted. |
| `vern` | int ≥ 0 | yes | **It must equal the stored `vern`.** Otherwise the server returns `409 vern-conflict` (someone restored a version). Re-fetch the file. |
| `features` | string[] | no | If you send it, it must cover the team's enabled features, or the server returns `feature-not-supported`. **Omit it.** |
| `changes` | Change[] | no* | The change list (section 6). |
| `changesWithMetadata` | `[{changes: Change[], hintOrigin?: string, hintEvents?: string[]}]` | no* | An alternative to `changes`. If it is present, `changes` is ignored. |
| `skipValidate` | bool | no | Skips only the optional whole-file validation. The per-shape validation still runs. |

### 5.2 Response (JSON)

```json
{ "revn": 12, "lagged": [ {"id": "...", "revn": 13, "fileId": "...", "sessionId": "...", "changes": [ ... ]} ] }
```

- `revn` is the file revn **before** this update was applied. The value now stored is `revn + 1`.
- `lagged` lists every stored change with a revn greater than the `revn` you sent. It includes the change you just made, so after a stale revn this list can be long.

[SOURCE] The OpenAPI document describes the result as a vector, but the code returns this map. Trust the map, and parse the response defensively.

### 5.3 Recommended revn/vern discipline for an MCP server

1. Call `get-file {id}` and keep `revn` and `vern`.
2. Send `update-file` with that `revn`. After a 200 response, set your local `revn` to `response.revn + 1`. It is always safe to send a revn that is lower than the stored one.
3. On `409 vern-conflict`, re-fetch the file and rebuild your changes. On `400 revn-conflict`, which should not happen if you only ever increment by 1, also re-fetch.
4. **Serialize your calls.** The server allows 1 concurrent `update-file` per profile, with a queue of 5. Batch many changes into one call where you can.
5. The backend does **not** recompute geometry, layout or text layout:
   - It stores exactly what you send.
   - When you move or resize a shape, send consistent `x`, `y`, `width`, `height`, `selrect` and `points` together.
   - Flex and grid layouts are **not** reflowed when the file is opened [VERIFIED 2026-10-19]: the children stay where you put them until someone edits the layout in the editor. Place them yourself. Penpot lays out the **top** layer (last in `shapes`) first.
   - For shapes created by the API, run `reg-objects` on the parent group or bool ids so that their selrect is recalculated. Boards do not need this.

### 5.4 Operations (inside `mod-obj.operations`)

| `type` | fields | effect |
|---|---|---|
| `assign` | `value: {attr: val, ...}`, `ignoreTouched?`, `ignoreGeometry?` | Sets several attributes at once. **`value` is decoded with the full shape-attribute schema (JSON-safe), so this is the operation to use from JSON.** |
| `set` | `attr: string` (kebab or camel), `val: any`, `ignoreTouched?`, `ignoreGeometry?` | Sets or removes one attribute (`val: null` removes it). **`val` is NOT type-decoded** (section 5.5). |
| `set-touched` | `touched: string[]` or null | Component-copy override flags. |
| `set-remote-synced` | `remoteSynced?: bool` | Component sync flag. |

### 5.5 JSON pitfalls [SOURCE, partly UNCERTAIN]

The JSON decoder only converts values whose schema type is known. These fields are typed `any`, so values in them stay as raw JSON (strings, numbers and maps):

- **`mod-obj` → `{type:"set", val}`:** UUIDs stay strings and enum values stay strings, so they are not keywords. Shape validation then fails, for example on `"constraintsH":"scale"` or a gradient `type`, with **500 `assertion` / `data-validation`**.
  - **Fix: always use `assign`** from JSON. Plain numbers, booleans and hex color strings are fine in `set`.
- **`mov-objects.shapes`, `mov-objects.afterShape` and `reorder-children.shapes`:** these are `any`. UUID strings are not converted, so they match no objects, and the move does nothing: JSON `mov-objects` answers 200 and leaves the shape where it was [VERIFIED 2026-10-19, `scripts/prove/penpot-move.sh`]. The transit form below works.
  - **Workaround:** send these change types with `Content-Type: application/transit+json`, encoding the ids as `"~u<uuid>"`.
  - **Alternatively,** avoid moves: `del-obj` the shape and `add-obj` it again under the new parent at the index you want. This changes nothing about the shape, since it keeps the same id.
- **`add-page.page`:** typed `any`. Use the `{id, name}` form instead, which builds an empty page on the server.
- **`add-component.variantProperties`:** these are decoded properly.

Everything else is decoded correctly from JSON, including `add-obj.obj`, which is decoded with the full shape schema: points, matrices, rect records, keyword enums and path content.

---

## 6. Change types (`changes[]` items) [SOURCE `common/src/app/common/files/changes.cljc`]

**Field conventions:**

- Every change has a `type` field (kebab-case string).
- Field names are given in kebab-case. In JSON you can send them as camelCase (`page-id` → `pageId`).
- "opt" means optional.
- For shape-level changes, `page-id` targets a page. Use `component-id` instead to edit a deleted component's stored objects. Normally you send `page-id`.

### 6.1 Shapes

| type | fields | semantics |
|---|---|---|
| `add-obj` | `id` uuid, `obj` Shape, `frame-id` uuid, `page-id` opt, `component-id` opt, `parent-id` opt/nullable, `index` opt int/null, `ignore-touched` opt bool | Adds `obj` under `parent-id` (default: the root `00000000-…`), at `index` (default: appended, which puts it on top). **The change-level `id`, `frame-id` and `parent-id` override the values inside `obj`.** A parent or frame that does not exist falls back to the root. |
| `mod-obj` | `id`, `page-id` opt, `component-id` opt, `operations` Operation[] | Applies operations (section 5.4). The shape is re-validated afterwards. If the id is not found, the change does nothing. |
| `del-obj` | `id`, `page-id` opt, `component-id` opt, `ignore-touched` opt | Deletes the shape and its children, and removes it from the parent's `shapes`. |
| `mov-objects` | `parent-id` uuid, `shapes` uuid[], `index` opt int/null, `after-shape` opt uuid, `page-id` opt, `component-id` opt, `ignore-touched` opt, `allow-altering-copies` opt | Re-parents or reorders shapes (see the JSON pitfall in 5.5). |
| `reorder-children` | `parent-id`, `shapes` uuid[] (the new order), `page-id` opt, `component-id` opt, `ignore-touched` opt, `allow-altering-copies` opt | Sorts the parent's `shapes` to follow the given order. |
| `reg-objects` | `shapes` uuid[], `page-id` opt, `component-id` opt | Recalculates the geometry (selrect/points) of the listed groups and bools from their children. Send it after changing a group's children. |
| `fix-obj` | `id`, `fix` opt (`broken-children`), `page-id` opt, `component-id` opt | Repairs broken child references. |

### 6.2 Pages

| type | fields | semantics |
|---|---|---|
| `add-page` | `id` opt uuid, `name` opt string, `page` opt Page | Send **either** `id`+`name` (which creates an empty page with a root frame) **or** a full `page`. Sending both raises `conflict`. The page is appended to `pages`. |
| `mod-page` | `id`, `name` opt, `background` opt hex/null, `background-token` opt, `pixel-grid-color` opt, `pixel-grid-opacity` opt | Renames the page or sets its background (`null` removes it). |
| `del-page` | `id` | |
| `mov-page` | `id`, `index` int | Reorders pages. |
| `set-guide` | `page-id`, `id`, `params` Guide or null | Guide: `{id, axis: "x"\|"y", position: number, frame-id?: uuid, color?: hex}`. `null` deletes the guide. |
| `set-flow` | `page-id`, `id`, `params` Flow or null | Flow: `{id, name, starting-frame: uuid}`. `null` deletes the flow. |
| `set-default-grid` | `page-id`, `grid-type` `square`\|`column`\|`row`, `params` or null | Sets the default grid params for the page. |
| `set-comment-thread-position` | `comment-thread-id`, `page-id`, `frame-id` nullable, `position` point nullable | Used by comments. |

### 6.3 Library assets (local library of the file)

| type | fields |
|---|---|
| `add-color` / `mod-color` | `color`: `{id, name, path?, opacity?, color?: "#rrggbb", gradient?: Gradient, image?: ImageColor, modified-at?, plugin-data?}`. Exactly one of `color`, `gradient` or `image` must be set. |
| `del-color` | `id` |
| `add-typography` / `mod-typography` | `typography`: `{id, name, font-id, font-family, font-variant-id, font-size, font-weight, font-style, line-height, letter-spacing, text-transform, path?, modified-at?}`. **All values are strings**, for example `"font-size":"16"`, `"font-weight":"400"`, `"line-height":"1.2"`. |
| `del-typography` | `id` |
| `add-media` / `mod-media` | `object`: `{id, name, width, height, mtype, media-id, file-id?, thumbnail-id?, created-at?}` (a file-media-object row) |
| `del-media` | `id` |
| `add-component` | `id`, `name`, `path` (group path, `""` for none), `main-instance-id` (the shape that is the main instance, usually a board), `main-instance-page`, `variant-id` opt, `variant-properties` opt `[{name, value}]` |
| `mod-component` | `id`, `name` opt, `path` opt, `variant-id` opt, `variant-properties` opt |
| `del-component` | `id`, `delta` opt point, `skip-undelete?` opt (soft delete) |
| `restore-component` | `id`, `page-id` |
| `purge-component` | `id` (hard delete) |

**To make a component:**

1. `add-obj` a board (the main instance) with `componentId: <cid>`, `componentFile: <fileId>`, `componentRoot: true` and `mainInstance: true`.
2. `add-component` with `{id: cid, name, path: "", mainInstanceId: <boardId>, mainInstancePage: <pageId>}`.

**To instantiate a copy:** add the copy shapes with `componentId`, `componentFile` and `componentRoot: true` on the root. Each copy child needs a `shapeRef` that points to the corresponding main child. [UNCERTAIN: complex; the frontend logic is in `common/src/app/common/logic/libraries.cljc`]

### 6.4 Design tokens

| type | fields |
|---|---|
| `set-tokens-lib` | `tokens-lib` (the full lib) or null; over JSON, a DTCG JSON *string*: an object has its keys keywordized and fails "check error" (seen 2026-10-20) |
| `set-token` | `set-id`, `token-id`, `attrs`: `{id?, name, type, value, description?, modified-at?}` or null (null deletes the token) |
| `set-token-set` | `id`, `attrs`: `{id?, name, description?, modified-at?, tokens?}` or null |
| `set-token-theme` | `id`, `attrs`: `{id?, name, group?, description?, is-source?, external-id?, modified-at?, sets?: string[]}` or null |
| `set-tokens-status` | `theme-ids` uuid set, `set-ids` uuid set (newer than 2.18) |
| `rename-token-set-group` | `set-group-path` string[], `set-group-fname` string |
| `move-token-set` / `move-token-set-group` | `from-path` string[], `to-path` string[], `before-path` string[] or null, `before-group` bool or null |
| `set-base-font-size` | `base-font-size` string |
| `set-tokens-source` | `file-id`, `library-id` or null (newer) |
| `set-active-token-themes` | Appears in the 2.18 OpenAPI but not in current source [UNCERTAIN] |

### 6.5 Plugin data

`set-plugin-data`:

- `object-type`: one of `file`, `page`, `shape`, `color`, `typography`, `component`
- `object-id` opt
- `page-id`: required when `object-type` is `shape`
- `namespace`: string, which is decoded to a keyword
- `key`: string
- `value`: string or null

---

## 7. Shape object schema [SOURCE `common/src/app/common/types/shape.cljc` and friends]

### 7.1 Shape types

| `type` | UI name | specific attrs |
|---|---|---|
| `frame` | **Board** | `shapes` uuid[] (children), `hide-fill-on-export?`, `show-content?` (`true` = no clipping), `use-for-thumbnail?`, layout attrs (7.6), variant attrs |
| `group` | Group | `shapes` uuid[], `masked-group?` (when true, the first child is the mask) |
| `bool` | Boolean | `shapes` uuid[], `bool-type` `union`\|`difference`\|`exclude`\|`intersection`, `content` path content; `x`/`y`/`width`/`height` may be null |
| `rect` | Rectangle (images too: a rect whose fill is a `fill-image`) | `r1`..`r4` radii, legacy `rx`, `ry` |
| `circle` | **Ellipse** (there is no `ellipse` type) | `rx?`, `ry?` (legacy) |
| `path` | Path / curve | `content` (7.5); `x`/`y`/`width`/`height` may be null |
| `text` | Text | `content` (7.7), `position-data?`, `grow-type` |
| `image` | Legacy image shape | `metadata: {id, width, height, mtype?}`. The current frontend creates images as a `rect` with a `fill-image` fill instead. |
| `svg-raw` | Raw SVG node (from an SVG import) | `content` (`{tag, attrs, content}` map or string), `shapes?` |

### 7.2 Required base attributes (every type)

| attr | type | notes |
|---|---|---|
| `id` | uuid | |
| `name` | string | |
| `type` | enum above | |
| `x`, `y`, `width`, `height` | number | Required. Use null only for `path` and `bool`. Values are absolute page coordinates of the *unrotated* box. |
| `selrect` | Rect `{x, y, width, height, x1, y1, x2, y2}` | The unrotated bounding box: `x1=x`, `y1=y`, `x2=x+width`, `y2=y+height`. |
| `points` | Point[4] `[{x,y}…]` | The 4 corners after `transform`, in this order: top-left, top-right, bottom-right, bottom-left. With no rotation they are the selrect corners. |
| `transform` | Matrix `{a,b,c,d,e,f}` | Identity is `{a:1,b:0,c:0,d:1,e:0,f:0}`. Rotation is stored here, around the shape centre. |
| `transform-inverse` | Matrix | The inverse of `transform` (also identity when there is no rotation). |
| `parent-id` | uuid | Overridden by `add-obj`. |
| `frame-id` | uuid | Overridden by `add-obj`. |

**Rotation:**

- To rotate a shape by θ around its centre (cx, cy):
  - `transform` = `[cos θ, sin θ, -sin θ, cos θ, 0, 0]`
  - `transform-inverse` = `[cos θ, -sin θ, sin θ, cos θ, 0, 0]`
  - `points` = the selrect corners, each rotated around (cx, cy)
  - `rotation` = θ in degrees
- Note that `e` and `f` are 0. Penpot applies the matrix relative to the centre. [UNCERTAIN; check against a shape rotated in the UI.]

### 7.3 Common optional attributes

| attr | type / values |
|---|---|
| `rotation` | number (degrees), nullable |
| `flip-x`, `flip-y` | bool, nullable |
| `opacity` | number 0..1 |
| `blend-mode` | `normal darken multiply color-burn lighten screen color-dodge overlay soft-light hard-light difference exclusion hue saturation color luminosity` |
| `hidden`, `blocked`, `locked`, `collapsed` | bool |
| `proportion` (w/h), `proportion-lock` | number, bool |
| `constraints-h` | `left right leftright center scale` |
| `constraints-v` | `top bottom topbottom center scale` |
| `fixed-scroll` | bool |
| `r1 r2 r3 r4` | ≥0 numbers (corner radii: top-left, top-right, bottom-right, bottom-left). Valid on rect, frame and image. |
| `fills` | Fill[] (7.4) |
| `strokes` | Stroke[] (7.4) |
| `shadow` | `[{id: uuid\|null, style: "drop-shadow"\|"inner-shadow", offset-x, offset-y, blur ≥0, spread, hidden: bool, color: Color}]`. Here `Color` = `{color?: "#hex", opacity?: 0..1, gradient?, image?, ref-id?, ref-file?}`, with exactly one of `color`, `gradient` or `image`. |
| `blur` | `{id: uuid, type: "layer-blur", value ≥0, hidden: bool}` |
| `background-blur` | Similar to `blur` (newer; see `shape/background_blur.cljc`) |
| `exports` | `[{type: png\|jpeg\|webp\|svg\|pdf, scale >0, suffix: string}]`. Export presets only; nothing is rendered. |
| `grids` | Board layout grids: `[{type: column\|row\|square, display: bool, params: {...}}]` |
| `interactions` | Prototype interactions (`shape/interactions.cljc`) |
| `grow-type` | `auto-width auto-height fixed` (used mainly on text) |
| `hide-in-viewer` | bool |
| `component-id`, `component-file`, `component-root`, `main-instance`, `shape-ref`, `touched`, `remote-synced` | Component linkage |
| `applied-tokens` | `{attr-name: token-name}` |
| `plugin-data` | `{namespace: {key: string}}` |
| `page-id` | uuid (optional, informative) |

### 7.4 Fills and strokes

**Fill:** a closed map, which must contain **exactly one** of `fill-color`, `fill-color-gradient` or `fill-image`:

```json
{"fillColor": "#FF5733", "fillOpacity": 1}
{"fillColorGradient": {"type": "linear", "startX": 0.5, "startY": 0, "endX": 0.5, "endY": 1, "width": 1,
                        "stops": [{"color": "#000000", "opacity": 1, "offset": 0}, {"color": "#FFFFFF", "opacity": 1, "offset": 1}]},
 "fillOpacity": 1}
{"fillImage": {"id": "<file-media-object-id>", "width": 800, "height": 600, "mtype": "image/png", "name": "photo", "keepAspectRatio": true}, "fillOpacity": 1}
```

**Fill fields:**

- `fill-opacity`: 0..1
- `fill-color-ref-id` and `fill-color-ref-file`: link the fill to a library color. Use the color id and the file id that holds the color.
- Gradient coordinates are relative to the shape, from 0 to 1. `type` is `linear` or `radial`.

**Stroke:** a closed map with **exactly one** of `stroke-color`, `stroke-color-gradient` or `stroke-image`:

```json
{"strokeColor": "#000000", "strokeOpacity": 1, "strokeWidth": 2, "strokeStyle": "solid", "strokeAlignment": "inner"}
```

**Stroke fields:**

- `stroke-style`: `solid`, `dotted`, `dashed` or `mixed`
- `stroke-alignment`: `center`, `inner` or `outer`
- `stroke-width` ≥ 0. Per-side widths (`stroke-width-top`, `-right`, `-bottom`, `-left`) are also accepted.
- `stroke-dash` and `stroke-gap`
- `stroke-cap-start` and `stroke-cap-end`:
  - line caps: `round`, `square`
  - marker caps: `line-arrow`, `triangle-arrow`, `square-marker`, `circle-marker`, `diamond-marker`
  - `null` for none
  - Caps apply only to open paths.
- `stroke-color-ref-id` and `stroke-color-ref-file`
- `hidden`

**Defaults the UI uses:**

- rect and ellipse: fill `#B1B2B5`
- board: fill `#FFFFFF`
- path: stroke `#000000`, width 1, alignment `center`
- text: fill `#000000`

### 7.5 Path content (`content` on `path` and `bool`)

Send either an array of segments or an SVG path string. The backend converts both to its binary `PathData` [SOURCE `types/path/impl.cljc`]:

```json
"content": [
  {"command": "move-to",  "params": {"x": 0,   "y": 0}},
  {"command": "line-to",  "params": {"x": 100, "y": 0}},
  {"command": "curve-to", "params": {"x": 100, "y": 100, "c1x": 150, "c1y": 25, "c2x": 150, "c2y": 75}},
  {"command": "close-path"}
]
```

- The first segment must be `move-to`.
- You can also send an SVG path string such as `"M0,0 L100,0 Z"`. [UNCERTAIN] The string form is parsed by `path/from-string`; check which SVG commands it accepts.
- The segment array is accepted from JSON in `add-obj` [VERIFIED 2026-10-19]. `get-page` returns `content` as an absolute SVG string such as `"M1620.0,120.0L1720.0,20.0C...Z"`.
- `reg-objects` on a `bool` computes its `content` from the children, so a new bool can be sent with `content: []` [VERIFIED 2026-10-19].
- For paths, set `selrect` and `points` to the bounding box of the content. `x`, `y`, `width` and `height` can be null or match the box.
- `get-file` may return `content` in an encoded form (binary/base64 or string) when the file has `fdata/path-data`. [UNCERTAIN]

### 7.6 Layout (flex and grid) [SOURCE `types/shape/layout.cljc`]

**Container attributes** (on a `frame`):

| attr | values |
|---|---|
| `layout` | `flex` \| `grid` (absent = no layout) |
| `layout-flex-dir` | `row row-reverse column column-reverse` |
| `layout-grid-dir` | `row column` |
| `layout-gap` | `{row-gap ≥0, column-gap ≥0}` |
| `layout-gap-type` | `simple multiple` |
| `layout-wrap-type` | `wrap nowrap` |
| `layout-padding-type` | `simple multiple` |
| `layout-padding` | `{p1, p2, p3, p4}` (top, right, bottom, left), all required, ≥0 |
| `layout-justify-content` | `start center end space-between space-around space-evenly stretch` |
| `layout-align-content` | `start end center space-between space-around space-evenly stretch` |
| `layout-align-items` | `start end center stretch` |
| `layout-justify-items` | `start end center stretch` |
| `layout-grid-rows`, `layout-grid-columns` | `[{type: percent\|flex\|auto\|fixed, value?: ≥0}]` |
| `layout-grid-cells` | `{cellId: {id, row, row-span, column, column-span, position?: auto\|manual\|area, area-name?, align-self?, justify-self?, shapes: uuid[]}}` |

**Child attributes** (on any shape inside a layout board):

| attr | values |
|---|---|
| `layout-item-h-sizing`, `layout-item-v-sizing` | `fill fix auto` |
| `layout-item-margin-type` | `simple multiple` |
| `layout-item-margin` | `{m1, m2, m3, m4}` |
| `layout-item-min-w`, `layout-item-max-w`, `layout-item-min-h`, `layout-item-max-h` | ≥0 |
| `layout-item-align-self` | `start end center stretch` |
| `layout-item-absolute` | bool (when true, the child ignores the flow) |
| `layout-item-z-index` | number |

In a flex row, children are laid out in the **reverse** order of the parent's `shapes` array. The last item in the array is drawn on top and comes first in the flow. [UNCERTAIN; this matches the UI behaviour.]

### 7.7 Text content [SOURCE `types/shape/text.cljc`, `types/text.cljc`]

```json
"content": {
  "type": "root", "verticalAlign": "top",
  "children": [{
    "type": "paragraph-set",
    "children": [{
      "type": "paragraph", "textAlign": "left", "textDirection": "ltr",
      "fontId": "sourcesanspro", "fontFamily": "sourcesanspro", "fontVariantId": "regular",
      "fontSize": "24", "fontWeight": "400", "fontStyle": "normal",
      "lineHeight": "1.2", "letterSpacing": "0", "textTransform": "none", "textDecoration": "none",
      "fills": [{"fillColor": "#000000", "fillOpacity": 1}],
      "children": [{
        "text": "Hello Penpot",
        "fontId": "sourcesanspro", "fontFamily": "sourcesanspro", "fontVariantId": "regular",
        "fontSize": "24", "fontWeight": "400", "fontStyle": "normal",
        "lineHeight": "1.2", "letterSpacing": "0", "textTransform": "none", "textDecoration": "none",
        "fills": [{"fillColor": "#000000", "fillOpacity": 1}]
      }]
    }]
  }]
}
```

**Levels:**

- **root:** `vertical-align` (`top`, `center` or `bottom`)
- **paragraph:** `text-align` (`left`, `center`, `right` or `justify`), `text-direction`
- **leaf (span):** `text` plus the style attributes

**Style rules:**

- Paragraph and leaf nodes both carry the style attributes.
- `paragraph-set`, `paragraph` and leaf `children` arrays need **at least 1** item.
- **All typography values are strings:** `font-size`, `font-weight`, `line-height` and `letter-spacing`.
- Allowed ranges: `font-size` from 3 to 1000; `line-height` and `letter-spacing` from -200 to 200.
- `text-transform`: `uppercase`, `capitalize`, `lowercase`, `none` or `unset`.
- To link a typography, use `typography-ref-id` and `typography-ref-file`.

**Fonts:**

- The default font is `sourcesanspro` (Source Sans Pro), with `font-variant-id` `"regular"`.
- **Google fonts:** `font-id` is `"gfont-<slug>"`, for example `"gfont-inter"` with `font-family` `"Inter"`. [UNCERTAIN; check against a text styled in the UI.]
- **Custom team fonts:** `font-id` is `"custom-<font-id-uuid>"`.

**`position-data`** is optional. It holds the computed glyph runs (`[{x, y, width, height, fills, text, font-*}]`) that the viewer, thumbnails and the exporter use to render text without a layout engine.

- If you omit it, the workspace recomputes it when a user opens the file.
- [UNCERTAIN] Text from the API may render empty or wrong in the exporter, thumbnails or view mode until a user opens the file in the workspace.
- Set `grow-type` to `auto-width` so that the editor resizes the box to fit the text.

### 7.8 Minimal valid shapes (JSON, for `add-obj.obj`)

**Rectangle** (x=100, y=100, size 200×120):

```json
{"id":"<ID>","type":"rect","name":"Rectangle","x":100,"y":100,"width":200,"height":120,"rotation":0,
 "selrect":{"x":100,"y":100,"width":200,"height":120,"x1":100,"y1":100,"x2":300,"y2":220},
 "points":[{"x":100,"y":100},{"x":300,"y":100},{"x":300,"y":220},{"x":100,"y":220}],
 "transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},
 "transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},
 "parentId":"00000000-0000-0000-0000-000000000000","frameId":"00000000-0000-0000-0000-000000000000",
 "fills":[{"fillColor":"#FF5733","fillOpacity":1}],"strokes":[],"r1":8,"r2":8,"r3":8,"r4":8}
```

**Ellipse:** the same as the rectangle but with `"type":"circle"` and no radii.

**Board:** the same geometry with these changes:

```json
{"type":"frame","name":"Board","shapes":[],"fills":[{"fillColor":"#FFFFFF","fillOpacity":1}],"strokes":[],
 "hideFillOnExport":false,"r1":0,"r2":0,"r3":0,"r4":0, ...geometry...}
```

**A child of a board:**

- In the `add-obj` change, set `frameId` and `parentId` to the board id.
- The child keeps absolute page coordinates.
- The server appends the child id to the board's `shapes` array.

**Group:**

1. Add the children first.
2. Add the group with `shapes: [childIds]`. Give it geometry that covers its children.
3. Send `mov-objects` to put the children inside the group. Section 5.5 explains why JSON is a problem here, so simply add the children *after* the group, with `parentId` set to the group id.
4. Send `reg-objects` with `shapes:[groupId]` so that the group's selrect is recalculated.

---

## 8. Media and images

1. **Upload:** call `upload-file-media-object` as **multipart/form-data** with these fields:
   - `file-id`
   - `is-local` (`true` for an image used only in this file)
   - `name`
   - `content` (the file part)
   - Accepted types: png, jpeg, webp, gif and svg.
   - **Or use** `create-file-media-object-from-url` with JSON `{fileId, isLocal: true, url, name?}` (rate-limited, 100 per 5 min).
2. **Response:** a file-media-object `{id, fileId, name, width, height, mtype, mediaId, thumbnailId, isLocal, createdAt}`.
3. **Draw it:** `add-obj` a `rect` with these fills:

   ```json
   [{"fillOpacity":1,"fillImage":{"id":"<media.id>","width":W,"height":H,"mtype":"image/png","keepAspectRatio":true}}]
   ```

   The server checks and fixes media references on `add-obj`.
4. **Download:** `GET /assets/by-file-media-id/{id}` (the `/thumbnail` suffix gives the thumbnail). `GET /assets/by-id/{mediaId}` gives the raw storage object.
5. **Large files:** use chunked upload: `create-upload-session` → `upload-chunk` (multipart; repeat it) → `assemble-file-media-object`.

---

## 9. Comments, share links, webhooks (overview)

**Comments:**

- `create-comment-thread {fileId, pageId, frameId, position:{x,y}, content, shareId?, mentions?}` creates a thread and its first comment.
- `create-comment {threadId, content}` adds a reply.
- `get-comment-threads {fileId}` (or `{teamId}`) and `get-comments {threadId}` read them.
- `update-comment-thread {id, isResolved}` resolves or reopens a thread.
- `update-comment-thread-status {id}` only marks the thread as read for the caller.
- `frameId` can be the root frame, the zero UUID.

**Share links:**

- Create one with `create-share-link {fileId, whoComment: "team"|"all", whoInspect: "team"|"all", pages:[pageId...]}`. The response holds the link `id`.
- The view URL is `{base}/#/view?file-id=…&page-id=…&share-id=<id>` [UNCERTAIN: route format].
- Anonymous viewers read the file with `get-view-only-bundle {fileId, shareId}`.

**Webhooks:**

- Create one with `create-webhook {teamId, uri, mtype: "application/json"|"application/transit+json"}`. Each call is validated against the target URI.
- Penpot then sends a `POST` to `uri` with `content-type: <mtype>` and `user-agent: penpot/<version>`, with a 4 s timeout and up to 3 retries.
- The body is the audit event for the RPC call, roughly `{name: "<command>", props: {...params / result props...}, profileId, type: "action"|"trigger", timestamp, ...}`. [UNCERTAIN: exact keys. Inspect a real delivery with webhook.site.]
- Commands marked **WEBHOOK event** in section 13 trigger deliveries. These include `create-file`, `update-file` (batched for 2 min per profile and file), `rename-file`, `delete-file`, `create-project`, `create-comment`, `create-comment-thread`, the font commands and the binfile commands.
- After repeated failures the webhook is deactivated (`isActive: false`, `errorCode`, `errorCount`).

---
## 10. Working examples (curl)

### Setup

```bash
export PENPOT=https://design.penpot.app
export TOKEN=...your-access-token...
H=(-H "Authorization: Token $TOKEN" -H "Content-Type: application/json" -H "Accept: application/json" -H "X-Client: penpot-mcp/0.1")
rpc() { curl -sS -X POST "$PENPOT/api/main/methods/$1" "${H[@]}" -d "${2:-{\}}"; }
ROOT=00000000-0000-0000-0000-000000000000
uuid() { uuidgen | tr A-Z a-z; }
```

### 10.1 Check the token / who am I

```bash
rpc get-profile
# {"id":"…","fullname":"…","email":"…","defaultTeamId":"…","defaultProjectId":"…",…}
# id == 00000000-… means the token was NOT accepted
```

### 10.2 List teams

```bash
rpc get-teams
# [{"id":"…","name":"Default","isDefault":true,"permissions":{"type":"owner",…},"features":[…],…}]
```

### 10.3 List projects

**For one team:**

```bash
rpc get-projects '{"teamId":"'$TEAM'"}'
# [{"id":"…","teamId":"…","name":"Drafts","isDefault":true,"isPinned":false,"modifiedAt":"…",…}]
```

**For all teams:**

```bash
rpc get-all-projects
```

### 10.4 Create a project

```bash
rpc create-project '{"teamId":"'$TEAM'","name":"MCP Project"}'
# {"id":"<PROJECT>","teamId":"…","name":"MCP Project",…}
```

- Do not send `id`: since 2.19 the server rejects it.
- Other project calls: `rename-project {id,name}` and `delete-project {id}`.

### 10.5 List files / create a file

**List the files in a project:**

```bash
rpc get-project-files '{"projectId":"'$PROJECT'"}'
```

**Create a file:**

```bash
rpc create-file '{"projectId":"'$PROJECT'","name":"MCP File"}'
# {"id":"<FILE>","projectId":"…","name":"MCP File","revn":0,"vern":0,"features":[…],"isShared":false,…}
```

The new file already has one page, "Page 1". Call `get-file` to get the page id.

### 10.6 Get a file (pages, objects, revn, vern)

```bash
rpc get-file '{"id":"'$FILE'"}' > file.json
jq '{revn, vern, pages: .data.pages, names: [.data.pagesIndex[] | {id, name}]}' file.json
PAGE=$(jq -r '.data.pages[0]' file.json); REVN=$(jq '.revn' file.json); VERN=$(jq '.vern' file.json)
```

**Response shape:**

- `{id, name, revn, vern, features, projectId, isShared, permissions, data: {pages, pagesIndex: {<pageId>: {id, name, objects: {<shapeId>: Shape}}}, colors, typographies, components, …}}`
- Map keys that are UUIDs are returned as strings. Attribute keys are camelCased.
- The response can be large.
  - **Lighter alternatives:**
    - `get-page {fileId, pageId?, objectId?}` returns one page, or one subtree when `objectId` is given.
    - `get-file-summary {id}` and `get-file-info {id}` return only metadata.

### 10.7 Add a page

```bash
SID=$(uuid)   # session id: reuse for the whole MCP session
NEWPAGE=$(uuid)
rpc update-file '{"id":"'$FILE'","sessionId":"'$SID'","revn":'$REVN',"vern":'$VERN',
  "changes":[{"type":"add-page","id":"'$NEWPAGE'","name":"Page 2"}]}'
# {"revn":0,"lagged":[…]}    -> local REVN = REVN+1
REVN=$((REVN+1))
```

To rename the page:

```json
{"type":"mod-page","id":"<PAGE>","name":"Renamed"}
```

### 10.8 Add a rectangle

```bash
RECT=$(uuid)
rpc update-file '{"id":"'$FILE'","sessionId":"'$SID'","revn":'$REVN',"vern":'$VERN',"changes":[{
  "type":"add-obj","id":"'$RECT'","pageId":"'$PAGE'","frameId":"'$ROOT'","parentId":"'$ROOT'",
  "obj":{"id":"'$RECT'","type":"rect","name":"Card","x":100,"y":100,"width":200,"height":120,"rotation":0,
    "selrect":{"x":100,"y":100,"width":200,"height":120,"x1":100,"y1":100,"x2":300,"y2":220},
    "points":[{"x":100,"y":100},{"x":300,"y":100},{"x":300,"y":220},{"x":100,"y":220}],
    "transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},"transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},
    "parentId":"'$ROOT'","frameId":"'$ROOT'",
    "fills":[{"fillColor":"#FF5733","fillOpacity":1}],
    "strokes":[{"strokeColor":"#000000","strokeOpacity":1,"strokeWidth":2,"strokeStyle":"solid","strokeAlignment":"inner"}],
    "r1":8,"r2":8,"r3":8,"r4":8}}]}'
REVN=$((REVN+1))
```

**Change the fill and move the shape, using `assign`:**

```json
{"type":"mod-obj","id":"<RECT>","pageId":"<PAGE>","operations":[{"type":"assign","value":{
  "fills":[{"fillColor":"#2E86DE","fillOpacity":1}],
  "x":150,"y":100,
  "selrect":{"x":150,"y":100,"width":200,"height":120,"x1":150,"y1":100,"x2":350,"y2":220},
  "points":[{"x":150,"y":100},{"x":350,"y":100},{"x":350,"y":220},{"x":150,"y":220}]}}]}
```

**Delete the shape:**

```json
{"type":"del-obj","id":"<RECT>","pageId":"<PAGE>"}
```

### 10.9 Add a board containing a text

This example sends both shapes in **one** `update-file` call. The board must come first. The text uses `frameId` and `parentId` = the board.

```bash
BOARD=$(uuid); TEXT=$(uuid)
cat > changes.json <<EOF
[{"type":"add-obj","id":"$BOARD","pageId":"$PAGE","frameId":"$ROOT","parentId":"$ROOT",
  "obj":{"id":"$BOARD","type":"frame","name":"Hero","x":0,"y":0,"width":1440,"height":900,"rotation":0,
    "selrect":{"x":0,"y":0,"width":1440,"height":900,"x1":0,"y1":0,"x2":1440,"y2":900},
    "points":[{"x":0,"y":0},{"x":1440,"y":0},{"x":1440,"y":900},{"x":0,"y":900}],
    "transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},"transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},
    "parentId":"$ROOT","frameId":"$ROOT","shapes":[],"fills":[{"fillColor":"#FFFFFF","fillOpacity":1}],"strokes":[],
    "hideFillOnExport":false,"r1":0,"r2":0,"r3":0,"r4":0}},
 {"type":"add-obj","id":"$TEXT","pageId":"$PAGE","frameId":"$BOARD","parentId":"$BOARD",
  "obj":{"id":"$TEXT","type":"text","name":"Title","x":100,"y":100,"width":400,"height":58,"rotation":0,
    "selrect":{"x":100,"y":100,"width":400,"height":58,"x1":100,"y1":100,"x2":500,"y2":158},
    "points":[{"x":100,"y":100},{"x":500,"y":100},{"x":500,"y":158},{"x":100,"y":158}],
    "transform":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},"transformInverse":{"a":1,"b":0,"c":0,"d":1,"e":0,"f":0},
    "parentId":"$BOARD","frameId":"$BOARD","growType":"auto-width",
    "content":{"type":"root","verticalAlign":"top","children":[{"type":"paragraph-set","children":[{
      "type":"paragraph","textAlign":"left","fontId":"sourcesanspro","fontFamily":"sourcesanspro",
      "fontVariantId":"bold","fontSize":"48","fontWeight":"700","fontStyle":"normal","lineHeight":"1.2",
      "letterSpacing":"0","textTransform":"none","textDecoration":"none",
      "fills":[{"fillColor":"#111111","fillOpacity":1}],
      "children":[{"text":"Hello from MCP","fontId":"sourcesanspro","fontFamily":"sourcesanspro",
        "fontVariantId":"bold","fontSize":"48","fontWeight":"700","fontStyle":"normal","lineHeight":"1.2",
        "letterSpacing":"0","textTransform":"none","textDecoration":"none",
        "fills":[{"fillColor":"#111111","fillOpacity":1}]}]}]}]}}}]
EOF
rpc update-file "{\"id\":\"$FILE\",\"sessionId\":\"$SID\",\"revn\":$REVN,\"vern\":$VERN,\"changes\":$(cat changes.json)}"
REVN=$((REVN+1))
```

[UNCERTAIN]

- `fontVariantId:"bold"` with `fontWeight:"700"` is how the UI names the Source Sans Pro bold variant.
- No `positionData` is sent, so the text renders in the workspace (recomputed there), but it may not render in the exporter, view mode or thumbnails until the file is opened once (section 7.7).

### 10.10 Upload an image and use it as a fill

```bash
curl -sS -X POST "$PENPOT/api/main/methods/upload-file-media-object" \
  -H "Authorization: Token $TOKEN" -H "Accept: application/json" \
  -F "file-id=$FILE" -F "is-local=true" -F "name=photo" -F "content=@photo.png;type=image/png"
# {"id":"<MEDIA>","width":800,"height":600,"mtype":"image/png",…}
```

Then `add-obj` a rect whose `fills` holds `fillImage: {id: MEDIA, width, height, mtype, keepAspectRatio: true}`.

### 10.11 Add a library color and a typography

```json
[{"type":"add-color","color":{"id":"<uuid>","name":"Brand / Primary","path":"","color":"#2E86DE","opacity":1}},
 {"type":"add-typography","typography":{"id":"<uuid>","name":"H1","path":"","fontId":"sourcesanspro","fontFamily":"sourcesanspro",
   "fontVariantId":"bold","fontSize":"48","fontWeight":"700","fontStyle":"normal","lineHeight":"1.2","letterSpacing":"0","textTransform":"none"}}]
```

To publish the file as a shared library:

```bash
rpc set-file-shared '{"id":"'$FILE'","isShared":true}'
rpc link-file-to-library '{"fileId":"<consumer>","libraryId":"'$FILE'"}'
```

### 10.12 Export a shape to PNG (exporter service)

**The exporter does not accept access tokens.** [SOURCE `exporter/src/app/http.cljs`, `app/auth.cljs`; partly VERIFIED]

- It reads only the `auth-token` **cookie**, which must hold a session token.
- It passes the cookie to a headless browser, which opens the render page.
- It parses **only `application/transit+json`** request bodies. A JSON body is treated as a raw string and fails validation.

**Option A: session cookie.** This needs the account's email and password.

```bash
curl -sS -c jar.txt -X POST "$PENPOT/api/main/methods/login-with-password" \
  -H "Content-Type: application/json" -H "Accept: application/json" \
  -d '{"email":"me@example.com","password":"…"}'     # sets cookie auth-token
PROFILE=$(rpc get-profile | jq -r .id)

curl -sS -b jar.txt -X POST "$PENPOT/api/export" -H "Content-Type: application/transit+json" -d '["^ ",
  "~:cmd","~:export-shapes",
  "~:profile-id","~u'$PROFILE'",
  "~:wait",true,
  "~:exports",[["^ ","~:file-id","~u'$FILE'","~:page-id","~u'$PAGE'","~:object-id","~u'$BOARD'",
                 "~:type","~:png","~:scale",2,"~:suffix","","~:name","hero"]]]'
# transit response, e.g. ["^ ","~:id","~u…","~:name","hero","~:filename","hero.png","~:mtype","image/png","~:uri","https://…/assets/by-id/<id>",…]
curl -sS -o hero.png "<uri from response>"   # assets/by-id is downloadable (temp object, ~10 min)
```

**Export parameters:**

- **`exports[]` items:**
  - required: `file-id`, `page-id`, `object-id`, `type`, `scale` (number), `suffix` (string), `name` (string)
  - optional: `share-id`
  - `type` is one of `~:png`, `~:jpeg`, `~:webp`, `~:svg` or `~:pdf`.
- **Top level:**
  - required: `cmd` (`~:export-shapes`; `~:export-frames` gives a multi-page PDF of boards) and `profile-id`
  - optional: `wait` (bool), `name`, `skip-children`, `force-multiple`, `is-wasm`
- **Results:**
  - Several exports give one zip.
  - With `wait:false`, you get a resource handle at once, and progress goes through the websocket.
- [VERIFIED] `POST /api/export` with a transit body that lacks `exports` and `profile-id` returns `400 spec-validation`.
- **Newer servers (source only):** `POST /api/export/jobs` (same body, the profile comes from the session) → `GET /api/export/jobs/{id}` to poll → the job result holds the resource.

**Option B: share link** [UNCERTAIN, not tested]

1. `create-share-link {fileId, whoComment:"all", whoInspect:"all", pages:[PAGE]}`
2. Call `/api/export` with no cookie and `~:share-id` inside each export item.

Because the render page can load an anonymous view-only bundle with the share id, this may work with nothing but an access token, which is used to create the link. Delete the link afterwards.

**Option C: render locally** [no exporter]

Fetch the subtree with `get-page {fileId, pageId, objectId}` and render it to SVG yourself.

- This is enough for rects, ellipses, paths, fills and strokes.
- Text needs `positionData`, or your own layout.

### 10.13 Export or import a whole `.penpot` file

**Export:**

```bash
curl -sS -N -X POST "$PENPOT/api/main/methods/export-binfile" "${H[@]}" \
  -d '{"fileId":"'$FILE'","includeLibraries":false,"embedAssets":false}'
# text/event-stream; events: "event: progress" …, final "event: end" with "data: <transit string uri>"
# then GET that uri (assets/by-id/…, zip, valid ~60 min)
```

- `includeLibraries` and `embedAssets` are required on 2.18. Newer servers deprecate them in favour of `type`: one of `include-libraries`, `merge-libraries`, `detach-libraries` or `link-later`.

**Import:** call `import-binfile` (multipart) with these fields:

- `name`: a string, or a map from file id to name
- `project-id`
- `file`: the part
- Or send `upload-id` from the chunked upload instead of `file`.

The response is an SSE stream that ends with the ids of the created files.

### 10.14 Comments

```bash
rpc create-comment-thread '{"fileId":"'$FILE'","pageId":"'$PAGE'","frameId":"'$ROOT'","position":{"x":120,"y":80},"content":"Review this"}'
rpc get-comment-threads '{"fileId":"'$FILE'"}'
```

### 10.15 Search

```bash
rpc search-files '{"teamId":"'$TEAM'","searchTerm":"landing"}'
```

---

## 11. MCP implementation notes

**Client setup:**

- Keep one HTTP client that always sends `Authorization: Token`, `Accept: application/json`, `Content-Type: application/json` and `X-Client`.
- Keep one `sessionId` per MCP process.

**Per-file cache:** keep `{revn, vern, features, pages}` for each file.

- After each successful `update-file`, set `revn` to `response.revn + 1`.
- On `409` or `revn-conflict`, re-fetch the file.

**Build shapes in code:**

- Write helper functions that take `{type, x, y, w, h, rotation?}` and produce `selrect`, `points`, `transform` and `transformInverse` (section 7.2).
- Merge the type defaults from section 7.4, so that the LLM supplies only the semantic attributes.

**Validation:**

- Validate shapes on the client against the OpenAPI shape schema (`AddObjChange.obj` in `openapi.json`), or keep to a small subset you know works.
- The server returns `500 assertion/data-validation` with an `explain` string for an invalid shape.

**Writing changes:**

- Use `assign`, not `set`, from JSON.
- Avoid `mov-objects` and `reorder-children` over JSON, or send those changes in transit.

**Reading files:**

- `get-file` can be several MB. For tool output, prefer:
  - `get-page` with `objectId`
  - or a summary you compute yourself from the file: name, type, geometry, and child ids per shape
- `get-file-fragment` is only needed when you request pointer-maps on purpose.

**Rate limits:**

- Expect 429 responses with `Retry-After`, and honour the header.
- `update-file` is serialized per profile (1 in flight, with a queue of 5).

**Live editors:** changes are pushed live to open workspace tabs through the websocket, so a user watching in the browser sees MCP edits right away.

**Self-hosted servers:**

- The flag `enable-access-tokens` is required.
- `client-header-check-middleware` makes the `X-Client` header mandatory.
- `rpc-rlimit` enables rate limits.
- The base URL is `PENPOT_PUBLIC_URI`.
- The exporter must be deployed for exports (`penpotapp/exporter`).

**Version drift:** check the server version with `GET /api/main/doc/openapi.json` (`info.version`). Gate newer commands on that version: the jobs API, `get-file-tokens`, `get-file-plugin-data`, and the `since 2.19` / `2.20` commands.

**Uncertain points to test first:**

1. Whether `mov-objects` silently does nothing over JSON (section 5.5).
2. Whether text renders in the exporter without `positionData`.
3. Whether a share-link export works without a session (section 10.12, option B).
4. The `rotation` and `transform` convention.
5. The Google font `fontId` format.
6. The exact keys of the webhook payload.
7. The update-file response format: a `{revn, lagged}` map, not the vector the OpenAPI document describes.

---

## 12. Exporter service summary

| Endpoint | Method | Auth | Body | Notes |
|---|---|---|---|---|
| `/api/export` | POST | cookie `auth-token` (session) | transit `{cmd, profile-id, exports[], wait?, name?, skip-children?, force-multiple?, is-wasm?}` | `cmd` = `export-shapes` or `export-frames`. Returns a resource `{id, name, filename, mtype, uri?}`. With `wait:true` it returns once the render is done. |
| `/api/export/jobs` | POST | cookie (required) | same | Newer servers only. Returns a job `{id, state, filename, mtype, …}`. |
| `/api/export/jobs/{id}` | GET / DELETE | cookie | n/a | Poll or cancel a job. Only the job's owner can see it. |
| `/readyz` | GET | none | n/a | Health check (direct exporter port 6061). |

**Errors from the exporter:**

| Status | Meaning |
|---|---|
| 400 | `validation` (spec explain) |
| 401 | `authentication` |
| 404 | not found |
| 429 | `queue-full` |
| 503 | `browser-not-ready` |
| 500 | other server errors |

All error bodies are transit.

---

## 13. Command catalog (all RPC commands)

**How to read this catalog:**

- **Where it comes from:** it is generated from the live OpenAPI document (design.penpot.app, v2.18) together with the backend source (`backend/src/app/rpc/commands/*.clj`).
- **Endpoint:** `POST /api/main/methods/<name>`.
- **Param names:** they are shown as camelCase JSON keys. kebab-case is accepted too.
- **Types:**
  - `uuid` is a UUID string.
  - `point` is `{x,y}`.
  - `datetime` is ISO-8601.
  - `duration` is a string that the server parses as `"PT"+value`, so `"720h"` means 30 days. Integer milliseconds also work. For `create-access-token.expiration`, the UI sends `"720h"`, `"1440h"`, `"2160h"` or `"4320h"`; leave the field out for a token that never expires.
  - A trailing `?` on a type means nullable. In nested objects, `key?` means optional.
- **Flags:**
  - **no auth required:** callable without a token.
  - **WEBHOOK event:** the command triggers webhook deliveries.
  - **GET allowed:** the command also accepts GET with query params.
  - **NOT in … 2.18:** the command exists only in newer source.

### Auth & session

#### `check-nitrate-sso`

_since 2.18_

Check if a user needs to login into the organization SSO. Accepts either team-id (to look up the organization via the team) or organization-id directly. Returns {:authorized true :reason :sso-satisfied} when SSO is not active or the session already holds a valid entry for the organization, and {:authorized true :reason :no-team-access} when the gate was skipped because the user cannot access the team; the reason lets the client tell a usable session apart from a plain permission failure. Returns {:authorized false :redirect-uri <url>} when SSO is active; the client must redirect there. The OIDC provider itself handles re-authentication transparently if the user already has an active SSO session. A nil :redirect-uri means SSO is required but the provider is not usable.

| param | type | req |
|---|---|---|
| teamId | uuid | no |
| organizationId | uuid | no |
| url | uri | yes |

#### `create-demo-profile`

_no auth required; since 1.15_

A command that is responsible of creating a demo purpose profile. It only works if the `demo-users` flag is enabled in the configuration.

- change 1.15: This method is migrated from mutations to commands.
- change 2.18: Add optional `skip-onboarding` param. When true, the profile is created with `onboarding-viewed` and `release-notes-viewed` (current version) set, skipping the onboarding flow.
- change 2.18: Add optional `expires-in` param. When set, the demo purge is scheduled that long after creation instead of the global deletion delay. Only values between 5 minutes and the global delay are accepted.
- change 2.18.1: Add optional `renderer` param. When set to `wasm` or `svg`, the profile is created with that `renderer` in props. When omitted, no renderer is stored and the Penpot default applies.

| param | type | req |
|---|---|---|
| skipOnboarding | bool | no |
| expiresIn | duration | no |

#### `get-sso-provider`

_no auth required; since 2.12; GET allowed_

| param | type | req |
|---|---|---|
| email | email | yes |

Returns: `{id: uuid}`

#### `login-with-ldap`

_no auth required; since 1.15_

Performs the authentication using LDAP backend. Only works if LDAP is properly configured and enabled with `login-with-ldap` flag.

| param | type | req |
|---|---|---|
| email | email | yes |
| password | string | yes |
| invitationToken | string | no |

#### `login-with-password`

_no auth required; since 1.15_

Performs authentication using penpot password.

| param | type | req |
|---|---|---|
| email | email | yes |
| password | string | yes |
| invitationToken | string | no |

#### `logout`

_no auth required; since 1.0_

Clears the authentication cookie and logout the current session.

- change 2.1: Now requires profile-id passed in the body

| param | type | req |
|---|---|---|
| profileId | uuid | no |

#### `prepare-register-profile`

_no auth required; since 1.15_

| param | type | req |
|---|---|---|
| fullname | string | yes |
| email | email | yes |
| password | string | yes |
| acceptNewsletterUpdates | boolean | no |
| invitationToken | string | no |

#### `recover-profile`

_no auth required; since 1.15_

| param | type | req |
|---|---|---|
| token | string | yes |
| password | string | yes |

#### `register-profile`

_no auth required; since 1.15_

| param | type | req |
|---|---|---|
| token | string | yes |
| acceptNewsletterUpdates | boolean | no |

#### `request-profile-recovery`

_no auth required; since 1.15_

| param | type | req |
|---|---|---|
| email | email | yes |

#### `verify-token`

_no auth required; since 1.15_

| param | type | req |
|---|---|---|
| token | string max 5000 | yes |

### Profile

#### `add-profile-plugin`

_since 2.18; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "add-profile-plugin"}
   [:plugin ctp/schema:registry-entry]])
```

#### `delete-profile`

_since 1.0; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "request-email-change"}
   [:email ::sm/email]])
```

#### `delete-profile-photo`

_since 2.17_

Params: none (`{}`)

Returns: `null`

#### `get-profile`

_no auth required; since 1.18; GET allowed_

Params: none (`{}`)

Returns: `{id: uuid, fullname: string, email: email, theme?: string, isAdmin?: bool, isActive?: bool, isBlocked?: bool, isDemo?: bool, isMuted?: bool, createdAt?: datetime(ISO), modifiedAt?: datetime(ISO), defaultProjectId?: uuid, defaultTeamId?: uuid, props?: ProfileProps{plugins,renderer,mcpEnabled,newsletterUpdates,newsletterNews,onboardingTeamId,onboardingViewed,onboardingQuestions,onboardingQuestionsAnswered,nitrateOnboardingViewed,v2InfoShown,releaseNotesViewed,notifications,workspaceVisited,customS...`

#### `get-profiles-for-file-comments`

_since 1.15; GET allowed_

Retrieves a list of profiles with limited set of properties of all participants on comment threads of the file.

- change 1.15: Imported from queries and renamed.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| shareId | uuid? | no |

#### `get-subscription-usage`

_since 2.9; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

#### `get-subscription-warning`

_since 2.18; GET allowed_

Params: none (`{}`)

Returns: `{type?: string, daysFromExpiry?: int, daysUntilExpiry?: int, expirationDate?: string(iso)}`

#### `remove-profile-plugin`

_since 2.18; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "remove-profile-plugin"}
   [:plugin-id ::sm/uuid]])
```

#### `request-email-change`

_since 1.0_

| param | type | req |
|---|---|---|
| email | email | yes |

#### `send-user-feedback`

_since 1.18_

| param | type | req |
|---|---|---|
| subject | string max 500 | yes |
| content | string max 2500 | yes |
| type | string | no |
| errorHref | string max 2500 | no |
| errorReport | string max 1048576 | no |

#### `update-profile`

_since 1.0_

| param | type | req |
|---|---|---|
| fullname | string | yes |
| lang | string max 8 | no |
| theme | string max 250 | no |

Returns: `{id: uuid, fullname: string, email: email, theme?: string, isAdmin?: bool, isActive?: bool, isBlocked?: bool, isDemo?: bool, isMuted?: bool, createdAt?: datetime(ISO), modifiedAt?: datetime(ISO), defaultProjectId?: uuid, defaultTeamId?: uuid, props?: ProfileProps{plugins,renderer,mcpEnabled,newsletterUpdates,newsletterNews,onboardingTeamId,onboardingViewed,onboardingQuestions,onboardingQuestionsAnswered,nitrateOnboardingViewed,v2InfoShown,releaseNotesViewed,notifications,workspaceVisited,customS...`

#### `update-profile-notifications`

_since 2.4.0_

| param | type | req |
|---|---|---|
| dashboardComments | enum(all\|partial\|none) | yes |
| emailComments | enum(all\|partial\|none) | yes |
| emailInvites | enum(all\|none) | yes |

#### `update-profile-password`

_since 1.0_

| param | type | req |
|---|---|---|
| password | string | yes |
| oldPassword | string | no |

#### `update-profile-photo`

_multipart/form-data_

| param | type | req |
|---|---|---|
| file | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |

Returns: `null`

#### `update-profile-props`

_since 1.0_

| param | type | req |
|---|---|---|
| props | {plugins?: object{ids,data}, renderer?: enum(svg\|wasm), mcpEnabled?: bool, newsletterUpdates?: bool, newsletterNews?: bool, onboardingTeamId?: uuid, onboardingViewed?: bool, onboardingQuestions?: map<string,string>, onboardingQuestionsAnswered?: bool, nitrateOnboardingViewed?: bool, v2InfoShown?: b... | yes |

### Access tokens

#### `create-access-token`

_since 1.18_

| param | type | req |
|---|---|---|
| name | string max 250 | yes |
| expiration | duration | no |
| type | string | no |

#### `delete-access-token`

_since 1.18_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-access-tokens`

_since 1.18; GET allowed_

Params: none (`{}`)

#### `get-current-mcp-token`

_DEPRECATED; since 2.15; GET allowed_

Params: none (`{}`)

### Teams, members & invitations

#### `check-team-external-invitations`

_since 2.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| organizationId | uuid | yes |

Returns: `{hasExternalInvitations: bool, allowsAnybody: bool}`

#### `create-team`

_since 1.17_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| name | string | yes |
| features | string[] (features set) | no |
| id | uuid | no |
| organizationId | uuid | no |
| isDefault | boolean | no |

#### `create-team-access-request`

_since 2.2.0_

A rpc call that allow to request for an invitations to join the team.

| param | type | req |
|---|---|---|
| fileId | uuid | no |
| teamId | uuid | no |
| isViewer | bool | no |

#### `create-team-invitations`

_since 1.17_

A rpc call that allows to send single or multiple invitations to join the team. Supports two parameter formats: 1. emails (set) + role (single role for all emails) 2. invitations (vector of {:email :role} maps for individual roles)

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| emails | email[] | no |
| role | enum(viewer\|admin\|editor\|owner) | no |
| invitations | object{email,role}[] | no |

#### `create-team-with-invitations`

_since 1.17_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| name | string max 250 | yes |
| features | string[] (features set) | no |
| id | uuid | no |
| emails | email[] | yes |
| role | enum(viewer\|admin\|editor\|owner) | yes |

#### `delete-team`

_since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `delete-team-invitation`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| email | email | yes |

#### `delete-team-member`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| memberId | uuid | yes |

#### `get-owned-teams`

_since 2.8.0; GET allowed_

Params: none (`{}`)

#### `get-team`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| id | uuid | no |
| fileId | uuid | no |

#### `get-team-info`

_no auth required; since 2.2.0; GET allowed_

Retrieve minimal team info by its ID.

| param | type | req |
|---|---|---|
| id | uuid | no |
| fileId | uuid | no |

#### `get-team-invitation-token`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| email | email | yes |

#### `get-team-invitations`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `get-team-members`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `get-team-stats`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `get-team-users`

_since 1.17; GET allowed_

Get team users by team-id or by file-id

| param | type | req |
|---|---|---|
| teamId | uuid | no |
| fileId | uuid | no |

#### `get-teams`

_since 1.17; GET allowed_

Params: none (`{}`)

#### `leave-team`

_since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| reassignTo | uuid | no |

#### `update-team`

_since 1.17_

| param | type | req |
|---|---|---|
| name | string | yes |
| id | uuid | yes |

#### `update-team-invitation-role`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| email | email | yes |
| role | enum(viewer\|admin\|editor\|owner) | yes |

#### `update-team-member-role`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| memberId | uuid | yes |
| role | enum(viewer\|admin\|editor\|owner) | yes |

#### `update-team-photo`

_since 1.17; multipart/form-data_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| file | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |

### Projects

#### `create-project`

_WEBHOOK event; since 1.18_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| name | string max 250 | yes |
| id | uuid | no |

#### `delete-project`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `duplicate-project`

_WEBHOOK event; since 1.16_

Duplicate an entire project with all the files

| param | type | req |
|---|---|---|
| projectId | uuid | yes |
| name | string max 250 | no |

#### `get-all-projects`

_since 1.18; GET allowed_

Params: none (`{}`)

#### `get-project`

_since 1.18; GET allowed_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-project-files`

_since 1.17; GET allowed_

Get all files for the specified project.

| param | type | req |
|---|---|---|
| projectId | uuid | yes |

Returns: `File{id,features,hasMediaTrimmed,commentThreadSeqn,name,revn,vern,modifiedAt,isShared,projectId,createdAt,data}[]`

#### `get-projects`

_since 1.18; GET allowed_

- change 2.12: This endpoint now return deleted but recoverable projects

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `move-project`

_WEBHOOK event; since 1.16_

Move projects between teams

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| projectId | uuid | yes |

#### `rename-project`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| name | string max 250 | yes |
| id | uuid | yes |

#### `update-project-pin`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| isPinned | bool | yes |
| id | uuid | yes |

### File snapshots (versions)

#### `create-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| label | string | no |

#### `delete-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-file-snapshot`

_since 2.16; GET allowed_

Retrieve a file bundle with data from a specific snapshot for read-only preview. Does not modify any database state.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| id | uuid | yes |
| features | string[] (features set) | no |

Returns: `{id: uuid, features: string[] (features set), hasMediaTrimmed: bool, commentThreadSeqn: integer, name: string max 250, revn: integer, vern: integer, modifiedAt: datetime(ISO), isShared: bool, projectId: uuid, createdAt: datetime(ISO), data?: any, permissions: Permissions{type,isOwner,isAdmin,canEdit,canRead,isLogged}}`

#### `get-file-snapshots`

_since 1.20; GET allowed_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

#### `lock-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `restore-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| id | uuid | yes |

#### `unlock-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `update-file-snapshot`

_since 1.20_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| label | string | yes |

### File thumbnails

#### `create-file-object-thumbnail`

_since 1.19_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| objectId | string max 250 | yes |
| media | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |
| tag | string max 50 | no |

#### `create-file-thumbnail`

_since 1.19_

Creates or updates the file thumbnail. Mainly used for paint the grid thumbnails.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| revn | int | yes |
| media | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |

#### `delete-file-object-thumbnail`

_since 1.19_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| objectId | string max 250 | yes |

#### `delete-file-object-thumbnails`

_since 1.19_

| param | type | req |
|---|---|---|
| objectIds | string max 250[] | yes |

#### `get-file-data-for-thumbnail`

_since 1.17; GET allowed_

Retrieves the data for generate the thumbnail of the file. Used mainly for render thumbnails on dashboard.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

Returns: `{id: uuid, revn: int, page: map<string,any>, stripFramesWithThumbnails?: bool}`

#### `get-file-object-thumbnails`

_since 1.17; GET allowed_

Retrieve a file object thumbnails.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| tag | string max 50 | no |

Returns: `map<string,string max 250>`

### Media / images

#### `assemble-file-media-object`

_since 2.17_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| sessionId | uuid | yes |
| fileId | uuid | yes |
| isLocal | bool | yes |
| name | string max 250 | yes |
| mtype | string | yes |
| id | uuid | no |

#### `clone-file-media-object`

_since 1.17_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| isLocal | bool | yes |
| id | uuid | yes |

#### `create-file-media-object-from-url`

_since 1.17_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| isLocal | bool | yes |
| url | uri | yes |
| id | uuid | no |
| name | string max 250 | no |

#### `create-upload-session`

_since 2.17_

| param | type | req |
|---|---|---|
| totalChunks | integer | yes |

Returns: `{sessionId: uuid}`

#### `upload-chunk`

_since 2.17; multipart/form-data_

| param | type | req |
|---|---|---|
| sessionId | uuid | yes |
| index | int | yes |
| content | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |

Returns: `{sessionId: uuid, index: int}`

#### `upload-file-media-object`

_since 1.17; multipart/form-data_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| id | uuid | no |
| fileId | uuid | yes |
| isLocal | bool | yes |
| name | string max 250 | yes |
| content | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | yes |

### Fonts

#### `create-font-variant`

_WEBHOOK event; since 1.18; multipart/form-data_

Upload a font variant. Font data must be provided as an `:uploads` map (keyed by mime-type, values are upload-session UUIDs from the chunked-upload API).

- change 2.16: Add :uploads param for chunked upload support
- change 2.18: Remove :data param, use :uploads exclusively

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| fontId | uuid | yes |
| fontFamily | string | yes |
| fontWeight | enum(950\|300\|600\|500\|100\|800\|200\|900\|700\|400) | yes |
| fontStyle | enum(normal\|italic) | yes |
| uploads | map<string,uuid> | yes |

#### `delete-font`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| id | uuid | yes |

#### `delete-font-variant`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| id | uuid | yes |

#### `download-font`

_since 2.15_

Download the font file. Returns a http redirect to the asset resource uri.

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `download-font-family`

_since 2.15_

Download the entire font family as a zip file. Returns the zip bytes on the body, without encoding it on transit or json.

| param | type | req |
|---|---|---|
| fontId | uuid | yes |

#### `get-font-variants`

_since 1.18; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | no |
| fileId | uuid | no |
| projectId | uuid | no |
| shareId | uuid | no |

#### `update-font`

_WEBHOOK event; since 1.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| id | uuid | yes |
| name | string | yes |

### Comments

#### `create-comment`

_WEBHOOK event; since 1.15_

| param | type | req |
|---|---|---|
| threadId | uuid | yes |
| content | string max 750 | yes |
| shareId | uuid? | no |
| mentions | uuid[] | no |

#### `create-comment-thread`

_WEBHOOK event; since 1.15_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| position | point | yes |
| content | string max 750 | yes |
| pageId | uuid | yes |
| frameId | uuid | yes |
| shareId | uuid? | no |
| mentions | uuid[] | no |

#### `delete-comment`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| shareId | uuid? | no |

#### `delete-comment-thread`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| shareId | uuid? | no |

#### `get-comment-thread`

_since 1.15; GET allowed_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| id | uuid | yes |
| shareId | uuid? | no |

#### `get-comment-threads`

_since 1.15; GET allowed_

| param | type | req |
|---|---|---|
| fileId | uuid | no |
| teamId | uuid | no |
| shareId | uuid? | no |

#### `get-comments`

_since 1.15; GET allowed_

| param | type | req |
|---|---|---|
| threadId | uuid | yes |
| shareId | uuid? | no |

#### `get-unread-comment-threads`

_since 1.15; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `mark-all-threads-as-read`

_since 1.15_

| param | type | req |
|---|---|---|
| threads | uuid[] | yes |

#### `update-comment`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| content | string max 750 | yes |
| shareId | uuid? | no |
| mentions | uuid[] | no |

#### `update-comment-thread`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| isResolved | boolean | yes |
| shareId | uuid? | no |

#### `update-comment-thread-frame`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| frameId | uuid | yes |
| shareId | uuid? | no |

#### `update-comment-thread-position`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| position | point | yes |
| frameId | uuid | yes |
| shareId | uuid? | no |

#### `update-comment-thread-status`

_since 1.15_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| shareId | uuid? | no |

### Viewer & share links

#### `create-share-link`

_since 1.18_

Creates a share-link object. Share links are resources that allows external users access to specific pages of a file with specific permissions (who-comment and who-inspect).

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| whoComment | string max 250 | yes |
| whoInspect | string max 250 | yes |
| pages | uuid[] | yes |

#### `delete-share-link`

_since 1.18_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-view-only-bundle`

_no auth required; since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| shareId | uuid | no |
| features | string[] (features set) | no |

### Webhooks

#### `create-webhook`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| uri | uri | yes |
| mtype | enum(application/json\|application/transit+json) | yes |

#### `delete-webhook`

_since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-webhooks`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `update-webhook`

_since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| uri | uri | yes |
| mtype | enum(application/json\|application/transit+json) | yes |
| isActive | bool | yes |

### Export / import (binfile) & jobs

#### `cancel-job`

_since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Cancel a job the caller created, whether it is still waiting or already running. A running job stops at its next beat and rolls its work back; a job that already finished between the request and the update is answered with the state it is in, never an error. Like every other command here, a job of another profile is indistinguishable from one that does not exist.

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "cancel-job" :closed true}
   [:id ::sm/uuid]])
```

#### `clone-template`

_WEBHOOK event; since 1.16_

Clone into the specified project the template by its id.

| param | type | req |
|---|---|---|
| projectId | uuid | yes |
| templateId | string | yes |

#### `create-export-binfile-job`

_WEBHOOK event; since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Create a durable job that exports a set of files as a `.penpot` package. Its params are `:file-ids`, the set of files to export, and `:export-type`, how the libraries of those files are handled: what the job-def declares, and nothing more. There is no default for the type, because a caller that does not say it would get a package that is not the one it asked for. The job is created, not run: the caller follows it by its id. The read permission of every file is checked here so a refusal arrives synchronously, and the handler checks it again before reading.

#### `create-import-binfile-job`

_WEBHOOK event; since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only)_

Create a durable job that imports a `.penpot` package into a project. The package arrives as a chunked upload, which the server assembles itself: the command never reads a path the caller names. The package is then checked and stored as the resource of the job, so any worker can consume it by `resource_id`. The job is created, not run, and the caller follows it by its id.

#### `export-binfile`

_WEBHOOK event; since 1.15_

Export a penpot file in a binary format.

- change 2.12: Remove version parameter, only one version is supported
- change 2.19: Deprecated `include-libraries` and `embed-assets` params

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| includeLibraries | bool | yes |
| embedAssets | bool | yes |

#### `get-builtin-templates`

_since 1.19; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

#### `get-job`

_since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

Read a job the caller created: its state and, once it is over, what it produced or why it failed. The result of a finished job is the JSON the row holds, decoded with the result schema its job-def declares: a job that keys its result by uuid (the import resolution) answers with the types it stored, and a job whose job-def declares no schema answers with the JSON as it is on disk, where a uuid is text and a uuid used as a map key is a keyword. A result that does not conform to the declared schema never reaches the column: the substrate validates it when the job is completed. The progress and the history of the job are not part of the answer: a client that follows a job takes its progress from the websocket, and the event log still has no read surface.

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "get-job" :closed true}
   [:id ::sm/uuid]])
```

#### `import-binfile`

_WEBHOOK event; since 1.15_

Import a penpot file in a binary format. The file content may be provided either as a multipart `file` upload or as an `upload-id` referencing a completed chunked-upload session, which allows importing files larger than the multipart size limit.

- change 1.20: Set default version to 3
- change 2.15: Add upload-id param for chunked upload support

| param | type | req |
|---|---|---|
| name | string max 250 or map<string,string max 250> | yes |
| projectId | uuid | yes |
| version | enum(1\|3) | no |
| file | {filename: string, size: int, path: path, mtype?: string, headers?: map<string,string>} | no |
| uploadId | uuid | no |

### Libraries

#### `get-file-libraries`

_since 1.17; GET allowed_

Get libraries used by the specified file.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

#### `get-library-file-references`

_since 1.17; GET allowed_

Returns all the file references that use specified file (library) id.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

#### `get-library-usage`

_since 2.10.0; GET allowed_

Gets the number of files that use the specified library.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

Returns: `int`

#### `get-team-shared-files`

_since 1.17; GET allowed_

Get all file (libraries) for the specified team.

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `has-file-libraries`

_since 1.15.1_

Checks if the file has libraries. Returns a boolean

| param | type | req |
|---|---|---|
| fileId | uuid | yes |

Returns: `bool`

#### `ignore-file-library-sync-status`

_since 1.17_

Ignore updates in linked files

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| date | datetime(ISO) | yes |

#### `link-file-to-library`

_WEBHOOK event; since 1.17_

Link a file to a library. Returns the recursive list of libraries used by that library

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| libraryId | uuid | yes |

#### `set-file-shared`

_WEBHOOK event; since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| isShared | bool | yes |

#### `unlink-file-from-library`

_WEBHOOK event; since 1.17_

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| libraryId | uuid | yes |

#### `update-file-library-sync-status`

_since 1.17_

Update the synchronization status of a file->library link

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| libraryId | uuid | yes |

### Search

#### `search-files`

_since 1.17_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| searchTerm | string max 250 | no |

### Files & pages

#### `create-file`

_WEBHOOK event; since 1.17_

- change 2.19: The optional :id param is rejected with a params-validation error; the server always generates the identifier

| param | type | req |
|---|---|---|
| name | string max 250 | yes |
| projectId | uuid | yes |
| id | uuid | no |
| isShared | bool | no |
| features | string[] (features set) | no |

#### `delete-file`

_WEBHOOK event; since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `duplicate-file`

_WEBHOOK event; since 1.16_

Duplicate a single file in the same team.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| name | string max 250 | no |

#### `get-file`

_since 1.17; GET allowed_

Retrieve a file by its ID. Only authenticated users.

| param | type | req |
|---|---|---|
| features | string[] (features set) | no |
| id | uuid | yes |

Returns: `{id: uuid, features: string[] (features set), hasMediaTrimmed: bool, commentThreadSeqn: integer, name: string max 250, revn: integer, vern: integer, modifiedAt: datetime(ISO), isShared: bool, projectId: uuid, createdAt: datetime(ISO), data?: any, permissions: Permissions{type,isOwner,isAdmin,canEdit,canRead,isLogged}}`

#### `get-file-fragment`

_no auth required; since 1.17; GET allowed_

Retrieve a file fragment by its ID. Only authenticated users.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| fragmentId | uuid | yes |
| shareId | uuid | no |

Returns: `{id: uuid, fileId: uuid, createdAt: datetime(ISO), content: any}`

#### `get-file-info`

_no auth required; since 2.2.0; GET allowed_

Retrieve minimal file info by its ID.

| param | type | req |
|---|---|---|
| features | string[] (features set) | no |
| id | uuid | yes |

#### `get-file-plugin-data`

_since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

Return the file-level plugin data: the arbitrary key/value metadata stored on the file root through the Plugin API. Optionally narrowed to a single `namespace`. Lets external tooling read metadata such as issue or commit references without downloading the whole file.

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "get-file-plugin-data"}
   [:id ::sm/uuid]
   [:namespace {:optional true} :keyword]])
```

#### `get-file-stats`

_since 2.17; GET allowed_

Return aggregate statistics for a single file: page count, shape counts by type, component/color/typography counts, and inbound and outbound library reference counts. Cheap alternative to `get-file` when only metrics are needed.

| param | type | req |
|---|---|---|
| id | uuid | yes |

Returns: `{fileId: uuid, pageCount: integer, shapeCounts: FileStatsShapeCounts{total,byType}, componentCount: integer, deletedComponentCount: integer, colorCount: integer, typographyCount: integer, libraryCount: integer, referencedByCount: integer, revn: integer, updatedAt: datetime(ISO)}`

#### `get-file-summary`

_since 1.20; GET allowed_

Retrieve a file summary by its ID. Only authenticated users.

| param | type | req |
|---|---|---|
| id | uuid | yes |

#### `get-file-tokens`

_since 2.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

Return the design tokens of a file as a DTCG document, or nil when the file has none. Active themes and sets come from the file's tokens status. Cheap alternative to `get-file` when only the tokens are needed.

Params (malli schema from source, kebab-case):
```clojure
[:map {:title "get-file-tokens"}
   [:file-id ::sm/uuid]])
```

#### `get-page`

_since 1.17; GET allowed_

Retrieves the page data from file and returns it. If no page-id is specified, the first page will be returned. If object-id is specified, only that object and its children will be returned in the page objects data structure. If you specify the object-id, the page-id parameter becomes mandatory. Mainly used for rendering purposes on the exporter. It does not accepts client features.

| param | type | req |
|---|---|---|
| fileId | uuid | yes |
| pageId | uuid | no |
| shareId | uuid | no |
| objectId | uuid or set | no |
| features | string[] (features set) | no |

#### `get-team-deleted-files`

_since 2.12; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `get-team-recent-files`

_since 1.17; GET allowed_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |

#### `move-files`

_WEBHOOK event; since 1.16_

Move a set of files from one project to other.

| param | type | req |
|---|---|---|
| ids | uuid[] | yes |
| projectId | uuid | yes |

#### `permanently-delete-team-files`

_since 2.13_

Mark the specified files to be deleted immediatelly on the specified team. The team-id on params will be used to filter and check writable permissons on team.

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| ids | uuid[] | yes |

#### `rename-file`

_WEBHOOK event; since 1.17_

| param | type | req |
|---|---|---|
| name | string max 250 | yes |
| id | uuid | yes |

Returns: `{id: uuid, name: string max 250, createdAt: datetime(ISO), modifiedAt: datetime(ISO)}`

#### `restore-deleted-team-files`

_since 2.13_

Removes the deletion mark from the specified files (and respective projects) on the specified team.

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| ids | uuid[] | yes |

#### `update-file`

_WEBHOOK event; since 1.17_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| sessionId | uuid | yes |
| revn | int | yes |
| vern | int | yes |
| features | string[] (features set) | no |
| changes | Change[] (see update-file section) | no |
| changesWithMetadata | {changes: Change[], hintOrigin?, hintEvents?}[] | no |
| skipValidate | bool | no |

Returns: `object{changes,fileId,id,revn,sessionId}[]`

### Organizations / Nitrate (enterprise, mostly irrelevant)

#### `add-team-to-organization`

_since 2.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| organizationId | uuid | yes |

#### `all-organization-members-in-team`

_since 2.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| organizationId | uuid | yes |

Returns: `bool`

#### `all-team-members-in-organizations`

_since 2.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| organizationIds | uuid[] | yes |

Returns: `map<string,bool>`

#### `check-organization-members`

_since 2.18_

| param | type | req |
|---|---|---|
| organizationId | uuid | yes |
| emails | email[] | yes |

Returns: `map<string,boolean>`

#### `get-leave-organization-summary`

_since 2.18; GET allowed_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| defaultTeamId | uuid | yes |

Returns: `{teamsToDelete: int, teamsToTransfer: int, teamsToExit: int, teamsToDetach: int, memberAddedAt: string(iso), organizationMemberCountBefore: int}`

#### `get-nitrate-activation-code-request`

_since 2.18; GET allowed_

Returns a Base64-encoded JSON file requesting a Nitrate activation code. Payload includes nitrateId, publicKey, email and iat.

Params: none (`{}`)

Returns: `string`

#### `get-nitrate-connectivity`

_since 2.18; GET allowed_

Params: none (`{}`)

Returns: `{licenses: bool}`

#### `get-owned-organizations-summary`

_since 2.18; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

List organizations owned by the current profile with team and member counts. Used by the delete-account modal to warn the user about cascading deletion.

#### `leave-organization`

_since 2.18_

| param | type | req |
|---|---|---|
| id | uuid | yes |
| name | string | yes |
| defaultTeamId | uuid | yes |
| teamsToDelete | uuid[] | yes |
| teamsToLeave | object{id,reassignTo}[] | yes |

#### `redeem-nitrate-activation-code`

_since 2.18_

| param | type | req |
|---|---|---|
| activationCode | string | yes |

Returns: `{cancelAt: string(iso)}`

#### `remove-team-from-organization`

_since 2.18_

| param | type | req |
|---|---|---|
| teamId | uuid | yes |
| organizationId | uuid | yes |
| organizationName | string | yes |

### Misc / internal

#### `get-enabled-flags`

_no auth required; DEPRECATED; since 1.20; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

Deprecated. Use `get-environment-data` instead.

#### `get-environment-data`

_no auth required; since 2.19; NOT in design.penpot.app 2.18 OpenAPI (newer source only); GET allowed_

Returns the deployment type and the enabled environment flags.

#### `get-error-report`

_token perms "error-reports:read"; since 2.20; GET allowed_

| param | type | req |
|---|---|---|
| id | uuid | yes |

Returns: `{id: uuid, createdAt: string(iso), source: string, profileId?: string, kind?: string, tenant?: string, version?: string, hint?: string, report?: string, href?: string, context?: string}`

#### `get-error-reports`

_token perms "error-reports:read"; since 2.20; GET allowed_

| param | type | req |
|---|---|---|
| since | string(iso) | no |
| sinceId | uuid | no |
| limit | int | no |
| source | string | no |
| profileId | string | no |
| kind | string | no |
| tenant | string | no |
| version | string | no |
| hint | string | no |
| until | string(iso) | no |

Returns: `{items: object{id,createdAt,source,profileId,kind,tenant,version,hint}[], nextSince?: string(iso), nextId?: uuid}`

#### `push-audit-events`

_since 1.17_

| param | type | req |
|---|---|---|
| events | Event{name,type,props,timestamp,context}[] | yes |


---

## Appendix A. Source pointers (penpot/penpot `main`)

| Topic | Path |
|---|---|
| RPC routes, auth wrapping, GET restriction | `backend/src/app/rpc.clj` |
| Body parsing, response format, auth header parsing, CORS | `backend/src/app/http/middleware.clj` |
| Error to HTTP status mapping | `backend/src/app/http/errors.clj` |
| Access token middleware | `backend/src/app/http/access_token.clj` |
| Commands | `backend/src/app/rpc/commands/*.clj` |
| Rate and concurrency limits | `backend/resources/rlimit.edn`, `backend/resources/climit.edn` |
| update-file | `backend/src/app/rpc/commands/files_update.clj` |
| Change vocabulary and operations | `common/src/app/common/files/changes.cljc` |
| Shape schema and defaults | `common/src/app/common/types/shape.cljc`, `shape/*.cljc`, `fills.cljc`, `color.cljc`, `path/impl.cljc`, `text.cljc` |
| Features | `common/src/app/common/features.cljc` |
| How the frontend builds changes | `common/src/app/common/files/changes_builder.cljc` |
| Exporter | `exporter/src/app/{http,router,auth,handlers}.cljs`, `exporter/src/app/handlers/export_shapes.cljs` |
| Frontend API client | `frontend/src/app/main/repo.cljs` |
