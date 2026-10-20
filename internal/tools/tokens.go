package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/palette"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

type importPaletteIn struct {
	FileID string `json:"file_id" jsonschema:"the file whose library gets the colours"`
	Path   string `json:"path" jsonschema:"a palette file inside the working directory: .swift, .kt, .md token table, .css custom properties, .json design tokens, or a hex list"`
	Group  string `json:"group,omitempty" jsonschema:"group path to put every colour under, such as Brand"`
	Alpha  string `json:"alpha,omitempty" jsonschema:"how 8-digit hex is read: leading (#AARRGGBB, Android and Swift, the default) or trailing (#RRGGBBAA, CSS)"`
}

type importOut struct {
	FileID  string          `json:"file_id"`
	Source  string          `json:"source"`
	Added   int             `json:"added"`
	Updated int             `json:"updated"`
	Colors  []palette.Color `json:"colors"`
	Assets  []assetOut      `json:"assets"`
}

type syncTokensIn struct {
	FileID  string         `json:"file_id" jsonschema:"the file"`
	Tokens  map[string]any `json:"tokens,omitempty" jsonschema:"design tokens in DTCG form: {set name: {group: {token: {$value, $type}}}}, optionally with $themes and $metadata"`
	Path    string         `json:"path,omitempty" jsonschema:"a DTCG .json file inside the working directory, instead of tokens"`
	SetName string         `json:"set_name,omitempty" jsonschema:"when the tokens are one set without a set level, the name to give that set"`
}

type syncOut struct {
	FileID string   `json:"file_id"`
	Sets   []string `json:"sets"`
	Tokens int      `json:"tokens"`
}

type applyTokenIn struct {
	FileID    string   `json:"file_id" jsonschema:"the file"`
	PageID    string   `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	ShapeIDs  []string `json:"shape_ids" jsonschema:"the shapes"`
	Token     string   `json:"token" jsonschema:"the token's dotted name, such as color.brand.primary"`
	Attribute string   `json:"attribute" jsonschema:"what it sets: fill, stroke_color, radius or opacity"`
	Value     string   `json:"value,omitempty" jsonschema:"the token's value, when it cannot be read from the file: #RRGGBB for colours, a number for the rest"`
}

type updateColorIn struct {
	FileID  string   `json:"file_id" jsonschema:"the file"`
	ColorID string   `json:"color_id" jsonschema:"the colour, from list_library"`
	Name    string   `json:"name,omitempty" jsonschema:"new name; \"Brand / Primary\" also moves it into a group"`
	Path    *string  `json:"path,omitempty" jsonschema:"new group path; empty string moves it out of every group"`
	Hex     string   `json:"hex,omitempty" jsonschema:"new colour as #RRGGBB"`
	Opacity *float64 `json:"opacity,omitempty" jsonschema:"0 to 1"`
}

type updateTypographyIn struct {
	FileID        string   `json:"file_id" jsonschema:"the file"`
	TypographyID  string   `json:"typography_id" jsonschema:"the type style, from list_library"`
	Name          string   `json:"name,omitempty" jsonschema:"new name; \"Heading / H1\" also moves it into a group"`
	FontFamily    string   `json:"font_family,omitempty"`
	FontSize      float64  `json:"font_size,omitempty"`
	FontWeight    int      `json:"font_weight,omitempty"`
	Italic        *bool    `json:"italic,omitempty"`
	LineHeight    float64  `json:"line_height,omitempty"`
	LetterSpacing *float64 `json:"letter_spacing,omitempty"`
}

type deleteAssetIn struct {
	FileID  string `json:"file_id" jsonschema:"the file"`
	ID      string `json:"id" jsonschema:"the colour, type style or component, from list_library"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true; say so only after the user asked for this to be deleted"`
}

var tokenNameRE = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9$_-]*(\.[A-Za-z0-9$_-]+)*$`)

func registerTokens(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "import_palette",
		Description: "Read colours from a palette file in the repository and add them to a file's library in one write. " +
			"Re-running updates the same colours instead of adding twins.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in importPaletteIn) (*mcp.CallToolResult, importOut, error) {
		d := d.pick(ctx)
		out, err := importPalette(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "sync_tokens",
		Description: "Replace a file's design tokens (sets, tokens, themes) with the DTCG tokens given, " +
			"from the call or a .json file in the repository. Tokens not given are removed.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in syncTokensIn) (*mcp.CallToolResult, syncOut, error) {
		d := d.pick(ctx)
		out, err := syncTokens(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "apply_token",
		Description: "Apply a design token to shapes: their fill, stroke colour, corner radius or opacity takes the " +
			"token's value and stays linked to the token.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in applyTokenIn) (*mcp.CallToolResult, changedOut, error) {
		d := d.pick(ctx)
		out, err := applyToken(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_color",
		Description: "Change a library colour's name, group, hex or opacity. Shapes using it keep their own copy until refreshed in Penpot.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateColorIn) (*mcp.CallToolResult, assetOut, error) {
		d := d.pick(ctx)
		out, err := updateColor(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_typography",
		Description: "Change a library type style: name, font, size, weight, line height or letter spacing.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateTypographyIn) (*mcp.CallToolResult, assetOut, error) {
		d := d.pick(ctx)
		out, err := updateTypography(ctx, d, in)
		return nil, out, err
	})

	for _, k := range []struct{ tool, kind string }{
		{"delete_color", "colour"}, {"delete_typography", "type style"}, {"delete_component", "component"},
	} {
		mcp.AddTool(s, &mcp.Tool{
			Name:        k.tool,
			Description: "Delete a library " + k.kind + ". Needs confirm: true, given only after the user asked.",
			Annotations: destructive,
		}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteAssetIn) (*mcp.CallToolResult, assetOut, error) {
			d := d.pick(ctx)
			out, err := deleteAsset(ctx, d, in, k.kind)
			return nil, out, err
		})
	}
}

func importPalette(ctx context.Context, d Deps, in importPaletteIn) (importOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return importOut{}, err
	}
	alphaFirst := true
	switch in.Alpha {
	case "", "leading":
	case "trailing":
		alphaFirst = false
	default:
		return importOut{}, fmt.Errorf("alpha must be leading or trailing")
	}
	data, err := palette.Open(d.Root, in.Path)
	if err != nil {
		return importOut{}, err
	}
	colors, err := palette.Parse(in.Path, data, alphaFirst)
	if err != nil {
		return importOut{}, err
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return importOut{}, err
	}
	out := importOut{FileID: in.FileID, Source: filepath.Base(in.Path), Colors: colors}
	changes := make([]map[string]any, 0, len(colors))
	for _, c := range colors {
		name := c.Name
		if in.Group != "" {
			name = in.Group + " / " + name
		}
		path, short := splitPath(name, "")
		if err := checkName(short); err != nil {
			return importOut{}, fmt.Errorf("colour %q: %w", c.Name, err)
		}
		id := find(lib.colors, path, short)
		typ := "mod-color"
		if id == "" {
			id, typ = penpot.NewID(), "add-color"
			out.Added++
		} else {
			out.Updated++
		}
		changes = append(changes, map[string]any{"type": typ, "color": map[string]any{
			"id": id, "name": short, "path": path, "color": strings.ToLower(c.Hex), "opacity": c.Opacity,
		}})
		out.Assets = append(out.Assets, assetOut{FileID: in.FileID, ID: id, Name: short, Path: path, Created: typ == "add-color"})
	}
	if err := d.Writer.Apply(ctx, in.FileID, changes); err != nil {
		return importOut{}, err
	}
	return out, nil
}

func syncTokens(ctx context.Context, d Deps, in syncTokensIn) (syncOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return syncOut{}, err
	}
	doc := in.Tokens
	switch {
	case in.Path != "" && doc != nil:
		return syncOut{}, fmt.Errorf("give tokens or path, not both")
	case in.Path != "":
		data, err := palette.Open(d.Root, in.Path)
		if err != nil {
			return syncOut{}, err
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return syncOut{}, fmt.Errorf("%s is not a JSON object: %w", filepath.Base(in.Path), err)
		}
	case doc == nil:
		return syncOut{}, fmt.Errorf("give tokens or path")
	}
	if in.SetName != "" {
		if err := checkName(in.SetName); err != nil {
			return syncOut{}, fmt.Errorf("set_name: %w", err)
		}
		doc = map[string]any{in.SetName: doc}
	}
	out := syncOut{FileID: in.FileID, Sets: []string{}}
	for name, set := range doc {
		if strings.HasPrefix(name, "$") {
			continue
		}
		m, ok := set.(map[string]any)
		if !ok {
			return syncOut{}, fmt.Errorf("set %q must be an object of token groups", name)
		}
		if strings.TrimSpace(name) != name || name == "" {
			return syncOut{}, fmt.Errorf("set name %q has spaces at its ends", name)
		}
		if _, isToken := m["$value"]; isToken {
			return syncOut{}, fmt.Errorf("%q is a token, not a set; give set_name for a single set", name)
		}
		n, err := countTokens(name, m)
		if err != nil {
			return syncOut{}, err
		}
		out.Sets = append(out.Sets, name)
		out.Tokens += n
	}
	if len(out.Sets) == 0 {
		return syncOut{}, fmt.Errorf("no token sets found; give set_name for a single set")
	}
	// Sets Penpot holds inactive resolve nothing, so the synced sets are
	// made active unless the document says which are.
	if _, ok := doc["$metadata"]; !ok {
		sort.Strings(out.Sets)
		doc["$metadata"] = map[string]any{"tokenSetOrder": out.Sets, "activeSets": out.Sets}
	}
	// Penpot decodes the library from DTCG JSON itself (read-multi-set-dtcg),
	// which wants the keys as written. The JSON body's keys are kebab-cased
	// and keywordized on the way in, so the doc goes as a string, which that
	// decoder parses with the keys untouched.
	raw, err := json.Marshal(doc)
	if err != nil {
		return syncOut{}, err
	}
	change := map[string]any{"type": "set-tokens-lib", "tokensLib": string(raw)}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{change}); err != nil {
		return syncOut{}, err
	}
	return out, nil
}

// countTokens checks a set's tokens: every leaf is {$value, $type} under a
// valid dotted name.
func countTokens(prefix string, m map[string]any) (int, error) {
	n := 0
	for k, v := range m {
		if strings.HasPrefix(k, "$") {
			continue
		}
		child, ok := v.(map[string]any)
		if !ok {
			return 0, fmt.Errorf("%s.%s must be an object", prefix, k)
		}
		if _, ok := child["$value"]; ok {
			if _, ok := child["$type"].(string); !ok {
				return 0, fmt.Errorf("token %s.%s has no $type", prefix, k)
			}
			if !tokenNameRE.MatchString(k) {
				return 0, fmt.Errorf("token name %q may hold only letters, digits, _, - and $", k)
			}
			n++
			continue
		}
		c, err := countTokens(prefix+"."+k, child)
		if err != nil {
			return 0, err
		}
		n += c
	}
	return n, nil
}

// tokenValue looks a token up by dotted name in the file's tokens, as
// get-file returns them in DTCG form.
func tokenValue(raw any, name string) (string, bool) {
	sets, ok := raw.(map[string]any)
	if !ok {
		return "", false
	}
	for setName, set := range sets {
		if strings.HasPrefix(setName, "$") {
			continue
		}
		cur := set
		for _, part := range strings.Split(name, ".") {
			m, ok := cur.(map[string]any)
			if !ok {
				cur = nil
				break
			}
			cur = m[part]
		}
		if tok, ok := cur.(map[string]any); ok {
			switch v := tok["$value"].(type) {
			case string:
				return v, true
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64), true
			}
		}
	}
	return "", false
}

func applyToken(ctx context.Context, d Deps, in applyTokenIn) (changedOut, error) {
	ref := pageRef{FileID: in.FileID, PageID: in.PageID}
	if err := ref.validate(); err != nil {
		return changedOut{}, err
	}
	if err := checkIDs(in.ShapeIDs, 1); err != nil {
		return changedOut{}, err
	}
	if !tokenNameRE.MatchString(in.Token) {
		return changedOut{}, fmt.Errorf("token must be a dotted name such as color.brand.primary")
	}
	value := in.Value
	if value == "" {
		var f struct {
			Data struct {
				TokensLib any `json:"tokensLib"`
			} `json:"data"`
		}
		if err := d.Client.Call(ctx, "get-file", map[string]string{"id": in.FileID}, &f); err != nil {
			return changedOut{}, err
		}
		v, ok := tokenValue(f.Data.TokensLib, in.Token)
		if !ok {
			return changedOut{}, fmt.Errorf("token %q is not in the file's tokens; sync it first, or give value", in.Token)
		}
		value = v
	}
	alias := strings.HasPrefix(value, "{")
	if alias && d.Route == nil {
		return changedOut{}, fmt.Errorf("token %q refers to another token (%s); give the resolved value, "+
			"or connect a Penpot tab (PENPOT_MCP_URL) to let Penpot resolve it", in.Token, value)
	}
	if _, ok := tokenProps[in.Attribute]; !ok {
		return changedOut{}, fmt.Errorf("attribute must be fill, stroke_color, radius or opacity")
	}
	if alias {
		return tabApplyToken(ctx, d, in)
	}
	var attrs map[string]any
	var keys []string
	switch in.Attribute {
	case "fill", "stroke_color":
		if !hexRE.MatchString(value) {
			return changedOut{}, fmt.Errorf("token %q is %q, not a #RRGGBB colour", in.Token, value)
		}
		keys = []string{"fill"}
		if in.Attribute == "stroke_color" {
			keys = []string{"strokeColor"}
		}
	case "radius", "opacity":
		f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSuffix(value, "px"), "%"), 64)
		if err != nil {
			return changedOut{}, fmt.Errorf("token %q is %q, not a number", in.Token, value)
		}
		if in.Attribute == "radius" {
			if f < 0 || f > maxSize {
				return changedOut{}, fmt.Errorf("radius must be 0 to %d", maxSize)
			}
			keys = []string{"r1", "r2", "r3", "r4"}
			attrs = map[string]any{"r1": f, "r2": f, "r3": f, "r4": f}
		} else {
			if strings.HasSuffix(value, "%") || f > 1 {
				f /= 100
			}
			if f < 0 || f > 1 {
				return changedOut{}, fmt.Errorf("opacity must be 0 to 1")
			}
			keys = []string{"opacity"}
			attrs = map[string]any{"opacity": f}
		}
	default:
		return changedOut{}, fmt.Errorf("attribute must be fill, stroke_color, radius or opacity")
	}

	t, p, err := loadIDs(ctx, d, idsIn{pageRef: ref, ShapeIDs: in.ShapeIDs}, 1)
	if err != nil {
		return changedOut{}, err
	}
	changes := make([]map[string]any, 0, len(in.ShapeIDs))
	for _, id := range in.ShapeIDs {
		o := p.objects[id]
		typ := str(o, "type")
		a := map[string]any{}
		for k, v := range attrs {
			a[k] = v
		}
		switch in.Attribute {
		case "fill":
			if typ == penpot.TypeGroup {
				return changedOut{}, fmt.Errorf("groups have no fill; apply the token to their children")
			}
			if typ == penpot.TypeText {
				if d.Route != nil {
					return tabApplyToken(ctx, d, in)
				}
				return changedOut{}, fmt.Errorf("a token on a text's colour needs a connected Penpot tab (PENPOT_MCP_URL); " +
					"without one, use update_shape")
			}
			a["fills"] = fillList(value)
		case "stroke_color":
			strokes, _ := o["strokes"].([]any)
			if len(strokes) == 0 {
				return changedOut{}, fmt.Errorf("%q has no stroke; give it one first", str(o, "name"))
			}
			first, _ := strokes[0].(map[string]any)
			s := map[string]any{}
			for k, v := range first {
				s[k] = v
			}
			s["strokeColor"] = strings.ToLower(value)
			a["strokes"] = append([]any{s}, strokes[1:]...)
		case "radius":
			if typ != penpot.TypeBoard && typ != penpot.TypeRect {
				return changedOut{}, fmt.Errorf("radius is for boards and rects, not a %s", typ)
			}
		}
		applied := map[string]any{}
		if old, ok := o["appliedTokens"].(map[string]any); ok {
			for k, v := range old {
				applied[k] = v
			}
		}
		for _, k := range keys {
			applied[k] = in.Token
		}
		a["appliedTokens"] = applied
		changes = append(changes, penpot.ModObj(t.PageID, id, a))
	}
	if err := d.Writer.Apply(ctx, t.FileID, changes); err != nil {
		return changedOut{}, err
	}
	return changedOut{Target: t, Shapes: described(p, in.ShapeIDs)}, nil
}

func updateColor(ctx context.Context, d Deps, in updateColorIn) (assetOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return assetOut{}, err
	}
	if err := checkID("color_id", in.ColorID); err != nil {
		return assetOut{}, err
	}
	if err := checkColor("hex", in.Hex); err != nil {
		return assetOut{}, err
	}
	if in.Opacity != nil && (*in.Opacity < 0 || *in.Opacity > 1) {
		return assetOut{}, fmt.Errorf("opacity must be 0 to 1")
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return assetOut{}, err
	}
	old, ok := lib.colors[in.ColorID]
	if !ok {
		return assetOut{}, fmt.Errorf("colour %s is not in the library of %q", in.ColorID, lib.name)
	}
	c := map[string]any{"id": in.ColorID, "name": str(old, "name"), "path": str(old, "path"), "color": str(old, "color"), "opacity": 1.0}
	if v, ok := old["opacity"].(float64); ok {
		c["opacity"] = v
	}
	if in.Name != "" {
		path, name := splitPath(in.Name, "")
		if err := checkName(name); err != nil {
			return assetOut{}, err
		}
		c["name"] = name
		if strings.Contains(in.Name, "/") {
			c["path"] = path
		}
	}
	if in.Path != nil {
		c["path"] = strings.TrimSpace(*in.Path)
	}
	if in.Hex != "" {
		c["color"] = strings.ToLower(in.Hex)
	}
	if in.Opacity != nil {
		c["opacity"] = *in.Opacity
	}
	if c["color"] == "" {
		return assetOut{}, fmt.Errorf("colour %q is a gradient or image; only plain colours are changed here", str(old, "name"))
	}
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "mod-color", "color": c}}); err != nil {
		return assetOut{}, err
	}
	return assetOut{FileID: in.FileID, ID: in.ColorID, Name: str(c, "name"), Path: str(c, "path")}, nil
}

func updateTypography(ctx context.Context, d Deps, in updateTypographyIn) (assetOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return assetOut{}, err
	}
	if err := checkID("typography_id", in.TypographyID); err != nil {
		return assetOut{}, err
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return assetOut{}, err
	}
	old, ok := lib.typographies[in.TypographyID]
	if !ok {
		return assetOut{}, fmt.Errorf("type style %s is not in the library of %q", in.TypographyID, lib.name)
	}
	t := map[string]any{}
	for k, v := range old {
		t[k] = v
	}
	if in.Name != "" {
		path, name := splitPath(in.Name, "")
		if err := checkName(name); err != nil {
			return assetOut{}, err
		}
		t["name"] = name
		if strings.Contains(in.Name, "/") {
			t["path"] = path
		}
	}
	if in.FontFamily != "" || in.FontWeight != 0 || in.Italic != nil {
		family := orDefault(in.FontFamily, str(old, "fontFamily"))
		weight := in.FontWeight
		if weight == 0 {
			weight, _ = strconv.Atoi(str(old, "fontWeight"))
		}
		italic := str(old, "fontStyle") == "italic"
		if in.Italic != nil {
			italic = *in.Italic
		}
		face, err := fonts.Resolve(family, weight, italic)
		if err != nil {
			return assetOut{}, err
		}
		t["fontId"], t["fontFamily"], t["fontVariantId"] = face.FontID, face.Family, face.VariantID
		t["fontWeight"], t["fontStyle"] = face.Weight, face.Style
	}
	if in.FontSize != 0 {
		if in.FontSize < 3 || in.FontSize > 1000 {
			return assetOut{}, fmt.Errorf("font_size must be 3 to 1000")
		}
		t["fontSize"] = fnum(in.FontSize)
	}
	if in.LineHeight != 0 {
		if in.LineHeight < 0 || in.LineHeight > 10 {
			return assetOut{}, fmt.Errorf("line_height must be above 0 and at most 10")
		}
		t["lineHeight"] = fnum(in.LineHeight)
	}
	if in.LetterSpacing != nil {
		t["letterSpacing"] = fnum(*in.LetterSpacing)
	}
	t["id"] = in.TypographyID
	delete(t, "modifiedAt")
	if err := d.Writer.Apply(ctx, in.FileID, []map[string]any{{"type": "mod-typography", "typography": t}}); err != nil {
		return assetOut{}, err
	}
	return assetOut{FileID: in.FileID, ID: in.TypographyID, Name: str(t, "name"), Path: str(t, "path")}, nil
}

func deleteAsset(ctx context.Context, d Deps, in deleteAssetIn, kind string) (assetOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return assetOut{}, err
	}
	if err := checkID("id", in.ID); err != nil {
		return assetOut{}, err
	}
	if err := checkConfirm(in.Confirm, "a "+kind); err != nil {
		return assetOut{}, err
	}
	lib, err := loadLibrary(ctx, d, in.FileID)
	if err != nil {
		return assetOut{}, err
	}
	assets, change := lib.colors, "del-color"
	switch kind {
	case "type style":
		assets, change = lib.typographies, "del-typography"
	case "component":
		assets, change = lib.components, "del-component"
	}
	a, ok := assets[in.ID]
	if !ok || a["deleted"] == true {
		return assetOut{}, fmt.Errorf("%s %s is not in the library of %q", kind, in.ID, lib.name)
	}
	changes := []map[string]any{{"type": change, "id": in.ID}}
	if kind == "component" {
		// Penpot deletes the main instance with the component; copies stay
		// and can be restored with it.
		if main, page := str(a, "mainInstanceId"), str(a, "mainInstancePage"); main != "" && page != "" {
			changes = append([]map[string]any{penpot.DelObj(page, main)}, changes...)
		}
	}
	if err := d.Writer.Apply(ctx, in.FileID, changes); err != nil {
		return assetOut{}, err
	}
	d.log().Info("deleted "+kind, "file_id", in.FileID, "id", in.ID, "name", str(a, "name"))
	return assetOut{FileID: in.FileID, ID: in.ID, Name: str(a, "name"), Path: str(a, "path")}, nil
}

// tokenProps are the Plugin API's TokenProperty names for each attribute.
var tokenProps = map[string][]string{
	"fill":         {"fill"},
	"stroke_color": {"strokeColor"},
	"radius":       {"borderRadiusTopLeft", "borderRadiusTopRight", "borderRadiusBottomRight", "borderRadiusBottomLeft"},
	"opacity":      {"opacity"},
}

// tabApplyToken applies a token in the connected tab, for what the API
// path cannot do: a token that refers to another one, which Penpot
// resolves, and a text's colour. The token is looked up by name in the
// active sets, the last match winning, as Penpot resolves it.
func tabApplyToken(ctx context.Context, d Deps, in applyTokenIn) (changedOut, error) {
	ref := pageRef{FileID: in.FileID, PageID: in.PageID}
	t, p, err := loadIDs(ctx, d, idsIn{pageRef: ref, ShapeIDs: in.ShapeIDs}, 1)
	if err != nil {
		return changedOut{}, err
	}
	code := "if (!penpot.library.local.tokens) throw new Error('this Penpot has no design tokens in its Plugin API');\n" +
		shapesJS(in.ShapeIDs) + `
const sets = penpot.library.local.tokens.sets;
let token;
for (const set of sets.filter(s => s.active)) {
  token = set.tokens.find(x => x.name === ` + js(in.Token) + `) || token;
}
if (!token) {
  const off = sets.filter(s => !s.active && s.tokens.some(x => x.name === ` + js(in.Token) + `)).map(s => s.name);
  throw new Error("token " + ` + js(in.Token) + ` + (off.length ? " is only in inactive sets (" + off.join(", ") + "): activate one in Penpot" : " is in no set"));
}
token.applyToShapes(shapes, ` + js(tokenProps[in.Attribute]) + `);
return token.resolvedValueString || null;`
	if _, err := inTab(ctx, d, in.FileID, code, true); err != nil {
		return changedOut{}, err
	}
	return changedOut{Target: t, Shapes: described(p, in.ShapeIDs)}, nil
}
