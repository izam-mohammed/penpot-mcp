# library and tokens

colours, type styles, components and design tokens, on the [API
route](../routes.md). start with `list_library` on the file: it returns
every colour, type style and component with its id and group path.

## colours and type styles

| you want | tools |
| --- | --- |
| add | `add_color`, `add_typography` |
| change one, by its id from `list_library` | `update_color`, `update_typography` |
| delete (needs `confirm: true`) | `delete_color`, `delete_typography` |

a `/` in a name groups it: `Brand / Primary` lands in the Brand group.
shapes that use a changed colour keep their own copy until refreshed in
Penpot.

```json
{"file_id": "<file id>", "name": "Brand / Primary", "hex": "#2563EB"}
```

## your repository's palette

`import_palette` reads a palette file inside the working directory and adds
every colour in one write. it understands CSS custom properties, `.swift`,
`.kt`, a markdown token table, JSON design tokens and a plain hex list.

```json
{"file_id": "<file id>", "path": "src/theme.css", "group": "Brand"}
```

for 8-digit hex, `alpha` says where the alpha sits: `leading`
(`#AARRGGBB`, the default) or `trailing` (`#RRGGBBAA`, CSS).

## design tokens

`sync_tokens` replaces the file's tokens with DTCG tokens, given inline as
`tokens` or as a `.json` `path` in the working directory. it replaces every
set, so read what the file has first.

`apply_token` links shapes' fill, stroke colour, radius or opacity to a
token by its dotted name, such as `color.brand.primary`. it is a hybrid
tool: it works without a tab.

## components

draw the board first ([drawing](drawing.md)), then:

| you want | tool |
| --- | --- |
| make a board a component | `create_component` with `shape_id` |
| place a linked copy | `instantiate_component` with `component_id`, `x`, `y` |
| delete one (needs `confirm: true`) | `delete_component` |
| publish a file as a shared library, and use it | `set_file_shared`, `link_library`, `unlink_library` |

with a tab connected, `detach`, `reset_overrides`, `swap_component`,
`switch_variant` and `combine_as_variants` work on copies.

the full tables: [library, tokens and components](../reference/tools/library.md).
