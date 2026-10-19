package tools

import (
	"context"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// drawNode is one shape in a draw call. The list is flat: a node names its
// parent by the ref of an earlier board in the same list, or by the id of a
// board already on the page.
type drawNode struct {
	Ref    string  `json:"ref,omitempty" jsonschema:"a name for this node that later nodes use as parent"`
	Parent string  `json:"parent,omitempty" jsonschema:"ref of an earlier board in this list, or the id of a board on the page; the call's parent_id when empty"`
	Type   string  `json:"type" jsonschema:"board, rect, ellipse or text"`
	Name   string  `json:"name,omitempty" jsonschema:"layer name; required except for text"`
	Height float64 `json:"height,omitempty" jsonschema:"height in pixels; not used for text"`
	Fill   string  `json:"fill,omitempty" jsonschema:"fill colour as #RRGGBB (boards, rects, ellipses)"`
	Radius float64 `json:"radius,omitempty" jsonschema:"corner radius (boards, rects)"`
	Clip   bool    `json:"clip,omitempty" jsonschema:"boards only: clip children to the board"`
	textFields
}

type drawIn struct {
	target
	Shapes []drawNode `json:"shapes" jsonschema:"shapes bottom to top: each later shape is drawn above earlier ones"`
}

// drawn is one created shape.
type drawn struct {
	Ref      string  `json:"ref,omitempty"`
	ShapeID  string  `json:"shape_id"`
	ParentID string  `json:"parent_id"`
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Measured *bool   `json:"text_measured,omitempty"`
}

type drawOut struct {
	penpot.Target
	Shapes []drawn `json:"shapes"`
}

func (o drawOut) single() shapeOut {
	s := o.Shapes[0]
	return shapeOut{Target: o.Target, ShapeID: s.ShapeID, ParentID: s.ParentID, Width: s.Width, Height: s.Height}
}

func registerDrawTree(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "draw",
		Description: "Draw many shapes (boards, rects, ellipses, text) in one write: a whole screen in one call. " +
			"Shapes are a flat list, bottom to top; nest by giving a node the ref of an earlier board as parent. " +
			"Coordinates are absolute page pixels. Returns every shape's id and box.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in drawIn) (*mcp.CallToolResult, drawOut, error) {
		if err := in.validate(); err != nil {
			return nil, drawOut{}, err
		}
		if len(in.Shapes) == 0 || len(in.Shapes) > maxDrawSize {
			return nil, drawOut{}, fmt.Errorf("shapes must hold 1 to %d nodes", maxDrawSize)
		}
		refs := map[string]string{} // ref -> type
		for i, n := range in.Shapes {
			if err := n.validate(); err != nil {
				return nil, drawOut{}, fmt.Errorf("shapes[%d]: %w", i, err)
			}
			if n.Parent != "" && !uuidRE.MatchString(n.Parent) {
				typ, ok := refs[n.Parent]
				if !ok {
					return nil, drawOut{}, fmt.Errorf("shapes[%d]: parent %q is not the ref of an earlier node", i, n.Parent)
				}
				if typ != "board" {
					return nil, drawOut{}, fmt.Errorf("shapes[%d]: parent %q is a %s; only boards hold shapes", i, n.Parent, typ)
				}
			}
			if n.Ref != "" {
				if _, dup := refs[n.Ref]; dup {
					return nil, drawOut{}, fmt.Errorf("shapes[%d]: ref %q is used twice", i, n.Ref)
				}
				refs[n.Ref] = n.Type
			}
		}
		out, err := drawNodes(ctx, d, in.target, in.Shapes)
		return nil, out, err
	})
}

func (n drawNode) validate() error {
	switch n.Type {
	case "board", "rect", "ellipse":
		if err := checkName(n.Name); err != nil {
			return err
		}
		if err := checkSize("width", n.Width); err != nil {
			return err
		}
		if err := checkSize("height", n.Height); err != nil {
			return err
		}
		if err := checkColor("fill", n.Fill); err != nil {
			return err
		}
		if math.IsNaN(n.Radius) || n.Radius < 0 || n.Radius > maxSize {
			return fmt.Errorf("radius must be 0 to %d", maxSize)
		}
	case "text":
		if n.Text == "" || len(n.Text) > maxTextLen {
			return fmt.Errorf("text must be 1 to %d bytes", maxTextLen)
		}
		if n.Name != "" {
			if err := checkName(n.Name); err != nil {
				return err
			}
		}
		if n.Width != 0 {
			if err := checkSize("width", n.Width); err != nil {
				return err
			}
		}
		if n.FontSize != 0 && (math.IsNaN(n.FontSize) || n.FontSize < 3 || n.FontSize > 1000) {
			return fmt.Errorf("font_size must be 3 to 1000")
		}
		if n.LineHeight != 0 && (math.IsNaN(n.LineHeight) || n.LineHeight <= 0 || n.LineHeight > 200) {
			return fmt.Errorf("line_height must be above 0 and at most 200")
		}
		if math.IsNaN(n.LetterSpacing) || math.Abs(n.LetterSpacing) > 200 {
			return fmt.Errorf("letter_spacing must be within ±200")
		}
		if err := checkColor("color", n.Color); err != nil {
			return err
		}
		switch n.Align {
		case "", "left", "center", "right", "justify":
		default:
			return fmt.Errorf("align must be left, center, right or justify")
		}
		if _, err := n.face(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("type must be board, rect, ellipse or text")
	}
	if n.Ref != "" && !refRE.MatchString(n.Ref) {
		return fmt.Errorf("ref must be 1 to 64 letters, digits, dots, dashes or underscores")
	}
	if err := checkCoord("x", n.X); err != nil {
		return err
	}
	return checkCoord("y", n.Y)
}

// drawNodes builds every node into one update-file call.
func drawNodes(ctx context.Context, d Deps, tg target, nodes []drawNode) (drawOut, error) {
	t, err := d.Writer.Resolve(ctx, tg.FileID, tg.PageID)
	if err != nil {
		return drawOut{}, err
	}
	// Existing boards named as parents are checked once each.
	root := penpot.ZeroID
	checked := map[string]bool{}
	check := func(id string) error {
		if id == penpot.ZeroID || checked[id] {
			return nil
		}
		if err := checkParent(ctx, d, t, id); err != nil {
			return err
		}
		checked[id] = true
		return nil
	}
	if tg.ParentID != "" {
		if err := check(tg.ParentID); err != nil {
			return drawOut{}, err
		}
		root = tg.ParentID
	}

	ids := map[string]string{} // ref -> new id
	out := drawOut{Target: t}
	changes := make([]map[string]any, 0, len(nodes))
	for _, n := range nodes {
		parent := root
		switch {
		case n.Parent == "":
		case uuidRE.MatchString(n.Parent):
			if err := check(n.Parent); err != nil {
				return drawOut{}, err
			}
			parent = n.Parent
		default:
			parent = ids[n.Parent]
		}
		id := penpot.NewID()
		if n.Ref != "" {
			ids[n.Ref] = id
		}
		obj, res := build(ctx, d, n, id, parent)
		changes = append(changes, penpot.AddObj(t.PageID, obj))
		res.Ref = n.Ref
		out.Shapes = append(out.Shapes, res)
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return drawOut{}, err
	}
	return out, nil
}

func build(ctx context.Context, d Deps, n drawNode, id, parent string) (map[string]any, drawn) {
	box := penpot.Box{ID: id, ParentID: parent, Name: n.Name, X: n.X, Y: n.Y, W: n.Width, H: n.Height}
	res := drawn{ShapeID: id, ParentID: parent, Type: n.Type, Name: n.Name, X: n.X, Y: n.Y}
	switch n.Type {
	case "board":
		fill := n.Fill
		if fill == "" {
			fill = "#FFFFFF"
		}
		res.Width, res.Height = n.Width, n.Height
		return penpot.BoardObject(box, fill, n.Radius, n.Clip), res
	case "rect":
		res.Width, res.Height = n.Width, n.Height
		return penpot.RectObject(box, orDefault(n.Fill, "#D9D9D9"), n.Radius), res
	case "ellipse":
		res.Width, res.Height = n.Width, n.Height
		return penpot.EllipseObject(box, orDefault(n.Fill, "#D9D9D9")), res
	}

	// text
	face, _ := n.face() // validated
	size := orDefaultF(n.FontSize, 16)
	lh := orDefaultF(n.LineHeight, 1.2)
	grow := "auto-width"
	var m sizeResult
	if n.Width > 0 {
		grow = "auto-height"
		s := d.Fonts.MeasureWrapped(ctx, face, n.Text, size, lh, n.LetterSpacing, n.Width)
		m = sizeResult{s.Width, s.Height, s.Measured}
	} else {
		s := d.Fonts.Measure(ctx, face, n.Text, size, lh, n.LetterSpacing)
		m = sizeResult{s.Width, s.Height, s.Measured}
	}
	if box.Name == "" {
		box.Name = firstLine(n.Text)
		res.Name = box.Name
	}
	box.W, box.H = m.w, m.h
	res.Width, res.Height, res.Measured = m.w, m.h, &m.measured
	style := penpot.TextStyle{
		FontID: face.FontID, FontFamily: face.Family, FontVariantID: face.VariantID,
		FontWeight: face.Weight, FontStyle: face.Style,
		FontSize: size, LineHeight: lh, LetterSpacing: n.LetterSpacing,
		Color: orDefault(n.Color, "#000000"), Align: n.Align,
	}
	return penpot.TextObject(box, n.Text, style, grow), res
}

type sizeResult struct {
	w, h     float64
	measured bool
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orDefaultF(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

func firstLine(s string) string {
	for i, r := range s {
		if r == '\n' {
			s = s[:i]
			break
		}
	}
	r := []rune(s)
	if len(r) > 60 {
		return string(r[:60])
	}
	return s
}
