package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type nodes = []map[string]any

func TestDrawGroupsAndPathsInsideLayouts(t *testing.T) {
	cs, b := session(t)
	var out drawOut
	res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": nodes{
		{"ref": "row", "type": "board", "name": "row", "x": 0, "y": 0, "width": 400, "height": 200, "layout": "flex", "gap": 10, "padding": 10},
		{"ref": "card", "parent": "row", "type": "group", "name": "card"},
		{"parent": "card", "type": "rect", "name": "bg", "x": 300, "y": 300, "width": 60, "height": 40, "fill": "#eeeeee", "stroke": "#000000", "stroke_width": 2, "rotation": 10},
		{"parent": "card", "type": "path", "name": "tick", "d": "M300 300 L320 320 L360 290", "stroke": "#00ff00"},
		{"ref": "shape", "parent": "row", "type": "bool", "name": "shape", "bool_type": "exclude"},
		{"parent": "shape", "type": "ellipse", "name": "a", "x": 0, "y": 0, "width": 30, "height": 30},
		{"parent": "shape", "type": "ellipse", "name": "b", "x": 10, "y": 10, "width": 30, "height": 30},
		{"type": "ellipse", "name": "free", "x": 500, "y": 500, "width": 10, "height": 10, "rotation": 45},
	}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if len(out.Shapes) != 8 {
		t.Fatalf("drew %d shapes, want 8", len(out.Shapes))
	}
	card := out.Shapes[1]
	if card.Type != "group" || card.X < 10 || card.X > 11 {
		t.Errorf("the card group should sit at the board's padding, got %+v", card)
	}
	if len(b.raw) == 0 || !strings.Contains(b.raw[0], `"boolType":"exclude"`) {
		t.Errorf("the bool's type is missing from the write")
	}
}

func TestDrawGridLayout(t *testing.T) {
	cs, _ := session(t)
	var out drawOut
	shapes := nodes{{"ref": "g", "type": "board", "name": "grid", "x": 0, "y": 0, "width": 300, "height": 300, "layout": "grid", "columns": 3, "gap": 5}}
	for range 5 {
		shapes = append(shapes, map[string]any{"parent": "g", "type": "rect", "name": "cell", "x": 0, "y": 0, "width": 20, "height": 20})
	}
	if res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": shapes}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if len(out.Shapes) != 6 {
		t.Errorf("drew %d shapes, want 6", len(out.Shapes))
	}
}

func TestDrawRefusals(t *testing.T) {
	cs, b := session(t)
	for name, shapes := range map[string]nodes{
		"empty":             {},
		"bad ref":           {{"ref": "no spaces", "type": "board", "name": "b", "width": 1, "height": 1}},
		"layout on rect":    {{"type": "rect", "name": "r", "width": 1, "height": 1, "layout": "flex"}},
		"bad layout":        {{"type": "board", "name": "b", "width": 1, "height": 1, "layout": "stack"}},
		"rotated board":     {{"type": "board", "name": "b", "width": 1, "height": 1, "rotation": 5}},
		"huge rotation":     {{"type": "rect", "name": "r", "width": 1, "height": 1, "rotation": 999}},
		"bad stroke":        {{"type": "rect", "name": "r", "width": 1, "height": 1, "stroke": "red"}},
		"bad stroke width":  {{"type": "rect", "name": "r", "width": 1, "height": 1, "stroke_width": -1}},
		"bad fill":          {{"type": "rect", "name": "r", "width": 1, "height": 1, "fill": "red"}},
		"bad radius":        {{"type": "rect", "name": "r", "width": 1, "height": 1, "radius": -1}},
		"bad width":         {{"type": "rect", "name": "r", "width": 0, "height": 1}},
		"bad height":        {{"type": "rect", "name": "r", "width": 1, "height": 0}},
		"no name":           {{"type": "rect", "name": "", "width": 1, "height": 1}},
		"bad path":          {{"type": "path", "name": "p", "d": "L1 1"}},
		"image no path":     {{"type": "image", "name": "i"}},
		"image bad size":    {{"type": "image", "name": "i", "image_path": "/x.png", "width": -1}},
		"bool op":           {{"ref": "b", "type": "bool", "name": "b", "bool_type": "xor"}, {"parent": "b", "type": "rect", "name": "r", "width": 1, "height": 1}},
		"group no children": {{"ref": "g", "type": "group", "name": "g"}},
		"parent unknown":    {{"parent": "nope", "type": "rect", "name": "r", "width": 1, "height": 1}},
		"unknown type":      {{"type": "star", "name": "s"}},
		"duplicate ref":     {{"ref": "a", "type": "board", "name": "a", "width": 1, "height": 1}, {"ref": "a", "type": "board", "name": "b", "width": 1, "height": 1}},
		"parent after":      {{"parent": "late", "type": "rect", "name": "r", "width": 1, "height": 1}, {"ref": "late", "type": "board", "name": "b", "width": 1, "height": 1}},
		"parent is rect":    {{"ref": "r", "type": "rect", "name": "r", "width": 1, "height": 1}, {"parent": "r", "type": "rect", "name": "c", "width": 1, "height": 1}},
	} {
		if res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": shapes}, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if len(b.raw) != 0 {
		t.Errorf("refused drawings reached update-file: %d", len(b.raw))
	}
}

func TestDrawImageFromFile(t *testing.T) {
	cs, b := session(t)
	dir := t.TempDir()
	png := filepath.Join(dir, "swatch.png")
	if err := os.WriteFile(png, pngOf(t, true), 0o600); err != nil {
		t.Fatal(err)
	}
	svg := filepath.Join(dir, "logo.svg")
	if err := os.WriteFile(svg, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{png, svg} {
		if res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": nodes{{"type": "image", "image_path": p, "x": 0, "y": 0, "width": 40, "height": 30, "radius": 4}}}, nil); res.IsError {
			t.Errorf("%s: %s", filepath.Base(p), errText(res))
		}
	}
	if b.count("upload-file-media-object") != 2 {
		t.Errorf("uploads = %d, want 2", b.count("upload-file-media-object"))
	}
}

func TestImageType(t *testing.T) {
	if got := imageType(pngOf(t, true), "a.png"); got != "image/png" {
		t.Errorf("png = %q", got)
	}
	if got := imageType([]byte(`<svg></svg>`), "a.SVG"); got != "image/svg+xml" {
		t.Errorf("svg = %q", got)
	}
	if got := imageType([]byte(`<svg></svg>`), "a.txt"); got != "" {
		t.Errorf("svg text without the extension = %q", got)
	}
	if got := imageType([]byte(strings.Repeat("a", 5000)+"<svg>"), "a.svg"); got != "" {
		t.Errorf("an svg tag past the head = %q", got)
	}
}

// An agent colours a text with `fill` as often as with `color`; both work.
func TestDrawTextFillIsItsColour(t *testing.T) {
	cs, b := session(t)
	var out drawOut
	res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": nodes{
		{"type": "text", "text": "by fill", "x": 0, "y": 0, "fill": "#ab1234"},
		{"type": "text", "text": "both", "x": 0, "y": 40, "fill": "#ab1234", "color": "#cd5678"},
	}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	raw := strings.Join(b.raw, "")
	if !strings.Contains(raw, "#AB1234") {
		t.Errorf("a text's fill was not used as its colour")
	}
	if !strings.Contains(raw, "#CD5678") {
		t.Errorf("color should win over fill on a text")
	}
}
