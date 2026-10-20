package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// library is a file's local library as get-file returns it.
type library struct {
	name         string
	colors       map[string]map[string]any
	typographies map[string]map[string]any
	components   map[string]map[string]any
}

func loadLibrary(ctx context.Context, d Deps, fileID string) (library, error) {
	var f struct {
		Name string `json:"name"`
		Data struct {
			Colors       map[string]map[string]any `json:"colors"`
			Typographies map[string]map[string]any `json:"typographies"`
			Components   map[string]map[string]any `json:"components"`
		} `json:"data"`
	}
	if err := d.Client.Call(ctx, "get-file", map[string]string{"id": fileID}, &f); err != nil {
		return library{}, err
	}
	return library{f.Name, f.Data.Colors, f.Data.Typographies, f.Data.Components}, nil
}

// find returns the id of the asset with this path and name, or "".
func find(assets map[string]map[string]any, path, name string) string {
	for id, a := range assets {
		if str(a, "name") == name && str(a, "path") == path && a["deleted"] != true {
			return id
		}
	}
	return ""
}

// splitPath turns "Brand / Primary" or "Brand/Primary" into the path
// "Brand" and the name "Primary", unless path is given.
func splitPath(name, path string) (string, string) {
	if path != "" || !strings.Contains(name, "/") {
		return strings.TrimSpace(path), strings.TrimSpace(name)
	}
	i := strings.LastIndex(name, "/")
	parts := strings.Split(name[:i], "/")
	for j := range parts {
		parts[j] = strings.TrimSpace(parts[j])
	}
	return strings.Join(parts, " / "), strings.TrimSpace(name[i+1:])
}

type colorIn struct {
	FileID  string   `json:"file_id" jsonschema:"the file whose library gets the colour"`
	Name    string   `json:"name" jsonschema:"colour name; \"Brand / Primary\" puts it in the Brand group"`
	Path    string   `json:"path,omitempty" jsonschema:"group path, such as Brand or Brand / Dark; taken from name when empty"`
	Hex     string   `json:"hex" jsonschema:"the colour as #RRGGBB"`
	Opacity *float64 `json:"opacity,omitempty" jsonschema:"0 to 1, default 1"`
}

type typographyIn struct {
	FileID        string  `json:"file_id" jsonschema:"the file whose library gets the type style"`
	Name          string  `json:"name" jsonschema:"style name; \"Heading / H1\" puts it in the Heading group"`
	Path          string  `json:"path,omitempty" jsonschema:"group path; taken from name when empty"`
	FontFamily    string  `json:"font_family,omitempty" jsonschema:"a Google font name like Inter, or Source Sans Pro (the default)"`
	FontSize      float64 `json:"font_size,omitempty" jsonschema:"size in pixels, 3 to 1000, default 16"`
	FontWeight    int     `json:"font_weight,omitempty" jsonschema:"100 to 900 in steps of 100, default 400"`
	Italic        bool    `json:"italic,omitempty"`
	LineHeight    float64 `json:"line_height,omitempty" jsonschema:"multiple of the font size, default 1.2"`
	LetterSpacing float64 `json:"letter_spacing,omitempty" jsonschema:"pixels between letters"`
	TextTransform string  `json:"text_transform,omitempty" jsonschema:"none (default), uppercase, lowercase or capitalize"`
}

type assetOut struct {
	FileID  string `json:"file_id"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path,omitempty"`
	Created bool   `json:"created"`
}

type fileIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
}

type libColor struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Path    string  `json:"path,omitempty"`
	Hex     string  `json:"hex,omitempty"`
	Opacity float64 `json:"opacity"`
}

type libTypography struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path,omitempty"`
	FontFamily string `json:"font_family"`
	FontSize   string `json:"font_size"`
	FontWeight string `json:"font_weight"`
	LineHeight string `json:"line_height"`
}

type libComponent struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Path           string `json:"path,omitempty"`
	MainInstanceID string `json:"main_instance_id"`
	MainPageID     string `json:"main_page_id"`
}

type libraryOut struct {
	FileID       string          `json:"file_id"`
	FileName     string          `json:"file_name"`
	Colors       []libColor      `json:"colors"`
	Typographies []libTypography `json:"typographies"`
	Components   []libComponent  `json:"components"`
}

type componentIn struct {
	FileID  string `json:"file_id" jsonschema:"the file"`
	PageID  string `json:"page_id,omitempty" jsonschema:"the board's page; the first page when empty"`
	ShapeID string `json:"shape_id" jsonschema:"the board that becomes the component's main instance"`
	Name    string `json:"name,omitempty" jsonschema:"component name; the board's name when empty; \"Buttons / Primary\" groups it"`
	Path    string `json:"path,omitempty" jsonschema:"group path; taken from name when empty"`
}

type componentOut struct {
	penpot.Target
	ComponentID string `json:"component_id"`
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	ShapeID     string `json:"shape_id"`
}

type instanceOut struct {
	penpot.Target
	ShapeID     string  `json:"shape_id"`
	ComponentID string  `json:"component_id"`
	Name        string  `json:"name"`
	ParentID    string  `json:"parent_id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Shapes      int     `json:"shapes"`
}

type instanceIn struct {
	FileID      string  `json:"file_id" jsonschema:"the file"`
	PageID      string  `json:"page_id,omitempty" jsonschema:"the page to place the copy on; the first page when empty"`
	ParentID    string  `json:"parent_id,omitempty" jsonschema:"a board to place the copy inside; the page root when empty"`
	ComponentID string  `json:"component_id" jsonschema:"the component, from list_library"`
	X           float64 `json:"x" jsonschema:"left edge of the copy, absolute page pixels"`
	Y           float64 `json:"y" jsonschema:"top edge of the copy, absolute page pixels"`
	Name        string  `json:"name,omitempty" jsonschema:"layer name; the component's name when empty"`
}

func registerLibrary(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "add_color",
		Description: "Add a colour to a file's library. A colour with the same group and name is updated, " +
			"not duplicated, so re-running is safe.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in colorIn) (*mcp.CallToolResult, assetOut, error) {
		d := d.pick(ctx)
		out, err := addColor(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "add_typography",
		Description: "Add a type style to a file's library. A style with the same group and name is updated, " +
			"not duplicated.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in typographyIn) (*mcp.CallToolResult, assetOut, error) {
		d := d.pick(ctx)
		out, err := addTypography(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_library",
		Description: "A file's library: colours, type styles and components, each with its id and group path.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in fileIn) (*mcp.CallToolResult, libraryOut, error) {
		d := d.pick(ctx)
		out, err := listLibrary(ctx, d, in.FileID)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_component",
		Description: "Turn a board into a component: the board becomes its main instance. " +
			"Draw the board first (draw or create_board), then place copies with instantiate_component.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in componentIn) (*mcp.CallToolResult, componentOut, error) {
		d := d.pick(ctx)
		out, err := createComponent(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "instantiate_component",
		Description: "Place a copy of a component at x, y. The copy stays linked to the main instance, " +
			"so later changes to the main show in every copy.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in instanceIn) (*mcp.CallToolResult, instanceOut, error) {
		d := d.pick(ctx)
		out, err := instantiate(ctx, d, in)
		return nil, out, err
	})
}

func addColor(ctx context.Context, d Deps, in colorIn) (assetOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return assetOut{}, err
	}
	path, name := splitPath(in.Name, in.Path)
	if err := checkName(name); err != nil {
		return assetOut{}, err
	}
	if in.Hex == "" {
		return assetOut{}, fmt.Errorf("hex is required")
	}
	if err := checkColor("hex", in.Hex); err != nil {
		return assetOut{}, err
	}
	opacity := 1.0
	if in.Opacity != nil {
		opacity = *in.Opacity
	}
	if opacity < 0 || opacity > 1 {
		return assetOut{}, fmt.Errorf("opacity must be 0 to 1")
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return assetOut{}, err
	}
	id := find(lib.colors, path, name)
	created := id == ""
	if created {
		id = penpot.NewID()
	}
	color := map[string]any{"id": id, "name": name, "path": path, "color": strings.ToLower(in.Hex), "opacity": opacity}
	change := map[string]any{"type": "mod-color", "color": color}
	if created {
		change["type"] = "add-color"
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{change}); err != nil {
		return assetOut{}, err
	}
	return assetOut{FileID: in.FileID, ID: id, Name: name, Path: path, Created: created}, nil
}

func addTypography(ctx context.Context, d Deps, in typographyIn) (assetOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return assetOut{}, err
	}
	path, name := splitPath(in.Name, in.Path)
	if err := checkName(name); err != nil {
		return assetOut{}, err
	}
	size := orDefaultF(in.FontSize, 16)
	if size < 3 || size > 1000 {
		return assetOut{}, fmt.Errorf("font_size must be 3 to 1000")
	}
	lh := orDefaultF(in.LineHeight, 1.2)
	if lh <= 0 || lh > 10 {
		return assetOut{}, fmt.Errorf("line_height must be above 0 and at most 10")
	}
	transform := orDefault(in.TextTransform, "none")
	switch transform {
	case "none", "uppercase", "lowercase", "capitalize":
	default:
		return assetOut{}, fmt.Errorf("text_transform must be none, uppercase, lowercase or capitalize")
	}
	face, err := fonts.Resolve(in.FontFamily, in.FontWeight, in.Italic)
	if err != nil {
		return assetOut{}, err
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return assetOut{}, err
	}
	id := find(lib.typographies, path, name)
	created := id == ""
	if created {
		id = penpot.NewID()
	}
	// Penpot stores every typography value as a string.
	typo := map[string]any{
		"id": id, "name": name, "path": path,
		"fontId": face.FontID, "fontFamily": face.Family, "fontVariantId": face.VariantID,
		"fontSize": fnum(size), "fontWeight": face.Weight, "fontStyle": face.Style,
		"lineHeight": fnum(lh), "letterSpacing": fnum(in.LetterSpacing), "textTransform": transform,
	}
	change := map[string]any{"type": "mod-typography", "typography": typo}
	if created {
		change["type"] = "add-typography"
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{change}); err != nil {
		return assetOut{}, err
	}
	return assetOut{FileID: in.FileID, ID: id, Name: name, Path: path, Created: created}, nil
}

func listLibrary(ctx context.Context, d Deps, fileID string) (libraryOut, error) {
	if err := checkID("file_id", fileID); err != nil {
		return libraryOut{}, err
	}
	lib, err := loadLibrary(ctx, d, fileID)
	if err != nil {
		return libraryOut{}, err
	}
	out := libraryOut{
		FileID: fileID, FileName: lib.name,
		Colors: []libColor{}, Typographies: []libTypography{}, Components: []libComponent{},
	}
	for id, c := range lib.colors {
		op := 1.0
		if v, ok := c["opacity"].(float64); ok {
			op = v
		}
		out.Colors = append(out.Colors, libColor{id, str(c, "name"), str(c, "path"), str(c, "color"), op})
	}
	for id, t := range lib.typographies {
		out.Typographies = append(out.Typographies, libTypography{
			id, str(t, "name"), str(t, "path"),
			str(t, "fontFamily"), str(t, "fontSize"), str(t, "fontWeight"), str(t, "lineHeight"),
		})
	}
	for id, c := range lib.components {
		if c["deleted"] == true {
			continue
		}
		out.Components = append(out.Components, libComponent{
			id, str(c, "name"), str(c, "path"),
			str(c, "mainInstanceId"), str(c, "mainInstancePage"),
		})
	}
	sort.Slice(out.Colors, func(i, j int) bool {
		return out.Colors[i].Path+"/"+out.Colors[i].Name < out.Colors[j].Path+"/"+out.Colors[j].Name
	})
	sort.Slice(out.Typographies, func(i, j int) bool {
		return out.Typographies[i].Path+"/"+out.Typographies[i].Name < out.Typographies[j].Path+"/"+out.Typographies[j].Name
	})
	sort.Slice(out.Components, func(i, j int) bool {
		return out.Components[i].Path+"/"+out.Components[i].Name < out.Components[j].Path+"/"+out.Components[j].Name
	})
	return out, nil
}

func fnum(v float64) string { return fmt.Sprint(v) }

func createComponent(ctx context.Context, d Deps, in componentIn) (componentOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return componentOut{}, err
	}
	if in.PageID != "" {
		if err := checkID("page_id", in.PageID); err != nil {
			return componentOut{}, err
		}
	}
	if err := checkID("shape_id", in.ShapeID); err != nil {
		return componentOut{}, err
	}
	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return componentOut{}, err
	}
	p, err := loadRaw(ctx, d, t)
	if err != nil {
		return componentOut{}, err
	}
	board, err := p.get(in.ShapeID, t)
	if err != nil {
		return componentOut{}, err
	}
	if typ := str(board, "type"); typ != penpot.TypeBoard {
		return componentOut{}, fmt.Errorf("%q is a %s; only a board becomes a component", str(board, "name"), typ)
	}
	if cid := str(board, "componentId"); cid != "" {
		return componentOut{}, fmt.Errorf("%q is already part of component %s", str(board, "name"), cid)
	}
	for _, a := range p.ancestors(in.ShapeID) {
		if str(p.objects[a], "componentId") != "" {
			return componentOut{}, fmt.Errorf("%q sits inside component %q; make components from boards outside one",
				str(board, "name"), str(p.objects[a], "name"))
		}
	}
	name := in.Name
	if name == "" {
		name = str(board, "name")
	}
	path, name := splitPath(name, in.Path)
	if err := checkName(name); err != nil {
		return componentOut{}, err
	}
	cid := penpot.NewID()
	changes := []map[string]any{
		penpot.ModObj(t.PageID, in.ShapeID, map[string]any{
			"componentId": cid, "componentFile": t.FileID, "componentRoot": true, "mainInstance": true,
		}),
		{
			"type": "add-component", "id": cid, "name": name, "path": path,
			"mainInstanceId": in.ShapeID, "mainInstancePage": t.PageID,
		},
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return componentOut{}, err
	}
	return componentOut{Target: t, ComponentID: cid, Name: name, Path: path, ShapeID: in.ShapeID}, nil
}

func instantiate(ctx context.Context, d Deps, in instanceIn) (instanceOut, error) {
	tg := target{FileID: in.FileID, PageID: in.PageID, ParentID: in.ParentID}
	if err := tg.validate(); err != nil {
		return instanceOut{}, err
	}
	if err := checkID("component_id", in.ComponentID); err != nil {
		return instanceOut{}, err
	}
	if err := checkCoord("x", in.X); err != nil {
		return instanceOut{}, err
	}
	if err := checkCoord("y", in.Y); err != nil {
		return instanceOut{}, err
	}
	if in.Name != "" {
		if err := checkName(in.Name); err != nil {
			return instanceOut{}, err
		}
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return instanceOut{}, err
	}
	comp, ok := lib.components[in.ComponentID]
	if !ok || comp["deleted"] == true {
		return instanceOut{}, fmt.Errorf("component %s is not in the library of %q", in.ComponentID, lib.name)
	}
	mainID, mainPage := str(comp, "mainInstanceId"), str(comp, "mainInstancePage")
	src, err := d.Writer.Resolve(ctx, in.FileID, mainPage)
	if err != nil {
		return instanceOut{}, err
	}
	mp, err := loadRaw(ctx, d, src)
	if err != nil {
		return instanceOut{}, err
	}
	main, err := mp.get(mainID, src)
	if err != nil {
		return instanceOut{}, fmt.Errorf("the main instance of %q: %w", str(comp, "name"), err)
	}

	t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
	if err != nil {
		return instanceOut{}, err
	}
	parent, frame := penpot.ZeroID, penpot.ZeroID
	if in.ParentID != "" {
		o, err := checkParent(ctx, d, t, in.ParentID)
		if err != nil {
			return instanceOut{}, err
		}
		parent, frame = in.ParentID, o.FrameID
		if o.Type == penpot.TypeBoard {
			frame = in.ParentID
		}
	}

	changes, rootID, err := copyTree(mp, mainID, t.PageID, parent, frame, in.X-num(main, "x"), in.Y-num(main, "y"))
	if err != nil {
		return instanceOut{}, err
	}
	root := changes[0]["obj"].(map[string]any)
	root["componentId"], root["componentFile"], root["componentRoot"] = in.ComponentID, in.FileID, true
	if in.Name != "" {
		root["name"] = in.Name
	} else {
		root["name"] = str(comp, "name")
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return instanceOut{}, err
	}
	return instanceOut{
		Target: t, ShapeID: rootID, ComponentID: in.ComponentID, Name: str(root, "name"),
		ParentID: parent, X: num(root, "x"), Y: num(root, "y"), Width: num(root, "width"), Height: num(root, "height"),
		Shapes: len(changes),
	}, nil
}

// skipOnCopy are main-instance attributes a copy must not carry.
var skipOnCopy = map[string]bool{"mainInstance": true, "id": true, "parentId": true, "frameId": true, "shapes": true}

// copyTree returns add-obj changes for a copy of the shape mainID and
// everything inside it, moved by dx, dy, under parent. Every copied shape
// points at its original with shapeRef, which is what links a component
// copy to its main.
func copyTree(p rawPage, mainID, pageID, parent, frame string, dx, dy float64) ([]map[string]any, string, error) {
	ids := append([]string{mainID}, p.descendants(mainID)...)
	newID := make(map[string]string, len(ids))
	for _, id := range ids {
		newID[id] = penpot.NewID()
	}
	changes := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		o := p.objects[id]
		obj := make(map[string]any, len(o)+4)
		for k, v := range o {
			if !skipOnCopy[k] {
				obj[k] = v
			}
		}
		moved, err := shiftRaw(o, dx, dy)
		if err != nil {
			return nil, "", err
		}
		for k, v := range moved {
			obj[k] = v
		}
		obj["id"] = newID[id]
		obj["shapeRef"] = id
		if id == mainID {
			obj["parentId"], obj["frameId"] = parent, frame
		} else {
			obj["parentId"] = newID[str(o, "parentId")]
			obj["frameId"] = newID[str(o, "frameId")]
			if obj["frameId"] == "" {
				obj["frameId"] = frame
			}
		}
		var kids []string
		for _, c := range children(o) {
			if n, ok := newID[c]; ok {
				kids = append(kids, n)
			}
		}
		obj["shapes"] = kids
		changes = append(changes, penpot.AddObj(pageID, obj))
	}
	return changes, newID[mainID], nil
}
