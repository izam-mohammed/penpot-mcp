<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->

# Drawing and editing

Whole screens in one `draw` call, single shapes, edits, and the tab-only helpers for what a drawing looks like.

## `create_board`

Route: API: needs only the access token.

Add a board (frame) to a Penpot page, at the root or inside another board. Returns the new shape id and the file and page it landed in.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `height` | number | yes | height in pixels, above 0 |
| `name` | string | yes | layer name |
| `width` | number | yes | width in pixels, above 0 |
| `x` | number | yes | left edge, absolute page pixels (also inside a board) |
| `y` | number | yes | top edge, absolute page pixels |
| `fill` | string | no | fill colour as #RRGGBB; white for boards and grey for other shapes when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `radius` | number | no | corner radius in pixels (boards and rectangles) |
| `rotation` | number | no | degrees clockwise about the centre (rectangles and ellipses) |
| `stroke` | string | no | stroke colour as #RRGGBB; none when empty |
| `stroke_width` | number | no | stroke width in pixels, default 1 |

## `create_bool`

Route: API: needs only the access token.

Combine shapes that share a parent into a boolean shape: union, difference, intersection or exclude.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `name` | string | yes | layer name |
| `shape_ids` | array of string | yes | the shapes |
| `bool_type` | string | no | create_bool only: union (default), difference, intersection or exclude |
| `fill` | string | no | create_bool only: fill colour as #RRGGBB |
| `page_id` | string | no | the page; the first page when empty |

## `create_ellipse`

Route: API: needs only the access token.

Add an ellipse that fills the given box, at the root or inside a board.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `height` | number | yes | height in pixels, above 0 |
| `name` | string | yes | layer name |
| `width` | number | yes | width in pixels, above 0 |
| `x` | number | yes | left edge, absolute page pixels (also inside a board) |
| `y` | number | yes | top edge, absolute page pixels |
| `fill` | string | no | fill colour as #RRGGBB; white for boards and grey for other shapes when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `radius` | number | no | corner radius in pixels (boards and rectangles) |
| `rotation` | number | no | degrees clockwise about the centre (rectangles and ellipses) |
| `stroke` | string | no | stroke colour as #RRGGBB; none when empty |
| `stroke_width` | number | no | stroke width in pixels, default 1 |

## `create_group`

Route: API: needs only the access token.

Group shapes that share a parent. The group takes the place of the top one.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `name` | string | yes | layer name |
| `shape_ids` | array of string | yes | the shapes |
| `bool_type` | string | no | create_bool only: union (default), difference, intersection or exclude |
| `fill` | string | no | create_bool only: fill colour as #RRGGBB |
| `page_id` | string | no | the page; the first page when empty |

## `create_image`

Route: API: needs only the access token.

Upload a local image into the file and place it as an image-filled rectangle. Only PNG, JPEG, WebP, GIF and SVG files are read.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `image_path` | string | yes | absolute path of a local PNG, JPEG, WebP, GIF or SVG file, up to 20 MB |
| `x` | number | yes | left edge, absolute page pixels |
| `y` | number | yes | top edge, absolute page pixels |
| `height` | number | no |  |
| `name` | string | no | layer name; the file name when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `radius` | number | no | corner radius in pixels |
| `rotation` | number | no | degrees clockwise about the centre |
| `width` | number | no | width; with only one of width and height the other keeps the aspect ratio; the image's own size when both are empty |

## `create_path`

Route: API: needs only the access token.

Add a vector path from SVG path data (absolute page pixels). Every SVG command works; arcs and quadratics become cubic curves. Returns the path's bounding box.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `d` | string | yes | SVG path data in absolute page pixels, such as M10 10 L90 10 L50 80 Z; every SVG command works |
| `file_id` | string | yes | the file to draw in |
| `name` | string | yes | layer name |
| `fill` | string | no | fill colour as #RRGGBB; none when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `stroke` | string | no | stroke colour as #RRGGBB; #000000 when there is no fill |
| `stroke_width` | number | no | stroke width in pixels, default 1 |

## `create_rect`

Route: API: needs only the access token.

Add a rectangle to a Penpot page, at the root or inside a board. Returns the new shape id and the file and page it landed in.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `height` | number | yes | height in pixels, above 0 |
| `name` | string | yes | layer name |
| `width` | number | yes | width in pixels, above 0 |
| `x` | number | yes | left edge, absolute page pixels (also inside a board) |
| `y` | number | yes | top edge, absolute page pixels |
| `fill` | string | no | fill colour as #RRGGBB; white for boards and grey for other shapes when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `radius` | number | no | corner radius in pixels (boards and rectangles) |
| `rotation` | number | no | degrees clockwise about the centre (rectangles and ellipses) |
| `stroke` | string | no | stroke colour as #RRGGBB; none when empty |
| `stroke_width` | number | no | stroke width in pixels, default 1 |

## `create_text`

Route: hybrid: runs on the API and adds a step in the tab when one is connected.

Add a text box. Without width the box fits the text; with width the text wraps and the height grows. Sizes are measured with the real font. Returns the box size.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `align` | string | no | left, center, right or justify |
| `color` | string | no | text colour as #RRGGBB, default #000000 |
| `font_family` | string | no | a Google font name like Inter, or Source Sans Pro (the default) |
| `font_size` | number | no | size in pixels, 3 to 1000, default 16 |
| `font_weight` | integer | no | 100 to 900 in steps of 100, default 400 |
| `italic` | boolean | no |  |
| `letter_spacing` | number | no | pixels between letters |
| `line_height` | number | no | multiple of the font size, default 1.2 |
| `name` | string | no | layer name; the text itself when empty |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |
| `text` | string | no | the text (required for text shapes); a newline starts a new paragraph |
| `width` | number | no | box width; when empty the box fits the text on one line per paragraph |
| `x` | number | no | left edge, absolute page pixels (not for path, group or bool) |
| `y` | number | no | top edge, absolute page pixels |

## `current_file`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Which file and page the connected Penpot tab has open.

Takes no parameters.

## `delete_shapes`

Route: API: needs only the access token.

Delete shapes and everything inside them. Returns the names removed. The root frame is never deleted.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `shape_ids` | array of string | yes | the shapes |
| `page_id` | string | no | the page; the first page when empty |

## `draw`

Route: hybrid: runs on the API and adds a step in the tab when one is connected.

Draw many shapes in one write: a whole screen per call. Types: board, rect, ellipse, text, path (SVG d), image (local file), group and bool. Shapes are a flat list, bottom to top; nest by giving a node the ref of an earlier board, group or bool as parent. A board with layout places its children itself, first listed first. Coordinates are absolute page pixels. Returns every shape's id and box.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file to draw in |
| `shapes` | array of object | yes | shapes bottom to top: each later shape is drawn above earlier ones |
| `page_id` | string | no | the page; the first page when empty |
| `parent_id` | string | no | a board to draw inside; the page root when empty |

## `generate_font_faces`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

@font-face CSS for the fonts the shapes use.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |

## `generate_markup`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

HTML or SVG markup for shapes, generated by Penpot in the connected tab.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |
| `type` | string | no | html (default) or svg |

## `generate_style`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

CSS for shapes, generated by Penpot in the connected tab.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `shape_ids` | array of string | yes | the shapes, on the tab's current page |
| `with_children` | boolean | no | include the children's CSS |

## `get_selection`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

The shapes selected in the connected tab: id, name and type.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |

## `import_svg`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Turn SVG markup into editable Penpot shapes with Penpot's own SVG parser, in the connected tab. For a single path without a tab, create_path does it through the API.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `x` | number | yes | left edge, absolute page pixels |
| `y` | number | yes | top edge, absolute page pixels |
| `name` | string | no | layer name |
| `path` | string | no | an .svg file inside the working directory, instead of svg |
| `svg` | string | no | SVG markup |

## `move_shape`

Route: API: needs only the access token.

Move shapes by dx, dy pixels; everything inside them moves too, and groups around them refit.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `dx` | number | yes | pixels to move right (negative: left) |
| `dy` | number | yes | pixels to move down (negative: up) |
| `file_id` | string | yes | the file |
| `shape_ids` | array of string | yes | the shapes |
| `page_id` | string | no | the page; the first page when empty |

## `reflow`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Let the connected tab lay out flex and grid boards and size text now, after API writes. Without a tab, Penpot does this when the file is next opened.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |

## `reparent_shape`

Route: API: needs only the access token.

Move shapes into another board, group or bool, or reorder them in their own parent with index. They keep their page position.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `parent_id` | string | yes | the board, group or bool to move into; 00000000-0000-0000-0000-000000000000 for the page root |
| `shape_ids` | array of string | yes | the shapes |
| `index` | integer | no | position among the parent's children, 0 is the bottom; the top when empty |
| `page_id` | string | no | the page; the first page when empty |

## `set_layout`

Route: hybrid: runs on the API and adds a step in the tab when one is connected.

Give a board a flex (row or column) or grid layout and place its children. Children go in layer order, top layer first, as Penpot lays them out. Child sizes are kept.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `board_id` | string | yes | the board to lay out |
| `file_id` | string | yes |  |
| `align_items` | string | no | flex cross axis: start (default), center or end |
| `columns` | integer | no | grid: number of equal columns, default 2 |
| `direction` | string | no | flex: row (default) or column |
| `gap` | number | no | pixels between children |
| `justify` | string | no | flex main axis: start (default), center, end, space-between, space-around or space-evenly |
| `layout` | string | no | boards only: flex or grid; children are placed in list order |
| `padding` | number | no | pixels inside the board edge, all four sides |
| `page_id` | string | no | the page; the first page when empty |

## `update_shape`

Route: API: needs only the access token.

Change one shape: name, position (children follow), size, rotation, fill, stroke, radius, opacity, visibility, or a text shape's text. Only the fields given change.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file |
| `shape_id` | string | yes | the shape to change |
| `fill` | string | no | fill colour as #RRGGBB, or none |
| `height` | number | no | new height (boards, rects, ellipses, images) |
| `hidden` | boolean | no | hide or show the layer |
| `name` | string | no | new layer name |
| `opacity` | number | no | 0 to 1 |
| `page_id` | string | no | the page; the first page when empty |
| `radius` | number | no | corner radius (boards, rects, images) |
| `rotation` | number | no | degrees clockwise about the centre (rects, ellipses, images, text) |
| `stroke` | string | no | stroke colour as #RRGGBB, or none |
| `stroke_width` | number | no | stroke width in pixels |
| `text` | string | no | text shapes: new text, keeping the first run's style; the box is measured again |
| `width` | number | no | new width (boards, rects, ellipses, images, text); text then wraps at it |
| `x` | number | no | new left edge, absolute page pixels; children move with it |
| `y` | number | no | new top edge; children move with it |

## `viewport`

Route: tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.

Read the tab's viewport, or move it: zoom, centre, or fit shapes into view.

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_id` | string | yes | the file open in the connected tab |
| `center_x` | number | no | page x to centre on |
| `center_y` | number | no | page y to centre on |
| `shape_ids` | array of string | no | zoom to fit these shapes instead |
| `zoom` | number | no | new zoom, 0.01 to 200; unchanged when empty |

