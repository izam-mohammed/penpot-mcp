package tools

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

var (
	uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	hexRE  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

const (
	maxCoord   = 1_000_000
	maxSize    = 100_000
	maxNameLen = 200
)

type shapeIn struct {
	FileID   string  `json:"file_id" jsonschema:"the file to draw in"`
	PageID   string  `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	ParentID string  `json:"parent_id,omitempty" jsonschema:"a board to draw inside; the page root when empty"`
	Name     string  `json:"name" jsonschema:"layer name"`
	X        float64 `json:"x" jsonschema:"left edge, absolute page pixels (also inside a board)"`
	Y        float64 `json:"y" jsonschema:"top edge, absolute page pixels"`
	Width    float64 `json:"width" jsonschema:"width in pixels, above 0"`
	Height   float64 `json:"height" jsonschema:"height in pixels, above 0"`
	Fill     string  `json:"fill,omitempty" jsonschema:"fill colour as #RRGGBB; white for boards and grey for rectangles when empty"`
	Radius   float64 `json:"radius,omitempty" jsonschema:"corner radius in pixels"`
}

type shapeOut struct {
	penpot.Target
	ShapeID  string `json:"shape_id"`
	ParentID string `json:"parent_id"`
}

func registerDraw(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_board",
		Description: "Add a board (frame) to a Penpot page, at the root or inside another board. " +
			"Returns the new shape id and the file and page it landed in.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shapeIn) (*mcp.CallToolResult, shapeOut, error) {
		out, err := createShape(ctx, d, "frame", in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_rect",
		Description: "Add a rectangle to a Penpot page, at the root or inside a board. " +
			"Returns the new shape id and the file and page it landed in.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shapeIn) (*mcp.CallToolResult, shapeOut, error) {
		out, err := createShape(ctx, d, "rect", in)
		return nil, out, err
	})
}

func createShape(ctx context.Context, d Deps, kind string, in shapeIn) (shapeOut, error) {
	if err := in.validate(); err != nil {
		return shapeOut{}, err
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return shapeOut{}, err
	}
	parent := penpot.ZeroID
	if in.ParentID != "" && in.ParentID != penpot.ZeroID {
		if err := checkParent(ctx, d, t, in.ParentID); err != nil {
			return shapeOut{}, err
		}
		parent = in.ParentID
	}

	id := penpot.NewID()
	obj := shapeObject(kind, id, parent, in)
	change := map[string]any{
		"type":     "add-obj",
		"id":       id,
		"pageId":   t.PageID,
		"frameId":  parent,
		"parentId": parent,
		"obj":      obj,
	}
	if err := d.Writer.Apply(ctx, t.FileID, []map[string]any{change}); err != nil {
		return shapeOut{}, err
	}
	return shapeOut{Target: t, ShapeID: id, ParentID: parent}, nil
}

// checkParent makes sure the parent exists on the page and is a board.
func checkParent(ctx context.Context, d Deps, t penpot.Target, parentID string) error {
	var p penpot.Page
	if err := d.Client.Call(ctx, "get-page", map[string]string{
		"fileId": t.FileID, "pageId": t.PageID, "objectId": parentID,
	}, &p); err != nil {
		return err
	}
	o, ok := p.Objects[parentID]
	if !ok {
		return fmt.Errorf("parent %s is not on page %q", parentID, t.PageName)
	}
	if o.Type != "frame" {
		return fmt.Errorf("parent %q is a %s; only boards can hold shapes here", o.Name, o.Type)
	}
	return nil
}

func shapeObject(kind, id, parent string, in shapeIn) map[string]any {
	g := penpot.Box(in.X, in.Y, in.Width, in.Height)
	fill := in.Fill
	if fill == "" {
		fill = "#FFFFFF"
		if kind == "rect" {
			fill = "#D9D9D9"
		}
	}
	obj := map[string]any{
		"id":               id,
		"type":             kind,
		"name":             in.Name,
		"x":                g.X,
		"y":                g.Y,
		"width":            g.Width,
		"height":           g.Height,
		"rotation":         g.Rotation,
		"selrect":          g.Selrect,
		"points":           g.Points,
		"transform":        g.Transform,
		"transformInverse": g.TransformInverse,
		"parentId":         parent,
		"frameId":          parent,
		"fills":            []map[string]any{{"fillColor": strings.ToUpper(fill), "fillOpacity": 1}},
		"strokes":          []any{},
		"r1":               in.Radius,
		"r2":               in.Radius,
		"r3":               in.Radius,
		"r4":               in.Radius,
	}
	if kind == "frame" {
		obj["shapes"] = []string{}
		obj["hideFillOnExport"] = false
	}
	return obj
}

func (in shapeIn) validate() error {
	if err := checkID("file_id", in.FileID); err != nil {
		return err
	}
	if in.PageID != "" {
		if err := checkID("page_id", in.PageID); err != nil {
			return err
		}
	}
	if in.ParentID != "" {
		if err := checkID("parent_id", in.ParentID); err != nil {
			return err
		}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLen {
		return fmt.Errorf("name must be 1 to %d characters", maxNameLen)
	}
	for _, v := range []struct {
		label    string
		val, max float64
	}{{"x", in.X, maxCoord}, {"y", in.Y, maxCoord}} {
		if math.IsNaN(v.val) || math.Abs(v.val) > v.max {
			return fmt.Errorf("%s must be within ±%d", v.label, maxCoord)
		}
	}
	for _, v := range []struct {
		label string
		val   float64
	}{{"width", in.Width}, {"height", in.Height}} {
		if math.IsNaN(v.val) || v.val <= 0 || v.val > maxSize {
			return fmt.Errorf("%s must be above 0 and at most %d", v.label, maxSize)
		}
	}
	if in.Fill != "" && !hexRE.MatchString(in.Fill) {
		return fmt.Errorf("fill must look like #RRGGBB")
	}
	if math.IsNaN(in.Radius) || in.Radius < 0 || in.Radius > maxSize {
		return fmt.Errorf("radius must be 0 to %d", maxSize)
	}
	return nil
}

func checkID(label, v string) error {
	if !uuidRE.MatchString(v) {
		return fmt.Errorf("%s must be a lowercase UUID", label)
	}
	return nil
}
