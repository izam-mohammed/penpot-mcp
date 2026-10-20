package penpot

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testToken = "secret-token-value-123"

// fake is a stand-in Penpot backend that routes by command name.
type fake struct {
	t        *testing.T
	mu       sync.Mutex
	calls    []string
	bodies   []map[string]any
	headers  []http.Header
	handlers map[string]http.HandlerFunc
}

func newFake(t *testing.T) (*fake, *httptest.Server) {
	f := &fake{t: t, handlers: map[string]http.HandlerFunc{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fake) on(command string, h http.HandlerFunc) { f.handlers[command] = h }

func (f *fake) serve(w http.ResponseWriter, r *http.Request) {
	command := strings.TrimPrefix(r.URL.Path, "/api/main/methods/")
	data, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(data, &body)
	f.mu.Lock()
	f.calls = append(f.calls, command)
	f.bodies = append(f.bodies, body)
	f.headers = append(f.headers, r.Header.Clone())
	h, ok := f.handlers[command]
	f.mu.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"not-found","code":"object-not-found"}`))
		return
	}
	h(w, r)
}

func (f *fake) callCount(command string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if c == command {
			n++
		}
	}
	return n
}

func reply(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
}

func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(srv.URL, testToken, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// fileReply is a get-file reply with one page.
func fileReply(revn, vern int64, canEdit bool) map[string]any {
	return map[string]any{
		"id": "f0000000-0000-4000-8000-000000000001", "name": "test file",
		"revn": revn, "vern": vern,
		"permissions": map[string]any{"canEdit": canEdit, "canRead": true},
		"data": map[string]any{
			"pages":      []string{"a0000000-0000-4000-8000-000000000001"},
			"pagesIndex": map[string]any{"a0000000-0000-4000-8000-000000000001": map[string]any{"id": "a0000000-0000-4000-8000-000000000001", "name": "Page 1"}},
		},
	}
}
