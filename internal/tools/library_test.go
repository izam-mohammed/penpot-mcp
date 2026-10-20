package tools

import (
	"strings"
	"testing"
)

const (
	colorID     = "c1000000-0000-4000-8000-000000000001"
	componentID = "c2000000-0000-4000-8000-000000000001"
)

func TestColorIdempotent(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"Primary","path":"Brand","color":"#2563eb","opacity":1}}`

	var again assetOut
	if res := call(t, cs, "add_color", map[string]any{"file_id": fileID, "name": "Brand / Primary", "hex": "#1D4ED8"}, &again); res.IsError {
		t.Fatal(errText(res))
	}
	if again.Created || again.ID != colorID {
		t.Errorf("same group and name should update %s: %+v", colorID, again)
	}
	if !strings.Contains(b.raw[0], `"type":"mod-color"`) || !strings.Contains(b.raw[0], `"color":"#1d4ed8"`) {
		t.Errorf("want a mod-color with the new hex, lowercased: %s", b.raw[0])
	}

	var fresh assetOut
	if res := call(t, cs, "add_color", map[string]any{"file_id": fileID, "name": "Accent", "path": "Brand", "hex": "#F59E0B", "opacity": 0.5}, &fresh); res.IsError {
		t.Fatal(errText(res))
	}
	if !fresh.Created || fresh.ID == colorID || fresh.Path != "Brand" {
		t.Errorf("a new name should add a colour: %+v", fresh)
	}
	if !strings.Contains(b.raw[1], `"type":"add-color"`) {
		t.Errorf("want add-color: %s", b.raw[1])
	}

	res := call(t, cs, "add_color", map[string]any{"file_id": fileID, "name": "Bad", "hex": "blue"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "#RRGGBB") {
		t.Errorf("bad hex: got %q", errText(res))
	}
}

func TestTypographyStrings(t *testing.T) {
	cs, b := session(t)
	var out assetOut
	if res := call(t, cs, "add_typography", map[string]any{
		"file_id": fileID, "name": "Heading / H1", "font_family": "Inter", "font_size": 32, "font_weight": 700,
	}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Path != "Heading" || out.Name != "H1" || !out.Created {
		t.Errorf("typography = %+v", out)
	}
	for _, want := range []string{`"type":"add-typography"`, `"fontSize":"32"`, `"fontWeight":"700"`, `"lineHeight":"1.2"`, `"fontId":"gfont-inter"`} {
		if !strings.Contains(b.raw[0], want) {
			t.Errorf("typography change lacks %s: %s", want, b.raw[0])
		}
	}
}

func TestCreateComponent(t *testing.T) {
	cs, b := session(t)
	var out componentOut
	if res := call(t, cs, "create_component", map[string]any{"file_id": fileID, "shape_id": boardID, "name": "Screens / Login"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Name != "Login" || out.Path != "Screens" {
		t.Errorf("component = %+v", out)
	}
	for _, want := range []string{`"type":"add-component"`, `"mainInstanceId":"` + boardID + `"`, `"mainInstance":true`, `"componentRoot":true`} {
		if !strings.Contains(b.raw[0], want) {
			t.Errorf("changes lack %s: %s", want, b.raw[0])
		}
	}
	res := call(t, cs, "create_component", map[string]any{"file_id": fileID, "shape_id": rectID}, nil)
	if !res.IsError || !strings.Contains(errText(res), "only a board") {
		t.Errorf("rect: got %q", errText(res))
	}
}

func TestInstanceRefs(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"components":{"` + componentID + `":{"id":"` + componentID + `","name":"Login","path":"","mainInstanceId":"` +
		boardID + `","mainInstancePage":"` + pageID + `"}}`
	var out instanceOut
	if res := call(t, cs, "instantiate_component", map[string]any{"file_id": fileID, "component_id": componentID, "x": 500, "y": 100}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.X != 500 || out.Y != 100 || out.Shapes != 2 || out.Name != "Login" {
		t.Errorf("instance = %+v", out)
	}
	changes := b.updates[0]["changes"].([]any)
	root := changes[0].(map[string]any)["obj"].(map[string]any)
	kid := changes[1].(map[string]any)["obj"].(map[string]any)
	if root["shapeRef"] != boardID || root["componentId"] != componentID || root["componentFile"] != fileID || root["componentRoot"] != true {
		t.Errorf("copy root does not point at the main: %v", root)
	}
	if _, ok := root["mainInstance"]; ok {
		t.Error("a copy must not be a main instance")
	}
	if kid["shapeRef"] != rectID || kid["parentId"] != out.ShapeID || kid["frameId"] != out.ShapeID {
		t.Errorf("copy child should point at the main child and sit in the copy: %v", kid)
	}
	if kid["x"] != 550.0 || kid["y"] != 150.0 {
		t.Errorf("copy child at %v, %v, want 550, 150", kid["x"], kid["y"])
	}
	shapes := root["shapes"].([]any)
	if len(shapes) != 1 || shapes[0] != kid["id"] {
		t.Errorf("copy root's children = %v, want the copied child", shapes)
	}

	res := call(t, cs, "instantiate_component", map[string]any{"file_id": fileID, "component_id": colorID, "x": 0, "y": 0}, nil)
	if !res.IsError || !strings.Contains(errText(res), "not in the library") {
		t.Errorf("unknown component: got %q", errText(res))
	}
}

func TestListLibrary(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"Primary","path":"Brand","color":"#2563eb"}},` +
		`"components":{"` + componentID + `":{"id":"` + componentID + `","name":"Old","deleted":true}}`
	var out libraryOut
	if res := call(t, cs, "list_library", map[string]any{"file_id": fileID}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if len(out.Colors) != 1 || out.Colors[0].Hex != "#2563eb" || out.Colors[0].Opacity != 1 {
		t.Errorf("colors = %+v", out.Colors)
	}
	if len(out.Components) != 0 {
		t.Errorf("deleted components should not be listed: %+v", out.Components)
	}
}

// TestInstanceTabStep: with the file open in a tab, placing a copy lets the
// tab reflow; without a tab the reply has no tab step.
func TestInstanceTabStep(t *testing.T) {
	lib := `,"components":{"` + componentID + `":{"id":"` + componentID + `","name":"Login","path":"","mainInstanceId":"` +
		boardID + `","mainInstancePage":"` + pageID + `"}}`
	args := map[string]any{"file_id": fileID, "component_id": componentID, "x": 500, "y": 100}
	cs, b := session(t)
	b.lib = lib
	var out instanceOut
	if res := call(t, cs, "instantiate_component", args, &out); res.IsError || out.Tab != nil {
		t.Errorf("without a tab: %+v %q", out.Tab, errText(res))
	}
	tb := &tab{openFile: fileID, answer: `{"result":{}}`}
	cs, b = sessionWith(t, withRoute(tb.serve(t)))
	b.lib = lib
	if res := call(t, cs, "instantiate_component", args, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Tab == nil {
		t.Fatal("with the file open in a tab the reply should have a tab step")
	}
}
