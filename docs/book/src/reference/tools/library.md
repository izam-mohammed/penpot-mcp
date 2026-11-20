<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->

# Library, tokens and components

Colours, type styles, components, design tokens and shared libraries.

## `add_color`

Route: API: needs only the access token.

Add a colour to a file's library. A colour with the same group and name is updated, not duplicated, so re-running is safe.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file whose library gets the colour |
| `hex` | string | yes | the colour as #RRGGBB |
| `name` | string | yes | colour name; "Brand / Primary" puts it in the Brand group |
| `opacity` | number | no | 0 to 1, default 1 |
| `path` | string | no | group path, such as Brand or Brand / Dark; taken from name when empty |

## `add_typography`

Route: API: needs only the access token.

Add a type style to a file's library. A style with the same group and name is updated, not duplicated.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file whose library gets the type style |
| `name` | string | yes | style name; "Heading / H1" puts it in the Heading group |
| `font_family` | string | no | a Google font name like Inter, or Source Sans Pro (the default) |
| `font_size` | number | no | size in pixels, 3 to 1000, default 16 |
| `font_weight` | integer | no | 100 to 900 in steps of 100, default 400 |
| `italic` | boolean | no |  |
| `letter_spacing` | number | no | pixels between letters |
| `line_height` | number | no | multiple of the font size, default 1.2 |
| `path` | string | no | group path; taken from name when empty |
| `text_transform` | string | no | none (default), uppercase, lowercase or capitalize |

## `apply_token`

Route: hybrid: runs on the API and adds a step in the tab when one is connected.

Apply a design token to shapes: their fill, stroke colour, corner radius or opacity takes the token's value and stays linked to the token.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `attribute` | string | yes | what it sets: fill, stroke_color, radius or opacity |
| `file_id` | string | yes | the file |
| `shape_ids` | array of string | yes | the shapes |
| `token` | string | yes | the token's dotted name, such as color.brand.primary |
| `page_id` | string | no | the page; the first page when empty |
| `value` | string | no | the token's value, when it cannot be read from the file: #RRGGBB for colours, a number for the rest |

## `combine_as_variants`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Combine component main boards into one component with variants.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |

## `create_component`

Route: API: needs only the access token.

Turn a board into a component: the board becomes its main instance. Draw the board first (draw or create_board), then place copies with instantiate_component.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `shape_id` | string | yes | the board that becomes the component's main instance |
| `name` | string | no | component name; the board's name when empty; "Buttons / Primary" groups it |
| `page_id` | string | no | the board's page; the first page when empty |
| `path` | string | no | group path; taken from name when empty |

## `delete_color`

Route: API: needs only the access token.

Delete a library colour. Needs confirm: true, given only after the user asked.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `id` | string | yes | the colour, type style or component, from list_library |
| `confirm` | boolean | no | must be true; say so only after the user asked for this to be deleted |

## `delete_component`

Route: API: needs only the access token.

Delete a library component. Needs confirm: true, given only after the user asked.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `id` | string | yes | the colour, type style or component, from list_library |
| `confirm` | boolean | no | must be true; say so only after the user asked for this to be deleted |

## `delete_typography`

Route: API: needs only the access token.

Delete a library type style. Needs confirm: true, given only after the user asked.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `id` | string | yes | the colour, type style or component, from list_library |
| `confirm` | boolean | no | must be true; say so only after the user asked for this to be deleted |

## `detach`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Detach component copies: they become plain shapes and stop following the main.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |

## `import_palette`

Route: API: needs only the access token.

Read colours from a palette file in the repository and add them to a file's library in one write. Re-running updates the same colours instead of adding twins.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file whose library gets the colours |
| `path` | string | yes | a palette file inside the working directory: .swift, .kt, .md token table, .css custom properties, .json design tokens, or a hex list |
| `alpha` | string | no | how 8-digit hex is read: leading (#AARRGGBB, Android and Swift, the default) or trailing (#RRGGBBAA, CSS) |
| `group` | string | no | group path to put every colour under, such as Brand |

## `instantiate_component`

Route: hybrid: runs on the API and adds a step in the tab when one is connected.

Place a copy of a component at x, y. The copy stays linked to the main instance, so later changes to the main show in every copy.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `component_id` | string | yes | the component, from list_library |
| `file_id` | string | yes | the file |
| `x` | number | yes | left edge of the copy, absolute page pixels |
| `y` | number | yes | top edge of the copy, absolute page pixels |
| `name` | string | no | layer name; the component's name when empty |
| `page_id` | string | no | the page to place the copy on; the first page when empty |
| `parent_id` | string | no | a board to place the copy inside; the page root when empty |

## `link_library`

Route: API: needs only the access token.

Link a shared library to a file, so its colours, type styles and components can be used there.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file that uses the library |
| `library_id` | string | yes | the shared library file |

## `list_library`

Route: API: needs only the access token.

A file's library: colours, type styles and components, each with its id and group path.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |

## `reset_overrides`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Reset component copies to their main: every override on them is lost.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |

## `set_file_shared`

Route: API: needs only the access token.

Publish a file as a shared library, or unpublish it.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `shared` | boolean | yes | true publishes the file as a shared library, false unpublishes it |

## `swap_component`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Swap a component copy for a copy of another component, keeping its place.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `component_id` | string | yes | the component to swap to, from list_library |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_id` | string | yes | the component copy |

## `switch_variant`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Switch a variant copy to another value of one of its properties.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `property` | string | yes | the variant property to change |
| `shape_id` | string | yes | the variant copy |
| `value` | string | yes | its new value |

## `sync_tokens`

Route: API: needs only the access token.

Replace a file's design tokens (sets, tokens, themes) with the DTCG tokens given, from the call or a .json file in the repository. Tokens not given are removed.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `path` | string | no | a DTCG .json file inside the working directory, instead of tokens |
| `set_name` | string | no | when the tokens are one set without a set level, the name to give that set |
| `tokens` | object | no | design tokens in DTCG form: {set name: {group: {token: {$value, $type}}}}, optionally with $themes and $metadata |

## `unlink_library`

Route: API: needs only the access token.

Unlink a shared library from a file.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file that uses the library |
| `library_id` | string | yes | the shared library file |

## `update_color`

Route: API: needs only the access token.

Change a library colour's name, group, hex or opacity. Shapes using it keep their own copy until refreshed in Penpot.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `color_id` | string | yes | the colour, from list_library |
| `file_id` | string | yes | the file |
| `hex` | string | no | new colour as #RRGGBB |
| `name` | string | no | new name; "Brand / Primary" also moves it into a group |
| `opacity` | number | no | 0 to 1 |
| `path` | string | no | new group path; empty string moves it out of every group |

## `update_typography`

Route: API: needs only the access token.

Change a library type style: name, font, size, weight, line height or letter spacing.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `typography_id` | string | yes | the type style, from list_library |
| `font_family` | string | no |  |
| `font_size` | number | no |  |
| `font_weight` | integer | no |  |
| `italic` | boolean | no |  |
| `letter_spacing` | number | no |  |
| `line_height` | number | no |  |
| `name` | string | no | new name; "Heading / H1" also moves it into a group |

