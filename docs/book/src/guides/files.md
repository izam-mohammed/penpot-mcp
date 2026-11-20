# files and projects

finding what to work on, and keeping the dashboard tidy. everything here
runs on the [API route](../routes.md), so the token is all you need.

## find your way around

| you want | tool | it takes |
| --- | --- | --- |
| who the token is | `whoami` | nothing |
| your teams, and who is in one | `list_teams`, `list_members` | `team_id` for members |
| projects | `list_projects` | `team_id`, optional; all teams without it |
| files in a project | `list_files` | `project_id` |
| a file by name | `search_files` | `query`, optional `team_id` |
| pages of a file | `list_pages` | `file_id` |
| what is on a page | `get_page_tree` | `file_id`, optional `page_id` and `depth` |

`get_page_tree` is a flat list, top-most shape first, parents before their
children. each shape has `parent_id`, `depth`, `type`, its box and, for
text, the text. `more_children` counts what the depth cut off; ask for a
deeper `depth` rather than guessing. boards show as type `frame`.

two files with the same name? the agent should ask which one.

## make and change things

| you want | tools |
| --- | --- |
| projects | `create_project`, `rename_project`, `move_project`, `pin_project`, `delete_project` |
| files | `create_file`, `rename_file`, `duplicate_file`, `move_files`, `delete_file`, `restore_file` |
| pages | `add_page`, `rename_page`, `reorder_pages`, `delete_page` |
| a note kept in the design | `plugin_data` |

every delete answers `needs confirm: true; ask the user first`. the agent
asks you, and only then calls again with `confirm: true`. a deleted file
comes back with `restore_file` until Penpot purges it.

## an example

ask:

> make a project called "mobile app" in my default team, with a file
> called "login" in it.

the agent calls `create_project`; `team_id` is optional and the account's
default team is used when it is left out:

```json
{"name": "mobile app"}
```

and then, with the project id it gets back:

```json
{"project_id": "<project id>", "name": "login"}
```

for `create_file`. then `list_files` on the project shows the new file.

the full parameter tables are in the reference:
[files, projects and pages](../reference/tools/files.md).
