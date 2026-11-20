<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->

# Files, projects and pages

Finding what to work on, and managing teams, projects, files and pages. Deletes ask for `confirm: true` first.

## `add_page`

Route: API: needs only the access token.

Add an empty page at the end of a file. Returns its id and the file's pages in order.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `name` | string | yes | page name |

## `create_file`

Route: API: needs only the access token.

Create a file in a project. It starts with one page; returns the file's id and its pages.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | file name |
| `project_id` | string | yes | the project to create the file in |

## `create_project`

Route: API: needs only the access token.

Create a project in a team. Returns the new project's id, made by Penpot.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | project name |
| `team_id` | string | no | the team; the account's default team when empty |

## `delete_file`

Route: API: needs only the access token.

Delete a file. Needs confirm: true, given only after the user asked. Penpot keeps deleted files for a while; restore_file brings it back.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to delete |
| `confirm` | boolean | no | must be true; say so only after the user asked for this file to be deleted |

## `delete_page`

Route: API: needs only the access token.

Delete a page and everything on it. Needs confirm: true, given only after the user asked. A file's last page is never deleted.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `page_id` | string | yes | the page to delete, with everything on it |
| `confirm` | boolean | no | must be true; say so only after the user asked for this page to be deleted |

## `delete_project`

Route: API: needs only the access token.

Delete a project and every file in it. Needs confirm: true, given only after the user asked. Penpot keeps deleted files for a while; restore_file brings them back.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `project_id` | string | yes | the project |
| `confirm` | boolean | no | must be true; say so only after the user asked for this project to be deleted |

## `duplicate_file`

Route: API: needs only the access token.

Copy a file within its project. Returns the copy's id, made by Penpot.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to copy |
| `name` | string | no | the copy's name; Penpot names it when empty |

## `get_page_tree`

Route: API: needs only the access token.

A page's shapes as a flat list in drawing order (top-most first, parents before children): id, parent_id, depth, name, type, x, y, width, height, and for text shapes their text. Coordinates are absolute page pixels.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `depth` | integer | no | levels below the page root to include, 1 to 10, default 3 |
| `page_id` | string | no | the page; the first page when empty |

## `list_files`

Route: API: needs only the access token.

List the files in a Penpot project.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `project_id` | string | yes | the project to list files of |

## `list_members`

Route: API: needs only the access token.

List a team's members with their role: owner, admin, editor or viewer.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `team_id` | string | no | the team; the account's default team when empty |

## `list_pages`

Route: API: needs only the access token.

List the pages of a Penpot file, in order.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to list pages of |

## `list_projects`

Route: API: needs only the access token.

List Penpot projects, across all teams or for one team.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `team_id` | string | no | only projects of this team; all teams when empty |

## `list_teams`

Route: API: needs only the access token.

List the Penpot teams this account belongs to.

Takes no parameters.

## `move_files`

Route: API: needs only the access token.

Move files into another project of the same team.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_ids` | array of string | yes | the files to move |
| `project_id` | string | yes | the project to move them into |

## `move_project`

Route: API: needs only the access token.

Move a project, with its files, to another team.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `project_id` | string | yes | the project |
| `team_id` | string | yes | the team to move the project to |

## `pin_project`

Route: API: needs only the access token.

Pin or unpin a project on the dashboard.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `pinned` | boolean | yes | true pins the project on the dashboard, false unpins it |
| `project_id` | string | yes | the project |

## `plugin_data`

Route: API: needs only the access token.

Read or write a small key-value note on a file or a shape, under a namespace, for tools that need to remember things in the design itself.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `key` | string | yes | the key |
| `namespace` | string | yes | your namespace, such as a plugin or tool name |
| `page_id` | string | no | the shape's page; the first page when empty |
| `shape_id` | string | no | the shape; the file itself when empty |
| `value` | string | no | set it to this; omit to read it, empty string to remove it |

## `rename_file`

Route: API: needs only the access token.

Rename a file.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to rename |
| `name` | string | yes | new file name |

## `rename_page`

Route: API: needs only the access token.

Rename a page.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `name` | string | yes | new page name |
| `page_id` | string | yes | the page to rename |

## `rename_project`

Route: API: needs only the access token.

Rename a project.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | new project name |
| `project_id` | string | yes | the project |

## `reorder_pages`

Route: API: needs only the access token.

Move a page to another position in the file's page list.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `index` | integer | yes | its new position, 0 is first |
| `page_id` | string | yes | the page to move |

## `restore_file`

Route: API: needs only the access token.

Bring back deleted files (and their projects) before Penpot purges them. Without file_ids, lists the team's deleted files.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_ids` | array of string | no | the deleted files to restore; when empty, lists the team's deleted files instead |
| `team_id` | string | no | the team; the account's default team when empty |

## `search_files`

Route: API: needs only the access token.

Find files in a team by name.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | yes | text to look for in file names |
| `team_id` | string | no | the team; the account's default team when empty |

## `whoami`

Route: API: needs only the access token.

The Penpot account this server acts as: id, name, email, default team and project.

Takes no parameters.

