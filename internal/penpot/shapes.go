package penpot

import (
	"strconv"
	"strings"
)

// Shape types as Penpot stores them: a board is a "frame", an ellipse a
// "circle".
const (
	TypeBoard   = "frame"
	TypeRect    = "rect"
	TypeEllipse = "circle"
	TypeText    = "text"
)

// Box is the common part of every shape object.
type Box struct {
	ID       string
	ParentID string // the containing board, or ZeroID
	Name     string
	X, Y     float64
	W, H     float64
}

func (b Box) base(kind string) map[string]any {
	g := b.geometry()
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
		"frameId":          b.ParentID,
	}
}

func (b Box) geometry() Geometry { return BoxGeometry(b.X, b.Y, b.W, b.H) }

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
