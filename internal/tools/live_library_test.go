//go:build live

package tools

import "testing"

// TestLiveLibrary adds two colours and a type style to the live file, makes
// a button component (once; later runs reuse it) and places two copies,
// replacing the copies of an earlier run.
func TestLiveLibrary(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)

	for _, c := range []map[string]any{
		{"file_id": fileID, "name": "live / Primary", "hex": "#2563EB"},
		{"file_id": fileID, "name": "live / Ink", "hex": "#101828"},
	} {
		var out assetOut
		mustCall(t, cs, "add_color", c, &out)
		var again assetOut
		mustCall(t, cs, "add_color", c, &again)
		if again.Created || again.ID != out.ID {
			t.Errorf("re-adding %v made a new colour: %+v then %+v", c["name"], out, again)
		}
	}
	var typo assetOut
	mustCall(t, cs, "add_typography", map[string]any{
		"file_id": fileID, "name": "live / Button", "font_family": "Inter", "font_size": 16, "font_weight": 600,
	}, &typo)

	var lib libraryOut
	mustCall(t, cs, "list_library", map[string]any{"file_id": fileID}, &lib)
	componentID := ""
	for _, c := range lib.Components {
		if c.Path == "live" && c.Name == "Button" {
			componentID = c.ID
		}
	}
	if componentID == "" {
		var drawn drawOut
		mustCall(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": []map[string]any{
			{
				"ref": "b", "type": "board", "name": "live button", "x": 3900, "y": 0, "width": 160, "height": 48,
				"radius": 8, "fill": "#2563EB", "layout": "flex", "justify": "center", "align_items": "center",
			},
			{"parent": "b", "type": "text", "text": "Continue", "font_family": "Inter", "font_size": 16, "font_weight": 600, "color": "#FFFFFF"},
		}}, &drawn)
		var comp componentOut
		mustCall(t, cs, "create_component", map[string]any{
			"file_id": fileID, "shape_id": drawn.Shapes[0].ShapeID, "name": "live / Button",
		}, &comp)
		componentID = comp.ComponentID
		t.Logf("made component %s", componentID)
	}

	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID, "depth": 1}, &tree)
	var old []string
	for _, s := range tree.Shapes {
		if s.Name == "live button copy" {
			old = append(old, s.ID)
		}
	}
	if len(old) > 0 {
		var gone changedOut
		mustCall(t, cs, "delete_shapes", map[string]any{"file_id": fileID, "shape_ids": old}, &gone)
	}
	for _, y := range []float64{100, 180} {
		var copyOut instanceOut
		mustCall(t, cs, "instantiate_component", map[string]any{
			"file_id": fileID, "component_id": componentID, "x": 3900, "y": y, "name": "live button copy",
		}, &copyOut)
		if copyOut.Shapes != 2 {
			t.Errorf("copy has %d shapes, want 2 (board and text)", copyOut.Shapes)
		}
	}
	mustCall(t, cs, "list_library", map[string]any{"file_id": fileID}, &lib)
	t.Logf("library: %d colours, %d type styles, %d components", len(lib.Colors), len(lib.Typographies), len(lib.Components))
}
