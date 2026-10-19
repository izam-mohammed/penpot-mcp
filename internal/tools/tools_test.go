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

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
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
	raw     []string // update-file bodies as sent
	ctypes  []string // their content types
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
		b.raw = append(b.raw, string(data))
		b.ctypes = append(b.ctypes, r.Header.Get("Content-Type"))
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
			"` + boardID + `":{"id":"` + boardID + `","name":"Login","type":"frame","parentId":"00000000-0000-0000-0000-000000000000","frameId":"00000000-0000-0000-0000-000000000000","x":0,"y":0,"width":400,"height":300,"shapes":["` + rectID + `"]},
			"` + rectID + `":{"id":"` + rectID + `","name":"Button","type":"rect","parentId":"` + boardID + `","frameId":"` + boardID + `","x":50,"y":50,"width":100,"height":40}}}`))
	case "update-file":
		_, _ = w.Write([]byte(`{"revn":3,"lagged":[]}`))
	case "upload-file-media-object":
		_, _ = w.Write([]byte(`{"id":"d0000000-0000-4000-8000-000000000001","name":"swatch","width":120,"height":80,"mtype":"image/png"}`))
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
	// No font server: text falls back to the estimate, which is enough here.
	Register(s, Deps{Client: c, Writer: penpot.NewWriter(c), Fonts: fonts.NewMeasurer("http://127.0.0.1:1", "", nil)})

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

func TestListsTools(t *testing.T) {
	cs, _ := session(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	want := "apply_changes create_board create_bool create_ellipse create_group create_image create_path create_rect create_text " +
		"delete_shapes draw get_page_tree list_files list_pages list_projects list_teams move_shape reparent_shape " +
		"set_layout update_shape whoami"
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
	if out.FileName != "test file" || out.PageName != "Page 1" || out.ParentID != boardID || out.ShapeID == "" ||
		out.Width != 100 || out.Height != 100 {
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

func TestDrawOneBatch(t *testing.T) {
	cs, b := session(t)
	var out drawOut
	res := call(t, cs, "draw", map[string]any{
		"file_id": fileID,
		"shapes": []map[string]any{
			{"ref": "screen", "type": "board", "name": "Login", "x": 1000, "y": 0, "width": 393, "height": 852},
			{
				"ref": "title", "parent": "screen", "type": "text", "text": "Sign in", "x": 1024, "y": 80,
				"font_family": "Inter", "font_size": 28, "font_weight": 700,
			},
			{"parent": "screen", "type": "rect", "name": "Email", "x": 1024, "y": 160, "width": 345, "height": 48, "radius": 8},
			{"parent": "screen", "type": "ellipse", "name": "Avatar", "x": 1160, "y": 300, "width": 72, "height": 72},
			{"parent": boardID, "type": "rect", "name": "Into existing", "x": 10, "y": 10, "width": 10, "height": 10},
		},
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if n := b.count("update-file"); n != 1 {
		t.Fatalf("update-file calls = %d, want one batch", n)
	}
	changes := b.updates[0]["changes"].([]any)
	if len(changes) != 5 || len(out.Shapes) != 5 {
		t.Fatalf("changes %d, shapes %d; want 5 and 5", len(changes), len(out.Shapes))
	}
	screen := out.Shapes[0].ShapeID
	for i, want := range []string{penpot.ZeroID, screen, screen, screen, boardID} {
		if out.Shapes[i].ParentID != want {
			t.Errorf("shapes[%d] parent = %s, want %s", i, out.Shapes[i].ParentID, want)
		}
		ch := changes[i].(map[string]any)
		if ch["parentId"] != want || ch["frameId"] != want {
			t.Errorf("change %d parent/frame = %v/%v, want %s", i, ch["parentId"], ch["frameId"], want)
		}
	}
	text := changes[1].(map[string]any)["obj"].(map[string]any)
	if text["type"] != "text" || text["growType"] != "auto-width" || text["name"] != "Sign in" {
		t.Errorf("text obj = %v", text)
	}
	para := text["content"].(map[string]any)["children"].([]any)[0].(map[string]any)["children"].([]any)[0].(map[string]any)
	if para["fontId"] != "gfont-inter" || para["fontWeight"] != "700" || para["fontSize"] != "28" {
		t.Errorf("paragraph style = %v", para)
	}
	if out.Shapes[1].Width <= 0 || out.Shapes[1].Height != 34 { // ceil(28 * 1.2)
		t.Errorf("text box = %v x %v", out.Shapes[1].Width, out.Shapes[1].Height)
	}
	if ellipse := changes[3].(map[string]any)["obj"].(map[string]any); ellipse["type"] != "circle" {
		t.Errorf("ellipse type = %v, want circle", ellipse["type"])
	}
}

func TestDrawCaps(t *testing.T) {
	big := make([]map[string]any, maxDrawSize+1)
	for i := range big {
		big[i] = map[string]any{"type": "rect", "name": "r", "x": 0, "y": 0, "width": 1, "height": 1}
	}
	cases := map[string][]map[string]any{
		"empty":            {},
		"too many":         big,
		"unknown parent":   {{"parent": "nope", "type": "rect", "name": "r", "x": 0, "y": 0, "width": 1, "height": 1}},
		"parent not board": {{"ref": "a", "type": "rect", "name": "r", "x": 0, "y": 0, "width": 1, "height": 1}, {"parent": "a", "type": "rect", "name": "r", "x": 0, "y": 0, "width": 1, "height": 1}},
		"parent later":     {{"parent": "b", "type": "rect", "name": "r", "x": 0, "y": 0, "width": 1, "height": 1}, {"ref": "b", "type": "board", "name": "b", "x": 0, "y": 0, "width": 1, "height": 1}},
		"duplicate ref":    {{"ref": "a", "type": "board", "name": "a", "x": 0, "y": 0, "width": 1, "height": 1}, {"ref": "a", "type": "board", "name": "a", "x": 0, "y": 0, "width": 1, "height": 1}},
		"bad type":         {{"type": "star", "name": "s", "x": 0, "y": 0, "width": 1, "height": 1}},
		"empty text":       {{"type": "text", "text": "", "x": 0, "y": 0}},
		"tiny font":        {{"type": "text", "text": "a", "x": 0, "y": 0, "font_size": 1}},
		"bad weight":       {{"type": "text", "text": "a", "x": 0, "y": 0, "font_weight": 450}},
		"bad align":        {{"type": "text", "text": "a", "x": 0, "y": 0, "align": "middle"}},
		"bad font family":  {{"type": "text", "text": "a", "x": 0, "y": 0, "font_family": "../../etc"}},
	}
	for name, shapes := range cases {
		t.Run(name, func(t *testing.T) {
			cs, b := session(t)
			res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": shapes}, nil)
			if !res.IsError {
				t.Fatal("want a tool error")
			}
			if len(b.calls) != 0 {
				t.Errorf("made %d HTTP calls before refusing", len(b.calls))
			}
		})
	}
}

func TestCreateTextWraps(t *testing.T) {
	cs, b := session(t)
	var out shapeOut
	res := call(t, cs, "create_text", map[string]any{
		"file_id": fileID, "text": "one two three four five six seven eight", "x": 0, "y": 0,
		"width": 80, "font_size": 10,
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	obj := b.updates[0]["changes"].([]any)[0].(map[string]any)["obj"].(map[string]any)
	if obj["growType"] != "auto-height" || out.Width != 80 || out.Height <= 12 {
		t.Errorf("wrapped text: grow %v, box %v x %v; want auto-height, width 80, several lines", obj["growType"], out.Width, out.Height)
	}
}
