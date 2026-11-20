# Troubleshooting

Errors the server returns, what each one means, and what to do. If
`penpot` is listed in `/mcp` but has no tools, the server failed to
start: run `penpot-mcp </dev/null` in your own shell, with the token set,
to see the error.

## Setup

| Error | Meaning | Do |
| --- | --- | --- |
| `PENPOT_ACCESS_TOKEN is not set (or PENPOT_PROFILES for several accounts)` | The server started with no token. | Add `-e PENPOT_ACCESS_TOKEN=<your token>` to the `claude mcp add` line. |
| `PENPOT_ACCESS_TOKEN was not accepted` | The token is wrong or expired. | Make a new one in Penpot and update the MCP config. On a self-hosted Penpot, check `PENPOT_BASE_URL` and the `enable-access-tokens` flag. |
| `--http wants host:port, such as 127.0.0.1:8787` | The `--http` value has no port. | Give `host:port`. |
| `--http on a non-loopback address needs PENPOT_MCP_HTTP_TOKEN (16 characters or more)` | The server will not listen off loopback without a bearer token. | Set `PENPOT_MCP_HTTP_TOKEN`, or bind `127.0.0.1`. |
| `PENPOT_PROFILES is not a JSON list of profiles` | The profiles file is not a JSON list. | See [several accounts](configuration.md#several-accounts). |
| `profile "x": PENPOT_X_TOKEN is not set` | A profile names a variable that is empty. | Set that variable in the server's environment. |
| `PENPOT_MCP_URL is not a valid URL` | The MCP URL is malformed. | Copy the whole URL again, key included. |

## Rate limits and conflicts

| Error | Meaning | Do |
| --- | --- | --- |
| `this token cannot edit file` | The account can only view this file. | Pick a file the account can edit. |
| `HTTP 429 rate-limit` | Penpot's rate limit, already retried once. | Wait the seconds it says, then go on. Do not write in parallel. |
| `HTTP 409 validation/vern-conflict` | Someone restored a version in the browser. The server re-read once. | Read the page again with `get_page_tree`, then redo the change. |
| `HTTP 500 .../data-validation` | Penpot rejected a shape. | Read the `explain` text in the reply. Do not retry blindly. |

## Shapes and pages

| Error | Meaning | Do |
| --- | --- | --- |
| `parent ... is a rect` | The parent is not a board, group or bool. | Use the id of a board. |
| `the shapes must share a parent` | A group or bool across different parents. | Move them together first with `reparent_shape`. |
| `the root frame cannot be edited, moved or deleted` | The page's root id was given. | Use ids of shapes on the page. |
| `... is ignored by Penpot over JSON` | `mov-objects` was sent to `apply_changes`. | Use `reparent_shape`. |
| `a shape cannot go inside itself` | A reparent loop. | Pick another parent. |
| `page ... is not in file ...` | The page id is wrong or from another file. | Read the ids with `list_pages`. |
| `"..." is the last page of "..."; Penpot needs one` | The file would have no page. | Add another page first. |
| `nothing to change: give at least one field` | `update_shape` got only an id. | Say what to change. |

## Confirmations

Deletes, restores, share links and webhooks answer
`... needs confirm: true; ask the user first`. This is on purpose: ask
the person, then call again with `confirm: true`.

## Exports

| Error | Meaning | Do |
| --- | --- | --- |
| `exports need PENPOT_EMAIL and PENPOT_PASSWORD ... or PENPOT_MCP_URL` | Board exports need a login session or a tab. | See [routes](routes.md#login-or-tab). |
| a path outside the working directory is refused | Files are read and written only under the directory the server started in. | Use a path inside it, or start the server there. |

## The tab

| Error | Meaning | Do |
| --- | --- | --- |
| `no Penpot tab is connected` | The tab is closed, still loading, or asleep. | Open the file in Penpot, open the MCP plugin and click *Connect*. See [routes](routes.md#tab). |
| `the Penpot tab has "X" open, not "Y"` | The file guard: another file is open in the tab. | Open Y in the tab and reconnect, or work on X. |
| `the code may have run anyway` | A tab call timed out after it was sent. | Read the file back first. Never repeat it blind. |
| `this needs the Penpot MCP route: set PENPOT_MCP_URL ...` | A tab tool was used with no `PENPOT_MCP_URL`. | Set it, see [routes](routes.md#tab). |
| `the Penpot MCP server no longer offers execute_code; this server needs an update` | The official MCP changed. | Update penpot-mcp. |

## Still stuck

Open an issue on
[GitHub](https://github.com/izam-mohammed/penpot-mcp/issues) with the error
text. Leave the token, the MCP URL and the password out.
