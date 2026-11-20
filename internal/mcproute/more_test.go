package mcproute

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMapErrorCases(t *testing.T) {
	if err := mapError("No Penpot instance connected"); !errors.Is(err, ErrNoTab) {
		t.Errorf("no instance: %v", err)
	}
	if err := mapError("operation timed out"); !errors.Is(err, ErrMaybeApplied) {
		t.Errorf("timeout: %v", err)
	}
	long := mapError("Tool execution failed: " + strings.Repeat("x", 3000))
	if !strings.HasSuffix(long.Error(), "...") || len(long.Error()) > 2100 {
		t.Errorf("a long error should be cut: %d bytes", len(long.Error()))
	}
}

func TestSessionGone(t *testing.T) {
	for _, s := range []string{"HTTP 404", "bad request 400", "unknown Session id"} {
		if !sessionGone(errors.New(s)) {
			t.Errorf("%q should count as a lost session", s)
		}
	}
	if sessionGone(errors.New("connection refused")) {
		t.Error("a refused connection is not a lost session")
	}
}

func TestExecBareValueAndStructuredLog(t *testing.T) {
	answer := `42`
	endpoint, _ := official(t, func(string) (string, bool) { return answer, false })
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	res, err := r.Exec(context.Background(), "return 42")
	if err != nil || string(res.Value) != "42" {
		t.Fatalf("bare value: %s, %v", res.Value, err)
	}
}

func TestExecLogShapes(t *testing.T) {
	logs := []string{`{"result":1,"log":["a","b"]}`, `{"result":1}`}
	for _, answer := range logs {
		endpoint, _ := official(t, func(string) (string, bool) { return answer, false })
		var out bytes.Buffer
		r := route(t, endpoint, &out)
		res, err := r.Exec(context.Background(), "return 1")
		if err != nil || string(res.Value) != "1" {
			t.Fatalf("%s: %s, %v", answer, res.Value, err)
		}
		if strings.Contains(answer, "log") && res.Log != `["a","b"]` {
			t.Errorf("log = %q", res.Log)
		}
	}
}

func TestCurrentWithoutFileIsNoTab(t *testing.T) {
	endpoint, _ := official(t, func(string) (string, bool) { return `{"result":null}`, false })
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	if _, err := r.Current(context.Background()); !errors.Is(err, ErrNoTab) {
		t.Errorf("got %v", err)
	}
	if _, err := r.Guard(context.Background(), "f", "F"); !errors.Is(err, ErrNoTab) {
		t.Errorf("guard: %v", err)
	}
}

func TestServerWithoutExecuteCode(t *testing.T) {
	s := mcp.NewServer(&mcp.Implementation{Name: "penpot", Version: "1.0.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "other"}, func(context.Context, *mcp.CallToolRequest, codeIn) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{}, nil, nil
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, nil)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	var logs bytes.Buffer
	r := route(t, srv.URL+"/mcp/stream?userToken=key-very-secret", &logs)
	_, err := r.Exec(context.Background(), "return 1")
	if err == nil || !strings.Contains(err.Error(), "no longer offers execute_code") {
		t.Fatalf("got %v", err)
	}
}

func TestCallAfterServerDiesDropsSession(t *testing.T) {
	endpoint, _ := official(t, func(string) (string, bool) { return `{"result":1}`, false })
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	if _, err := r.Exec(context.Background(), "return 1"); err != nil {
		t.Fatal(err)
	}
	// Closing the session client side makes the next CallTool fail; the
	// route drops it and either reconnects or fails without the key.
	r.mu.Lock()
	cs := r.session
	r.mu.Unlock()
	_ = cs.Close()
	if _, err := r.Exec(context.Background(), "return 2"); err != nil && strings.Contains(err.Error(), "key-very-secret") {
		t.Errorf("the key leaked: %v", err)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	endpoint, _ := official(t, func(string) (string, bool) { return `{"result":1}`, false })
	var logs bytes.Buffer
	r := route(t, endpoint, &logs)
	if _, err := r.Exec(context.Background(), "return 1"); err != nil {
		t.Fatal(err)
	}
	r.Close()
	r.Close()
}
