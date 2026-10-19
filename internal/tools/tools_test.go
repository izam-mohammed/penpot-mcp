package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

const (
	fileID  = "f0000000-0000-4000-8000-000000000001"
	pageID  = "a0000000-0000-4000-8000-000000000001"
	boardID = "b0000000-0000-4000-8000-000000000001"
	rectID  = "c0000000-0000-4000-8000-000000000001"
)

// backend replays recorded Penpot replies by command name.
type backend struct {
	mu      sync.Mutex
	calls   []string
	updates []map[string]any
}

func (b *backend) serve(w http.ResponseWriter, r *http.Request) {
	command := strings.TrimPrefix(r.URL.Path, "/api/main/methods/")
	data, _ := io.ReadAll(r.Body)
	b.mu.Lock()
	b.calls = append(b.calls, command)
	if command == "update-file" {
		var body map[string]any
		_ = json.Unmarshal(data, &body)
		b.updates = append(b.updates, body)
	}
	b.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch command {
	case "get-file":
		_, _ = w.Write([]byte(`{"id":"` + fileID + `","name":"test file","revn":3,"vern":0,
			"permissions":{"canEdit":true,"canRead":true},
			"data":{"pages":["` + pageID + `"],"pagesIndex":{"` + pageID + `":{"id":"` + pageID + `","name":"Page 1"}}}}`))
	case "get-page":
		_, _ = w.Write([]byte(`{"id":"` + pageID + `","name":"Page 1","objects":{
			"00000000-0000-0000-0000-000000000000":{"id":"00000000-0000-0000-0000-000000000000","name":"Root Frame","type":"frame","shapes":["` + boardID + `"]},
			"` + boardID + `":{"id":"` + boardID + `","name":"Login","type":"frame","x":0,"y":0,"width":400,"height":300,"shapes":["` + rectID + `"]},
			"` + rectID + `":{"id":"` + rectID + `","name":"Button","type":"rect","x":50,"y":50,"width":100,"height":40}}}`))
	case "update-file":
		_, _ = w.Write([]byte(`{"revn":3,"lagged":[]}`))
	case "get-all-projects":
		_, _ = w.Write([]byte(`[{"id":"p1","name":"Drafts","teamId":"t1"}]`))
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"not-found","code":"object-not-found"}`))
	}
}

func (b *backend) count(command string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, c := range b.calls {
		if c == command {
			n++
		}
	}
	return n
}

func session(t *testing.T) (*mcp.ClientSession, *backend) {
	t.Helper()
	b := &backend{}
	srv := httptest.NewServer(http.HandlerFunc(b.serve))
	t.Cleanup(srv.Close)
	c, err := penpot.NewClient(srv.URL, "test-token", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	Register(s, Deps{Client: c, Writer: penpot.NewWriter(c)})

	ctx := context.Background()
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, b
}

func call(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if out != nil && !res.IsError {
		data, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatal(err)
		}
	}
	return res
}

func errText(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestListsEightTools(t *testing.T) {
	cs, _ := session(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	want := "create_board create_rect get_page_tree list_files list_pages list_projects list_teams whoami"
	if got := strings.Join(sortStrings(names), " "); got != want {
		t.Errorf("tools = %s\nwant    %s", got, want)
	}
}

func TestPageTree(t *testing.T) {
	cs, _ := session(t)
	var out treeOut
	res := call(t, cs, "get_page_tree", map[string]any{"file_id": fileID}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.PageName != "Page 1" || len(out.Shapes) != 2 {
		t.Fatalf("tree = %+v", out)
	}
	board, button := out.Shapes[0], out.Shapes[1]
	if board.Name != "Login" || board.Depth != 1 || board.ParentID != penpot.ZeroID {
		t.Errorf("board = %+v", board)
	}
	if button.Name != "Button" || button.Depth != 2 || button.ParentID != boardID {
		t.Errorf("button = %+v", button)
	}
	call(t, cs, "get_page_tree", map[string]any{"file_id": fileID, "depth": 1}, &out)
	if len(out.Shapes) != 1 || out.Shapes[0].More != 1 {
		t.Errorf("depth 1 should cut children and count them: %+v", out.Shapes)
	}
}

func TestWriteEchoesTarget(t *testing.T) {
	cs, b := session(t)
	var out shapeOut
	res := call(t, cs, "create_rect", map[string]any{
		"file_id": fileID, "parent_id": boardID, "name": "Red",
		"x": 60, "y": 60, "width": 100, "height": 100, "fill": "#e53935", "radius": 8,
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.FileName != "test file" || out.PageName != "Page 1" || out.ParentID != boardID || out.ShapeID == "" {
		t.Errorf("result = %+v", out)
	}
	if len(b.updates) != 1 {
		t.Fatalf("update-file calls = %d", len(b.updates))
	}
	u := b.updates[0]
	if u["revn"].(float64) != 3 {
		t.Errorf("sent revn %v, want the file's 3", u["revn"])
	}
	ch := u["changes"].([]any)[0].(map[string]any)
	obj := ch["obj"].(map[string]any)
	if ch["parentId"] != boardID || ch["frameId"] != boardID || obj["type"] != "rect" {
		t.Errorf("change = %v", ch)
	}
	fill := obj["fills"].([]any)[0].(map[string]any)["fillColor"]
	if fill != "#E53935" {
		t.Errorf("fill = %v", fill)
	}
}

func TestCreateBoardAtRoot(t *testing.T) {
	cs, b := session(t)
	var out shapeOut
	res := call(t, cs, "create_board", map[string]any{
		"file_id": fileID, "name": "Screen", "x": 0, "y": 0, "width": 393, "height": 852,
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.ParentID != penpot.ZeroID || b.count("get-page") != 0 {
		t.Errorf("root board: parent %s, get-page calls %d", out.ParentID, b.count("get-page"))
	}
	obj := b.updates[0]["changes"].([]any)[0].(map[string]any)["obj"].(map[string]any)
	if obj["type"] != "frame" || obj["shapes"] == nil {
		t.Errorf("board obj = %v", obj)
	}
}

func TestCreateRectRejectsBadInput(t *testing.T) {
	base := map[string]any{"file_id": fileID, "name": "R", "x": 0, "y": 0, "width": 10, "height": 10}
	cases := map[string]map[string]any{
		"bad file id":    {"file_id": "../etc"},
		"zero width":     {"width": 0},
		"huge height":    {"height": 1e9},
		"far x":          {"x": -2e6},
		"bad fill":       {"fill": "red"},
		"empty name":     {"name": "  "},
		"long name":      {"name": strings.Repeat("n", 201)},
		"negative round": {"radius": -1},
		"bad parent":     {"parent_id": "nope"},
	}
	for name, patch := range cases {
		t.Run(name, func(t *testing.T) {
			cs, b := session(t)
			args := map[string]any{}
			for k, v := range base {
				args[k] = v
			}
			for k, v := range patch {
				args[k] = v
			}
			res := call(t, cs, "create_rect", args, nil)
			if !res.IsError {
				t.Fatal("want a tool error")
			}
			if n := len(b.calls); n != 0 {
				t.Errorf("made %d HTTP calls before refusing", n)
			}
		})
	}
}

func TestParentMustBeBoard(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "create_rect", map[string]any{
		"file_id": fileID, "parent_id": rectID, "name": "R", "x": 0, "y": 0, "width": 10, "height": 10,
	}, nil)
	if !res.IsError || !strings.Contains(errText(res), "only boards") {
		t.Fatalf("want a refusal, got %q", errText(res))
	}
	if b.count("update-file") != 0 {
		t.Error("wrote despite a bad parent")
	}
}

func sortStrings(s []string) []string {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
	return s
}
