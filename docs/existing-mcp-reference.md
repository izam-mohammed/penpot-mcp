# Existing Penpot MCP server and Penpot skill: reference

Written 2026-10-19. This is a read-only survey of the Penpot MCP setup on this machine. It covers:

- the hosted Penpot MCP server at `design.penpot.app`, which this machine's config points to;
- its open-source code (`penpot/penpot`, `mcp/` directory, `develop` branch);
- the local Claude Code skill at `~/.claude/skills/penpot`, symlinked from `~/.claude-<org>/skills/penpot`.

It is input for building our own server, which will call Penpot's backend HTTP API directly instead of going through the plugin bridge.

Sources and how each was read:

| source | how it was read |
| --- | --- |
| live hosted server (`design.penpot.app/mcp/stream`) | JSON-RPC over streamable HTTP with the skill's `penpot_mcp.py` helpers: `initialize`, `tools/list`, `resources/list`, `prompts/list`, `high_level_overview`, and `penpot_api_info` for about 40 types. No `execute_code` or `export_shape` call was made, and nothing in Penpot was changed. |
| upstream source | `gh api repos/penpot/penpot/contents/mcp/packages/...?ref=develop`, read 2026-10-19 |
| upstream docs | `github.com/penpot/penpot-mcp` README (archived), `penpot/penpot/mcp/README.md`, `help.penpot.app/mcp/` |
| skill | every file under `~/.claude/skills/penpot` |

Uncertain points are marked **[uncertain]**.

---

## 1. Overview and architecture

### 1.1 What it is

The Penpot MCP server does not talk to Penpot's backend API. It is a **relay** between an MCP client (an LLM agent) and a **Penpot plugin running inside a browser tab** that has a Penpot file open. Every design operation runs as JavaScript inside that tab, against the **Penpot Plugin API** (the `penpot` global). The server holds no design data of its own.

```
MCP client (Claude Code)
   │  streamable HTTP, JSON-RPC 2.0 (POST /mcp, or /mcp/stream on the hosted service)
   ▼
MCP server (Node, express + @modelcontextprotocol/server)
   │  WebSocket (default port 4402); task = {id, type, params}; response = {id, success, data|error}
   ▼
Penpot MCP plugin (iframe UI + plugin sandbox) in the user's browser tab
   │  new Function(penpot, penpotMgmt, storage, console, penpotUtils, code)
   ▼
Penpot Plugin API -> the workspace's in-browser state -> Penpot backend (persisted by the web app)
```

Consequences that define its limits:

- A browser tab with the file open and the plugin connected is **required**. No tab means no answers.
- The agent acts only on the file open in that tab, and on whichever page is active there.
- The server can only do what the Plugin API exposes. The Plugin API cannot create projects or files, rename files, list teams or projects, or read comments outside the open file. It also cannot export without the export service.
- Writes are persisted by the Penpot web app's normal change pipeline, not by the server. A call that "returned" is not proof that a change was committed (see the traps in §6).

### 1.2 Components (upstream monorepo `mcp/`)

| path | role |
| --- | --- |
| `packages/server/` | MCP server: tool registry, HTTP endpoint, WebSocket `PluginBridge`, task dispatch, REPL, logging |
| `packages/plugin/` | Penpot plugin: WebSocket client, heartbeat, `ExecuteCodeTaskHandler`, `PenpotUtils` |
| `packages/common/` | shared TypeScript request and response types |
| `types-generator/` | builds the API docs YAML that `penpot_api_info` serves |
| `bin/`, `scripts/`, `resources/` | packaging and bootstrap; `resources/` holds the instruction texts (the high-level overview) **[uncertain: exact file names]** |

The old standalone repo, `github.com/penpot/penpot-mcp`, is **archived**. Its code now lives in `penpot/penpot` under `mcp/`. It is published on npm as `@penpot/mcp` (`npx -y @penpot/mcp@latest`).

### 1.3 Server internals (from `PenpotMcpServer.ts`, `PluginBridge.ts`)

- **MCP server identity:** `{name: "penpot", version: "1.0.0"}`. Capabilities are `tools` (`listChanged: true`) only. There are **no resources and no prompts**: `resources/list`, `resources/templates/list` and `prompts/list` all return `-32601 Method not found` on the hosted server.
- **`initialize` instructions** (live, verbatim): "You have access to Penpot tools in order to interact with Penpot designs. Before working with these tools, be sure to read the 'Penpot High-Level Overview' via the `high_level_overview` tool."
- **HTTP:** express, `app.all("/mcp")`. A fresh `McpServer` is created per request (`createMcpHandler`). The `userToken` query parameter goes into an `AsyncLocalStorage` session context, and logs show only a token fingerprint. The legacy `/sse` and `/messages` endpoints are no longer supported upstream.
- **Hosted path:** `https://design.penpot.app/mcp/stream?userToken=<MCP key>`. The `/mcp/stream` prefix comes from the hosted deployment's reverse proxy. **[uncertain]** whether it maps to `/mcp` one to one.
- **The server sets an `Mcp-Session-Id` header.** The skill's client stores it in `$TMPDIR/penpot-mcp-session-<uid>` (mode 0600), and re-runs `initialize` on HTTP 400 or 404.
- **The server needs a `User-Agent` header.** Without one, the hosted endpoint answered **403 Forbidden** (seen 2026-10-17, Penpot 2.18.4).
- **WebSocket bridge:**
  - The plugin connects to `ws://<host>:4402` (`?userToken=` in multi-user mode, where a missing token is rejected with close code 1008).
  - Within 10 s, the plugin must send `{type: "initialize", session: {sessionId, fileId, fileName, workspaceState: "none"|"loading"|"ready"}}`, or the socket is closed.
  - Message types from the plugin: `heartbeat` (every 10 s), `freeze` (Chrome page freeze), `context-update` (file or page changed), and task responses.
  - The server pings every 30 s for keep-alive.
  - A connection is **stale** when no plugin message has arrived for 30 s. Calls then fail with "The Penpot plugin tab appears to be suspended by the browser (no heartbeat for Ns)...". A frozen tab gives "The Penpot plugin tab has been frozen by the browser...".
  - No connection gives: "No Penpot instance connected for user token. Please ensure that Penpot is connected and that the MCP client connection is using the correct token."
- **Task timeout:** `PENPOT_MCP_TOOL_TIMEOUT_S`, default **120 s**. On timeout, the work may still have been fully applied in the tab.
- **Response size cap (hosted only):** the plugin refuses to send a serialised response over **15,000,000 chars** (`MAX_TASK_RESPONSE_SIZE_REMOTE_MCP`) and sends an error instead.
- **Sessions (develop branch):**
  - Each tab has a short session id derived from `penpot.currentUser.sessionId`, shown in the plugin UI with a Copy button.
  - Tools take an optional `sessionId`. With exactly one connected session it is chosen automatically. With several, the error lists them and the agent must ask the user to choose.
  - **The hosted server's live schema has no `sessionId` parameter yet** (see §3). The deployed version is older than `develop`.
- **Multi-instance:** with `PENPOT_MCP_REDIS_URI` set and in multi-user mode, tasks are routed through Redis to the instance that holds the plugin's socket. Channel names are qualified by `PENPOT_TENANT` (default `default`).

### 1.4 Plugin internals (from `ExecuteCodeTaskHandler.ts`, `main.ts`, `PenpotUtils.ts`)

- The execution context is created **once per plugin load**: `{penpot, penpotMgmt (or null), storage: {}, console: ExecuteCodeTaskConsole, penpotUtils: PenpotUtils}`.
- `storage` persists across calls. It is lost when the plugin reloads, when the tab is suspended and woken, or when the user clicks Connect again.
- **For each call:**
  1. The console log is reset.
  2. `penpot.flags.naturalChildOrdering` and `penpot.flags.throwValidationErrors` are set to `true` for the call, then restored afterwards.
  3. `penpot.waitForLayoutUpdate()` is awaited first, if it exists. With `penpotMgmt`, this happens only when `workspace.status === "ready"`.
  4. The code runs as `new Function(...keys, "return (async () => { CODE })();")`, so top-level `await` and `return` work.
  5. If the result is a top-level `Uint8Array`, it is sent as `{__type: "base64", data}`.
  6. The response is `{result, log}`.
- If `penpot.flags` is missing, the call fails with an "incompatible Penpot version" error.
- `penpotMgmt` (`PenpotMgmt` type) is new on `develop`: "If `penpotMgmt` is null, file management is unavailable and only the open file can be used." **[uncertain]** It is not exposed on the hosted server as of 2026-10-19: the live tool description lists only `penpot`, `penpotUtils` and `storage`.
- Reconnect policy: exponential backoff from 1 s up to 30 s. On `visibilitychange` or `resume`, the plugin sends a heartbeat or reconnects.

---

## 2. Install and connect configuration

### 2.1 What is configured on this machine (redacted)

The only Penpot MCP entry found is at **project scope** in `~/.claude-<org>/.claude.json`, under project `~/<another-project>`:

```json
"penpot": { "type": "http", "url": "https://design.penpot.app/mcp/stream?userToken=<REDACTED>" }
```

None was found in `~/.claude.json`, user-scope `mcpServers`, `settings*.json`, or any `.mcp.json` up to depth 4 under `~`. The `penpot` MCP tools are **not loaded** in this session (`ToolSearch "penpot"` returns nothing), because the entry is scoped to another project. The skill's `penpot_mcp.py` still finds it through its "any project" fallback.

No local copy of the server source is installed: no npm global, no clone under `~` or `~/fun`. Repositories with Penpot design sources (data, not server code) exist in `~/fun/izam/{money-logs,car-mate,browser-hand}/design/penpot/`.

### 2.2 Hosted server (design.penpot.app)

1. In Penpot, go to **Your account → Integrations → MCP Server** and turn on Status. Generate the MCP key; it is shown once, and each user has one key. Regenerating the key invalidates the old one immediately.
2. Copy the URL: `https://design.penpot.app/mcp/stream?userToken=<MCP key>`. The URL is the credential.
3. Add it to Claude Code at user scope. Never put it in a project `.mcp.json`:
   `claude mcp add -s user --transport http penpot '<URL>'`
   Penpot also suggests `npx -y add-mcp -g -n penpot <URL>`.
4. Open the design file in a browser, then connect the plugin. The help centre says **File → MCP Server → Connect**. The skill observed an **MCP** button in the file toolbar, with a grey dot when disconnected and a **Connect here** option. Only one tab per user is active at a time, and connecting a new tab takes the connection from the old one.
5. Hosted limits: no local file system, so `import_image` and `export_shape.filePath` are absent. Multi-user mode is always remote mode.

### 2.3 Self-hosted or local server (upstream)

- Start: `npx -y @penpot/mcp@latest`, or from source with `pnpm run bootstrap`.
- Load the plugin in Penpot from `http://localhost:4400/manifest.json` and click "Connect to MCP server". Keep the plugin UI open: closing it ends the connection.
- Client: `claude mcp add penpot -t http http://localhost:4401/mcp`. Stdio-only clients need `npx -y mcp-remote <url>`.
- Ports: 4400 plugin web server, 4401 MCP HTTP, 4402 WebSocket, 4403 REPL.
- Browser notes: Chromium 142+ asks for local-network permission, Brave Shields may block the connection, and Firefox works.

| env var | default | meaning |
| --- | --- | --- |
| `PENPOT_MCP_SERVER_HOST` | `localhost` | MCP HTTP bind address (older README: `PENPOT_MCP_SERVER_LISTEN_ADDRESS`) |
| `PENPOT_MCP_SERVER_PORT` | `4401` | MCP HTTP port |
| `PENPOT_MCP_WEBSOCKET_PORT` | `4402` | plugin WebSocket port |
| `PENPOT_MCP_REPL_HOST` / `PENPOT_MCP_REPL_PORT` | `localhost` / `4403` | dev REPL |
| `PENPOT_MCP_REPL_ENABLE` | unset (falls back to `DEVENV`) | REPL on or off; never on in multi-user mode |
| `PENPOT_MCP_REMOTE_MODE` | `false` | `true` disables file-system tools |
| `PENPOT_MCP_DEVENV` | `false` | registers the developer tools (§3.6) in local single-user mode |
| `PENPOT_MCP_TOOL_TIMEOUT_S` | `120` | per-task timeout |
| `PENPOT_MCP_EXPORT_SHAPE_MAX_PARALLEL_REQUESTS` | `0` (unbounded) | export concurrency, multi-user only |
| `PENPOT_MCP_REDIS_URI` | unset | multi-instance routing, multi-user only |
| `PENPOT_TENANT` | `default` | Redis channel namespace |
| `PENPOT_MCP_LOG_LEVEL` | `info` | `trace` \| `debug` \| `info` \| `warn` \| `error` |
| `PENPOT_MCP_LOG_DIR` | unset | file logging only when set |
| `PENPOT_MCP_PLUGIN_SERVER_HOST` | local only | plugin web server address(es), comma-separated |
| `PENPOT_MCP_SERVER_ADDRESS` (older README) | `localhost` | address the plugin uses to build `ws://<addr>:4402` |

Multi-user mode is started with the `--multi-user` flag. It needs user tokens on both the client URL and the plugin socket.

---

## 3. Tools

### 3.1 Summary

| tool | hosted (live) | local single-user | dev env only |
| --- | --- | --- | --- |
| `execute_code` | yes | yes | |
| `high_level_overview` | yes | yes | |
| `penpot_api_info` | yes | yes | |
| `export_shape` | yes (no `filePath`) | yes (with `filePath`) | |
| `import_image` | no | yes | |
| `cljs_repl`, `import_penpot_file`, `cljs_compiler_output`, `clj_check_parentheses`, `read_taiga_issue` | no | no | yes (`PENPOT_MCP_DEVENV=true`, not multi-user) **[uncertain: exact registered names; class names are `CljsReplTool`, `ImportPenpotFileTool`, `CljsCompilerOutputTool`, `CljCheckParentheses`, `ReadTaigaIssueTool`]** |

Every tool returns MCP `content`: `text`, or `image` (base64 PNG) for `export_shape`. A thrown error does **not** become a JSON-RPC error. It comes back as a normal text result, `"Tool execution failed: <error>"`, and the hosted server sometimes adds `isError: true`. Clients must string-match these messages (the skill does: "no penpot instance connected", "heartbeat"). All live tool schemas carry `"execution": {"taskSupport": "forbidden"}`.

### 3.2 `execute_code`

- **Purpose:** run arbitrary JavaScript in the plugin context. This is the only way to read or write design data.
- **Input (live hosted schema):**

```json
{ "type": "object",
  "properties": { "code": { "type": "string", "minLength": 1,
                            "description": "The JavaScript code to execute in the plugin context." } },
  "required": ["code"] }
```

  On `develop` there is also `sessionId?: string` (1 to 128 chars): "Penpot session to use. Optional when exactly one session is connected. If several sessions are listed in an error, ask the user to select one and pass its sessionId."
- **Globals:** `penpot` (type `Penpot`), `penpotUtils`, `storage`, and `console` (captured); `penpotMgmt` on develop. The code is a function body: `return` gives the result, and `await` is allowed.
- **Returns:** text containing `JSON.stringify({result, log}, null, 2)`. `result` is whatever was returned; non-JSON values such as Shapes serialise as plain objects of their enumerable props, or `{}` **[uncertain]**. `log` holds the captured console output. With no return value the text is "Code executed successfully with no return value." An exception returns its message.
- **Description guidance (verbatim points):** read the overview first; keep intermediates and functions in `storage`; never log what you return; try a simple approach first.
- **Example:**

```js
// code
return { file: penpot.currentFile.name, id: penpot.currentFile.id,
         page: penpot.currentPage.name,
         pages: penpotUtils.getPages().map(p => p.name), version: penpot.version };
```

  Result text: `{"result": {"file": "...", "id": "...", "page": "...", "pages": [...], "version": "2.18.4"}, "log": ""}`.

### 3.3 `high_level_overview`

- **Purpose:** returns the "Penpot High-Level Overview": about 29 KB of usage instructions for the Plugin API, summarised in §5. The description says not to call it again once read.
- **Input:** `{}` (no properties).
- **Returns:** one text block. It needs no plugin connection: it worked live with no file open.

### 3.4 `penpot_api_info`

- **Purpose:** API documentation lookup, served from YAML bundled with the server. It needs no plugin connection.
- **Input:** `{type: string (minLength 1, required), member?: string}`.
- **Returns:** if the full text of the type is at most 2,000 chars, it is returned whole. Otherwise the overview (the TypeScript interface) is returned, followed by "Member details not provided (too long). Call this tool with a member name for more information." With `member`, the member's docs are returned. An unknown type or member gives `API type "X" not found` or `Member "m" not found in type "X"`.
- **Example:** `{type: "Export"}` returns the `Export` interface (`type: "svg"|"png"|"jpeg"|"webp"|"pdf"; scale?; suffix?; skipChildren?`) with member docs.
- **Known types** (from the overview): Penpot, ActiveUser, Blur, Board, VariantContainer, Boolean, CloseOverlay, Color, ColorShapeInfo, ColorShapeInfoEntry, Comment, CommentThread, CommonLayout, Context, ContextGeometryUtils, ContextTypesUtils, ContextUtils, Dissolve, Ellipse, EventsMap, Export, File, FileValidationError, FileVersion, Fill, Flags, FlexLayout, Flow, Font, FontVariant, FontsContext, GridLayout, Group, GuideColumn, GuideColumnParams, GuideRow, GuideSquare, GuideSquareParams, HistoryContext, Image, Interaction, LayoutCellProperties, LayoutChildProperties, Library, LibraryColor, LibraryComponent, LibraryVariantComponent, LibraryElement, LibrarySummary, LibraryTypography, LocalStorage, NavigateTo, OpenOverlay, OpenUrl, OverlayAction, Page, Path, PathCommand, PluginData, PreviousScreen, Push, Rectangle, RulerGuide, Shadow, ShapeBase, Slide, Stroke, SvgRaw, Text, TextRange, ToggleOverlay, Track, TokenBase, TokenBorderRadius, TokenShadowValue, TokenShadowValueString, TokenShadow, TokenColor, TokenDimension, TokenFontFamilies, TokenFontSizes, TokenFontWeights, TokenLetterSpacing, TokenNumber, TokenOpacity, TokenRotation, TokenSizing, TokenSpacing, TokenBorderWidth, TokenTextCase, TokenTextDecoration, TokenTypographyValue, TokenTypographyValueString, TokenTypography, TokenCatalog, TokenSet, TokenTheme, User, Variants, Viewport, Action, Animation, BooleanType, Bounds, Gradient, Guide, ImageData, LibraryContext, Point, RulerGuideOrientation, Shape, StrokeCap, Theme, TrackType, Trigger, TokenValueString, Token, and the `Token*Props`, TokenProperty and TokenType aliases.

### 3.5 `export_shape`

- **Purpose:** render a shape to PNG or SVG so the agent can see it.
- **Input (live hosted schema):**

```json
{ "type": "object",
  "properties": {
    "shapeId": { "type": "string", "minLength": 1,
      "description": "Identifier of the shape to export. Special identifiers you can use: 'selection' (first shape currently selected by the user), 'page' (entire current page)" },
    "format": { "type": "string", "enum": ["svg", "png"], "default": "png" },
    "mode":   { "type": "string", "enum": ["shape", "fill"], "default": "shape",
      "description": "'shape' = full shape incl. descendants; 'fill' = the raw image used as the shape's image fill (PNG only)" } },
  "required": ["shapeId"] }
```

  Local mode adds `filePath?: string` (absolute path; the file is written and a text confirmation returned). Develop adds `sessionId?`.
- **How it works:**
  1. The server generates `return penpotUtils.exportImage(<shape>, mode, asSvg)`. The shape is `penpot.selection[0]`, `penpot.root`, or `penpotUtils.findShapeById(id)`.
  2. This runs as an `executeCode` task. The plugin calls `shape.export({type: "png"|"svg"})`, which goes to the Penpot **export service** (`/api/export`, a headless-browser renderer). For `fill` mode it reads `fillImage.data()`.
  3. The server converts non-PNG bytes to PNG with `sharp`.
- **Returns:** PNG as an MCP `image` content block (`{type: "image", data: <base64>, mimeType: "image/png"}`), or SVG as text.
- **Known failures:**
  - `ResourceRequest timed out` (500 from the export service, intermittent, even on an 8 × 8 rect; retry works).
  - `shapeId: "page"` times out (`locator.waitFor: Timeout`).
  - An id from an earlier session, or a board on a non-active page, gives `Cannot read properties of null (reading 'export')`. `"selection"` with nothing selected gives the same error on `undefined`.
  - The real reason sits on `e.data` (a ClojureScript map). `e.message` says only "http error".

### 3.6 `import_image` (local mode only)

- **Input:** `{filePath: string (absolute, required), x?: number, y?: number, width?: number (>0), height?: number (>0), sessionId?}`. If only one dimension is given, the other keeps the aspect ratio.
- **Formats:** `.jpg`/`.jpeg`/`.png`/`.gif`/`.webp`. SVG is not supported.
- **Behaviour:** the server reads the file and base64-encodes it. The plugin then runs `penpotUtils.importImage(b64, mime, fileName, x, y, w, h)`, which calls `penpot.uploadMediaData` and creates a Rectangle with `fills: [{fillImage}]`.
- **Returns:** `{"shapeId": "<id>"}`.
- **On the hosted server**, do the same by hand: `penpot.uploadMediaData(name, Uint8Array, mime)` inside `execute_code`, with the bytes inlined as base64. This is what the skill's `upload` command does.

### 3.7 Developer tools (`PENPOT_MCP_DEVENV=true`, local single-user only)

These are for people working on Penpot itself, not design work. They connect to Penpot's ClojureScript nREPL:

- the CLJS REPL tool evaluates ClojureScript in the frontend;
- import a `.penpot` file;
- read the CLJS compiler output;
- check Clojure parentheses;
- read a Taiga issue.

Not studied further; out of scope.

---

## 4. Penpot Plugin API surface (what `execute_code` can reach)

Taken from live `penpot_api_info` against Penpot 2.18.4.

### 4.1 `penpot` (type `Penpot` = `Context` plus `ui`, `utils`, `closePlugin`, `on`, `off`)

- **State:** `version`, `root` (current page root), `currentFile: File|null`, `currentPage: Page|null`, `selection: Shape[]`, `viewport`, `flags`, `history`, `library`, `fonts`, `currentUser`, `activeUsers`, `theme`, `localStorage`.
- **Create:** `createRectangle()`, `createBoard()`, `createEllipse()`, `createPath()`, `createText(text)` (can be null), `createBoolean(boolType, shapes)`, `createShapeFromSvg(svg): Group|null`, `createShapeFromSvgWithImages(svg): Promise<Group|null>`, `createPage(): Page`, `createVariantFromComponents(boards)`.
- **Structure:** `group(shapes)`, `ungroup(group, ...)`, `flatten(shapes): Path[]`, `alignHorizontal(shapes, "left"|"center"|"right")`, `alignVertical(shapes, "top"|"center"|"bottom")`, `distributeHorizontal(shapes)`, `distributeVertical(shapes)`.
- **Media:** `uploadMediaUrl(name, url): Promise<ImageData>`, `uploadMediaData(name, Uint8Array, mime): Promise<ImageData>`.
- **Code generation:** `generateMarkup(shapes, {type: "html"|"svg"})`, `generateStyle(shapes, {type: "css", withPrelude?, includeChildren?})`, `generateFontFaces(shapes): Promise<string>`.
- **Colours:** `shapesColors(shapes)`, `replaceColor(shapes, old, new)`.
- **Navigation:** `openPage(page|id, newWindow?)`, `openViewer()`.
- **Events:** `on(type, cb)` and `off(id)`, for example `pagechange`.
- **Undocumented in the typings:** `penpot.waitForLayoutUpdate(timeoutMs?)`, added by 2.18.0. It resolves when pending layout work is done.

### 4.2 `File`

`id`, `name` (**read-only**), `revn`, `pages: Page[]`, `export("penpot"|"zip", libraryExportType?: "all"|"merge"|"detach"): Promise<Uint8Array>`, `findVersions({createdBy?})`, `saveVersion(label)`, `validate(): FileValidationError[]`, plugin data.

`FileVersion` has `label`, `createdBy?`, `createdAt`, `isAutosave`, `restore()` (reloads the workspace and closes the plugin), `remove()` and `pin()`.

### 4.3 `Page`

`id`, `name` (writable), `root`, `rulerGuides`, `remove()`, `getShapeById(id)`, `findShapes({name?, nameLike?, type?})`, `flows`, `createFlow(name, board)`, `removeFlow`, `addRulerGuide(orientation, value, board?)`, `removeRulerGuide`, `addCommentThread(content, position)`, `removeCommentThread`, `findCommentThreads({onlyYours, showResolved})`, plugin data.

### 4.4 Shapes (`ShapeBase`; concrete types `Board`, `Group`, `Rectangle`, `Ellipse`, `Path`, `Text`, `Image` (legacy), `Boolean`, `SvgRaw`)

- **Geometry:** `x`, `y` (absolute page coordinates, writable), `width`, `height`, `bounds`, `center` (read-only), `parentX`/`parentY`/`boardX`/`boardY` (read-only), `resize(w, h)`, `rotate(angle, center?)`, `rotation`, `flipX`, `flipY`, `proportionLock`, `constraintsHorizontal`, `constraintsVertical`, `fixedWhenScrolling`.
- **Style:**
  - `fills: Fill[] | "mixed"`, where a `Fill` is `{fillColor?, fillOpacity?, fillColorGradient?, fillColorRefFile?, fillColorRefId?, fillImage?}`;
  - `strokes: Stroke[]`, where a `Stroke` is `{strokeColor?, strokeOpacity?, strokeStyle?: "none"|"svg"|"mixed"|"solid"|"dotted"|"dashed", strokeWidth?, strokeAlignment?: "center"|"inner"|"outer", strokeCapStart?, strokeCapEnd?, strokeColorGradient?, ref fields}`;
  - `shadows: Shadow[]`, where a `Shadow` is `{style: "drop-shadow"|"inner-shadow", offsetX, offsetY, blur, spread, hidden, color}`;
  - `blur?`, `backgroundBlur?` (accepted but not rendered), `opacity`, `blendMode` (16 values), `borderRadius` and four per-corner radii, `hidden`, `visible`, `blocked`.
- **Tree and z-order:** `parent`, `parentIndex`, `setParentIndex(i)`, `bringToFront()`, `bringForward()`, `sendToBack()`, `sendBackward()`, `clone()`, `remove()`.
- **Components:** `isComponentInstance()`, `isComponentMainInstance()`, `isComponentCopyInstance()`, `isComponentRoot()`, `isComponentHead()`, `componentRefShape()`, `componentRoot()`, `componentHead()`, `component()`, `detach()`, `swapComponent(c)`, `resetOverrides()`, `switchVariant(pos, value)`, `combineAsVariants(ids)`, `isVariantHead()`.
- **Export:** `exports: Export[]`, `export({type: "svg"|"png"|"jpeg"|"webp"|"pdf", scale?, suffix?, skipChildren?}): Promise<Uint8Array>`.
- **Prototyping:** `interactions`, `addInteraction(trigger, action, delay?)`, `removeInteraction`.
- **Tokens:** `tokens` (map from property to token name), `applyToken(token, properties?)`.
- **Layout child:** `layoutChild?` (`absolute`, margins, `horizontalSizing`/`verticalSizing` `"fix"|"auto"|"fill"`, min and max sizes, `zIndex`), `layoutCell?`.
- **Plugin data:** `get/setPluginData`, `get/setSharedPluginData(namespace, ...)`.

### 4.5 `Board`

All of `ShapeBase`, plus:

- `type: "board"`, `clipContent`, `showInViewMode`, `children`, `appendChild`, `insertChild(i, s)`;
- `addFlexLayout(): FlexLayout`, `addGridLayout(): GridLayout`, `flex?`, `grid?`;
- `guides`, `rulerGuides`, `addRulerGuide`, `removeRulerGuide`;
- `horizontalSizing?`/`verticalSizing?` (`"auto"|"fix"`), `isVariantContainer()`.

`FlexLayout` has `dir` (`row`, `row-reverse`, `column`, `column-reverse`), `wrap`, `rowGap`, `columnGap`, the paddings, `alignItems`, `alignContent`, `justifyItems`, `justifyContent`, `horizontalSizing`/`verticalSizing` (`fill|auto|fix`), `appendChild`, `remove()`.

`GridLayout` has the same, plus `dir: row|column`, `rows`/`columns: Track[]`, `addRow`, `addRowAtIndex`, `addColumn`, `addColumnAtIndex`, `removeRow`, `removeColumn`, `setRow`, `setColumn`, and `appendChild(child, row, column)` with 1-based indices.

### 4.6 `Text`

- **Content and sizing:** `characters`, `growType` (`fixed`|`auto-width`|`auto-height`).
- **Font:** `fontId`, `fontFamily`, `fontVariantId`, `fontSize` (**string**, or `"mixed"`), `fontWeight` (string), `fontStyle`.
- **Spacing:** `lineHeight` and `letterSpacing`, both strings.
- **Formatting:** `textTransform`, `textDecoration`, `direction`, `align`, `verticalAlign`.
- **Methods and bounds:** `textBounds`, `getRange(start, end): TextRange`, `applyTypography(t)`.
- **No OpenType feature settings.**

### 4.7 Fonts

`penpot.fonts` has `all: Font[]`, `findById`, `findByName` (**substring match**), `findAllById`, `findAllByName`. A `Font` has `{name, fontId, fontFamily, fontStyle, fontVariantId, fontWeight, variants: FontVariant[], applyToText(text, variant?), applyToRange(range, variant?)}`.

### 4.8 Libraries

- `penpot.library`:
  - `local: Library`;
  - `connected: Library[]`;
  - `availableLibraries(): Promise<LibrarySummary[]>`;
  - `connectLibrary(id): Promise<Library>`.
- `Library`:
  - `id`, `name`, `colors`, `typographies`, `components`, `tokens: TokenCatalog`;
  - `createColor()`, `createTypography()`, `createComponent(shapes)`.
- `LibraryColor`:
  - `{id, libraryId, name, path, color?, opacity?, gradient?, image?, asFill(), asStroke()}`;
  - `path` holds the group: setting `name = "a/b"` stores `path "a"` and `name "b"`.
- `LibraryTypography`:
  - `{id, libraryId, name, path, fontId, fontFamily, fontVariantId, fontSize, fontWeight, fontStyle?, lineHeight, letterSpacing, textTransform?}`;
  - the metrics are strings;
  - methods `applyToText(shape)`, `applyToTextRange(range)`, `setFont(font, variant?)`.
- `LibraryComponent`:
  - `{id, libraryId, name, path, instance(), mainInstance(), isVariant(), transformInVariant()}`.
- Variants:
  - `VariantContainer` (a board);
  - `Variants` has `properties`, `addProperty()`, `renameProperty(pos, name)`, `currentValues(prop)`, `variantComponents()`;
  - `LibraryVariantComponent` has `variants`, `variantProps`, `variantError`, `setVariantProperty(pos, value)`.

### 4.9 Design tokens

- `penpot.library.local.tokens` (`TokenCatalog`):
  - `{themes, sets, addTheme({group, name}), addSet({name, active?}), getThemeById, getSetById}`.
- `TokenSet`:
  - `{id, name, active, tokens, tokensByType, toggleActive(), getTokenById, addToken({type, name, value}), duplicate(), remove()}`.
- `TokenTheme`:
  - `{id, externalId, group, name, active, toggleActive(), activeSets, addSet(set), removeSet(set), duplicate(), remove()}`;
  - pass the set object to `addSet`, not its name.
- Token types: color, dimension, spacing, typography, shadow, opacity, borderRadius, borderWidth, fontWeights, fontSizes, fontFamilies, letterSpacing, textDecoration, textCase, and also number, rotation and sizing.
- Values can be references (`"{color.primary}"`). `resolvedValue` follows them.
- Applying a token is asynchronous (about 100 ms). Setting a property directly removes the binding.

### 4.10 Other contexts

- `Viewport`: `{center, zoom, bounds, zoomReset(), zoomToFitAll(), zoomIntoView(shapes)}`.
- `HistoryContext`: `{undoBlockBegin(): Symbol, undoBlockFinish(id)}`.
- `Flags`: `{naturalChildOrdering, throwValidationErrors}`. The plugin forces both to `true` during a call.
- `User`: `{id, name?, avatarUrl?, color, sessionId?}`. `ActiveUser` is also available.
- `LocalStorage`: `getItem`, `setItem`, `removeItem`, `getKeys`.
- `CommentThread`: `{seqNumber, board?, owner?, position, resolved, findComments(), reply(content), remove()}`.
- `ContextUtils`: `{geometry, types}`.
- `penpot.ui`: `open`, `resize`, `sendMessage`, `onMessage`. This is the plugin iframe UI and is not useful to an agent.

### 4.11 `penpotUtils` (the plugin's helper library, `PenpotUtils.ts`)

- **Find shapes and pages:**
  - `shapeStructure(shape, maxDepth?)` returns `{id, name, type, children?, layout?}`;
  - `findShapes(predicate, root=null)` and `findShape(predicate, root=null)` search **all pages** when there is no root (slow; can wedge the plugin);
  - `findShapeById(id)`, `findPage(pred)`;
  - `getPages()` returns `{id, name}` **descriptors**, not `Page` objects;
  - `getPageById(id)`, `getPageByName(name)`, `getPageForShape(shape)`.
- **Geometry:** `generateCss(shape)`, `getBounds(shape)`, `isContainedIn(child, parent)`, `setParentXY(shape, px, py)`.
- **Layout and analysis:** `addFlexLayout(container, "row"|"column")` keeps the visual order; `analyzeDescendants(root, evaluator, maxDepth?)`.
- **Images:** `base64ToByteArray(b64)`, `importImage(b64, mime, name, x?, y?, w?, h?)`, `exportImage(shape, "shape"|"fill", asSVG)`.
- **Tokens:** `findTokensByName(name)`, `findTokenByName(name)`, `getTokenSet(token)`, `tokenOverview()`.
- **Variants:** `createVariantContainer([{shape, properties: {Prop: "Value"}}])`.

---

## 5. Guidance carried by `high_level_overview` (condensed)

- **Structure:** pages, then `page.root`, then boards, groups, and leaf shapes.
- **Positioning:** `x` and `y` are absolute. `width`, `height` and `bounds` are read-only, so use `resize`. `parentX`/`parentY` are read-only, so use `penpotUtils.setParentXY`.
- **Fills and strokes:** the `fills`, `strokes` and `shadows` array contents are read-only. Replace the whole array. Colours are upper-case hex.
- **Z-order:** the `children` order is the z-order; the last child is on top.
- **Removing:** `remove()` on a descendant of a component main only hides it.
- **Reparenting:** `appendChild` or `insertChild` moves a shape and keeps its absolute position.
- **Images:** images are `fillImage` on a fill. The `Image` type is legacy.
- **Layouts:**
  - Under a layout, positions are controlled by the layout.
  - `layoutChild.absolute` opts a child out. The overview says x and y are then relative, but the skill measured them as absolute.
  - Use `auto` sizing for containers that fit their content.
  - Use `penpotUtils.addFlexLayout` on a board that already has children.
  - Grid children use 1-based cells.
- **Text:** `resize` resets `growType` to `fixed`. Set it back afterwards, and sleep about 100 ms before reading sizes. Apply fonts with `font.applyToText(text, variant)`.
- **Selection:** copy `penpot.selection` into `storage` straight away.
- **Code export:** `generateStyle` and `generateMarkup` turn designs into CSS, HTML or SVG.
- **Libraries and components:** create colours and typographies; `component.instance()`, `mainInstance()`, `createComponent(shapes)`; `detach()` before restructuring a copy.
- **Variants:** use the `penpotUtils.createVariantContainer` workflow; the overview gives the low-level order if that is not used.
- **Tokens:** sets, themes and `applyToken`, with the list of `TokenProperty` names.
- **Design practice:** look at designs with `export_shape`. Do not invent colours when translating a design to code. Use flex or grid and semantic names.

---

## 6. Known limitations and traps

Collected from the server source, the help centre, and the skill's `references/api-traps.md`. Each row there has a date and a version; most were observed on Penpot 2.18.0 or 2.18.4.

### 6.1 Architectural limits (of the plugin-bridge design)

1. **A browser tab must be open with the file, and the plugin must be connected.** Closing the plugin UI or the tab, or opening another tab that takes the connection, cuts the agent off.
2. **Tab suspension:**
   - Chrome freezes background tabs, and the server then answers "no heartbeat for Ns" (stale after 30 s). The cure is a human focusing the tab.
   - After a wake, `storage` is empty, and the first calls are slow and often hit the 120 s timeout.
   - The plugin can also drop its connection while the tab is awake (after long idle). The toolbar MCP button then shows a grey dot, and only clicking **Connect here** restores it.
3. **One active tab per user.** The connected file can change under the agent: after a wake, the plugin answered from a different file with the same page names. Always check `penpot.currentFile.id`.
4. **No projects or files from the API.** Creating a project or file, renaming a file, and team or project navigation are dashboard clicks, done by a human or by browser automation.
5. **Only the active page is writable.** "Cannot modify a page that is not currently active", and the active page can revert between calls. `openPage` in one call, then draw in the next.
6. **Timeouts do not roll back.** A 120 s timeout can hide a fully applied call, so re-running it duplicates shapes. A call that throws part-way keeps its earlier statements. A call that returned may not be committed: a `detach()` on a nested copy crashed the workspace, and the file reloaded to its last server-saved state.
7. **The export depends on the export service.** It times out under load (`ResourceRequest timed out`) whatever the size or format. `penpot.generateMarkup([shape], {type: "svg"})` is a fast fallback that needs no export service.
8. **Large results.** Hosted responses are capped at about 15 MB. Base64 PNGs bloat the agent's context unless written straight to disk, which the skill does.
9. **The hosted server has no file-system tools.** Images must be inlined as base64 in `execute_code`.
10. **Version skew.** The hosted server lags behind `develop`: no `sessionId`, no `penpotMgmt`. The plugin refuses to run if `penpot.flags` is missing.
11. **Auth.** The URL carries the token. A missing `User-Agent` gives a 403. A server added mid-session to Claude Code shows "Connected", but its tools never load until a restart.

### 6.2 Plugin API traps

| area | trap | what to do |
| --- | --- | --- |
| Layout and measuring | Geometry is not computed on create: text reads 1×1, shapes 100×100 at (0,0). | `await penpot.waitForLayoutUpdate()`, with a fallback of `viewport.zoomIntoView` plus a 1,200 ms sleep. |
| | The flush does not wait for font loading: the first text in a new face stays 1×1. | Poll, or read in a later call. |
| Clipping | Boards clip children by default. | Set `clipContent = false` where needed. |
| | Clipping does **not** clip rotated children in `export_shape`. | |
| Text | `resize()` resets `growType` to `"fixed"`. | Set the grow type after resizing. |
| | Auto-height reads `height: 1` until it settles. | |
| | `createText("")` throws `:applyToText`. | Use `" "`. |
| | `Text.fontSize` is a string, or `"mixed"`. | |
| | `fills` can be the string `"mixed"`. | |
| | `findByName` substring-matches (`"Inter"` gives `"Inter Tight"`). | Search `fonts.all` for an exact name. |
| | No OpenType features (no tabular figures). | |
| Libraries and tokens | `LibraryColor` and `createComponent` split names at `/` into `path`, which breaks naive idempotency guards and doubles component prefixes. | |
| | Token names are paths: a leaf cannot also be a group. | |
| | `rgba({ref}, a)` is rejected. | |
| | Binding a shadow token does nothing. | |
| | An opacity token on a board fades all its children. | |
| | `TokenTheme.addSet("name")` fails with "should be an uuid". | Pass the set object. |
| | Removing a set and re-adding one with the same name in one call rolls the call back. | |
| Effects and SVG | `backgroundBlur` is not rendered. | |
| | Gradient fills need `width`. | |
| | `createShapeFromSvg` drops gradients. | |
| | Setting `rotation` moves `x`/`y`. | |
| Components | `remove()` on a component descendant only hides it. | |
| | `appendChild` on a copy throws "Cannot change the structure of a component copy". | |
| | `detach()` on a nested copy can crash the workspace. | |
| Flex layout | `addFlexLayout()` on a board that has children **reverses** them. | `penpotUtils.addFlexLayout` keeps the order. |
| | `leftPadding` was seen to have no effect. | |
| | `verticalSizing = "fill"` breaks auto height. | |
| | An absolute flex child takes page coordinates. | |
| Finding shapes | `page.findShapes(criteria)` takes an object; `penpotUtils.findShapes(predicate, root)` takes a function. Mixing them fails silently or throws. | |
| | `findShapes({type: "board"})` returns nested boards and the page's `Root Frame` (id all zeros, 0.01 px, `parent` null). | Filter top-level boards by `s.parent && s.parent.id === page.root.id`. |
| | Shape wrappers are fresh objects, so `===` is always false. | Compare `id`. |
| Pages and ids | `penpot.getPages` does not exist. | Use `penpot.currentFile.pages`. |
| | `getPages()` returns descriptors. | |
| | `export_shape` with ids from an earlier session fails. | |
| Names | A leading `/` in a name is stripped. | |
| | `File.name` is read-only. | |
| Sandbox | Assigning `fn.apply`, `call`, `bind`, `name` or `length` on a function throws (read-only in the sandbox). | |
| | A helper that closes over a board keeps drawing onto that board in later calls. | Pass the board as an argument. |
| Batch size | **[unverified, third-party]** About 5 to 10 shape ops per call is the safe limit. | This machine saw complete state after timeouts; it suggests at most about 60 shapes per call after a wake. |

---

## 7. The `penpot` skill

`~/.claude/skills/penpot`, symlinked into `~/.claude-<org>/skills/`. Nothing Penpot-related is in `~/.claude-<org>/skills/synced`.

### 7.1 Purpose and route

"connect, then project and file, then read the repository, then draw, then check, then export, then cite." Penpot behaviour, the builder and the checks live in the skill. A repository's palette, fonts, folder and board size live in the repository, and `references/house-conventions.md` holds the six questions that find them.

### 7.2 Files

| file | what it is |
| --- | --- |
| `SKILL.md` | the workflow (sections 1 to 7 plus upkeep) |
| `scripts/penpot_mcp.py` | standalone MCP client, Python stdlib only (detail in §7.3) |
| `scripts/draw_screens.py` | draws a JSON screen spec through `storage.R`, one small `execute_code` per block |
| `scripts/make_bootstrap.py` | finds a repo's colour tokens and emits a paste-ready block (detail below) |
| `scripts/check_bootstrap.mjs` | runs the generated bootstrap three times against a fake library to prove it is idempotent |
| `scripts/check_boards.py` | checks exported PNGs: reads size from the header, finds single-flat-colour (blank) exports through zlib IDAT decoding, finds truncated files |
| `scripts/check_citations.py` | both orphan directions: PNGs that no markdown cites, and citations with no file. A `.png` inside a code fence is not a citation. |
| `scripts/check_no_rebuild.py` | scans Claude Code JSONL transcripts for Penpot helpers retyped instead of pasting `builder.js`; exits 0 only when none |
| `scripts/checks.test.py`, `assets/builder.test.mjs` | test suites (`python3 scripts/checks.test.py`, `node assets/builder.test.mjs`) |
| `scripts/decode.sh` | POSIX sh: a saved `execute_code` result `{result: {stem: b64}}` becomes `DIR/stem.png` |
| `scripts/pack_svgs.sh` | POSIX sh: SVG folders become `storage.SVG[stem] = "<svg…>"`; `--brands` also sets `storage.BRANDS` and `storage.BRAND_HEX` from `colours.txt` |
| `assets/builder.js` | `storage.pd` primitives (detail in §7.4) |
| `assets/screens.js` | `storage.R` declarative screen renderer (detail in §7.5) |
| `assets/tokens_board.js` | `storage.TB` tokens board, drawn in stages (detail in §7.5) |
| `references/api-traps.md` | the dated trap table summarised in §6.2 |
| `references/ui-defaults.md` | design defaults and snippets (detail below) |
| `references/house-conventions.md` | the six repo questions, how to find the answers, and to write them back into the repo |

`scripts/make_bootstrap.py <repo> [--source F] [--prefix P/] [--list] [--json] [--block NAME] [--alpha trailing]`:

- It reads Swift `static let x = token|tone|hex|color("#…")`, Kotlin `Color(0xAARRGGBB)`, and markdown `| Token | Value |` tables.
- It emits one `execute_code` body that sets `storage.T` (`{c, o}` pairs), sets `storage.F` (candidate fonts), and syncs library colours idempotently.
- Alpha is read leading by default (`#AARRGGBB`).
- When a Kotlin file holds more than one palette, it refuses to choose and asks for `--block`.

`references/ui-defaults.md` holds:

- spacing on base 8 with a half step of 4;
- type scale 12, 14, 16, 20, 24, 32, 40+, with 12 as the floor;
- WCAG 2.2 contrast (4.5:1 and 3:1), composited with opacity;
- touch targets: 44 pt (Apple), 48 dp (Material), 24 px (WCAG AA);
- frame presets: 393×852, 412×915, 375×667, 834×1194, 1440×1024;
- snippets: board-size census, next free row or column, stray shapes, text below the size floor, contrast for every token pair.

### 7.3 `penpot_mcp.py` (direct client)

**URL lookup order:**

1. `$PENPOT_MCP_URL`;
2. `mcpServers.penpot` in `$CLAUDE_CONFIG_DIR/.claude.json`, then `~/.claude.json`, then `~/.claude*/.claude.json`, checking the current project scope first, then user scope, then any project.

It never prints the URL.

**Protocol:**

- POST JSON-RPC with `Accept: application/json, text/event-stream` and `User-Agent: claude-code-penpot/1.0 (mcp)`.
- It parses either plain JSON or SSE `data:` lines.
- `initialize` uses protocol version `2025-06-18`, then sends `notifications/initialized`.
- The session id is cached in `$TMPDIR/penpot-mcp-session-<uid>`. On HTTP 400 or 404 it re-initialises once.
- Timeout is 300 s.

**Commands:**

| command | does |
| --- | --- |
| `where` | which config file the URL came from |
| `tools` | `tools/list` with names, descriptions and input properties |
| `overview [FILE]` | `high_level_overview`, optionally to a file |
| `api TYPE [MEMBER]` | `penpot_api_info` |
| `code FILE.js` or `code -` | `execute_code` with the file or stdin |
| `js '…'` | `execute_code` inline |
| `file` | `{file, page, pages, version}` |
| `boards` | top-level boards `{name, id, x, y, w, h}` |
| `export NAME/ID OUT.png` | resolves the board by id or name, waits for layout, calls `export_shape` (PNG), decodes the `image` content to disk; retries 3 times with a 4, 8, 12 s backoff |
| `export-all DIR [PREFIX]` | every top-level board (optionally only those whose name starts with PREFIX) to `DIR/<name>.png` |
| `upload NAME IMG [VAR]` | base64-inlines the file, calls `penpot.uploadMediaData`, keeps the `ImageData` in `storage[VAR]` (default `storage.img_<NAME>`) |

**Exit codes:** 0 ok, 1 tool error, 2 usage error or no server, 3 plugin not connected ("no penpot instance connected" or "heartbeat" in the message).

### 7.4 `assets/builder.js` (`storage.pd`)

Paste it whole in one call, after the bootstrap and again after every wake. Every colour option takes a token `{c, o}`, `"#RRGGBB"`, or `null`. An opacity passed by the caller wins over the token's own.

| helper | does |
| --- | --- |
| `pd.fonts(name)` | exact-name font lookup with a `findByName` fallback, returns `{font, weights, variant(w, style)}` |
| `pd.fonts.probe(names)` | which of the given faces exist |
| `pd.text(parent, x, y, chars, o)` | `o` = `{font, weight, size, lineHeight, tracking, color, opacity, width, height, grow, align, caps, name}`. Uses `applyToText`; sets `growType` after `resize`; `" "` for empty text; throws if the font is missing. |
| `pd.rect(parent, x, y, w, h, o)` | rectangle; `o` = `{fill, opacity, stroke, strokeWidth, strokeAlign, radius, name}` |
| `pd.box(...)` | board; `clipContent` defaults to `false` unless `o.clip` |
| `pd.ellipse(...)` | ellipse |
| `pd.card(...)` | rect with radius 6 and pressed-state fills |
| `pd.glass(...)` | clipping board with a blurred backdrop copy (absolute coordinates), a tint and an edge |
| `pd.glass.refit(surf, w, h, ground)` | puts glass layers back after a move or resize |
| `pd.rule(parent, x, y, len, o)` | hairline |
| `pd.glyph(parent, x, y, size, innerSvg, o)` | 24-grid line icon through `createShapeFromSvg`, recoloured by walking its children |
| `pd.typo(spec)` | idempotent library typography by `path` and `name`, string metrics |
| `pd.typo.onto(shape, name, path)` | binds a text to a typography (not named `apply`, which is read-only) |
| `pd.reflow(shapeOrArray, ms)` | `waitForLayoutUpdate`, with the zoom-and-sleep fallback, then polls texts until `width > 1` |
| `pd.audit(root)` | `{colours, opacities, radii, borderWidths, fontSizes, fonts, outside, mixedText, mixedFills}` |
| `pd.cloneScreen(src, {name, x, y, content, keep, set})` | clones a board, deletes non-kept children, sets text |

Positioning always appends first, then calls `penpotUtils.setParentXY`.

### 7.5 `assets/screens.js` (`storage.R`) and `assets/tokens_board.js` (`storage.TB`)

**What screens.js needs first:** `storage.THEMES = {light, dark}` (tokens with underscored names, such as `bg_grouped`, `label_primary`, `accent`, `bar_bg`, `keypad_bg`, `chrome_bar`), `storage.SVG`, `storage.BRANDS`, `storage.BRAND_HEX`, and `storage.APP = {name, icon, tabs, sidebar, colours, picker, …}`.

**What `storage.R` offers:**

- Theme and boards: `R.theme(name)`; `R.board(name, x, y, w, h, fill)`, which removes the old top-level board of that name first; `R.find`.
- Frames: `R.phone`, `R.phoneEnd`, `R.desk`, `R.modal`.
- Chrome: `R.statusbar`, `R.tabbar`, `R.home`, `R.browser`, `R.sidebar`.
- Marks: `R.icon`, `R.brand` (on a white tile), `R.glyph`, `R.badge`, `R.appIcon`, `R.keycap`.
- Touch targets: `R.hit` draws a named 44×44 "hit area" rect.
- List rows: `R.row`.
- Running blocks: `R.region`, `R.run(name, key, blocks)`.
- Block types (`R.blocks`): section, headline, summary, hero, button, segmented, search, chips, keypad, pin, bars, message, colours, icongrid, banner, toast, skeleton, empty, google.
- Text: `tx()` replaces en and em dashes with a hyphen.

**`storage.TB` (tokens board):**

- Stages: `TB.start()`, `TB.colours("light"|"dark", half?)`, `TB.type()`, `TB.sizes()`, `TB.note(text)`.
- It reads `storage.THEME_RAW {light, dark, assumed}`, the library typographies, and the token sets.
- Each swatch shows its contrast ratio against its theme's surface.

**`draw_screens.py SPEC.json [all|moves|NAME…] [--theme light|dark] [--suffix -dark] [--page NAME] [--setup FILE.js…] [--export DIR]`:**

- Spec: `{layout: {col, phoneY, phoneGap}, screens: [{g, j, name, kind: "desk"|"phone", spec, blocks|main|list|pane|modal…}], moves}`.
- `g` is the column; `j` is the phone slot (null means the desktop board).
- At most 14 icons go into each icongrid call.

### 7.6 Workflow rules in SKILL.md

1. **Connect:**
   - Add the server at user scope. Use the `mcp__penpot__*` tools if they are loaded, otherwise `$P`. Write each call's JS to a scratch file and run `$P code file.js`.
   - Read the overview once per session.
   - Check `penpot.currentFile.id` against the id the repo records before drawing.
   - Failure table:
     - heartbeat: retry once, then ask the operator to focus the tab;
     - not connected: open the file in a Claude in Chrome tab and click MCP, then **Connect here**;
     - 403: send a User-Agent;
     - export timeout: retry.
   - After a wake, re-paste the bootstrap, the builder and the helpers.
2. **Projects and files:**
   - Create them on the dashboard through Claude in Chrome (**+ NEW PROJECT**, the empty file tile), or ask the operator.
   - Use real names (no "New File 1"). Rename files from the dashboard tile's context menu, because `File.name` is read-only. Pages can be renamed in code.
   - Record `Design file: Penpot, project <P>, file <F>` in the repo's `CLAUDE.md`.
   - Do not open a second Penpot file while another session is drawing.
3. **Read the repository first:**
   - house conventions;
   - the plan for what to draw;
   - existing exports (sample their colours with `magick`);
   - colours through `make_bootstrap.py`;
   - fonts through `pd.fonts.probe`. SF Pro is not available on design.penpot.app; use Inter, and JetBrains Mono for mono.
   - Images through `$P upload`, taken from the repo's own assets.
4. **Draw:**
   - Make a Tokens page first: colours in light and dark, type, spacing, sizes, radii. Each exists as a board, as Penpot tokens (sets `Light`/`Dark` under themes `Mode/Light` and `Mode/Dark`), and as library colours and typographies.
   - Then draw the first screen only. Stop for operator approval with the PNGs. After approval, draw the rest.
   - Always two themes. Nothing hard-coded except white brand tiles.
   - Checks: contrast; 44 pt targets; quiet secondary actions; red only for warnings; right-aligned numbers; one name per thing; states as screens; desktop is not a stretched phone.
   - Layout: desktop boards across at `y = 0`, phone boards below each at `y = desktop height + 100`.
   - Use real icon sets (Phosphor or Lucide SVG) and real brand marks (Simple Icons first, then Wikimedia, then press kits). Use the official Google sign-in PNG, because `createShapeFromSvg` drops its gradients. No SF Symbols outside Apple platforms.
   - One script per board that removes its old board first. A few shapes per call. Reflow before reading. Run `pd.audit`.
   - No en or em dashes in any text. Name shapes semantically.
5. **Check:** export and look at the PNG; `audit.outside` must be empty; `check_no_rebuild.py` must report zero helpers rebuilt.
6. **Export:** `$P export-all <repo>/<design folder> screens--`, then `file *.png`, `check_boards.py` and `check_citations.py`. With the MCP tools loaded instead, export in `execute_code` (`shape.export`, then base64, 4 to 6 boards per call) and decode with `decode.sh`.
7. **Cite:**
   - Keep a design index (`design/README.md`) with project, file, page, naming, layout, and a table of file, what it shows, and plan.
   - Plans cite boards in prose, never as ticked checkboxes.
   - Amend plans when a design changes them.

### 7.7 Browser automation (outside the MCP)

The skill uses **Claude in Chrome** (`mcp__claude-in-chrome__*`) for everything the Plugin API cannot do: create projects and files, rename files, wake or reconnect the plugin.

Claude in Chrome sees only its own tab group, not the operator's existing Penpot tab. A new tab in the group must start the MCP plugin itself, and then it takes over the connection.

---

## 8. Implications for a direct-backend server (notes, not decisions)

What the current design gives that a backend-API server would need to replace or drop:

- **Covered only by the plugin today:**
  - computed geometry (layout engine, text measuring);
  - `generateStyle` and `generateMarkup`;
  - fonts in the browser;
  - component instance logic;
  - flex and grid reflow.
  A backend server writes the file's change log (`update-file` changes) and must compute or omit these itself. **[uncertain: how much layout the backend validates or computes server-side]**
- **Covered by neither, but available in the backend:**
  - team, project and file CRUD;
  - file rename;
  - listing files and pages without a tab;
  - comments across files.
  These are the gaps the skill fills today with Chrome clicks.
- **Export:**
  - `shape.export` already goes to the backend export service (`/api/export`), so a backend server can call that service directly.
  - The same `ResourceRequest timed out` behaviour should be expected.
  - SVG through `generateMarkup` is plugin-only.
- **Removed failure modes:** tab sleep, heartbeat, `storage` loss, the wrong file being connected, the one-tab rule, the 120 s plugin timeout, and the 15 MB response cap.
- **Auth:** the hosted MCP uses a per-user MCP key (a JWE) in `userToken`. Penpot's backend API uses access tokens (`Authorization: Token <token>`) **[uncertain: confirm against the backend RPC docs]**.
- **Keep the tool shape that worked:** a small number of tools plus a docs tool (`penpot_api_info`-like) and an overview, a disk-writing export, and image upload from a path. The skill's helpers (`file`, `boards`, `export-all`, `upload`) show what agents actually call most.
