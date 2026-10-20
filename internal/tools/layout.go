package tools

import (
	"context"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// layoutFields put a board's children in a row, a column or a grid. Penpot
// stores the attributes but only its frontend reflows, so this server places
// the children itself when it writes the layout.
type layoutFields struct {
	Layout     string  `json:"layout,omitempty" jsonschema:"boards only: flex or grid; children are placed in list order"`
	Direction  string  `json:"direction,omitempty" jsonschema:"flex: row (default) or column"`
	Gap        float64 `json:"gap,omitempty" jsonschema:"pixels between children"`
	Padding    float64 `json:"padding,omitempty" jsonschema:"pixels inside the board edge, all four sides"`
	Justify    string  `json:"justify,omitempty" jsonschema:"flex main axis: start (default), center, end, space-between, space-around or space-evenly"`
	AlignItems string  `json:"align_items,omitempty" jsonschema:"flex cross axis: start (default), center or end"`
	Columns    int     `json:"columns,omitempty" jsonschema:"grid: number of equal columns, default 2"`
}

func (l layoutFields) set() bool {
	return l != layoutFields{}
}

func (l layoutFields) validate() error {
	switch l.Layout {
	case "flex", "grid":
	default:
		return fmt.Errorf("layout must be flex or grid")
	}
	switch l.Direction {
	case "", "row", "column":
	default:
		return fmt.Errorf("direction must be row or column")
	}
	switch l.Justify {
	case "", "start", "center", "end", "space-between", "space-around", "space-evenly":
	default:
		return fmt.Errorf("justify must be start, center, end, space-between, space-around or space-evenly")
	}
	switch l.AlignItems {
	case "", "start", "center", "end":
	default:
		return fmt.Errorf("align_items must be start, center or end")
	}
	if math.IsNaN(l.Gap) || l.Gap < 0 || l.Gap > maxSize {
		return fmt.Errorf("gap must be 0 to %d", maxSize)
	}
	if math.IsNaN(l.Padding) || l.Padding < 0 || l.Padding > maxSize {
		return fmt.Errorf("padding must be 0 to %d", maxSize)
	}
	if l.Columns < 0 || l.Columns > 100 {
		return fmt.Errorf("columns must be 1 to 100")
	}
	return nil
}

// item is a child's box for placement.
type item struct {
	id         string
	x, y, w, h float64
}

// attrs are the board attributes Penpot stores for the layout.
func (l layoutFields) attrs(kids []item) map[string]any {
	a := map[string]any{
		"layout":            l.Layout,
		"layoutGap":         map[string]any{"rowGap": l.Gap, "columnGap": l.Gap},
		"layoutGapType":     "simple",
		"layoutPadding":     map[string]any{"p1": l.Padding, "p2": l.Padding, "p3": l.Padding, "p4": l.Padding},
		"layoutPaddingType": "simple",
		"layoutWrapType":    "nowrap",
	}
	if l.Layout == "flex" {
		a["layoutFlexDir"] = orDefault(l.Direction, "row")
		a["layoutJustifyContent"] = orDefault(l.Justify, "start")
		a["layoutAlignItems"] = orDefault(l.AlignItems, "start")
		a["layoutAlignContent"] = "stretch"
		return a
	}
	cols, rows := l.grid(len(kids))
	a["layoutGridDir"] = "row"
	a["layoutJustifyItems"] = "start"
	a["layoutAlignItems"] = "start"
	a["layoutJustifyContent"] = "stretch"
	a["layoutAlignContent"] = "stretch"
	a["layoutGridColumns"] = tracks(cols)
	a["layoutGridRows"] = tracks(rows)
	cells := map[string]any{}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			id := penpot.NewID()
			shapes := []string{}
			if i := r*cols + c; i < len(kids) {
				shapes = append(shapes, kids[i].id)
			}
			cells[id] = map[string]any{
				"id": id, "row": r + 1, "rowSpan": 1, "column": c + 1, "columnSpan": 1,
				"position": "auto", "shapes": shapes,
			}
		}
	}
	a["layoutGridCells"] = cells
	return a
}

func tracks(n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{"type": "flex", "value": 1}
	}
	return out
}

func (l layoutFields) grid(n int) (cols, rows int) {
	cols = l.Columns
	if cols == 0 {
		cols = 2
	}
	rows = (n + cols - 1) / cols
	if rows == 0 {
		rows = 1
	}
	return cols, rows
}

// place returns where each child goes inside board, in the order given.
func (l layoutFields) place(board item, kids []item) []item {
	out := make([]item, len(kids))
	copy(out, kids)
	if len(kids) == 0 {
		return out
	}
	inX, inY := board.x+l.Padding, board.y+l.Padding
	inW, inH := board.w-2*l.Padding, board.h-2*l.Padding

	if l.Layout == "grid" {
		cols, rows := l.grid(len(kids))
		cw := (inW - l.Gap*float64(cols-1)) / float64(cols)
		rh := (inH - l.Gap*float64(rows-1)) / float64(rows)
		for i := range out {
			r, c := i/cols, i%cols
			out[i].x = inX + float64(c)*(cw+l.Gap)
			out[i].y = inY + float64(r)*(rh+l.Gap)
		}
		return out
	}

	row := l.Direction != "column"
	main := func(it item) float64 {
		if row {
			return it.w
		}
		return it.h
	}
	cross := func(it item) float64 {
		if row {
			return it.h
		}
		return it.w
	}
	avail, crossAvail := inW, inH
	if !row {
		avail, crossAvail = inH, inW
	}
	sum := 0.0
	for _, k := range kids {
		sum += main(k)
	}
	n := float64(len(kids))
	gap, offset := l.Gap, 0.0
	free := avail - sum - gap*(n-1)
	switch l.Justify {
	case "center":
		offset = free / 2
	case "end":
		offset = free
	case "space-between":
		if n > 1 {
			gap = (avail - sum) / (n - 1)
		}
	case "space-around":
		gap = (avail - sum) / n
		offset = gap / 2
	case "space-evenly":
		gap = (avail - sum) / (n + 1)
		offset = gap
	}
	pos := offset
	for i, k := range kids {
		c := 0.0
		switch l.AlignItems {
		case "center":
			c = (crossAvail - cross(k)) / 2
		case "end":
			c = crossAvail - cross(k)
		}
		if row {
			out[i].x, out[i].y = inX+pos, inY+c
		} else {
			out[i].x, out[i].y = inX+c, inY+pos
		}
		pos += main(k) + gap
	}
	return out
}

type setLayoutIn struct {
	FileID  string `json:"file_id"`
	PageID  string `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	BoardID string `json:"board_id" jsonschema:"the board to lay out"`
	layoutFields
}

type setLayoutOut struct {
	penpot.Target
	BoardID string    `json:"board_id"`
	Placed  []drawn   `json:"placed" jsonschema:"each direct child's new box, in layout order"`
	Tab     *tabCheck `json:"tab,omitempty" jsonschema:"what the connected Penpot tab saw, when the MCP route is on"`
}

func registerLayout(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "set_layout",
		Description: "Give a board a flex (row or column) or grid layout and place its children. " +
			"Children go in layer order, top layer first, as Penpot lays them out. Child sizes are kept.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in setLayoutIn) (*mcp.CallToolResult, setLayoutOut, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, setLayoutOut{}, err
		}
		if in.PageID != "" {
			if err := checkID("page_id", in.PageID); err != nil {
				return nil, setLayoutOut{}, err
			}
		}
		if err := checkID("board_id", in.BoardID); err != nil {
			return nil, setLayoutOut{}, err
		}
		if err := in.validate(); err != nil {
			return nil, setLayoutOut{}, err
		}
		t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
		if err != nil {
			return nil, setLayoutOut{}, err
		}
		p, err := loadRaw(ctx, d, t)
		if err != nil {
			return nil, setLayoutOut{}, err
		}
		b, err := p.get(in.BoardID, t)
		if err != nil {
			return nil, setLayoutOut{}, err
		}
		if str(b, "type") != penpot.TypeBoard || in.BoardID == penpot.ZeroID {
			return nil, setLayoutOut{}, fmt.Errorf("%q is a %s; only boards take a layout", str(b, "name"), str(b, "type"))
		}
		// Penpot lays out the top layer first: the reverse of shapes.
		ids := children(b)
		kids := make([]item, 0, len(ids))
		for i := len(ids) - 1; i >= 0; i-- {
			kids = append(kids, rawItem(p.objects[ids[i]], ids[i]))
		}
		board := rawItem(b, in.BoardID)
		placed := in.place(board, kids)
		changes := []map[string]any{penpot.ModObj(t.PageID, in.BoardID, in.attrs(kids))}
		out := setLayoutOut{Target: t, BoardID: in.BoardID}
		for i, k := range kids {
			moved, err := moveTree(p, t.PageID, k.id, placed[i].x-k.x, placed[i].y-k.y)
			if err != nil {
				return nil, setLayoutOut{}, err
			}
			changes = append(changes, moved...)
			o := p.objects[k.id]
			out.Placed = append(out.Placed, drawn{
				ShapeID: k.id, ParentID: in.BoardID, Type: str(o, "type"), Name: str(o, "name"),
				X: placed[i].x, Y: placed[i].y, Width: k.w, Height: k.h,
			})
		}
		if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
			return nil, setLayoutOut{}, err
		}
		out.Tab = afterWrite(ctx, d, t.FileID, nil)
		return nil, out, nil
	})
}

func rawItem(o map[string]any, id string) item {
	return item{id: id, x: num(o, "x"), y: num(o, "y"), w: num(o, "width"), h: num(o, "height")}
}
