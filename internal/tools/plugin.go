package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/mcproute"
	"github.com/izam-mohammed/penpot-mcp/internal/palette"
)

// maxCode caps execute_code's input.
const maxCode = 100_000

type execIn struct {
	FileID string `json:"file_id" jsonschema:"the file the code is for; the call is refused unless this file is open in the connected tab"`
	Code   string `json:"code" jsonschema:"a Plugin API function body: await allowed, return gives the result"`
}

type tabFileIn struct {
	FileID string `json:"file_id" jsonschema:"the file open in the connected tab"`
}

type tabShapesIn struct {
	FileID   string   `json:"file_id" jsonschema:"the file open in the connected tab"`
	ShapeIDs []string `json:"shape_ids" jsonschema:"the shapes, on the tab's current page"`
}

type markupIn struct {
	tabShapesIn
	Type string `json:"type,omitempty" jsonschema:"html (default) or svg"`
}

type styleIn struct {
	tabShapesIn
	WithChildren bool `json:"with_children,omitempty" jsonschema:"include the children's CSS"`
}

type swapIn struct {
	FileID      string `json:"file_id" jsonschema:"the file open in the connected tab"`
	ShapeID     string `json:"shape_id" jsonschema:"the component copy"`
	ComponentID string `json:"component_id" jsonschema:"the component to swap to, from list_library"`
}

type variantIn struct {
	FileID   string `json:"file_id" jsonschema:"the file open in the connected tab"`
	ShapeID  string `json:"shape_id" jsonschema:"the variant copy"`
	Property string `json:"property" jsonschema:"the variant property to change"`
	Value    string `json:"value" jsonschema:"its new value"`
}

type viewportIn struct {
	FileID   string   `json:"file_id" jsonschema:"the file open in the connected tab"`
	Zoom     float64  `json:"zoom,omitempty" jsonschema:"new zoom, 0.01 to 200; unchanged when empty"`
	CenterX  *float64 `json:"center_x,omitempty" jsonschema:"page x to centre on"`
	CenterY  *float64 `json:"center_y,omitempty" jsonschema:"page y to centre on"`
	ShapeIDs []string `json:"shape_ids,omitempty" jsonschema:"zoom to fit these shapes instead"`
}

type apiInfoIn struct {
	Type   string `json:"type" jsonschema:"a Plugin API type, such as Shape, Board or Penpot"`
	Member string `json:"member,omitempty" jsonschema:"one member of that type"`
}

type svgIn struct {
	FileID string  `json:"file_id" jsonschema:"the file open in the connected tab"`
	SVG    string  `json:"svg,omitempty" jsonschema:"SVG markup"`
	Path   string  `json:"path,omitempty" jsonschema:"an .svg file inside the working directory, instead of svg"`
	X      float64 `json:"x" jsonschema:"left edge, absolute page pixels"`
	Y      float64 `json:"y" jsonschema:"top edge, absolute page pixels"`
	Name   string  `json:"name,omitempty" jsonschema:"layer name"`
}

type tabOut struct {
	mcproute.Open
	Result any    `json:"result,omitempty" jsonschema:"what the code returned"`
	Log    string `json:"log,omitempty"`

	raw json.RawMessage
}

type textOut struct {
	mcproute.Open
	Text string `json:"text"`
}

// js encodes v as a JavaScript literal (JSON is a subset), so ids and
// strings reach the tab as data, never as code.
func js(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// shapesJS is code that resolves ids on the current page into `shapes`,
// failing on any that is missing.
func shapesJS(ids []string) string {
	return `const ids = ` + js(ids) + `;
const shapes = ids.map(id => penpot.currentPage.getShapeById(id));
const missing = ids.filter((id, i) => !shapes[i]);
if (missing.length) throw new Error("not on the open page: " + missing.join(", "));
`
}

// need is code that fails clearly when this Penpot's Plugin API lacks a
// method.
func need(obj, method string) string {
	return fmt.Sprintf("if (typeof %s.%s !== 'function') throw new Error('this Penpot has no %s');\n", obj, method, method)
}

// inTab guards the file, then runs code in the tab; writes hold the API
// writer's lock and refresh the file's revision after.
func inTab(ctx context.Context, d Deps, fileID, code string, write bool) (tabOut, error) {
	if d.Route == nil {
		return tabOut{}, errNoRoute
	}
	if err := checkID("file_id", fileID); err != nil {
		return tabOut{}, err
	}
	name, err := d.Writer.FileName(ctx, fileID)
	if err != nil {
		return tabOut{}, err
	}
	var out tabOut
	run := func() error {
		open, err := d.Route.Guard(ctx, fileID, name)
		if err != nil {
			return err
		}
		res, err := d.Route.Exec(ctx, code)
		if err != nil {
			return err
		}
		out = tabOut{Open: open, Log: res.Log, raw: res.Value}
		if len(res.Value) > 0 {
			_ = json.Unmarshal(res.Value, &out.Result)
		}
		return nil
	}
	if write {
		err = d.Writer.Exclusive(fileID, run)
	} else {
		err = run()
	}
	return out, err
}

var errNoRoute = fmt.Errorf("this needs the Penpot MCP route: set PENPOT_MCP_URL (Your account > Integrations > MCP Server) and connect the plugin in a tab")

func registerPlugin(s *mcp.Server, d Deps) {
	if !d.anyRoute() {
		return
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "execute_code",
		Description: "Last resort: run Plugin API JavaScript in the connected Penpot tab, for what no other tool does. " +
			"Refused unless file_id is the file open in the tab. Prefer the API tools: they need no tab and are checked.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in execIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		if strings.TrimSpace(in.Code) == "" || len(in.Code) > maxCode {
			return nil, tabOut{}, fmt.Errorf("code must be 1 to %d characters", maxCode)
		}
		out, err := inTab(ctx, d, in.FileID, in.Code, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "current_file",
		Description: "Which file and page the connected Penpot tab has open.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, mcproute.Open, error) {
		d := d.pick(ctx)
		if d.Route == nil {
			return nil, mcproute.Open{}, errNoRoute
		}
		o, err := d.Route.Current(ctx)
		return nil, o, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_selection",
		Description: "The shapes selected in the connected tab: id, name and type.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabFileIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		out, err := inTab(ctx, d, in.FileID, `return penpot.selection.map(s => ({id: s.id, name: s.name, type: s.type}));`, false)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "reflow",
		Description: "Let the connected tab lay out flex and grid boards and size text now, after API writes. " +
			"Without a tab, Penpot does this when the file is next opened.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabFileIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		out, err := inTab(ctx, d, in.FileID, need("penpot", "waitForLayoutUpdate")+`await penpot.waitForLayoutUpdate(); return true;`, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "generate_markup",
		Description: "HTML or SVG markup for shapes, generated by Penpot in the connected tab.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in markupIn) (*mcp.CallToolResult, textOut, error) {
		d := d.pick(ctx)
		typ := orDefault(in.Type, "html")
		if typ != "html" && typ != "svg" {
			return nil, textOut{}, fmt.Errorf("type must be html or svg")
		}
		return tabText(ctx, d, in.tabShapesIn, need("penpot", "generateMarkup")+`return penpot.generateMarkup(shapes, {type: `+js(typ)+`});`)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "generate_style",
		Description: "CSS for shapes, generated by Penpot in the connected tab.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in styleIn) (*mcp.CallToolResult, textOut, error) {
		d := d.pick(ctx)
		return tabText(ctx, d, in.tabShapesIn, need("penpot", "generateStyle")+
			`return penpot.generateStyle(shapes, {type: "css", withChildren: `+js(in.WithChildren)+`});`)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "generate_font_faces",
		Description: "@font-face CSS for the fonts the shapes use.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabShapesIn) (*mcp.CallToolResult, textOut, error) {
		d := d.pick(ctx)
		return tabText(ctx, d, in, need("penpot", "generateFontFaces")+`return await penpot.generateFontFaces(shapes);`)
	})

	for _, op := range []struct{ name, method, desc string }{
		{"detach", "detach", "Detach component copies: they become plain shapes and stop following the main."},
		{"reset_overrides", "resetOverrides", "Reset component copies to their main: every override on them is lost."},
	} {
		mcp.AddTool(s, &mcp.Tool{Name: op.name, Description: op.desc, Annotations: destructive}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabShapesIn) (*mcp.CallToolResult, tabOut, error) {
			d := d.pick(ctx)
			if err := checkIDs(in.ShapeIDs, 1); err != nil {
				return nil, tabOut{}, err
			}
			code := shapesJS(in.ShapeIDs) + `for (const s of shapes) { ` + need("s", op.method) + ` s.` + op.method + `(); }
return shapes.length;`
			out, err := inTab(ctx, d, in.FileID, code, true)
			return nil, out, err
		})
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "swap_component",
		Description: "Swap a component copy for a copy of another component, keeping its place.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in swapIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		if err := checkID("shape_id", in.ShapeID); err != nil {
			return nil, tabOut{}, err
		}
		if err := checkID("component_id", in.ComponentID); err != nil {
			return nil, tabOut{}, err
		}
		code := shapesJS([]string{in.ShapeID}) + `const comp = penpot.library.local.components.find(c => c.id === ` + js(in.ComponentID) + `);
if (!comp) throw new Error("no component " + ` + js(in.ComponentID) + `);
` + need("shapes[0]", "swapComponent") + `shapes[0].swapComponent(comp); return true;`
		out, err := inTab(ctx, d, in.FileID, code, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "switch_variant",
		Description: "Switch a variant copy to another value of one of its properties.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in variantIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		if err := checkID("shape_id", in.ShapeID); err != nil {
			return nil, tabOut{}, err
		}
		if err := checkName(in.Property); err != nil {
			return nil, tabOut{}, fmt.Errorf("property: %w", err)
		}
		if err := checkName(in.Value); err != nil {
			return nil, tabOut{}, fmt.Errorf("value: %w", err)
		}
		code := shapesJS([]string{in.ShapeID}) + need("shapes[0]", "switchVariant") +
			`shapes[0].switchVariant(` + js(in.Property) + `, ` + js(in.Value) + `); return true;`
		out, err := inTab(ctx, d, in.FileID, code, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "combine_as_variants",
		Description: "Combine component main boards into one component with variants.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabShapesIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		if err := checkIDs(in.ShapeIDs, 2); err != nil {
			return nil, tabOut{}, err
		}
		code := shapesJS(in.ShapeIDs) + need("shapes[0]", "combineAsVariants") +
			`const v = shapes[0].combineAsVariants(ids.slice(1)); return v ? {id: v.id, name: v.name} : true;`
		out, err := inTab(ctx, d, in.FileID, code, true)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "active_users",
		Description: "Who else has the file open in Penpot right now.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in tabFileIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		out, err := inTab(ctx, d, in.FileID, `const users = penpot.activeUsers || [];
return users.map(u => ({id: u.id, name: u.name, page: u.position ? u.position.page : undefined}));`, false)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "viewport",
		Description: "Read the tab's viewport, or move it: zoom, centre, or fit shapes into view.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in viewportIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		code := ""
		if len(in.ShapeIDs) > 0 {
			if err := checkIDs(in.ShapeIDs, 1); err != nil {
				return nil, tabOut{}, err
			}
			code += shapesJS(in.ShapeIDs) + `penpot.viewport.zoomIntoView(shapes);
`
		}
		if in.Zoom != 0 {
			if in.Zoom < 0.01 || in.Zoom > 200 {
				return nil, tabOut{}, fmt.Errorf("zoom must be 0.01 to 200")
			}
			code += `penpot.viewport.zoom = ` + js(in.Zoom) + `;
`
		}
		if (in.CenterX == nil) != (in.CenterY == nil) {
			return nil, tabOut{}, fmt.Errorf("give center_x and center_y together")
		}
		if in.CenterX != nil {
			code += `penpot.viewport.center = {x: ` + js(*in.CenterX) + `, y: ` + js(*in.CenterY) + `};
`
		}
		code += `const v = penpot.viewport; return {zoom: v.zoom, center: v.center, bounds: v.bounds};`
		out, err := inTab(ctx, d, in.FileID, code, false)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "api_info",
		Description: "Penpot's Plugin API documentation for one type or member, from the official server.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in apiInfoIn) (*mcp.CallToolResult, textOut, error) {
		d := d.pick(ctx)
		if err := checkName(in.Type); err != nil {
			return nil, textOut{}, fmt.Errorf("type: %w", err)
		}
		if d.Route == nil {
			return nil, textOut{}, errNoRoute
		}
		args := map[string]any{"type": in.Type}
		if in.Member != "" {
			args["member"] = in.Member
		}
		t, err := d.Route.Call(ctx, "penpot_api_info", args)
		return nil, textOut{Text: t}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "import_svg",
		Description: "Turn SVG markup into editable Penpot shapes with Penpot's own SVG parser, in the connected tab. " +
			"For a single path without a tab, create_path does it through the API.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in svgIn) (*mcp.CallToolResult, tabOut, error) {
		d := d.pick(ctx)
		svg := in.SVG
		switch {
		case svg != "" && in.Path != "":
			return nil, tabOut{}, fmt.Errorf("give svg or path, not both")
		case in.Path != "":
			if !strings.EqualFold(filepath.Ext(in.Path), ".svg") {
				return nil, tabOut{}, fmt.Errorf("path must be an .svg file")
			}
			data, err := palette.Open(d.Root, in.Path)
			if err != nil {
				return nil, tabOut{}, err
			}
			svg = string(data)
		}
		if !strings.Contains(svg, "<svg") {
			return nil, tabOut{}, fmt.Errorf("give SVG markup with an <svg> element")
		}
		if err := checkCoord("x", in.X); err != nil {
			return nil, tabOut{}, err
		}
		if err := checkCoord("y", in.Y); err != nil {
			return nil, tabOut{}, err
		}
		code := need("penpot", "createShapeFromSvg") + `const g = penpot.createShapeFromSvg(` + js(svg) + `);
if (!g) throw new Error("Penpot could not read the SVG");
g.x = ` + js(in.X) + `; g.y = ` + js(in.Y) + `;
` + func() string {
			if in.Name == "" {
				return ""
			}
			return `g.name = ` + js(in.Name) + `;
`
		}() + `return {id: g.id, name: g.name, width: g.width, height: g.height};`
		out, err := inTab(ctx, d, in.FileID, code, true)
		return nil, out, err
	})
}

func tabText(ctx context.Context, d Deps, in tabShapesIn, body string) (*mcp.CallToolResult, textOut, error) {
	if err := checkIDs(in.ShapeIDs, 1); err != nil {
		return nil, textOut{}, err
	}
	out, err := inTab(ctx, d, in.FileID, shapesJS(in.ShapeIDs)+body, false)
	if err != nil {
		return nil, textOut{}, err
	}
	var s string
	if json.Unmarshal(out.raw, &s) != nil {
		s = string(out.raw)
	}
	return nil, textOut{Open: out.Open, Text: s}, nil
}
