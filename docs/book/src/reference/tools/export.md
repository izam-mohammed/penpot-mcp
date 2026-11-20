<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->

# Export and import

Boards and shapes to image files, whole files to `.penpot` and back.

## `export_boards`

Route: login or tab: needs `PENPOT_EMAIL` and `PENPOT_PASSWORD`, or a connected tab with the shape's page open.

Export every top-level board of a page (optionally only names with a prefix) into a folder, one file per board.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `dir` | string | yes | folder inside the working directory; each board is written as &lt;board name>.&lt;ext> |
| `file_id` | string | yes | the file |
| `format` | string | no | png (default), jpeg, webp, svg or pdf |
| `overwrite` | boolean | no | replace files already there |
| `page_id` | string | no | the page; the first page when empty |
| `prefix` | string | no | only boards whose name starts with this |
| `scale` | number | no | 0.25 to 4, default 1 |

## `export_file`

Route: API: needs only the access token.

Download a whole file as .penpot (Penpot's own format, re-importable) to a path in the working directory.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `path` | string | yes | where to write the .penpot file, inside the working directory |
| `overwrite` | boolean | no | replace a file already at path |

## `export_shape`

Route: login or tab: needs `PENPOT_EMAIL` and `PENPOT_PASSWORD`, or a connected tab with the shape's page open.

Render a board or shape to PNG, JPEG, WEBP, SVG or PDF and write it to a path in the working directory. Uses Penpot's exporter when PENPOT_EMAIL and PENPOT_PASSWORD are set (it takes only a login session); otherwise the connected Penpot tab renders it (png, jpeg, svg or pdf; the shape's page must be open there). A blank render is an error.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `path` | string | yes | where to write it, inside the working directory, with the format's extension |
| `shape_id` | string | yes | the board or shape to export |
| `format` | string | no | png (default), jpeg, webp, svg or pdf |
| `overwrite` | boolean | no | replace a file already at path |
| `page_id` | string | no | the page; the first page when empty |
| `scale` | number | no | 0.25 to 4, default 1; 2 for retina PNGs |

## `import_file`

Route: API: needs only the access token.

Import a .penpot file from the working directory into a project as a new file.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | a .penpot file inside the working directory |
| `project_id` | string | yes | the project to import into |
| `name` | string | no | the new file's name; the file name when empty |

