package tools

import (
	"strings"
	"testing"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

const (
	textID  = "c0000000-0000-4000-8000-000000000002"
	grpID   = "c0000000-0000-4000-8000-000000000003"
	grp1ID  = "c0000000-0000-4000-8000-000000000004"
	grp2ID  = "c0000000-0000-4000-8000-000000000005"
	ellipID = "c0000000-0000-4000-8000-000000000006"
	board2  = "b0000000-0000-4000-8000-000000000002"
)

func pts(x, y, w, h int) string {
	f := func(a, b int) string { return `{"x":` + itoa(a) + `,"y":` + itoa(b) + `}` }
	return `[` + f(x, y) + `,` + f(x+w, y) + `,` + f(x+w, y+h) + `,` + f(x, y+h) + `]`
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s, neg := "", n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// richPage is a page with a board holding a stroked rect and a text, a
// group of two rects, an ellipse and an empty second board.
func richPage() string {
	const zero = penpot.ZeroID
	obj := func(id, name, typ, parent, frame string, x, y, w, h int, extra string) string {
		return `"` + id + `":{"id":"` + id + `","name":"` + name + `","type":"` + typ + `","parentId":"` + parent + `","frameId":"` + frame +
			`","x":` + itoa(x) + `,"y":` + itoa(y) + `,"width":` + itoa(w) + `,"height":` + itoa(h) + `,"rotation":0,"points":` + pts(x, y, w, h) + extra + `}`
	}
	content := `"content":{"type":"root","children":[{"type":"paragraph-set","children":[{"type":"paragraph","textAlign":"center","children":[` +
		`{"text":"Hello","fontId":"sourcesanspro","fontFamily":"sourcesanspro","fontVariantId":"regular","fontWeight":"400","fontStyle":"normal",` +
		`"fontSize":"16","lineHeight":"1.2","letterSpacing":"0","fills":[{"fillColor":"#112233","fillOpacity":1}]}]}]}]}`
	return `{"id":"` + pageID + `","name":"Page 1","objects":{` +
		`"` + zero + `":{"id":"` + zero + `","name":"Root Frame","type":"frame","shapes":["` + boardID + `","` + grpID + `","` + ellipID + `","` + board2 + `"]},` +
		obj(boardID, "Login", "frame", zero, zero, 0, 0, 400, 300, `,"shapes":["`+rectID+`","`+textID+`"]`) + `,` +
		obj(rectID, "Button", "rect", boardID, boardID, 50, 50, 100, 40, `,"strokes":[{"strokeColor":"#000000","strokeWidth":1}],"fills":[{"fillColor":"#FF0000","fillOpacity":1}]`) + `,` +
		obj(textID, "Hello", "text", boardID, boardID, 10, 10, 60, 20, `,"growType":"auto-width",`+content) + `,` +
		obj(grpID, "Pair", "group", zero, zero, 200, 200, 120, 50, `,"shapes":["`+grp1ID+`","`+grp2ID+`"]`) + `,` +
		obj(grp1ID, "One", "rect", grpID, zero, 200, 200, 50, 50, ``) + `,` +
		obj(grp2ID, "Two", "rect", grpID, zero, 270, 200, 50, 50, ``) + `,` +
		obj(ellipID, "Dot", "circle", zero, zero, 400, 400, 30, 30, ``) + `,` +
		obj(board2, "Empty", "frame", zero, zero, 600, 0, 100, 100, `,"shapes":[]`) +
		`}}`
}

func richSession(t *testing.T) (csT func(string, map[string]any) string, b *backend) {
	t.Helper()
	cs, back := session(t)
	back.page = richPage()
	return func(name string, args map[string]any) string {
		t.Helper()
		if _, has := args["file_id"]; !has {
			args["file_id"] = fileID
		}
		res := call(t, cs, name, args, nil)
		if res.IsError {
			return "ERR: " + errText(res)
		}
		return ""
	}, back
}

func TestUpdateShapeStyles(t *testing.T) {
	do, b := richSession(t)
	for name, args := range map[string]map[string]any{
		"stroke":           {"shape_id": rectID, "stroke": "#00ff00", "stroke_width": 3},
		"keep stroke":      {"shape_id": rectID, "stroke_width": 2},
		"stroke none":      {"shape_id": rectID, "stroke": "none"},
		"fill none":        {"shape_id": rectID, "fill": "none"},
		"radius":           {"shape_id": rectID, "radius": 6},
		"opacity hidden":   {"shape_id": rectID, "opacity": 0.5, "hidden": true},
		"rename":           {"shape_id": rectID, "name": "Go"},
		"move":             {"shape_id": rectID, "x": 60, "y": 70},
		"move group":       {"shape_id": grpID, "x": 210},
		"resize":           {"shape_id": rectID, "width": 120, "height": 50, "rotation": 15},
		"ellipse rotation": {"shape_id": ellipID, "rotation": 45},
		"text change":      {"shape_id": textID, "text": "Goodbye\nthere"},
		"text width":       {"shape_id": textID, "width": 80},
		"text rotate":      {"shape_id": textID, "rotation": 10},
	} {
		before := len(b.raw)
		if got := do("update_shape", args); got != "" {
			t.Errorf("%s: %s", name, got)
		} else if len(b.raw) != before+1 {
			t.Errorf("%s: wrote %d times", name, len(b.raw)-before)
		}
	}
	if !strings.Contains(strings.Join(b.raw, "\n"), `"name":"Goodbye"`) {
		t.Error("a text layer named after its text should follow the new text")
	}
}

func TestUpdateShapeRefusals(t *testing.T) {
	do, b := richSession(t)
	for name, args := range map[string]map[string]any{
		"no stroke":         {"shape_id": grp1ID, "stroke_width": 2},
		"group fill":        {"shape_id": grpID, "fill": "#ffffff"},
		"radius on ellipse": {"shape_id": ellipID, "radius": 3},
		"height on text":    {"shape_id": textID, "height": 30},
		"width on group":    {"shape_id": grpID, "width": 30},
		"rotate board":      {"shape_id": boardID, "rotation": 30},
		"text on rect":      {"shape_id": rectID, "text": "x"},
		"unknown shape":     {"shape_id": "d0000000-0000-4000-8000-0000000000ff", "name": "x"},
		"bad coord":         {"shape_id": rectID, "x": 1e12},
		"bad size":          {"shape_id": rectID, "width": -5},
		"bad rotation":      {"shape_id": rectID, "rotation": 999},
		"bad stroke width":  {"shape_id": rectID, "stroke_width": -1},
		"bad radius":        {"shape_id": rectID, "radius": -1},
		"bad colour":        {"shape_id": rectID, "fill": "red"},
		"long text":         {"shape_id": textID, "text": strings.Repeat("x", 200_000)},
		"long name":         {"shape_id": rectID, "name": strings.Repeat("x", 500)},
	} {
		if got := do("update_shape", args); got == "" {
			t.Errorf("%s: was accepted", name)
		}
	}
	if len(b.raw) != 0 {
		t.Errorf("refused updates reached update-file: %d", len(b.raw))
	}
}

func TestMoveShapes(t *testing.T) {
	do, b := richSession(t)
	if got := do("move_shape", map[string]any{"shape_ids": []string{rectID, grpID}, "dx": 10, "dy": -5}); got != "" {
		t.Fatal(got)
	}
	// A shape listed with its own parent moves once, with the parent.
	if got := do("move_shape", map[string]any{"shape_ids": []string{grpID, grp1ID}, "dx": 1, "dy": 1}); got != "" {
		t.Fatal(got)
	}
	if got := do("move_shape", map[string]any{"shape_ids": []string{rectID}, "dx": 1e12, "dy": 0}); got == "" {
		t.Error("a huge dx was accepted")
	}
	if got := do("move_shape", map[string]any{"shape_ids": []string{penpot.ZeroID}, "dx": 1, "dy": 0}); got == "" {
		t.Error("moving the root was accepted")
	}
	if len(b.raw) != 2 {
		t.Errorf("writes = %d, want 2", len(b.raw))
	}
}

func TestReparentRules(t *testing.T) {
	do, b := richSession(t)
	for name, args := range map[string]map[string]any{
		"into rect":     {"shape_ids": []string{ellipID}, "parent_id": rectID},
		"into itself":   {"shape_ids": []string{boardID}, "parent_id": boardID},
		"into child":    {"shape_ids": []string{grpID}, "parent_id": grp1ID},
		"into own kid":  {"shape_ids": []string{boardID}, "parent_id": rectID},
		"bad parent id": {"shape_ids": []string{ellipID}, "parent_id": "x"},
		"bad index":     {"shape_ids": []string{ellipID}, "parent_id": board2, "index": -1},
		"unknown":       {"shape_ids": []string{ellipID}, "parent_id": "d0000000-0000-4000-8000-0000000000ff"},
	} {
		if got := do("reparent_shape", args); got == "" {
			t.Errorf("%s: was accepted", name)
		}
	}
	if len(b.raw) != 0 {
		t.Fatalf("refused reparents reached update-file: %d", len(b.raw))
	}
	for name, args := range map[string]map[string]any{
		"into a board": {"shape_ids": []string{ellipID}, "parent_id": board2, "index": 0},
		"into a group": {"shape_ids": []string{ellipID}, "parent_id": grpID},
		"to the root":  {"shape_ids": []string{rectID}, "parent_id": penpot.ZeroID},
		"a group out":  {"shape_ids": []string{grp1ID}, "parent_id": board2},
	} {
		if got := do("reparent_shape", args); got != "" {
			t.Errorf("%s: %s", name, got)
		}
	}
}

func TestDeleteShapesCleansEmptyGroups(t *testing.T) {
	do, b := richSession(t)
	if got := do("delete_shapes", map[string]any{"shape_ids": []string{grp1ID, grp2ID}}); got != "" {
		t.Fatal(got)
	}
	raw := b.raw[0]
	if !strings.Contains(raw, `"id":"`+grpID+`"`) || strings.Count(raw, `"del-obj"`) != 3 {
		t.Errorf("the emptied group should go too: %s", raw)
	}
	if got := do("delete_shapes", map[string]any{"shape_ids": []string{grp1ID}}); got != "" {
		t.Fatal(got)
	}
	if !strings.Contains(b.raw[1], `"reg-objects"`) {
		t.Errorf("a group with a child left should be refitted: %s", b.raw[1])
	}
	if got := do("delete_shapes", map[string]any{"shape_ids": []string{rectID, rectID}}); got == "" {
		t.Error("a repeated id was accepted")
	}
}

func TestGroupAndBool(t *testing.T) {
	do, b := richSession(t)
	if got := do("create_group", map[string]any{"shape_ids": []string{rectID, textID}, "name": "Pair"}); got != "" {
		t.Fatal(got)
	}
	if len(b.raw) != 2 || b.ctypes[1] != "application/transit+json" {
		t.Errorf("a group is a JSON add then a transit move: %v", b.ctypes)
	}
	if got := do("create_bool", map[string]any{"shape_ids": []string{grp1ID, grp2ID}, "name": "Cut", "bool_type": "difference"}); got != "" {
		t.Fatal(got)
	}
	if got := do("create_bool", map[string]any{"shape_ids": []string{grp1ID, grp2ID}, "name": "Fill", "fill": "#ff00ff"}); got != "" {
		t.Fatal(got)
	}
	for name, tc := range map[string]struct {
		tool string
		args map[string]any
	}{
		"group with bool type": {"create_group", map[string]any{"shape_ids": []string{rectID}, "name": "g", "bool_type": "union"}},
		"bool with one shape":  {"create_bool", map[string]any{"shape_ids": []string{rectID}, "name": "b"}},
		"bad bool op":          {"create_bool", map[string]any{"shape_ids": []string{grp1ID, grp2ID}, "name": "b", "bool_type": "xor"}},
		"different parents":    {"create_group", map[string]any{"shape_ids": []string{rectID, ellipID}, "name": "g"}},
		"empty name":           {"create_group", map[string]any{"shape_ids": []string{rectID}, "name": ""}},
		"bad fill":             {"create_bool", map[string]any{"shape_ids": []string{grp1ID, grp2ID}, "name": "b", "fill": "red"}},
	} {
		if got := do(tc.tool, tc.args); got == "" {
			t.Errorf("%s: was accepted", name)
		}
	}
}

func TestEditHelpers(t *testing.T) {
	if atof("12.5", 1) != 12.5 || atof("x", 7) != 7 || atof("NaN", 3) != 3 {
		t.Error("atof")
	}
	if firstFill(map[string]any{}) != "#D9D9D9" {
		t.Error("firstFill default")
	}
	if got := firstFill(map[string]any{"fills": []any{map[string]any{"fillColor": "#abcdef"}}}); got != "#abcdef" {
		t.Errorf("firstFill = %q", got)
	}
	if firstStroke(map[string]any{}) != "" {
		t.Error("firstStroke default")
	}
	if got := firstStroke(map[string]any{"strokes": []any{map[string]any{"strokeColor": "#010101"}}}); got != "#010101" {
		t.Errorf("firstStroke = %q", got)
	}
	if leaf, _ := firstLeaf(map[string]any{"type": "root"}); leaf != nil {
		t.Error("firstLeaf on empty content")
	}
	if got := plainText(map[string]any{"children": []any{
		map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "a"}, map[string]any{"text": "b"}}},
		map[string]any{"type": "paragraph", "children": []any{map[string]any{"text": "c"}}},
	}}); got != "ab\nc" {
		t.Errorf("plainText = %q", got)
	}
}
