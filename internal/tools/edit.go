package tools

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

const maxEditIDs = 200

// pageRef names a page; the first page when PageID is empty.
type pageRef struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	PageID string `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
}

func (r pageRef) validate() error {
	if err := checkID("file_id", r.FileID); err != nil {
		return err
	}
	if r.PageID != "" {
		return checkID("page_id", r.PageID)
	}
	return nil
}

func checkIDs(ids []string, min int) error {
	if len(ids) < min || len(ids) > maxEditIDs {
		return fmt.Errorf("shape_ids must hold %d to %d ids", min, maxEditIDs)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if err := checkID("shape_ids", id); err != nil {
			return err
		}
		if id == penpot.ZeroID {
			return fmt.Errorf("the root frame cannot be edited, moved or deleted")
		}
		if seen[id] {
			return fmt.Errorf("shape %s is listed twice", id)
		}
		seen[id] = true
	}
	return nil
}

type updateIn struct {
	pageRef
	ShapeID     string   `json:"shape_id" jsonschema:"the shape to change"`
	Name        string   `json:"name,omitempty" jsonschema:"new layer name"`
	X           *float64 `json:"x,omitempty" jsonschema:"new left edge, absolute page pixels; children move with it"`
	Y           *float64 `json:"y,omitempty" jsonschema:"new top edge; children move with it"`
	Width       *float64 `json:"width,omitempty" jsonschema:"new width (boards, rects, ellipses, images, text); text then wraps at it"`
	Height      *float64 `json:"height,omitempty" jsonschema:"new height (boards, rects, ellipses, images)"`
	Rotation    *float64 `json:"rotation,omitempty" jsonschema:"degrees clockwise about the centre (rects, ellipses, images, text)"`
	Fill        string   `json:"fill,omitempty" jsonschema:"fill colour as #RRGGBB, or none"`
	Stroke      string   `json:"stroke,omitempty" jsonschema:"stroke colour as #RRGGBB, or none"`
	StrokeWidth *float64 `json:"stroke_width,omitempty" jsonschema:"stroke width in pixels"`
	Radius      *float64 `json:"radius,omitempty" jsonschema:"corner radius (boards, rects, images)"`
	Opacity     *float64 `json:"opacity,omitempty" jsonschema:"0 to 1"`
	Hidden      *bool    `json:"hidden,omitempty" jsonschema:"hide or show the layer"`
	Text        string   `json:"text,omitempty" jsonschema:"text shapes: new text, keeping the first run's style; the box is measured again"`
}

type idsIn struct {
	pageRef
	ShapeIDs []string `json:"shape_ids" jsonschema:"the shapes"`
}

type moveIn struct {
	idsIn
	DX float64 `json:"dx" jsonschema:"pixels to move right (negative: left)"`
	DY float64 `json:"dy" jsonschema:"pixels to move down (negative: up)"`
}

type reparentIn struct {
	idsIn
	ParentID string `json:"parent_id" jsonschema:"the board, group or bool to move into; 00000000-0000-0000-0000-000000000000 for the page root"`
	Index    *int   `json:"index,omitempty" jsonschema:"position among the parent's children, 0 is the bottom; the top when empty"`
}

type groupIn struct {
	idsIn
	Name     string `json:"name" jsonschema:"layer name"`
	BoolType string `json:"bool_type,omitempty" jsonschema:"create_bool only: union (default), difference, intersection or exclude"`
	Fill     string `json:"fill,omitempty" jsonschema:"create_bool only: fill colour as #RRGGBB"`
}

type changedOut struct {
	penpot.Target
	Shapes []changed `json:"shapes"`
}

type changed struct {
	ShapeID string `json:"shape_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

type groupOut struct {
	penpot.Target
	ShapeID  string `json:"shape_id"`
	ParentID string `json:"parent_id"`
}

type applyIn struct {
	pageRef
	Changes []map[string]any `json:"changes" jsonschema:"raw update-file changes in Penpot's JSON form (camelCase keys); see docs/penpot-api-reference.md section 6"`
}

type applyOut struct {
	penpot.Target
	Applied int `json:"applied"`
}

func registerEdit(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "update_shape",
		Description: "Change one shape: name, position (children follow), size, rotation, fill, stroke, " +
			"radius, opacity, visibility, or a text shape's text. Only the fields given change.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateIn) (*mcp.CallToolResult, changedOut, error) {
		out, err := updateShape(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "move_shape",
		Description: "Move shapes by dx, dy pixels; everything inside them moves too, and groups around them refit.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in moveIn) (*mcp.CallToolResult, changedOut, error) {
		out, err := moveShapes(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "reparent_shape",
		Description: "Move shapes into another board, group or bool, or reorder them in their own parent " +
			"with index. They keep their page position.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in reparentIn) (*mcp.CallToolResult, changedOut, error) {
		out, err := reparent(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_shapes",
		Description: "Delete shapes and everything inside them. Returns the names removed. " +
			"The root frame is never deleted.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in idsIn) (*mcp.CallToolResult, changedOut, error) {
		out, err := deleteShapes(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_group",
		Description: "Group shapes that share a parent. The group takes the place of the top one.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in groupIn) (*mcp.CallToolResult, groupOut, error) {
		out, err := group(ctx, d, in, false)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_bool",
		Description: "Combine shapes that share a parent into a boolean shape: union, difference, intersection or exclude.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in groupIn) (*mcp.CallToolResult, groupOut, error) {
		out, err := group(ctx, d, in, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "apply_changes",
		Description: "Last resort: send raw update-file changes when no other tool fits. Use the shape tools " +
			"first. Only known change types are accepted; mov-objects is refused (use reparent_shape).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in applyIn) (*mcp.CallToolResult, applyOut, error) {
		out, err := applyChanges(ctx, d, in)
		return nil, out, err
	})
}

func updateShape(ctx context.Context, d Deps, in updateIn) (changedOut, error) {
	if err := in.validate(); err != nil {
		return changedOut{}, err
	}
	if err := checkIDs([]string{in.ShapeID}, 1); err != nil {
		return changedOut{}, err
	}
	if err := in.check(); err != nil {
		return changedOut{}, err
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return changedOut{}, err
	}
	p, err := loadRaw(ctx, d, t)
	if err != nil {
		return changedOut{}, err
	}
	o, err := p.get(in.ShapeID, t)
	if err != nil {
		return changedOut{}, err
	}
	typ := str(o, "type")
	attrs := map[string]any{}
	var changes []map[string]any

	x, y := num(o, "x"), num(o, "y")
	if in.X != nil || in.Y != nil {
		nx, ny := x, y
		if in.X != nil {
			nx = *in.X
		}
		if in.Y != nil {
			ny = *in.Y
		}
		moved, err := moveTree(p, t.PageID, in.ShapeID, nx-x, ny-y)
		if err != nil {
			return changedOut{}, err
		}
		changes = append(changes, moved...)
		x, y = nx, ny
	}

	w, h, rot := num(o, "width"), num(o, "height"), num(o, "rotation")
	resized := in.Width != nil || in.Height != nil || in.Rotation != nil
	if resized {
		boxy := typ == penpot.TypeBoard || typ == penpot.TypeRect || typ == penpot.TypeEllipse
		switch {
		case in.Height != nil && !boxy:
			return changedOut{}, fmt.Errorf("height can change on boards, rects, ellipses and images, not a %s", typ)
		case in.Width != nil && !boxy && typ != penpot.TypeText:
			return changedOut{}, fmt.Errorf("width can change on boards, rects, ellipses, images and text, not a %s", typ)
		case in.Rotation != nil && (typ == penpot.TypeBoard || (!boxy && typ != penpot.TypeText)):
			return changedOut{}, fmt.Errorf("rotation can change on rects, ellipses, images and text, not a %s", typ)
		}
		if in.Width != nil {
			w = *in.Width
		}
		if in.Height != nil {
			h = *in.Height
		}
		if in.Rotation != nil {
			rot = *in.Rotation
		}
	}

	if in.Text != "" || (typ == penpot.TypeText && in.Width != nil) {
		if typ != penpot.TypeText {
			return changedOut{}, fmt.Errorf("text can change only on text shapes, not a %s", typ)
		}
		tw, th, err := retext(ctx, d, o, in.Text, in.Width, attrs)
		if err != nil {
			return changedOut{}, err
		}
		w, h, resized = tw, th, true
	}
	if resized {
		g := penpot.RotatedGeometry(x, y, w, h, rot)
		attrs["x"], attrs["y"], attrs["width"], attrs["height"] = g.X, g.Y, g.Width, g.Height
		attrs["rotation"], attrs["selrect"], attrs["points"] = g.Rotation, g.Selrect, g.Points
		attrs["transform"], attrs["transformInverse"] = g.Transform, g.TransformInverse
	}

	if in.Name == "" && in.Text != "" && str(o, "name") == firstLine(plainText(o["content"])) {
		// The layer was named after its text: keep it that way.
		in.Name = firstLine(in.Text)
	}
	if in.Name != "" {
		attrs["name"] = in.Name
	}
	if in.Fill != "" {
		if typ == penpot.TypeGroup {
			return changedOut{}, fmt.Errorf("groups have no fill; change their children")
		}
		attrs["fills"] = fillList(in.Fill)
	}
	if in.Stroke != "" || in.StrokeWidth != nil {
		color := in.Stroke
		if color == "" {
			color = firstStroke(o)
			if color == "" {
				return changedOut{}, fmt.Errorf("the shape has no stroke; give stroke as well")
			}
		}
		width := 1.0
		if in.StrokeWidth != nil {
			width = *in.StrokeWidth
		}
		if color == "none" {
			attrs["strokes"] = []any{}
		} else {
			attrs["strokes"] = []map[string]any{penpot.Stroke(color, width)}
		}
	}
	if in.Radius != nil {
		if typ != penpot.TypeBoard && typ != penpot.TypeRect {
			return changedOut{}, fmt.Errorf("radius is for boards, rects and images, not a %s", typ)
		}
		for _, k := range []string{"r1", "r2", "r3", "r4"} {
			attrs[k] = *in.Radius
		}
	}
	if in.Opacity != nil {
		attrs["opacity"] = *in.Opacity
	}
	if in.Hidden != nil {
		attrs["hidden"] = *in.Hidden
	}
	if len(attrs) > 0 {
		changes = append(changes, penpot.ModObj(t.PageID, in.ShapeID, attrs))
	}
	if len(changes) == 0 {
		return changedOut{}, fmt.Errorf("nothing to change: give at least one field")
	}
	if regs := refit(p, []string{in.ShapeID}); len(regs) > 0 {
		changes = append(changes, penpot.RegObjects(t.PageID, regs))
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return changedOut{}, err
	}
	name := str(o, "name")
	if in.Name != "" {
		name = in.Name
	}
	return changedOut{Target: t, Shapes: []changed{{ShapeID: in.ShapeID, Name: name, Type: typ}}}, nil
}

func (in updateIn) check() error {
	if in.Name != "" {
		if err := checkName(in.Name); err != nil {
			return err
		}
	}
	for label, v := range map[string]*float64{"x": in.X, "y": in.Y} {
		if v != nil {
			if err := checkCoord(label, *v); err != nil {
				return err
			}
		}
	}
	for label, v := range map[string]*float64{"width": in.Width, "height": in.Height} {
		if v != nil {
			if err := checkSize(label, *v); err != nil {
				return err
			}
		}
	}
	if in.Rotation != nil && (math.IsNaN(*in.Rotation) || math.Abs(*in.Rotation) > 360) {
		return fmt.Errorf("rotation must be within ±360")
	}
	for label, v := range map[string]string{"fill": in.Fill, "stroke": in.Stroke} {
		if v != "none" {
			if err := checkColor(label, v); err != nil {
				return err
			}
		}
	}
	if in.StrokeWidth != nil && (math.IsNaN(*in.StrokeWidth) || *in.StrokeWidth < 0 || *in.StrokeWidth > 1000) {
		return fmt.Errorf("stroke_width must be 0 to 1000")
	}
	if in.Radius != nil && (math.IsNaN(*in.Radius) || *in.Radius < 0 || *in.Radius > maxSize) {
		return fmt.Errorf("radius must be 0 to %d", maxSize)
	}
	if in.Opacity != nil && (math.IsNaN(*in.Opacity) || *in.Opacity < 0 || *in.Opacity > 1) {
		return fmt.Errorf("opacity must be 0 to 1")
	}
	if len(in.Text) > maxTextLen {
		return fmt.Errorf("text must be at most %d bytes", maxTextLen)
	}
	return nil
}

func fillList(hex string) []map[string]any {
	if hex == "none" {
		return []map[string]any{}
	}
	return []map[string]any{{"fillColor": strings.ToUpper(hex), "fillOpacity": 1}}
}

func firstStroke(o map[string]any) string {
	list, _ := o["strokes"].([]any)
	if len(list) == 0 {
		return ""
	}
	m, _ := list[0].(map[string]any)
	return str(m, "strokeColor")
}

// retext rebuilds a text shape's content with its first run's style and
// measures the new box. It fills attrs with content and growType.
func retext(ctx context.Context, d Deps, o map[string]any, text string, width *float64, attrs map[string]any) (w, h float64, err error) {
	leaf, align := firstLeaf(o["content"])
	if leaf == nil {
		return 0, 0, fmt.Errorf("cannot read the text's style")
	}
	if text == "" {
		text = plainText(o["content"])
	}
	face := fonts.Face{
		FontID: str(leaf, "fontId"), Family: str(leaf, "fontFamily"), VariantID: str(leaf, "fontVariantId"),
		Weight: str(leaf, "fontWeight"), Style: str(leaf, "fontStyle"),
	}
	size := atof(str(leaf, "fontSize"), 16)
	lh := atof(str(leaf, "lineHeight"), 1.2)
	ls := atof(str(leaf, "letterSpacing"), 0)
	color := "#000000"
	if fl, ok := leaf["fills"].([]any); ok && len(fl) > 0 {
		if m, ok := fl[0].(map[string]any); ok && str(m, "fillColor") != "" {
			color = str(m, "fillColor")
		}
	}
	grow := str(o, "growType")
	if width != nil {
		grow = "auto-height"
	}
	var s fonts.Size
	switch grow {
	case "auto-height", "fixed":
		wrap := num(o, "width")
		if width != nil {
			wrap = *width
		}
		s = d.Fonts.MeasureWrapped(ctx, face, text, size, lh, ls, wrap)
		s.Width = wrap
		if grow == "fixed" {
			s.Height = num(o, "height")
		}
	default:
		grow = "auto-width"
		s = d.Fonts.Measure(ctx, face, text, size, lh, ls)
	}
	style := penpot.TextStyle{
		FontID: face.FontID, FontFamily: face.Family, FontVariantID: face.VariantID,
		FontWeight: face.Weight, FontStyle: face.Style, FontSize: size, LineHeight: lh,
		LetterSpacing: ls, Color: color, Align: align,
	}
	attrs["content"] = penpot.TextContent(text, style)
	attrs["growType"] = grow
	attrs["positionData"] = nil
	return s.Width, s.Height, nil
}

func atof(s string, def float64) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) {
		return def
	}
	return v
}

// firstLeaf finds the first text run and its paragraph's alignment.
func firstLeaf(content any) (map[string]any, string) {
	var leaf map[string]any
	align := ""
	var walk func(any, string)
	walk = func(n any, al string) {
		m, ok := n.(map[string]any)
		if !ok || leaf != nil {
			return
		}
		if a := str(m, "textAlign"); a != "" {
			al = a
		}
		if _, isText := m["text"]; isText {
			leaf, align = m, al
			return
		}
		kids, _ := m["children"].([]any)
		for _, k := range kids {
			walk(k, al)
		}
	}
	walk(content, "")
	return leaf, align
}

// plainText joins a content tree's runs, one line per paragraph.
func plainText(content any) string {
	var paras []string
	var walk func(any)
	walk = func(n any) {
		m, ok := n.(map[string]any)
		if !ok {
			return
		}
		if str(m, "type") == "paragraph" {
			var b strings.Builder
			kids, _ := m["children"].([]any)
			for _, k := range kids {
				if km, ok := k.(map[string]any); ok {
					b.WriteString(str(km, "text"))
				}
			}
			paras = append(paras, b.String())
			return
		}
		kids, _ := m["children"].([]any)
		for _, k := range kids {
			walk(k)
		}
	}
	walk(content)
	return strings.Join(paras, "\n")
}

// topmost drops ids that sit inside another listed shape.
func topmost(p rawPage, ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	var out []string
	for _, id := range ids {
		inside := false
		for _, a := range p.ancestors(id) {
			if set[a] {
				inside = true
				break
			}
		}
		if !inside {
			out = append(out, id)
		}
	}
	return out
}

func loadIDs(ctx context.Context, d Deps, in idsIn, min int) (penpot.Target, rawPage, error) {
	if err := in.validate(); err != nil {
		return penpot.Target{}, rawPage{}, err
	}
	if err := checkIDs(in.ShapeIDs, min); err != nil {
		return penpot.Target{}, rawPage{}, err
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return penpot.Target{}, rawPage{}, err
	}
	p, err := loadRaw(ctx, d, t)
	if err != nil {
		return penpot.Target{}, rawPage{}, err
	}
	for _, id := range in.ShapeIDs {
		if _, err := p.get(id, t); err != nil {
			return penpot.Target{}, rawPage{}, err
		}
	}
	return t, p, nil
}

func described(p rawPage, ids []string) []changed {
	out := make([]changed, 0, len(ids))
	for _, id := range ids {
		o := p.objects[id]
		out = append(out, changed{ShapeID: id, Name: str(o, "name"), Type: str(o, "type")})
	}
	return out
}

func moveShapes(ctx context.Context, d Deps, in moveIn) (changedOut, error) {
	if err := checkCoord("dx", in.DX); err != nil {
		return changedOut{}, err
	}
	if err := checkCoord("dy", in.DY); err != nil {
		return changedOut{}, err
	}
	t, p, err := loadIDs(ctx, d, in.idsIn, 1)
	if err != nil {
		return changedOut{}, err
	}
	ids := topmost(p, in.ShapeIDs)
	var changes []map[string]any
	for _, id := range ids {
		moved, err := moveTree(p, t.PageID, id, in.DX, in.DY)
		if err != nil {
			return changedOut{}, err
		}
		changes = append(changes, moved...)
	}
	if regs := refit(p, ids); len(regs) > 0 {
		changes = append(changes, penpot.RegObjects(t.PageID, regs))
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return changedOut{}, err
	}
	return changedOut{Target: t, Shapes: described(p, in.ShapeIDs)}, nil
}

func reparent(ctx context.Context, d Deps, in reparentIn) (changedOut, error) {
	if err := checkID("parent_id", in.ParentID); err != nil {
		return changedOut{}, err
	}
	if in.Index != nil && (*in.Index < 0 || *in.Index > 100_000) {
		return changedOut{}, fmt.Errorf("index must be 0 or more")
	}
	t, p, err := loadIDs(ctx, d, in.idsIn, 1)
	if err != nil {
		return changedOut{}, err
	}
	if in.ParentID != penpot.ZeroID {
		parent, err := p.get(in.ParentID, t)
		if err != nil {
			return changedOut{}, err
		}
		if !isContainer(str(parent, "type")) {
			return changedOut{}, fmt.Errorf("parent %q is a %s; only boards, groups and bools hold shapes", str(parent, "name"), str(parent, "type"))
		}
	}
	for _, id := range in.ShapeIDs {
		if id == in.ParentID {
			return changedOut{}, fmt.Errorf("a shape cannot go inside itself")
		}
		for _, a := range p.ancestors(in.ParentID) {
			if a == id {
				return changedOut{}, fmt.Errorf("%q cannot go inside its own child", str(p.objects[id], "name"))
			}
		}
	}
	ids := topmost(p, in.ShapeIDs)
	index := -1
	if in.Index != nil {
		index = *in.Index
	}
	regs := refit(p, append(append([]string{}, ids...), in.ParentID))
	if t := str(p.objects[in.ParentID], "type"); t == penpot.TypeGroup || t == penpot.TypeBool {
		regs = append([]string{in.ParentID}, regs...)
	}
	changes := []transit.Map{penpot.MovObjects(t.PageID, in.ParentID, ids, index)}
	if len(regs) > 0 {
		changes = append(changes, penpot.RegObjectsTransit(t.PageID, regs))
	}
	if err := d.Writer.ApplyTransit(ctx, t.FileID, changes); err != nil {
		return changedOut{}, err
	}
	return changedOut{Target: t, Shapes: described(p, in.ShapeIDs)}, nil
}

func deleteShapes(ctx context.Context, d Deps, in idsIn) (changedOut, error) {
	t, p, err := loadIDs(ctx, d, in, 1)
	if err != nil {
		return changedOut{}, err
	}
	ids := topmost(p, in.ShapeIDs)
	gone := map[string]bool{}
	var changes []map[string]any
	for _, id := range ids {
		changes = append(changes, penpot.DelObj(t.PageID, id))
		gone[id] = true
	}
	// A group or bool left with no children goes too; the others refit.
	var regs []string
	for _, a := range refit(p, ids) {
		left := 0
		for _, c := range children(p.objects[a]) {
			if !gone[c] {
				left++
			}
		}
		if left == 0 {
			changes = append(changes, penpot.DelObj(t.PageID, a))
			gone[a] = true
			ids = append(ids, a)
			continue
		}
		regs = append(regs, a)
	}
	if len(regs) > 0 {
		changes = append(changes, penpot.RegObjects(t.PageID, regs))
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return changedOut{}, err
	}
	return changedOut{Target: t, Shapes: described(p, ids)}, nil
}

func group(ctx context.Context, d Deps, in groupIn, isBool bool) (groupOut, error) {
	if err := checkName(in.Name); err != nil {
		return groupOut{}, err
	}
	if err := checkColor("fill", in.Fill); err != nil {
		return groupOut{}, err
	}
	op := orDefault(in.BoolType, "union")
	switch {
	case !isBool && (in.BoolType != "" || in.Fill != ""):
		return groupOut{}, fmt.Errorf("bool_type and fill are for create_bool")
	case isBool && op != "union" && op != "difference" && op != "intersection" && op != "exclude":
		return groupOut{}, fmt.Errorf("bool_type must be union, difference, intersection or exclude")
	}
	min := 1
	if isBool {
		min = 2
	}
	t, p, err := loadIDs(ctx, d, in.idsIn, min)
	if err != nil {
		return groupOut{}, err
	}
	parent := str(p.objects[in.ShapeIDs[0]], "parentId")
	for _, id := range in.ShapeIDs {
		if str(p.objects[id], "parentId") != parent {
			return groupOut{}, fmt.Errorf("the shapes must share a parent; move them first with reparent_shape")
		}
	}
	// Keep the parent's order and put the group where the top one was.
	pick := map[string]bool{}
	for _, id := range in.ShapeIDs {
		pick[id] = true
	}
	siblings := children(p.objects[parent])
	var ordered []string
	top := 0
	for i, id := range siblings {
		if pick[id] {
			ordered = append(ordered, id)
			top = i
		}
	}
	if len(ordered) != len(in.ShapeIDs) {
		return groupOut{}, fmt.Errorf("the page lists the shapes under no parent; reload and try again")
	}
	var pts []penpot.Point
	for _, id := range ordered {
		list, _ := p.objects[id]["points"].([]any)
		for _, v := range list {
			m, _ := v.(map[string]any)
			pts = append(pts, penpot.Point{X: num(m, "x"), Y: num(m, "y")})
		}
	}
	x, y, w, h := penpot.Bounds(pts)
	frame := parent
	if po := p.objects[parent]; str(po, "type") != penpot.TypeBoard {
		frame = str(po, "frameId")
	}
	id := penpot.NewID()
	box := penpot.Box{ID: id, ParentID: parent, FrameID: frame, Name: in.Name, X: x, Y: y, W: w, H: h}
	obj := penpot.GroupObject(box)
	if isBool {
		obj = penpot.BoolObject(box, op, orDefault(in.Fill, firstFill(p.objects[ordered[0]])))
	}
	// The group is added first (JSON), then the shapes move in (transit).
	if err := d.Writer.Apply(ctx, t.FileID, []map[string]any{penpot.AddObjAt(t.PageID, obj, top+1)}); err != nil {
		return groupOut{}, err
	}
	err = d.Writer.ApplyTransit(ctx, t.FileID, []transit.Map{
		penpot.MovObjects(t.PageID, id, ordered, -1),
		penpot.RegObjectsTransit(t.PageID, append([]string{id}, refit(p, []string{ordered[0]})...)),
	})
	if err != nil {
		// Do not leave an empty group behind.
		_ = d.Writer.Apply(ctx, t.FileID, []map[string]any{penpot.DelObj(t.PageID, id)})
		return groupOut{}, err
	}
	return groupOut{Target: t, ShapeID: id, ParentID: parent}, nil
}

func firstFill(o map[string]any) string {
	list, _ := o["fills"].([]any)
	if len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok && hexRE.MatchString(str(m, "fillColor")) {
			return str(m, "fillColor")
		}
	}
	return "#D9D9D9"
}

// allowedChanges are the change types apply_changes passes on. mov-objects
// and reorder-children are refused: Penpot ignores them over JSON.
var allowedChanges = map[string]bool{
	"add-obj": true, "mod-obj": true, "del-obj": true, "reg-objects": true,
	"add-page": true, "mod-page": true, "del-page": true, "mov-page": true,
	"add-color": true, "mod-color": true, "del-color": true,
	"add-typography": true, "mod-typography": true, "del-typography": true,
	"add-component": true, "mod-component": true, "del-component": true,
	"add-media": true, "mod-media": true, "del-media": true,
	"set-guide": true, "set-flow": true, "set-comment-thread-position": true,
	"set-token": true, "set-token-set": true, "set-token-theme": true,
	"set-plugin-data": true,
}

const maxApplyChanges = 500

func applyChanges(ctx context.Context, d Deps, in applyIn) (applyOut, error) {
	if err := in.validate(); err != nil {
		return applyOut{}, err
	}
	if len(in.Changes) == 0 || len(in.Changes) > maxApplyChanges {
		return applyOut{}, fmt.Errorf("changes must hold 1 to %d items", maxApplyChanges)
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return applyOut{}, err
	}
	for i, c := range in.Changes {
		typ := str(c, "type")
		switch {
		case typ == "mov-objects" || typ == "reorder-children":
			return applyOut{}, fmt.Errorf("changes[%d]: %s is ignored by Penpot over JSON; use reparent_shape", i, typ)
		case !allowedChanges[typ]:
			return applyOut{}, fmt.Errorf("changes[%d]: unknown change type %q", i, typ)
		}
		if id := str(c, "id"); id == penpot.ZeroID && (typ == "del-obj" || typ == "add-obj" || typ == "mod-obj") {
			return applyOut{}, fmt.Errorf("changes[%d]: the root frame cannot be changed", i)
		}
		if pid, ok := c["pageId"]; ok {
			if s, _ := pid.(string); !uuidRE.MatchString(s) {
				return applyOut{}, fmt.Errorf("changes[%d]: pageId must be a lowercase UUID", i)
			}
		} else if typ == "add-obj" || typ == "mod-obj" || typ == "del-obj" || typ == "reg-objects" {
			c["pageId"] = t.PageID
		}
	}
	if err := d.Writer.Apply(ctx, t.FileID, in.Changes); err != nil {
		return applyOut{}, err
	}
	return applyOut{Target: t, Applied: len(in.Changes)}, nil
}
