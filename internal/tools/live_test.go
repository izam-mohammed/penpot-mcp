//go:build live

package tools

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// liveFile is the file the live run draws into. Create it in any project
// first; `make live` never creates or deletes files.
const liveFile = "penpot-mcp test"

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
