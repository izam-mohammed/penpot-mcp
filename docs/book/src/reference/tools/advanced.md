<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->

# Last resorts and documentation

For what no other tool does, and the way to look up Penpot's own API.

## `api_info`

Route: API: needs only the access token.

Penpot's documentation: a backend method's parameters and reply (from Penpot's OpenAPI document), or a Plugin API type or member (from the official MCP server, needs PENPOT_MCP_URL).

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `type` | string | yes | a Plugin API type, such as Shape, Board or Penpot; or a backend method, such as create-project |
| `member` | string | no | one member of that type, or one parameter or reply field of that method |
| `source` | string | no | plugin or backend; by default a kebab-case name, or any name without a tab, is a backend method |

## `apply_changes`

Route: API: needs only the access token.

Last resort: send raw update-file changes when no other tool fits. Use the shape tools first. Only known change types are accepted; mov-objects is refused (use reparent_shape).

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `changes` | array of object | yes | raw update-file changes in Penpot's JSON form (camelCase keys); see docs/penpot-api-reference.md section 6 |
| `file_id` | string | yes | the file |
| `page_id` | string | no | the page; the first page when empty |

## `execute_code`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Last resort: run Plugin API JavaScript in the connected Penpot tab, for what no other tool does. Refused unless file_id is the file open in the tab. Prefer the API tools: they need no tab and are checked.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `code` | string | yes | a Plugin API function body: await allowed, return gives the result |
| `file_id` | string | yes | the file the code is for; the call is refused unless this file is open in the connected tab |

