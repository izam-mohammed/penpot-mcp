# drawing whole screens

prefer `draw`: one call, one write, a whole screen. the single-shape tools
(`create_board`, `create_rect`, `create_ellipse`, `create_text`,
`create_path`, `create_image`) are for one-off additions. `draw` runs on the
API; a connected tab only adds real text sizes. see [routes](../routes.md).

## before you draw

find the file and page with `list_projects`, `list_files` and `list_pages`,
then read the page with `get_page_tree`. put new boards beside the old ones:
find the right-most `x + width` and add 100.

redrawing something that already exists? save a version first with
`create_snapshot`, so you can go back
([collaboration](collaboration.md)).

## how `draw` thinks

`shapes` goes bottom to top: each later shape sits above the earlier ones.
a node names itself with `ref`; later nodes set `parent` to that ref, or to
the id of a board, group or bool already on the page. a board with
`layout: flex` or `grid` places its children in list order, with `gap`,
`padding`, `direction`, `justify`, `align_items` or `columns`.

```json
{"file_id": "<file id>", "shapes": [
  {"ref": "screen", "type": "board", "name": "Login", "x": 0, "y": 0, "width": 390, "height": 844,
   "layout": "flex", "direction": "column", "gap": 16, "padding": 24},
  {"parent": "screen", "type": "text", "text": "Welcome back", "font_family": "Inter", "font_size": 28, "font_weight": 700},
  {"parent": "screen", "type": "rect", "name": "Email", "width": 342, "height": 48, "radius": 8, "fill": "#FFFFFF", "stroke": "#D0D5DD"},
  {"parent": "screen", "type": "rect", "name": "Sign in", "width": 342, "height": 48, "radius": 8, "fill": "#2563EB"}
]}
```

ran for real on 2026-11-20 against a new file: the reply listed every shape
with its id, parent and box (the board at 390 x 640, the title text at
(24, 24) measured at 204 x 34, the three rects stacked 16 px apart from y =
74), and `get_page_tree` showed the same ids and parents. no tab was
connected, so the reply's `tab` note said it had not reflowed; the layout
had still been placed by the server.

the rules worth knowing:

- coordinates are absolute page pixels, also inside a board. a rect 20 px
  into a board at (500, 0) sits at x = 520. in a layout board the server
  places the children and ignores their x and y.
- only boards, groups and bools can hold shapes.
- colours are `#RRGGBB`. paths with no fill get a black stroke.
- text is sized with the font's real metrics. `text_measured: false` in the
  reply means the font could not be fetched and the size is an estimate.
- a call takes up to 500 shapes.
- write calls go one after another, never in parallel.

## edit

| you want | tool |
| --- | --- |
| rename, move, resize, recolour, change text | `update_shape` |
| move shapes by an offset, children along | `move_shape` |
| move shapes into another board, or reorder | `reparent_shape` |
| group, or combine into a boolean shape | `create_group`, `create_bool` |
| delete shapes and what is inside them | `delete_shapes` |
| flex or grid on an existing board | `set_layout` |

take ids from `get_page_tree` or from the reply of the call that drew them.
`apply_changes` sends raw changes and is the last resort; never use it to
move shapes between parents, because Penpot ignores that over JSON.

## check what landed

`get_page_tree` must show each new shape with the `parent_id` you meant.
to see it, open the file in Penpot (API edits show live) or
[export](export.md) the board to a PNG.

with a tab connected, `reflow` has Penpot lay out flex and grid boards
now, and `generate_markup` and `generate_style` give HTML, SVG and CSS from
shapes. see the reference: [drawing and editing](../reference/tools/drawing.md).
