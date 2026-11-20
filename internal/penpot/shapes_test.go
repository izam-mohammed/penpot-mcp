package penpot

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	pageA = "a0000000-0000-4000-8000-000000000001"
	shape = "b0000000-0000-4000-8000-000000000001"
)

func testBox() Box {
	return Box{ID: shape, ParentID: ZeroID, Name: "box", X: 10, Y: 20, W: 100, H: 50}
}

func TestBoxFrameDefaultsToParent(t *testing.T) {
	b := testBox()
	b.ParentID = "c0000000-0000-4000-8000-000000000001"
	if got := RectObject(b, "", 0)["frameId"]; got != b.ParentID {
		t.Errorf("frameId = %v, want the parent", got)
	}
	b.FrameID = "d0000000-0000-4000-8000-000000000001"
	if got := RectObject(b, "", 0)["frameId"]; got != b.FrameID {
		t.Errorf("frameId = %v, want the board", got)
	}
}

func TestSimpleShapeObjects(t *testing.T) {
	b := testBox()
	board := BoardObject(b, "#aabbcc", 4, true)
	if board["type"] != TypeBoard || board["showContent"] != false {
		t.Errorf("board = %v", board)
	}
	fl := board["fills"].([]map[string]any)
	if fl[0]["fillColor"] != "#AABBCC" || board["r3"] != 4.0 {
		t.Errorf("board fills or radius: %v", board)
	}
	if got := RectObject(b, "", 2)["fills"].([]map[string]any); len(got) != 0 {
		t.Errorf("empty fill should give no fills: %v", got)
	}
	if got := EllipseObject(b, "#fff")["type"]; got != TypeEllipse {
		t.Errorf("ellipse type = %v", got)
	}
	if got := GroupObject(b)["type"]; got != TypeGroup {
		t.Errorf("group type = %v", got)
	}
	bo := BoolObject(b, "union", "#000")
	if bo["type"] != TypeBool || bo["boolType"] != "union" {
		t.Errorf("bool = %v", bo)
	}
	st := Stroke("#abc", 2)
	if st["strokeColor"] != "#ABC" || st["strokeWidth"] != 2.0 {
		t.Errorf("stroke = %v", st)
	}
}

func TestTextObjectContent(t *testing.T) {
	s := TextStyle{
		FontID: "gfont-inter", FontFamily: "Inter", FontVariantID: "regular",
		FontWeight: "400", FontStyle: "normal", FontSize: 14.5, LineHeight: 1.2, Color: "#112233",
	}
	obj := TextObject(testBox(), "one\ntwo", s, "auto-height")
	if obj["growType"] != "auto-height" || obj["type"] != TypeText {
		t.Fatalf("obj = %v", obj)
	}
	data, err := json.Marshal(obj["content"])
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{`"fontSize":"14.5"`, `"lineHeight":"1.2"`, `"text":"one"`, `"text":"two"`, `"textAlign":"left"`} {
		if !strings.Contains(got, want) {
			t.Errorf("content missing %s in %s", want, got)
		}
	}
	s.Align = "center"
	data, _ = json.Marshal(TextContent("x", s))
	if !strings.Contains(string(data), `"textAlign":"center"`) {
		t.Errorf("align not kept: %s", data)
	}
}

func TestPathObjectFitsBoxToSegments(t *testing.T) {
	segs, err := ParsePath("M 0 0 L 40 0 L 40 30 Z")
	if err != nil {
		t.Fatal(err)
	}
	obj := PathObject(testBox(), segs, "#fff", []map[string]any{Stroke("#000", 1)})
	if obj["type"] != TypePath || obj["width"] != 40.0 || obj["height"] != 30.0 {
		t.Errorf("path = %v", obj)
	}
}

func TestImageObject(t *testing.T) {
	m := Media{ID: "e0000000-0000-4000-8000-000000000001", Name: "a.png", Width: 10, Height: 20, Mtype: "image/png"}
	obj := ImageObject(testBox(), m, 3)
	fill := obj["fills"].([]map[string]any)[0]["fillImage"].(map[string]any)
	if fill["id"] != m.ID || fill["keepAspectRatio"] != true || obj["r1"] != 3.0 {
		t.Errorf("image = %v", obj)
	}
}

func TestChangeBuilders(t *testing.T) {
	obj := RectObject(testBox(), "#fff", 0)
	add := AddObj(pageA, obj)
	if add["type"] != "add-obj" || add["id"] != shape || add["pageId"] != pageA {
		t.Errorf("add = %v", add)
	}
	if at := AddObjAt(pageA, obj, 3); at["index"] != 3 {
		t.Errorf("index = %v", at["index"])
	}
	mod := ModObj(pageA, shape, map[string]any{"name": "n"})
	if mod["type"] != "mod-obj" || len(mod["operations"].([]map[string]any)) != 1 {
		t.Errorf("mod = %v", mod)
	}
	if del := DelObj(pageA, shape); del["type"] != "del-obj" {
		t.Errorf("del = %v", del)
	}
	if reg := RegObjects(pageA, []string{shape}); reg["type"] != "reg-objects" {
		t.Errorf("reg = %v", reg)
	}
}

func TestTransitChangeBuilders(t *testing.T) {
	mov := MovObjects(pageA, shape, []string{shape}, 2)
	if len(mov) != 5 {
		t.Errorf("mov with index has %d pairs, want 5", len(mov))
	}
	if top := MovObjects(pageA, shape, []string{shape}, -1); len(top) != 4 {
		t.Errorf("mov on top has %d pairs, want 4", len(top))
	}
	if reg := RegObjectsTransit(pageA, []string{shape}); len(reg) != 3 {
		t.Errorf("reg has %d pairs, want 3", len(reg))
	}
}

func TestTranslatePath(t *testing.T) {
	segs, err := ParsePath("M 0 0 L 10 0 C 1 2 3 4 5 6 Z")
	if err != nil {
		t.Fatal(err)
	}
	moved := TranslatePath(segs, 5, 7)
	x, y, _, _ := PathBounds(moved)
	if x != 5 || y != 7 {
		t.Errorf("moved bounds start at %v,%v, want 5,7", x, y)
	}
	if x0, y0, _, _ := PathBounds(segs); x0 != 0 || y0 != 0 {
		t.Errorf("the input moved: %v,%v", x0, y0)
	}
}
