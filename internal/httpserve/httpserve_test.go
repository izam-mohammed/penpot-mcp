package httpserve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func server() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "0"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "ping"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "pong"}}}, nil, nil
	})
	return s
}

func TestHttpNeedsTokenOffLoopback(t *testing.T) {
	for _, ok := range []string{"127.0.0.1:8787", "localhost:8787", "[::1]:8787"} {
		if err := Check(ok, ""); err != nil {
			t.Errorf("%s without a token: %v", ok, err)
		}
	}
	for _, bad := range []string{"0.0.0.0:8787", ":8787", "192.168.1.5:8787"} {
		if err := Check(bad, ""); err == nil || !strings.Contains(err.Error(), "PENPOT_MCP_HTTP_TOKEN") {
			t.Errorf("%s without a token should be refused: %v", bad, err)
		}
		if err := Check(bad, "short"); err == nil {
			t.Errorf("%s with a short token should be refused", bad)
		}
		if err := Check(bad, "a-long-enough-token-1234"); err != nil {
			t.Errorf("%s with a token: %v", bad, err)
		}
	}
	if err := Check("8787", ""); err == nil {
		t.Error("a bare port should be refused")
	}

	srv := httptest.NewServer(Handler(server(), "a-long-enough-token-1234"))
	defer srv.Close()
	ctx := context.Background()
	connect := func(token string) error {
		hc := &http.Client{Transport: bearer{token}}
		cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil).Connect(ctx,
			&mcp.StreamableClientTransport{Endpoint: srv.URL + Path, HTTPClient: hc, MaxRetries: -1}, nil)
		if err != nil {
			return err
		}
		defer func() { _ = cs.Close() }()
		_, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "ping"})
		return err
	}
	if err := connect("wrong"); err == nil {
		t.Error("a wrong token should be refused")
	}
	if err := connect("a-long-enough-token-1234"); err != nil {
		t.Errorf("the right token: %v", err)
	}
}

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func TestHttpBodyCap(t *testing.T) {
	srv := httptest.NewServer(Handler(server(), ""))
	defer srv.Close()
	big := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ping","arguments":{"x":"` + strings.Repeat("a", MaxBody) + `"}}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+Path, strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Errorf("a body over %d bytes got HTTP %d", MaxBody, resp.StatusCode)
	}
}
