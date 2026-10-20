package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// drawNode is one shape in a draw call. The list is flat: a node names its
// parent by the ref of an earlier board, group or bool in the same list, or
// by the id of one already on the page.
type drawNode struct {
	Ref         string  `json:"ref,omitempty" jsonschema:"a name for this node that later nodes use as parent"`
	Parent      string  `json:"parent,omitempty" jsonschema:"ref of an earlier board, group or bool in this list, or the id of one on the page; the call's parent_id when empty"`
	Type        string  `json:"type" jsonschema:"board, rect, ellipse, text, path, image, group or bool"`
	Name        string  `json:"name,omitempty" jsonschema:"layer name; required except for text and image"`
	Height      float64 `json:"height,omitempty" jsonschema:"height in pixels; not used for text, path, group or bool"`
	Fill        string  `json:"fill,omitempty" jsonschema:"fill colour as #RRGGBB"`
	Radius      float64 `json:"radius,omitempty" jsonschema:"corner radius (boards, rects, images)"`
	Clip        bool    `json:"clip,omitempty" jsonschema:"boards only: clip children to the board"`
	Rotation    float64 `json:"rotation,omitempty" jsonschema:"degrees clockwise about the centre (rects, ellipses, text, images)"`
	Stroke      string  `json:"stroke,omitempty" jsonschema:"stroke colour as #RRGGBB; paths get #000000 when they have no fill"`
	StrokeWidth float64 `json:"stroke_width,omitempty" jsonschema:"stroke width in pixels, default 1"`
	D           string  `json:"d,omitempty" jsonschema:"path only: SVG path data in absolute page pixels, such as M10 10 L90 10"`
	ImagePath   string  `json:"image_path,omitempty" jsonschema:"image only: absolute path of a local PNG, JPEG, WebP, GIF or SVG"`
	BoolType    string  `json:"bool_type,omitempty" jsonschema:"bool only: union (default), difference, intersection or exclude"`
	textFields
	layoutFields
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
	Shapes []drawn   `json:"shapes"`
	Tab    *tabCheck `json:"tab,omitempty" jsonschema:"what the connected Penpot tab saw, when the MCP route is on"`
}

func (o drawOut) single() shapeOut {
	s := o.Shapes[0]
	return shapeOut{Target: o.Target, ShapeID: s.ShapeID, ParentID: s.ParentID, Width: s.Width, Height: s.Height, Tab: o.Tab}
}

func registerDrawTree(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "draw",
		Description: "Draw many shapes in one write: a whole screen per call. Types: board, rect, ellipse, text, " +
			"path (SVG d), image (local file), group and bool. Shapes are a flat list, bottom to top; nest by " +
			"giving a node the ref of an earlier board, group or bool as parent. A board with layout places " +
			"its children itself, first listed first. Coordinates are absolute page pixels. " +
			"Returns every shape's id and box.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in drawIn) (*mcp.CallToolResult, drawOut, error) {
		if err := in.validate(); err != nil {
			return nil, drawOut{}, err
		}
		if err := checkTree(in.Shapes); err != nil {
			return nil, drawOut{}, err
		}
		out, err := drawNodes(ctx, d, in.target, in.Shapes)
		if err == nil {
			out.Tab = afterWrite(ctx, d, out.FileID, textIDs(out))
		}
		return nil, out, err
	})
}

// checkTree validates every node and the refs between them.
func checkTree(nodes []drawNode) error {
	if len(nodes) == 0 || len(nodes) > maxDrawSize {
		return fmt.Errorf("shapes must hold 1 to %d nodes", maxDrawSize)
	}
	refs := map[string]string{} // ref -> type
	kids := map[string]int{}
	for i, n := range nodes {
		if err := n.validate(); err != nil {
			return fmt.Errorf("shapes[%d]: %w", i, err)
		}
		if n.Parent != "" && !uuidRE.MatchString(n.Parent) {
			typ, ok := refs[n.Parent]
			if !ok {
				return fmt.Errorf("shapes[%d]: parent %q is not the ref of an earlier node", i, n.Parent)
			}
			if typ != "board" && typ != "group" && typ != "bool" {
				return fmt.Errorf("shapes[%d]: parent %q is a %s; only boards, groups and bools hold shapes", i, n.Parent, typ)
			}
			kids[n.Parent]++
		}
		if n.Ref != "" {
			if _, dup := refs[n.Ref]; dup {
				return fmt.Errorf("shapes[%d]: ref %q is used twice", i, n.Ref)
			}
			refs[n.Ref] = n.Type
		}
	}
	for i, n := range nodes {
		if (n.Type == "group" || n.Type == "bool") && (n.Ref == "" || kids[n.Ref] == 0) {
			return fmt.Errorf("shapes[%d]: a %s needs a ref and at least one child naming it as parent", i, n.Type)
		}
	}
	return nil
}

func (n drawNode) validate() error {
	if n.Ref != "" && !refRE.MatchString(n.Ref) {
		return fmt.Errorf("ref must be 1 to 64 letters, digits, dots, dashes or underscores")
	}
	if n.set() {
		if n.Type != "board" {
			return fmt.Errorf("layout is for boards only")
		}
		if err := n.layoutFields.validate(); err != nil {
			return err
		}
	}
	if math.IsNaN(n.Rotation) || math.Abs(n.Rotation) > 360 {
		return fmt.Errorf("rotation must be within ±360")
	}
	if n.Rotation != 0 {
		switch n.Type {
		case "rect", "ellipse", "text", "image":
		default:
			return fmt.Errorf("rotation is for rects, ellipses, text and images")
		}
	}
	if err := checkColor("stroke", n.Stroke); err != nil {
		return err
	}
	if math.IsNaN(n.StrokeWidth) || n.StrokeWidth < 0 || n.StrokeWidth > 1000 {
		return fmt.Errorf("stroke_width must be 0 to 1000")
	}
	if err := checkColor("fill", n.Fill); err != nil {
		return err
	}
	if math.IsNaN(n.Radius) || n.Radius < 0 || n.Radius > maxSize {
		return fmt.Errorf("radius must be 0 to %d", maxSize)
	}
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
	case "path":
		if err := checkName(n.Name); err != nil {
			return err
		}
		if _, err := penpot.ParsePath(n.D); err != nil {
			return fmt.Errorf("d: %w", err)
		}
		return nil // the box comes from d
	case "image":
		if n.Name != "" {
			if err := checkName(n.Name); err != nil {
				return err
			}
		}
		if n.ImagePath == "" {
			return fmt.Errorf("image_path is required for images")
		}
		for label, v := range map[string]float64{"width": n.Width, "height": n.Height} {
			if v != 0 {
				if err := checkSize(label, v); err != nil {
					return err
				}
			}
		}
	case "group", "bool":
		if err := checkName(n.Name); err != nil {
			return err
		}
		switch n.BoolType {
		case "":
		case "union", "difference", "intersection", "exclude":
			if n.Type != "bool" {
				return fmt.Errorf("bool_type is for bools only")
			}
		default:
			return fmt.Errorf("bool_type must be union, difference, intersection or exclude")
		}
		return nil // the box comes from the children
	case "text":
		if err := n.validateText(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("type must be board, rect, ellipse, text, path, image, group or bool")
	}
	if err := checkCoord("x", n.X); err != nil {
		return err
	}
	return checkCoord("y", n.Y)
}

func (n drawNode) validateText() error {
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
	_, err := n.face()
	return err
}

// node is a drawNode once built.
type node struct {
	drawNode
	id, parent, frame string
	obj               map[string]any
	res               drawn
	kids              []int
}

// drawNodes builds every node into one update-file call.
func drawNodes(ctx context.Context, d Deps, tg target, nodes []drawNode) (drawOut, error) {
	t, err := d.Writer.Resolve(ctx, tg.FileID, tg.PageID)
	if err != nil {
		return drawOut{}, err
	}
	// Existing containers named as parents are checked once each.
	existing := map[string]penpot.Object{}
	frameOf := func(id string) (string, error) {
		if id == penpot.ZeroID {
			return penpot.ZeroID, nil
		}
		o, ok := existing[id]
		if !ok {
			if o, err = checkParent(ctx, d, t, id); err != nil {
				return "", err
			}
			existing[id] = o
		}
		if o.Type == penpot.TypeBoard {
			return id, nil
		}
		return o.FrameID, nil
	}
	root := penpot.ZeroID
	if tg.ParentID != "" {
		root = tg.ParentID
	}
	rootFrame, err := frameOf(root)
	if err != nil {
		return drawOut{}, err
	}

	built := make([]*node, 0, len(nodes))
	byRef := map[string]int{}
	byID := map[string]int{}
	for i, n := range nodes {
		b := &node{drawNode: n, id: penpot.NewID(), parent: root, frame: rootFrame}
		switch {
		case n.Parent == "":
		case uuidRE.MatchString(n.Parent):
			if b.frame, err = frameOf(n.Parent); err != nil {
				return drawOut{}, err
			}
			b.parent = n.Parent
		default:
			p := built[byRef[n.Parent]]
			b.parent, b.frame = p.id, p.frame
			if p.Type == "board" {
				b.frame = p.id
			}
			p.kids = append(p.kids, i)
		}
		if n.Ref != "" {
			byRef[n.Ref] = i
		}
		byID[b.id] = i
		if err := build(ctx, d, t.FileID, b); err != nil {
			return drawOut{}, fmt.Errorf("shapes[%d]: %w", i, err)
		}
		built = append(built, b)
	}

	// Inner layouts and groups first: children always come after parents.
	for i := len(built) - 1; i >= 0; i-- {
		b := built[i]
		switch {
		case b.Type == "board" && b.set():
			if err := layOut(built, b); err != nil {
				return drawOut{}, err
			}
		case b.Type == "group" || b.Type == "bool":
			if err := fitToKids(built, b); err != nil {
				return drawOut{}, err
			}
		}
	}

	out := drawOut{Target: t}
	changes := make([]map[string]any, 0, len(built)+1)
	var refit []string
	for _, b := range built {
		change := penpot.AddObj(t.PageID, b.obj)
		if pi, ok := byID[b.parent]; ok && built[pi].set() {
			// Penpot lays out the top layer first, so the first listed
			// child goes on top.
			change = penpot.AddObjAt(t.PageID, b.obj, 0)
		}
		changes = append(changes, change)
		if b.Type == "group" || b.Type == "bool" {
			refit = append([]string{b.id}, refit...)
		}
		b.res.Ref = b.Ref
		b.res.X, b.res.Y = num(b.obj, "x"), num(b.obj, "y")
		b.res.Width, b.res.Height = num(b.obj, "width"), num(b.obj, "height")
		out.Shapes = append(out.Shapes, b.res)
	}
	if len(refit) > 0 {
		changes = append(changes, penpot.RegObjects(t.PageID, refit))
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return drawOut{}, err
	}
	return out, nil
}

// layOut places a new board's children and moves their subtrees with them.
func layOut(built []*node, b *node) error {
	kids := make([]item, len(b.kids))
	for i, k := range b.kids {
		o := built[k].obj
		kids[i] = item{id: built[k].id, x: num(o, "x"), y: num(o, "y"), w: num(o, "width"), h: num(o, "height")}
	}
	board := item{x: num(b.obj, "x"), y: num(b.obj, "y"), w: num(b.obj, "width"), h: num(b.obj, "height")}
	placed := b.place(board, kids)
	for i, k := range b.kids {
		if err := shiftSubtree(built, k, placed[i].x-kids[i].x, placed[i].y-kids[i].y); err != nil {
			return err
		}
	}
	// Grid cells name the children top layer first, like the placement.
	for k, v := range b.attrs(kids) {
		b.obj[k] = v
	}
	return nil
}

func shiftSubtree(built []*node, i int, dx, dy float64) error {
	if dx == 0 && dy == 0 {
		return nil
	}
	raw, err := roundTrip(built[i].obj)
	if err != nil {
		return err
	}
	attrs, err := shiftRaw(raw, dx, dy)
	if err != nil {
		return err
	}
	for k, v := range attrs {
		built[i].obj[k] = v
	}
	for _, k := range built[i].kids {
		if err := shiftSubtree(built, k, dx, dy); err != nil {
			return err
		}
	}
	return nil
}

// fitToKids sets a group's or bool's box to its children's bounds.
func fitToKids(built []*node, b *node) error {
	var pts []penpot.Point
	for _, k := range b.kids {
		raw, err := roundTrip(built[k].obj)
		if err != nil {
			return err
		}
		list, _ := raw["points"].([]any)
		for _, v := range list {
			m, _ := v.(map[string]any)
			pts = append(pts, penpot.Point{X: num(m, "x"), Y: num(m, "y")})
		}
	}
	x, y, w, h := penpot.Bounds(pts)
	g := penpot.BoxGeometry(x, y, w, h)
	b.obj["x"], b.obj["y"], b.obj["width"], b.obj["height"] = g.X, g.Y, g.Width, g.Height
	b.obj["selrect"], b.obj["points"] = g.Selrect, g.Points
	return nil
}

func roundTrip(obj map[string]any) (map[string]any, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	return raw, json.Unmarshal(data, &raw)
}

func build(ctx context.Context, d Deps, fileID string, b *node) error {
	n := b.drawNode
	box := penpot.Box{
		ID: b.id, ParentID: b.parent, FrameID: b.frame, Name: n.Name,
		X: n.X, Y: n.Y, W: n.Width, H: n.Height, Rotation: n.Rotation,
	}
	b.res = drawn{ShapeID: b.id, ParentID: b.parent, Type: n.Type, Name: n.Name}
	switch n.Type {
	case "board":
		b.obj = penpot.BoardObject(box, orDefault(n.Fill, "#FFFFFF"), n.Radius, n.Clip)
	case "rect":
		b.obj = penpot.RectObject(box, orDefault(n.Fill, "#D9D9D9"), n.Radius)
	case "ellipse":
		b.obj = penpot.EllipseObject(box, orDefault(n.Fill, "#D9D9D9"))
	case "path":
		segs, err := penpot.ParsePath(n.D)
		if err != nil {
			return err
		}
		b.obj = penpot.PathObject(box, segs, n.Fill, nil)
	case "image":
		m, err := uploadImage(ctx, d, fileID, n.ImagePath)
		if err != nil {
			return err
		}
		switch {
		case box.W == 0 && box.H == 0:
			box.W, box.H = m.Width, m.Height
		case box.H == 0:
			box.H = box.W * m.Height / m.Width
		case box.W == 0:
			box.W = box.H * m.Width / m.Height
		}
		if box.Name == "" {
			box.Name = m.Name
			b.res.Name = m.Name
		}
		b.obj = penpot.ImageObject(box, m, n.Radius)
	case "group":
		b.obj = penpot.GroupObject(box)
	case "bool":
		b.obj = penpot.BoolObject(box, orDefault(n.BoolType, "union"), orDefault(n.Fill, "#D9D9D9"))
	case "text":
		buildText(ctx, d, b, box)
	}
	b.obj["strokes"] = strokes(n)
	return nil
}

// strokes is a node's stroke list: its stroke colour, or black for a path
// with neither fill nor stroke.
func strokes(n drawNode) []map[string]any {
	color := n.Stroke
	if color == "" && n.Type == "path" && n.Fill == "" {
		color = "#000000"
	}
	if color == "" {
		return []map[string]any{}
	}
	return []map[string]any{penpot.Stroke(color, orDefaultF(n.StrokeWidth, 1))}
}

func buildText(ctx context.Context, d Deps, b *node, box penpot.Box) {
	n := b.drawNode
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
		b.res.Name = box.Name
	}
	box.W, box.H = m.w, m.h
	b.res.Measured = &m.measured
	style := penpot.TextStyle{
		FontID: face.FontID, FontFamily: face.Family, FontVariantID: face.VariantID,
		FontWeight: face.Weight, FontStyle: face.Style,
		FontSize: size, LineHeight: lh, LetterSpacing: n.LetterSpacing,
		Color: orDefault(n.Color, "#000000"), Align: n.Align,
	}
	b.obj = penpot.TextObject(box, n.Text, style, grow)
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
