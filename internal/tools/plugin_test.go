package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/mcproute"
)

// tab fakes the official Penpot MCP server with a tab that has openFile
// open; answer, when set, replaces every reply.
type tab struct {
	mu       sync.Mutex
	openFile string
	answer   string
	codes    []string
}

func (tb *tab) serve(t *testing.T) *mcproute.Route {
	t.Helper()
	s := mcp.NewServer(&mcp.Implementation{Name: "penpot", Version: "1.0.0"}, nil)
	type codeIn struct {
		Code string `json:"code"`
	}
	mcp.AddTool(s, &mcp.Tool{Name: "execute_code"}, func(_ context.Context, _ *mcp.CallToolRequest, in codeIn) (*mcp.CallToolResult, any, error) {
		tb.mu.Lock()
		defer tb.mu.Unlock()
		tb.codes = append(tb.codes, in.Code)
		text := `{"result":true,"log":""}`
		switch {
		case tb.answer != "":
			text = tb.answer
		case strings.Contains(in.Code, "penpot.currentFile"):
			text = `{"result":{"file_id":"` + tb.openFile + `","file_name":"open file","page_id":"` + pageID + `","page_name":"Page 1"}}`
		case strings.Contains(in.Code, "out[id]"):
			text = `{"result":{"x":{"width":120,"height":19}}}`
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil, nil
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	r, err := mcproute.New(srv.URL+"?userToken=key-very-secret", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r
}

func (tb *tab) ran() []string {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return append([]string(nil), tb.codes...)
}

func withRoute(r *mcproute.Route) func(*Deps) { return func(d *Deps) { d.Route = r } }

func TestPluginToolsNeedRoute(t *testing.T) {
	cs, _ := session(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "execute_code" || tool.Name == "reflow" {
			t.Errorf("%s is listed without PENPOT_MCP_URL", tool.Name)
		}
	}
	var out drawOut
	call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": []map[string]any{{"type": "text", "text": "hi", "x": 0, "y": 0}}}, &out)
	if out.Tab != nil {
		t.Errorf("without a route the reply has no tab step: %+v", out.Tab)
	}
}

func TestExecuteCodeGuarded(t *testing.T) {
	tb := &tab{openFile: "a1000000-0000-4000-8000-000000000009"}
	cs, _ := sessionWith(t, withRoute(tb.serve(t)))
	res := call(t, cs, "execute_code", map[string]any{"file_id": fileID, "code": "penpot.currentPage.remove()"}, nil)
	if !res.IsError || !strings.Contains(errText(res), `not "test file"`) {
		t.Errorf("wrong file open: got %q", errText(res))
	}
	for _, c := range tb.ran() {
		if strings.Contains(c, "remove()") {
			t.Fatal("the code ran against the wrong file")
		}
	}

	tb.openFile = fileID
	var out tabOut
	if res := call(t, cs, "execute_code", map[string]any{"file_id": fileID, "code": "return 1"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.FileID != fileID || out.Result != true {
		t.Errorf("execute_code = %+v", out)
	}
}

func TestPluginIDsAreData(t *testing.T) {
	tb := &tab{openFile: fileID}
	cs, _ := sessionWith(t, withRoute(tb.serve(t)))
	res := call(t, cs, "detach", map[string]any{"file_id": fileID, "shape_ids": []string{`x"); penpot.closePlugin(); ("`}}, nil)
	if !res.IsError {
		t.Error("a non-UUID id should be refused")
	}
	if res := call(t, cs, "detach", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	codes := tb.ran()
	last := codes[len(codes)-1]
	if !strings.Contains(last, `const ids = ["`+rectID+`"]`) || !strings.Contains(last, "typeof s.detach") {
		t.Errorf("detach code: %s", last)
	}
}

func TestHybridFallsBackWithoutTab(t *testing.T) {
	tb := &tab{answer: "Tool execution failed: no heartbeat for 30s"}
	cs, b := sessionWith(t, withRoute(tb.serve(t)))
	var out drawOut
	res := call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": []map[string]any{{"type": "text", "text": "hi", "x": 0, "y": 0}}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if len(b.updates) != 1 || out.Tab == nil || out.Tab.Reflowed || !strings.Contains(out.Tab.Note, "no Penpot tab") {
		t.Errorf("the API write should stand with a note: %+v", out.Tab)
	}

	tb.answer = ""
	tb.openFile = fileID
	res = call(t, cs, "draw", map[string]any{"file_id": fileID, "shapes": []map[string]any{{"type": "text", "text": "hi", "x": 0, "y": 0}}}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.Tab == nil || !out.Tab.Reflowed {
		t.Errorf("with the file open the tab should reflow: %+v", out.Tab)
	}
}
