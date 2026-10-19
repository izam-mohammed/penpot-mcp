package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

func lastChanges(t *testing.T, b *backend) []any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.updates) == 0 {
		t.Fatal("no update-file call")
	}
	changes, _ := b.updates[len(b.updates)-1]["changes"].([]any)
	return changes
}

func TestPathRejectsGarbage(t *testing.T) {
	cs, b := session(t)
	for _, d := range []string{
		"",
		"L10 10",          // must start with M
		"M10 10 X20 20",   // unknown command
		"M10",             // missing y
		"M10 10 L1e999 0", // overflow
		"M0 0 " + strings.Repeat("L1 1 ", penpot.MaxPathSegments+1),
	} {
		res := call(t, cs, "create_path", map[string]any{"file_id": fileID, "name": "p", "d": d}, nil)
		if !res.IsError {
			t.Errorf("d %.30q was accepted", d)
		}
	}
	if b.count("update-file") != 0 {
		t.Error("a bad path reached update-file")
	}

	var out shapeOut
	res := call(t, cs, "create_path", map[string]any{"file_id": fileID, "name": "tri", "d": "M10 10 h80 l-40 60 z"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	obj := lastChanges(t, b)[0].(map[string]any)["obj"].(map[string]any)
	if obj["type"] != "path" || out.Width != 80 || out.Height != 60 {
		t.Errorf("path box %v x %v, type %v", out.Width, out.Height, obj["type"])
	}
	if s, _ := obj["strokes"].([]any); len(s) != 1 {
		t.Errorf("a path with no fill should get a default stroke, got %v", obj["strokes"])
	}
}

func TestImageRejectsNonImage(t *testing.T) {
	cs, b := session(t)
	dir := t.TempDir()
	text := filepath.Join(dir, "notes.png")
	if err := os.WriteFile(text, []byte("secret notes, not a picture"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{text, dir, "relative.png", filepath.Join(dir, "missing.png")} {
		res := call(t, cs, "create_image", map[string]any{"file_id": fileID, "image_path": p, "x": 0, "y": 0}, nil)
		if !res.IsError {
			t.Errorf("%s was accepted", p)
		}
		if strings.Contains(errText(res), "secret") {
			t.Errorf("error leaks the file's bytes: %s", errText(res))
		}
	}
	if b.count("upload-file-media-object") != 0 {
		t.Error("a non-image was uploaded")
	}

	img := filepath.Join(dir, "swatch.png")
	// The 8-byte PNG signature is enough for content sniffing.
	if err := os.WriteFile(img, []byte("\x89PNG\r\n\x1a\n0000"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out shapeOut
	res := call(t, cs, "create_image", map[string]any{"file_id": fileID, "image_path": img, "x": 0, "y": 0, "width": 60}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.Width != 60 || out.Height != 40 {
		t.Errorf("image box %v x %v, want 60 x 40 (aspect kept)", out.Width, out.Height)
	}
}

func TestDeleteRefusesRoot(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "delete_shapes", map[string]any{"file_id": fileID, "shape_ids": []string{penpot.ZeroID}}, nil)
	if !res.IsError {
		t.Fatal("deleting the root frame was accepted")
	}
	if b.count("update-file") != 0 {
		t.Fatal("update-file was called")
	}

	var out changedOut
	res = call(t, cs, "delete_shapes", map[string]any{"file_id": fileID, "shape_ids": []string{boardID, rectID}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	changes := lastChanges(t, b)
	if len(changes) != 1 || changes[0].(map[string]any)["id"] != boardID {
		t.Errorf("want one del-obj for the board (its rect goes with it), got %v", changes)
	}
	if len(out.Shapes) != 1 || out.Shapes[0].Name != "Login" {
		t.Errorf("deleted names = %+v", out.Shapes)
	}
}

func TestApplyChangesValidates(t *testing.T) {
	cs, b := session(t)
	bad := []map[string]any{
		{"type": "mov-objects", "parentId": boardID, "shapes": []string{rectID}},
		{"type": "drop-everything"},
		{"type": "del-obj", "id": penpot.ZeroID},
		{"type": "mod-obj", "id": rectID, "pageId": "not-a-page"},
	}
	for _, c := range bad {
		res := call(t, cs, "apply_changes", map[string]any{"file_id": fileID, "changes": []map[string]any{c}}, nil)
		if !res.IsError {
			t.Errorf("%v was accepted", c)
		}
	}
	if b.count("update-file") != 0 {
		t.Fatal("a refused change reached update-file")
	}
	res := call(t, cs, "apply_changes", map[string]any{"file_id": fileID, "changes": []map[string]any{
		{"type": "mod-obj", "id": rectID, "operations": []map[string]any{{"type": "assign", "value": map[string]any{"name": "B"}}}},
	}}, nil)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if got := lastChanges(t, b)[0].(map[string]any)["pageId"]; got != pageID {
		t.Errorf("pageId = %v, want the resolved page", got)
	}
}

func TestReparentSendsTransit(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "reparent_shape", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "parent_id": penpot.ZeroID, "index": 0}, nil)
	if res.IsError {
		t.Fatal(errText(res))
	}
	b.mu.Lock()
	ctypes, raw := b.ctypes, b.raw
	b.mu.Unlock()
	if len(ctypes) != 1 || ctypes[0] != "application/transit+json" {
		t.Fatalf("content types %v", ctypes)
	}
	for _, want := range []string{`"~:mov-objects"`, `"~u` + rectID + `"`, `"~:index",0`, `"~:session-id"`} {
		if !strings.Contains(raw[0], want) {
			t.Errorf("body lacks %s: %s", want, raw[0])
		}
	}

	if res := call(t, cs, "reparent_shape", map[string]any{"file_id": fileID, "shape_ids": []string{boardID}, "parent_id": rectID}, nil); !res.IsError {
		t.Error("moving into a rect was accepted")
	}
}

func TestMoveCarriesChildren(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "move_shape", map[string]any{"file_id": fileID, "shape_ids": []string{boardID, rectID}, "dx": 10, "dy": -5}, nil)
	if res.IsError {
		t.Fatal(errText(res))
	}
	changes := lastChanges(t, b)
	if len(changes) != 2 {
		t.Fatalf("want the board and its rect moved once each, got %d changes", len(changes))
	}
	rect := changes[1].(map[string]any)["operations"].([]any)[0].(map[string]any)["value"].(map[string]any)
	if rect["x"] != 60.0 || rect["y"] != 45.0 {
		t.Errorf("rect moved to %v,%v, want 60,45", rect["x"], rect["y"])
	}
}

func TestCreateGroupIsTwoWrites(t *testing.T) {
	cs, b := session(t)
	var out groupOut
	res := call(t, cs, "create_group", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "name": "G"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.ctypes) != 2 || b.ctypes[0] != "application/json" || b.ctypes[1] != "application/transit+json" {
		t.Fatalf("content types %v, want JSON add-obj then transit move", b.ctypes)
	}
	if !strings.Contains(b.raw[1], `"~:parent-id","~u`+out.ShapeID+`"`) {
		t.Errorf("transit body does not move into the group: %s", b.raw[1])
	}
}

func TestUpdateShapeRules(t *testing.T) {
	cs, b := session(t)
	for _, args := range []map[string]any{
		{"shape_id": rectID},                     // nothing to change
		{"shape_id": rectID, "text": "hi"},       // not a text shape
		{"shape_id": boardID, "rotation": 30},    // boards do not rotate
		{"shape_id": rectID, "opacity": 2},       // out of range
		{"shape_id": penpot.ZeroID, "name": "x"}, // root
	} {
		args["file_id"] = fileID
		if res := call(t, cs, "update_shape", args, nil); !res.IsError {
			t.Errorf("%v was accepted", args)
		}
	}
	if b.count("update-file") != 0 {
		t.Fatal("a refused update reached update-file")
	}
	res := call(t, cs, "update_shape", map[string]any{"file_id": fileID, "shape_id": rectID, "rotation": 90, "fill": "#00ff00"}, nil)
	if res.IsError {
		t.Fatal(errText(res))
	}
	v := lastChanges(t, b)[0].(map[string]any)["operations"].([]any)[0].(map[string]any)["value"].(map[string]any)
	if v["rotation"] != 90.0 || v["fills"].([]any)[0].(map[string]any)["fillColor"] != "#00FF00" {
		t.Errorf("update value %v", v)
	}
}

func TestDrawLayoutPlacesChildren(t *testing.T) {
	cs, b := session(t)
	var out drawOut
	res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": []map[string]any{
		{
			"ref": "row", "type": "board", "name": "row", "x": 0, "y": 0, "width": 300, "height": 100,
			"layout": "flex", "gap": 10, "padding": 20, "align_items": "center",
		},
		{"parent": "row", "type": "rect", "name": "a", "x": 500, "y": 500, "width": 50, "height": 60},
		{"parent": "row", "type": "rect", "name": "b", "x": 500, "y": 500, "width": 40, "height": 20},
	}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	a, bb := out.Shapes[1], out.Shapes[2]
	if a.X != 20 || a.Y != 20 || bb.X != 80 || bb.Y != 40 {
		t.Errorf("a at %v,%v and b at %v,%v; want 20,20 and 80,40", a.X, a.Y, bb.X, bb.Y)
	}
	changes := lastChanges(t, b)
	if changes[2].(map[string]any)["index"] != 0.0 {
		t.Error("layout children should be added bottom-first so the first listed is on top")
	}
}
