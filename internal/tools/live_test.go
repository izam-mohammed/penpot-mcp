//go:build live

package tools

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// liveFile is the file the live run draws into. Create it in any project
// first; `make live` never creates or deletes files.
const liveFile = "penpot-mcp test"

func liveSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	c, err := penpot.NewClient(os.Getenv("PENPOT_BASE_URL"), os.Getenv("PENPOT_ACCESS_TOKEN"), "live", nil)
	if err != nil {
		t.Fatal(err)
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "live", Version: "0"}, nil)
	Register(s, Deps{Client: c, Writer: penpot.NewWriter(c), Fonts: fonts.NewMeasurer(penpot.DefaultBaseURL, t.TempDir(), nil)})
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "live", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func liveFileID(t *testing.T, cs *mcp.ClientSession) string {
	t.Helper()
	var projects projectsOut
	mustCall(t, cs, "list_projects", nil, &projects)
	for _, p := range projects.Projects {
		var files filesOut
		mustCall(t, cs, "list_files", map[string]any{"project_id": p.ID}, &files)
		for _, f := range files.Files {
			if f.Name == liveFile {
				return f.ID
			}
		}
	}
	t.Fatalf("no file named %q; create it in Penpot first", liveFile)
	return ""
}

// TestLiveDraw draws a login screen in one draw call, with measured text.
func TestLiveDraw(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)
	var out drawOut
	mustCall(t, cs, "draw", map[string]any{
		"file_id": fileID,
		"shapes": []map[string]any{
			{"ref": "s", "type": "board", "name": "live login", "x": 1000, "y": 0, "width": 393, "height": 600, "radius": 16},
			{"parent": "s", "type": "text", "text": "Sign in", "x": 1024, "y": 48, "font_family": "Inter", "font_size": 32, "font_weight": 700, "color": "#111111"},
			{"parent": "s", "type": "text", "text": "Welcome back. Enter your email and password to continue.", "x": 1024, "y": 100, "width": 345, "font_family": "Inter", "font_size": 15, "color": "#555555"},
			{"parent": "s", "type": "rect", "name": "email field", "x": 1024, "y": 180, "width": 345, "height": 48, "radius": 8, "fill": "#F2F2F2"},
			{"parent": "s", "type": "text", "text": "Email", "x": 1040, "y": 194, "font_family": "Inter", "font_size": 15, "color": "#888888"},
			{"parent": "s", "type": "rect", "name": "password field", "x": 1024, "y": 244, "width": 345, "height": 48, "radius": 8, "fill": "#F2F2F2"},
			{"parent": "s", "type": "text", "text": "Password", "x": 1040, "y": 258, "font_family": "Inter", "font_size": 15, "color": "#888888"},
			{"ref": "btn", "parent": "s", "type": "rect", "name": "sign in button", "x": 1024, "y": 324, "width": 345, "height": 52, "radius": 26, "fill": "#2E6BE6"},
			{"parent": "s", "type": "text", "text": "Continue", "x": 1158, "y": 340, "font_family": "Inter", "font_size": 16, "font_weight": 600, "color": "#FFFFFF"},
			{"parent": "s", "type": "ellipse", "name": "logo", "x": 1323, "y": 40, "width": 46, "height": 46, "fill": "#2E6BE6"},
		},
	}, &out)
	for _, s := range out.Shapes {
		if s.Type == "text" {
			t.Logf("text %q: %.0f x %.0f measured=%v", s.Name, s.Width, s.Height, *s.Measured)
			if !*s.Measured {
				t.Errorf("text %q was estimated, not measured", s.Name)
			}
		}
	}
}

// TestLive drives the real tools against the account in PENPOT_ACCESS_TOKEN:
// lists teams, finds the live file and draws a 400 x 300 board holding a
// 100 x 100 rect.
func TestLive(t *testing.T) {
	c, err := penpot.NewClient(os.Getenv("PENPOT_BASE_URL"), os.Getenv("PENPOT_ACCESS_TOKEN"), "live", nil)
	if err != nil {
		t.Fatal(err)
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "live", Version: "0"}, nil)
	Register(s, Deps{Client: c, Writer: penpot.NewWriter(c)})
	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ss.Close() }()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "live", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()

	var teams teamsOut
	mustCall(t, cs, "list_teams", nil, &teams)
	t.Logf("teams: %d", len(teams.Teams))

	var projects projectsOut
	mustCall(t, cs, "list_projects", nil, &projects)
	fileID := ""
	for _, p := range projects.Projects {
		var files filesOut
		mustCall(t, cs, "list_files", map[string]any{"project_id": p.ID}, &files)
		for _, f := range files.Files {
			if f.Name == liveFile {
				fileID = f.ID
				t.Logf("found %q in project %q", liveFile, p.Name)
			}
		}
	}
	if fileID == "" {
		t.Fatalf("no file named %q; create it in Penpot first", liveFile)
	}

	var board shapeOut
	mustCall(t, cs, "create_board", map[string]any{
		"file_id": fileID, "name": "live board", "x": 500, "y": 0, "width": 400, "height": 300,
	}, &board)
	var rect shapeOut
	mustCall(t, cs, "create_rect", map[string]any{
		"file_id": fileID, "parent_id": board.ShapeID, "name": "live rect",
		"x": 550, "y": 50, "width": 100, "height": 100, "fill": "#E53935", "radius": 8,
	}, &rect)
	t.Logf("drew board %s and rect %s in %q / %q", board.ShapeID, rect.ShapeID, board.FileName, board.PageName)

	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID}, &tree)
	found := false
	for _, n := range tree.Shapes {
		if n.ID == rect.ShapeID && n.ParentID == board.ShapeID {
			found = true
		}
	}
	if !found {
		t.Errorf("the rect is not inside the board in the page tree")
	}
}

func mustCall(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("%s: %s", name, errText(res))
	}
	data, _ := json.Marshal(res.StructuredContent)
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}

// TestLiveShapes draws every new shape type at x 2600, then edits, moves,
// regroups and deletes some of them.
func TestLiveShapes(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)
	img := filepath.Join(t.TempDir(), "swatch.png")
	writePNG(t, img, 120, 80)

	var out drawOut
	mustCall(t, cs, "draw", map[string]any{
		"file_id": fileID,
		"shapes": []map[string]any{
			{"ref": "s", "type": "board", "name": "live shapes", "x": 2600, "y": 0, "width": 700, "height": 640},
			{"parent": "s", "type": "path", "name": "wave", "d": "M2620 120 Q2670 20 2720 120 T2820 120", "stroke": "#E53935", "stroke_width": 3},
			{"parent": "s", "type": "path", "name": "badge", "d": "M2900 40 a40 40 0 1 0 80 0 a40 40 0 1 0 -80 0 Z", "fill": "#FFB300"},
			{"parent": "s", "type": "image", "image_path": img, "x": 3020, "y": 20, "width": 150, "radius": 12},
			{"parent": "s", "type": "rect", "name": "tilted", "x": 3200, "y": 40, "width": 80, "height": 40, "rotation": 45, "fill": "#1E88E5"},
			{"ref": "g", "parent": "s", "type": "group", "name": "pair"},
			{"parent": "g", "type": "rect", "name": "pair a", "x": 2620, "y": 200, "width": 60, "height": 60, "fill": "#43A047"},
			{"parent": "g", "type": "ellipse", "name": "pair b", "x": 2700, "y": 200, "width": 60, "height": 60, "fill": "#43A047"},
			{"ref": "b", "parent": "s", "type": "bool", "name": "cut", "bool_type": "difference", "fill": "#8E24AA"},
			{"parent": "b", "type": "rect", "name": "cut base", "x": 2900, "y": 200, "width": 100, "height": 100},
			{"parent": "b", "type": "ellipse", "name": "cut hole", "x": 2930, "y": 230, "width": 40, "height": 40},
			{
				"ref": "row", "parent": "s", "type": "board", "name": "row", "x": 2620, "y": 360, "width": 400, "height": 100, "fill": "#EEEEEE",
				"layout": "flex", "gap": 12, "padding": 16, "align_items": "center",
			},
			{"parent": "row", "type": "rect", "name": "first", "x": 0, "y": 0, "width": 60, "height": 60, "fill": "#E53935"},
			{"parent": "row", "type": "rect", "name": "second", "x": 0, "y": 0, "width": 80, "height": 40, "fill": "#43A047"},
			{"parent": "row", "type": "text", "text": "third", "x": 0, "y": 0, "font_family": "Inter", "font_size": 18},
			{
				"ref": "grid", "parent": "s", "type": "board", "name": "grid", "x": 3060, "y": 360, "width": 220, "height": 220, "fill": "#EEEEEE",
				"layout": "grid", "columns": 2, "gap": 10, "padding": 10,
			},
			{"parent": "grid", "type": "rect", "name": "g1", "x": 0, "y": 0, "width": 40, "height": 40, "fill": "#E53935"},
			{"parent": "grid", "type": "rect", "name": "g2", "x": 0, "y": 0, "width": 40, "height": 40, "fill": "#43A047"},
			{"parent": "grid", "type": "rect", "name": "g3", "x": 0, "y": 0, "width": 40, "height": 40, "fill": "#1E88E5"},
			{"parent": "s", "type": "text", "text": "Old label", "x": 2620, "y": 520, "font_family": "Inter", "font_size": 20},
			{"parent": "s", "type": "rect", "name": "doomed", "x": 2800, "y": 520, "width": 40, "height": 40, "fill": "#000000"},
		},
	}, &out)
	ids := map[string]string{}
	for _, s := range out.Shapes {
		ids[s.Name] = s.ShapeID
		t.Logf("%-10s %-8s %.0f,%.0f %.0fx%.0f", s.Name, s.Type, s.X, s.Y, s.Width, s.Height)
	}

	var ch changedOut
	mustCall(t, cs, "update_shape", map[string]any{"file_id": fileID, "shape_id": ids["Old label"], "text": "New, longer label"}, &ch)
	mustCall(t, cs, "update_shape", map[string]any{"file_id": fileID, "shape_id": ids["tilted"], "rotation": 0, "fill": "#00897B"}, &ch)
	mustCall(t, cs, "move_shape", map[string]any{"file_id": fileID, "shape_ids": []string{ids["pair"]}, "dx": 0, "dy": 40}, &ch)
	mustCall(t, cs, "reparent_shape", map[string]any{"file_id": fileID, "shape_ids": []string{ids["doomed"]}, "parent_id": ids["row"]}, &ch)
	mustCall(t, cs, "set_layout", map[string]any{"file_id": fileID, "board_id": ids["row"], "layout": "flex", "gap": 8, "padding": 8, "justify": "space-between", "align_items": "end"}, &setLayoutOut{})
	var g groupOut
	mustCall(t, cs, "create_group", map[string]any{"file_id": fileID, "shape_ids": []string{ids["wave"], ids["badge"]}, "name": "top pair"}, &g)
	mustCall(t, cs, "delete_shapes", map[string]any{"file_id": fileID, "shape_ids": []string{ids["second"]}}, &ch)

	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID}, &tree)
	for _, n := range tree.Shapes {
		if n.ID == ids["second"] {
			t.Errorf("deleted shape is still on the page")
		}
		if n.ID == ids["wave"] && n.ParentID != g.ShapeID {
			t.Errorf("wave is not in the new group")
		}
		if n.ID == ids["doomed"] && n.ParentID != ids["row"] {
			t.Errorf("doomed did not move into row")
		}
	}
}

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 3), B: 200, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
