package tools

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

var (
	uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	hexRE  = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	refRE  = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
)

const (
	maxCoord    = 1_000_000
	maxSize     = 100_000
	maxNameLen  = 200
	maxTextLen  = 10_000
	maxDrawSize = 500
)

// target is where a shape goes: a file, an optional page, an optional board.
type target struct {
	FileID   string `json:"file_id" jsonschema:"the file to draw in"`
	PageID   string `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	ParentID string `json:"parent_id,omitempty" jsonschema:"a board to draw inside; the page root when empty"`
}

func (t target) validate() error {
	if err := checkID("file_id", t.FileID); err != nil {
		return err
	}
	if t.PageID != "" {
		if err := checkID("page_id", t.PageID); err != nil {
			return err
		}
	}
	if t.ParentID != "" {
		if err := checkID("parent_id", t.ParentID); err != nil {
			return err
		}
	}
	return nil
}

type shapeIn struct {
	target
	Name   string  `json:"name" jsonschema:"layer name"`
	X      float64 `json:"x" jsonschema:"left edge, absolute page pixels (also inside a board)"`
	Y      float64 `json:"y" jsonschema:"top edge, absolute page pixels"`
	Width  float64 `json:"width" jsonschema:"width in pixels, above 0"`
	Height float64 `json:"height" jsonschema:"height in pixels, above 0"`
	Fill   string  `json:"fill,omitempty" jsonschema:"fill colour as #RRGGBB; white for boards and grey for other shapes when empty"`
	Radius float64 `json:"radius,omitempty" jsonschema:"corner radius in pixels (boards and rectangles)"`
}

type textIn struct {
	target
	Name string `json:"name,omitempty" jsonschema:"layer name; the text itself when empty"`
	textFields
}

// textFields are a text shape's position, content and typography.
type textFields struct {
	Text          string  `json:"text,omitempty" jsonschema:"the text (required for text shapes); a newline starts a new paragraph"`
	X             float64 `json:"x" jsonschema:"left edge, absolute page pixels"`
	Y             float64 `json:"y" jsonschema:"top edge, absolute page pixels"`
	Width         float64 `json:"width,omitempty" jsonschema:"box width; when empty the box fits the text on one line per paragraph"`
	FontFamily    string  `json:"font_family,omitempty" jsonschema:"a Google font name like Inter, or Source Sans Pro (the default)"`
	FontSize      float64 `json:"font_size,omitempty" jsonschema:"size in pixels, 3 to 1000, default 16"`
	FontWeight    int     `json:"font_weight,omitempty" jsonschema:"100 to 900 in steps of 100, default 400"`
	Italic        bool    `json:"italic,omitempty"`
	LineHeight    float64 `json:"line_height,omitempty" jsonschema:"multiple of the font size, default 1.2"`
	LetterSpacing float64 `json:"letter_spacing,omitempty" jsonschema:"pixels between letters"`
	Color         string  `json:"color,omitempty" jsonschema:"text colour as #RRGGBB, default #000000"`
	Align         string  `json:"align,omitempty" jsonschema:"left, center, right or justify"`
}

type shapeOut struct {
	penpot.Target
	ShapeID  string  `json:"shape_id"`
	ParentID string  `json:"parent_id"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
}

func registerDraw(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_board",
		Description: "Add a board (frame) to a Penpot page, at the root or inside another board. " +
			"Returns the new shape id and the file and page it landed in.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shapeIn) (*mcp.CallToolResult, shapeOut, error) {
		out, err := createBox(ctx, d, "board", in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_rect",
		Description: "Add a rectangle to a Penpot page, at the root or inside a board. " +
			"Returns the new shape id and the file and page it landed in.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shapeIn) (*mcp.CallToolResult, shapeOut, error) {
		out, err := createBox(ctx, d, "rect", in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_ellipse",
		Description: "Add an ellipse that fills the given box, at the root or inside a board.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shapeIn) (*mcp.CallToolResult, shapeOut, error) {
		out, err := createBox(ctx, d, "ellipse", in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_text",
		Description: "Add a text box. Without width the box fits the text; with width the text wraps " +
			"and the height grows. Sizes are measured with the real font. Returns the box size.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in textIn) (*mcp.CallToolResult, shapeOut, error) {
		if err := in.validate(); err != nil {
			return nil, shapeOut{}, err
		}
		node := drawNode{Type: "text", Name: in.Name, textFields: in.textFields}
		out, err := drawNodes(ctx, d, in.target, []drawNode{node})
		if err != nil {
			return nil, shapeOut{}, err
		}
		return nil, out.single(), nil
	})

	registerDrawTree(s, d)
}

func createBox(ctx context.Context, d Deps, kind string, in shapeIn) (shapeOut, error) {
	if err := in.validate(); err != nil {
		return shapeOut{}, err
	}
	node := drawNode{
		Type: kind, Name: in.Name, Fill: in.Fill, Radius: in.Radius,
		textFields: textFields{X: in.X, Y: in.Y, Width: in.Width},
		Height:     in.Height,
	}
	if err := node.validate(); err != nil {
		return shapeOut{}, err
	}
	out, err := drawNodes(ctx, d, in.target, []drawNode{node})
	if err != nil {
		return shapeOut{}, err
	}
	return out.single(), nil
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
	if o.Type != penpot.TypeBoard {
		return fmt.Errorf("parent %q is a %s; only boards can hold shapes here", o.Name, o.Type)
	}
	return nil
}

func checkName(name string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > maxNameLen {
		return fmt.Errorf("name must be 1 to %d characters", maxNameLen)
	}
	return nil
}

func checkCoord(label string, v float64) error {
	if math.IsNaN(v) || math.Abs(v) > maxCoord {
		return fmt.Errorf("%s must be within ±%d", label, maxCoord)
	}
	return nil
}

func checkSize(label string, v float64) error {
	if math.IsNaN(v) || v <= 0 || v > maxSize {
		return fmt.Errorf("%s must be above 0 and at most %d", label, maxSize)
	}
	return nil
}

func checkColor(label, v string) error {
	if v != "" && !hexRE.MatchString(v) {
		return fmt.Errorf("%s must look like #RRGGBB", label)
	}
	return nil
}

func checkID(label, v string) error {
	if !uuidRE.MatchString(v) {
		return fmt.Errorf("%s must be a lowercase UUID", label)
	}
	return nil
}

// face resolves a node's font, for measuring and for Penpot's ids.
func (n drawNode) face() (fonts.Face, error) {
	return fonts.Resolve(n.FontFamily, n.FontWeight, n.Italic)
}
