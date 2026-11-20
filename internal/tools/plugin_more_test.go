package tools

import (
	"strings"
	"testing"
)

func TestTabToolsRunCode(t *testing.T) {
	tb := &tab{openFile: fileID}
	cs, b := sessionWith(t, withRoute(tb.serve(t)))
	shapes := []string{rectID, boardID}
	one := func(m map[string]any) map[string]any { m["file_id"] = fileID; return m }

	cases := []struct {
		name string
		tool string
		args map[string]any
		code string // a fragment the tab should have been sent
	}{
		{"execute", "execute_code", one(map[string]any{"code": "return 1"}), "return 1"},
		{"selection", "get_selection", one(map[string]any{}), "penpot.selection"},
		{"reflow", "reflow", one(map[string]any{}), "waitForLayoutUpdate"},
		{"markup", "generate_markup", one(map[string]any{"shape_ids": shapes, "type": "svg"}), "generateMarkup"},
		{"markup default", "generate_markup", one(map[string]any{"shape_ids": shapes}), `type: "html"`},
		{"style", "generate_style", one(map[string]any{"shape_ids": shapes, "with_children": true}), "generateStyle"},
		{"font faces", "generate_font_faces", one(map[string]any{"shape_ids": shapes}), "generateFontFaces"},
		{"detach", "detach", one(map[string]any{"shape_ids": shapes}), "detach"},
		{"reset", "reset_overrides", one(map[string]any{"shape_ids": shapes}), "resetOverrides"},
		{"swap", "swap_component", one(map[string]any{"shape_id": rectID, "component_id": componentID}), "swapComponent"},
		{"variant", "switch_variant", one(map[string]any{"shape_id": rectID, "property": "Size", "value": "Large"}), "switchVariant"},
		{"combine", "combine_as_variants", one(map[string]any{"shape_ids": shapes}), "combineAsVariants"},
		{"active users", "active_users", one(map[string]any{}), "activeUsers"},
		{"viewport read", "viewport", one(map[string]any{}), "penpot.viewport"},
		{"viewport move", "viewport", one(map[string]any{"shape_ids": shapes, "zoom": 2, "center_x": 10, "center_y": 20}), "zoomIntoView"},
		{"svg", "import_svg", one(map[string]any{"svg": "<svg></svg>", "x": 5, "y": 6, "name": "Logo"}), "createShapeFromSvg"},
		{"svg without name", "import_svg", one(map[string]any{"svg": "<svg></svg>", "x": 0, "y": 0}), "createShapeFromSvg"},
	}
	for _, c := range cases {
		if res := call(t, cs, c.tool, c.args, nil); res.IsError {
			t.Errorf("%s: %s", c.name, errText(res))
			continue
		}
		last := tb.ran()
		if got := last[len(last)-1]; !strings.Contains(got, c.code) {
			t.Errorf("%s: the tab was sent %q, want it to hold %q", c.name, got, c.code)
		}
	}
	if got := call(t, cs, "current_file", nil, nil); got.IsError {
		t.Errorf("current_file: %s", errText(got))
	}
	_ = b
}

func TestTabToolsRefuseBadInput(t *testing.T) {
	tb := &tab{openFile: fileID}
	cs, _ := sessionWith(t, withRoute(tb.serve(t)))
	long := strings.Repeat("x", 300_000)
	for name, tc := range map[string]struct {
		tool string
		args map[string]any
	}{
		"empty code":      {"execute_code", map[string]any{"file_id": fileID, "code": "  "}},
		"long code":       {"execute_code", map[string]any{"file_id": fileID, "code": long}},
		"bad file":        {"reflow", map[string]any{"file_id": "x"}},
		"markup type":     {"generate_markup", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "type": "pdf"}},
		"markup no ids":   {"generate_markup", map[string]any{"file_id": fileID, "shape_ids": []string{}}},
		"detach no ids":   {"detach", map[string]any{"file_id": fileID, "shape_ids": []string{}}},
		"swap bad shape":  {"swap_component", map[string]any{"file_id": fileID, "shape_id": "x", "component_id": componentID}},
		"swap bad comp":   {"swap_component", map[string]any{"file_id": fileID, "shape_id": rectID, "component_id": "x"}},
		"variant bad id":  {"switch_variant", map[string]any{"file_id": fileID, "shape_id": "x", "property": "a", "value": "b"}},
		"variant no prop": {"switch_variant", map[string]any{"file_id": fileID, "shape_id": rectID, "property": "", "value": "b"}},
		"variant no val":  {"switch_variant", map[string]any{"file_id": fileID, "shape_id": rectID, "property": "a", "value": ""}},
		"combine one":     {"combine_as_variants", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}}},
		"viewport ids":    {"viewport", map[string]any{"file_id": fileID, "shape_ids": []string{"x"}}},
		"viewport zoom":   {"viewport", map[string]any{"file_id": fileID, "zoom": 500}},
		"viewport centre": {"viewport", map[string]any{"file_id": fileID, "center_x": 1}},
		"svg both":        {"import_svg", map[string]any{"file_id": fileID, "svg": "<svg/>", "path": "a.svg", "x": 0, "y": 0}},
		"svg wrong ext":   {"import_svg", map[string]any{"file_id": fileID, "path": "a.png", "x": 0, "y": 0}},
		"svg missing":     {"import_svg", map[string]any{"file_id": fileID, "path": "none.svg", "x": 0, "y": 0}},
		"svg not svg":     {"import_svg", map[string]any{"file_id": fileID, "svg": "hello", "x": 0, "y": 0}},
		"svg bad x":       {"import_svg", map[string]any{"file_id": fileID, "svg": "<svg/>", "x": 1e12, "y": 0}},
		"svg bad y":       {"import_svg", map[string]any{"file_id": fileID, "svg": "<svg/>", "x": 0, "y": 1e12}},
	} {
		if res := call(t, cs, tc.tool, tc.args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if got := len(tb.ran()); got > 0 {
		t.Errorf("refused calls ran code in the tab: %d", got)
	}
}

func TestImportSVGFromFileAndTabText(t *testing.T) {
	tb := &tab{openFile: fileID, text: `"<div>hi</div>"`}
	cs, b := sessionWith(t, withRoute(tb.serve(t)))
	writeFile(t, b.root, "logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`)
	if res := call(t, cs, "import_svg", map[string]any{"file_id": fileID, "path": "logo.svg", "x": 0, "y": 0}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	var out textOut
	res := call(t, cs, "generate_markup", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}}, &out)
	if res.IsError || out.Text != "<div>hi</div>" {
		t.Errorf("markup = %q, %q", out.Text, errText(res))
	}
	tb.text = `{"not":"text"}`
	res = call(t, cs, "generate_style", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}}, &out)
	if res.IsError || !strings.Contains(out.Text, "not") {
		t.Errorf("a non-text result should come back as JSON: %q %q", out.Text, errText(res))
	}
}
