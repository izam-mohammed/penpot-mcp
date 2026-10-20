package mcproute

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type codeIn struct {
	Code string `json:"code"`
}

// official fakes the official Penpot MCP server: execute_code answers with
// reply(code), and requests without a User-Agent get 403 as on the real one.
func official(t *testing.T, reply func(code string) (string, bool)) (string, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	s := mcp.NewServer(&mcp.Implementation{Name: "penpot", Version: "1.0.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "execute_code"}, func(ctx context.Context, _ *mcp.CallToolRequest, in codeIn) (*mcp.CallToolResult, any, error) {
		calls.Add(1)
		text, isErr := reply(in.Code)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, IsError: isErr}, nil, nil
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "penpot-mcp/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/mcp/stream?userToken=key-very-secret", &calls
}

func route(t *testing.T, endpoint string, logs *bytes.Buffer) *Route {
	t.Helper()
	r, err := New(endpoint, "test", slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r
}

func TestFileGuardRefuses(t *testing.T) {
	var ran atomic.Bool
	endpoint, calls := official(t, func(code string) (string, bool) {
		if strings.Contains(code, "penpot.currentFile") {
			return `{"result":{"file_id":"other","file_name":"Other file","page_id":"p","page_name":"Page 1"},"log":""}`, false
		}
		ran.Store(true)
		return `{"result":null}`, false
	})
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	_, err := r.Guard(context.Background(), "mine", "My file")
	if err == nil || !strings.Contains(err.Error(), `has "Other file" open, not "My file"`) {
		t.Fatalf("guard: %v", err)
	}
	if ran.Load() || calls.Load() != 1 {
		t.Errorf("only the guard's read should have run: %d calls", calls.Load())
	}
	o, err := r.Guard(context.Background(), "other", "Other file")
	if err != nil || o.PageName != "Page 1" {
		t.Errorf("matching file: %+v, %v", o, err)
	}
}

func TestMcpUrlNeverLogged(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	endpoint := dead.URL + "/mcp/stream?userToken=key-very-secret"
	dead.Close()
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	_, err := r.Exec(context.Background(), "return 1")
	if err == nil {
		t.Fatal("a dead server should fail")
	}
	for _, out := range []string{err.Error(), logs.String()} {
		if strings.Contains(out, "key-very-secret") || strings.Contains(out, dead.URL) {
			t.Errorf("the MCP URL leaked: %s", out)
		}
	}
	if _, err := New("not a url", "", nil); err == nil || strings.Contains(err.Error(), "not a url") {
		t.Errorf("a bad URL should fail without echoing it: %v", err)
	}
}

func TestNoRetryAfterTimeout(t *testing.T) {
	endpoint, calls := official(t, func(string) (string, bool) {
		time.Sleep(300 * time.Millisecond)
		return `{"result":1}`, false
	})
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	r.timeout = 100 * time.Millisecond
	_, err := r.Exec(context.Background(), "penpot.createRectangle()")
	if !errors.Is(err, ErrMaybeApplied) {
		t.Fatalf("want ErrMaybeApplied, got %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	if calls.Load() != 1 {
		t.Errorf("a timed-out call was sent %d times", calls.Load())
	}
}

func TestErrorsMapped(t *testing.T) {
	answer := "Tool execution failed: Error: no heartbeat for 30s"
	endpoint, _ := official(t, func(code string) (string, bool) { return answer, false })
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	if _, err := r.Exec(context.Background(), "return 1"); !errors.Is(err, ErrNoTab) {
		t.Errorf("heartbeat: %v", err)
	}
	answer = "Tool execution failed: TypeError: x is undefined"
	if _, err := r.Exec(context.Background(), "return x.y"); err == nil || !strings.Contains(err.Error(), "TypeError: x is undefined") {
		t.Errorf("code error: %v", err)
	}
	answer = `{"result":{"a":1},"log":"hello"}`
	res, err := r.Exec(context.Background(), "return {a: 1}")
	if err != nil || string(res.Value) != `{"a":1}` || res.Log != "hello" {
		t.Errorf("result = %s %q %v", res.Value, res.Log, err)
	}
}

func TestNoURLNoRoute(t *testing.T) {
	r, err := New("", "", nil)
	if r != nil || err != nil {
		t.Errorf("no URL should mean no route: %v %v", r, err)
	}
}
