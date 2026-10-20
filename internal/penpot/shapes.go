package penpot

import (
	"strconv"
	"strings"

	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

// Shape types as Penpot stores them: a board is a "frame", an ellipse a
// "circle".
const (
	TypeBoard   = "frame"
	TypeRect    = "rect"
	TypeEllipse = "circle"
	TypeText    = "text"
	TypePath    = "path"
	TypeGroup   = "group"
	TypeBool    = "bool"
)

// Box is the common part of every shape object.
type Box struct {
	ID       string
	ParentID string // the containing board, group or bool, or ZeroID
	FrameID  string // the nearest board; ParentID when empty
	Name     string
	X, Y     float64
	W, H     float64
	Rotation float64 // degrees clockwise about the centre
}

func (b Box) base(kind string) map[string]any {
	g := b.geometry()
	frame := b.FrameID
	if frame == "" {
		frame = b.ParentID
	}
	return map[string]any{
		"id":               b.ID,
		"type":             kind,
		"name":             b.Name,
		"x":                g.X,
		"y":                g.Y,
		"width":            g.Width,
		"height":           g.Height,
		"rotation":         g.Rotation,
		"selrect":          g.Selrect,
		"points":           g.Points,
		"transform":        g.Transform,
		"transformInverse": g.TransformInverse,
		"parentId":         b.ParentID,
		"frameId":          frame,
	}
}

func (b Box) geometry() Geometry { return RotatedGeometry(b.X, b.Y, b.W, b.H, b.Rotation) }

func fills(hex string) []map[string]any {
	if hex == "" {
		return []map[string]any{}
	}
	return []map[string]any{{"fillColor": strings.ToUpper(hex), "fillOpacity": 1}}
}

func radii(obj map[string]any, r float64) {
	for _, k := range []string{"r1", "r2", "r3", "r4"} {
		obj[k] = r
	}
}

// BoardObject is a board with a fill and corner radius. Children are clipped
// unless clip is false.
func BoardObject(b Box, fill string, radius float64, clip bool) map[string]any {
	obj := b.base(TypeBoard)
	obj["fills"] = fills(fill)
	obj["strokes"] = []any{}
	obj["shapes"] = []string{}
	obj["hideFillOnExport"] = false
	obj["showContent"] = !clip
	radii(obj, radius)
	return obj
}

// RectObject is a rectangle with a fill and corner radius.
func RectObject(b Box, fill string, radius float64) map[string]any {
	obj := b.base(TypeRect)
	obj["fills"] = fills(fill)
	obj["strokes"] = []any{}
	radii(obj, radius)
	return obj
}

// EllipseObject is an ellipse filling its box.
func EllipseObject(b Box, fill string) map[string]any {
	obj := b.base(TypeEllipse)
	obj["fills"] = fills(fill)
	obj["strokes"] = []any{}
	return obj
}

// TextStyle is one run's typography. Penpot stores the numbers as strings.
type TextStyle struct {
	FontID        string
	FontFamily    string
	FontVariantID string
	FontWeight    string
	FontStyle     string
	FontSize      float64
	LineHeight    float64 // multiple of the font size
	LetterSpacing float64 // px
	Color         string  // #RRGGBB
	Align         string  // left, center, right, justify
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func (s TextStyle) attrs() map[string]any {
	return map[string]any{
		"fontId":         s.FontID,
		"fontFamily":     s.FontFamily,
		"fontVariantId":  s.FontVariantID,
		"fontSize":       num(s.FontSize),
		"fontWeight":     s.FontWeight,
		"fontStyle":      s.FontStyle,
		"lineHeight":     num(s.LineHeight),
		"letterSpacing":  num(s.LetterSpacing),
		"textTransform":  "none",
		"textDecoration": "none",
		"fills":          fills(s.Color),
	}
}

// TextContent is Penpot's content tree for plain text: one paragraph per
// line, every node carrying the style.
func TextContent(text string, s TextStyle) map[string]any {
	align := s.Align
	if align == "" {
		align = "left"
	}
	var paras []map[string]any
	for _, line := range strings.Split(text, "\n") {
		leaf := s.attrs()
		leaf["text"] = line
		p := s.attrs()
		p["type"] = "paragraph"
		p["textAlign"] = align
		p["textDirection"] = "ltr"
		p["children"] = []map[string]any{leaf}
		paras = append(paras, p)
	}
	return map[string]any{
		"type":          "root",
		"verticalAlign": "top",
		"children": []map[string]any{{
			"type":     "paragraph-set",
			"children": paras,
		}},
	}
}

// TextObject is a text shape. grow is "auto-width", "auto-height" or
// "fixed".
func TextObject(b Box, text string, s TextStyle, grow string) map[string]any {
	obj := b.base(TypeText)
	obj["content"] = TextContent(text, s)
	obj["growType"] = grow
	return obj
}

// Stroke is a solid stroke.
func Stroke(hex string, width float64) map[string]any {
	return map[string]any{
		"strokeColor": strings.ToUpper(hex), "strokeOpacity": 1, "strokeWidth": width,
		"strokeStyle": "solid", "strokeAlignment": "center",
	}
}

// PathObject is a path. Its box is the bounds of segs; fill may be empty for
// an open line.
func PathObject(b Box, segs []Segment, fill string, strokes []map[string]any) map[string]any {
	b.X, b.Y, b.W, b.H = PathBounds(segs)
	obj := b.base(TypePath)
	obj["content"] = segs
	obj["fills"] = fills(fill)
	obj["strokes"] = strokes
	return obj
}

// GroupObject is an empty group; children are added under it and a
// reg-objects change then fits its box to them.
func GroupObject(b Box) map[string]any {
	obj := b.base(TypeGroup)
	obj["shapes"] = []string{}
	obj["fills"] = []any{}
	obj["strokes"] = []any{}
	return obj
}

// BoolObject is an empty boolean shape; reg-objects computes its content
// from its children.
func BoolObject(b Box, op, fill string) map[string]any {
	obj := b.base(TypeBool)
	obj["shapes"] = []string{}
	obj["boolType"] = op
	obj["content"] = []any{}
	obj["fills"] = fills(fill)
	obj["strokes"] = []any{}
	return obj
}

// Media is an uploaded file media object.
type Media struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Mtype  string  `json:"mtype"`
}

// ImageObject is a rectangle filled with an uploaded image.
func ImageObject(b Box, m Media, radius float64) map[string]any {
	obj := b.base(TypeRect)
	obj["fills"] = []map[string]any{{
		"fillOpacity": 1,
		"fillImage": map[string]any{
			"id": m.ID, "name": m.Name, "width": m.Width, "height": m.Height,
			"mtype": m.Mtype, "keepAspectRatio": true,
		},
	}}
	obj["strokes"] = []any{}
	radii(obj, radius)
	return obj
}

// AddObjAt is AddObj at a position in the parent's children (0 is the
// bottom).
func AddObjAt(pageID string, obj map[string]any, index int) map[string]any {
	c := AddObj(pageID, obj)
	c["index"] = index
	return c
}

// ModObj is the change that assigns attrs to a shape. assign is decoded with
// the shape schema, so it is safe from JSON (set is not).
func ModObj(pageID, id string, attrs map[string]any) map[string]any {
	return map[string]any{
		"type": "mod-obj", "id": id, "pageId": pageID,
		"operations": []map[string]any{{"type": "assign", "value": attrs}},
	}
}

// DelObj is the change that deletes a shape and its children.
func DelObj(pageID, id string) map[string]any {
	return map[string]any{"type": "del-obj", "id": id, "pageId": pageID}
}

// RegObjects is the change that refits groups and bools to their children.
func RegObjects(pageID string, ids []string) map[string]any {
	return map[string]any{"type": "reg-objects", "pageId": pageID, "shapes": ids}
}

// MovObjects is the transit change that moves shapes under parentID, at
// index when index >= 0, else on top.
func MovObjects(pageID, parentID string, ids []string, index int) transit.Map {
	shapes := make([]transit.UUID, len(ids))
	for i, id := range ids {
		shapes[i] = transit.UUID(id)
	}
	m := transit.Map{
		{K: "type", V: transit.Keyword("mov-objects")},
		{K: "page-id", V: transit.UUID(pageID)},
		{K: "parent-id", V: transit.UUID(parentID)},
		{K: "shapes", V: shapes},
	}
	if index >= 0 {
		m = append(m, transit.Pair{K: "index", V: index})
	}
	return m
}

// RegObjectsTransit is RegObjects for a transit batch.
func RegObjectsTransit(pageID string, ids []string) transit.Map {
	shapes := make([]transit.UUID, len(ids))
	for i, id := range ids {
		shapes[i] = transit.UUID(id)
	}
	return transit.Map{
		{K: "type", V: transit.Keyword("reg-objects")},
		{K: "page-id", V: transit.UUID(pageID)},
		{K: "shapes", V: shapes},
	}
}

// AddObj is the update-file change that adds obj under its parent.
func AddObj(pageID string, obj map[string]any) map[string]any {
	return map[string]any{
		"type":     "add-obj",
		"id":       obj["id"],
		"pageId":   pageID,
		"frameId":  obj["frameId"],
		"parentId": obj["parentId"],
		"obj":      obj,
	}
}
