# export

getting designs out of Penpot and into your repository. every path is
inside the directory the server was started in; anything outside is
refused.

## set it up first

boards render through Penpot's exporter, which needs a login session, not
the access token. pick one:

- `PENPOT_EMAIL` and `PENPOT_PASSWORD` on the `claude mcp add` line:
  exports run with no browser, in every format.
- no password (Google or SSO account)? set `PENPOT_MCP_URL`, connect a tab
  and keep the shape's page open there. no WEBP that way.

without either, `export_shape` answers `exports need PENPOT_EMAIL and
PENPOT_PASSWORD ... or PENPOT_MCP_URL`. details in [routes](../routes.md#login-or-tab).

## boards and shapes

| tool | does |
| --- | --- |
| `export_shape` | one board or shape (`shape_id`) to `path`, in `format` png (default), jpeg, webp, svg or pdf, at `scale` 0.25 to 4 |
| `export_boards` | every top-level board of a page into `dir`, one `<board name>.<ext>` each; `prefix` keeps only names that start with it |

neither overwrites a file unless `overwrite: true`. use `scale: 2` for
retina PNGs.

## an example: screens into the repo

1. pick the folder the project keeps designs in, say `docs/designs/`.
2. ask the agent to export the page's boards there as 2x PNGs:

   ```json
   {"file_id": "<file id>", "dir": "docs/designs", "format": "png", "scale": 2}
   ```

3. cite each image by its relative path from the plan or doc that needs
   it: `![Login](docs/designs/Login.png)`.
4. look at one of the images before trusting them.

board names become file names, so name boards for the file you want
(`Login`, not `Board 3`).

## whole files

| tool | does |
| --- | --- |
| `export_file` | the whole file as `.penpot` (Penpot's own format) to `path`; needs only the token |
| `import_file` | a `.penpot` file from the working directory into `project_id`, as a new file |

`export_file` is the backup before a risky change that a saved version does
not cover, such as deleting the file.

the full tables: [export and import](../reference/tools/export.md).
