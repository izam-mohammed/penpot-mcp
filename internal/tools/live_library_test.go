//go:build live

package tools

import (
	"os"
	"path/filepath"
	"testing"
)

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

// liveRoot is a per-test working directory holding a small palette file.
func liveRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "palette.css"), []byte(":root{--live-sky:#38BDF8;--live-sand:#F5D0A9}"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestLiveTokens imports a CSS palette twice (the second run adds nothing),
// replaces the live file's tokens with one set, and applies a colour and a
// radius token to the copies TestLiveLibrary placed.
func TestLiveTokens(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)

	var first, second importOut
	mustCall(t, cs, "import_palette", map[string]any{"file_id": fileID, "path": "palette.css", "group": "live"}, &first)
	mustCall(t, cs, "import_palette", map[string]any{"file_id": fileID, "path": "palette.css", "group": "live"}, &second)
	if second.Added != 0 || second.Updated != len(second.Colors) {
		t.Errorf("re-import added colours: first %+v, second %+v", first, second)
	}

	var synced syncOut
	mustCall(t, cs, "sync_tokens", map[string]any{"file_id": fileID, "set_name": "live", "tokens": map[string]any{
		"color":  map[string]any{"accent": map[string]any{"$value": "#E11D48", "$type": "color"}},
		"radius": map[string]any{"pill": map[string]any{"$value": "24", "$type": "borderRadius"}},
	}}, &synced)
	t.Logf("synced %d tokens in %v", synced.Tokens, synced.Sets)

	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID, "depth": 1}, &tree)
	var copies []string
	for _, s := range tree.Shapes {
		if s.Name == "live button copy" {
			copies = append(copies, s.ID)
		}
	}
	if len(copies) == 0 {
		t.Skip("no live button copies; run TestLiveLibrary first")
	}
	var applied changedOut
	mustCall(t, cs, "apply_token", map[string]any{"file_id": fileID, "shape_ids": copies[:1], "token": "color.accent", "attribute": "fill"}, &applied)
	mustCall(t, cs, "apply_token", map[string]any{"file_id": fileID, "shape_ids": copies[:1], "token": "radius.pill", "attribute": "radius"}, &applied)
}
