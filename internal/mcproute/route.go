// Package mcproute is the MCP route: this server as a client of the official
// Penpot MCP server, which runs Plugin API code in the user's Penpot tab.
// See docs/bridge.md § 2 and docs/external/penpot-official-mcp.md.
package mcproute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	// ErrNoTab means the official server has no Penpot tab with the MCP
	// plugin connected.
	ErrNoTab = errors.New("no Penpot tab is connected: open the file in Penpot, then Plugins > MCP > Connect " +
		"(a tab in the background can fall asleep; bring it to the front)")

	// ErrMaybeApplied means a call timed out after it was sent: the tab may
	// have run it. Never retried.
	ErrMaybeApplied = errors.New("the Penpot tab did not answer in time; the code may have run anyway, " +
		"so read the file again before trying once more")
)

// CallTimeout bounds one call; the official server stops at 120 s.
const CallTimeout = 125 * time.Second

// Route talks to the official MCP server. The URL carries the user's MCP
// key, so it is held only here and scrubbed from every error.
type Route struct {
	endpoint string
	secrets  []string
	version  string
	log      *slog.Logger
	connect  func(ctx context.Context) (*mcp.ClientSession, error)
	timeout  time.Duration

	mu      sync.Mutex
	session *mcp.ClientSession
	checked bool
}

// New returns nil when rawURL is empty.
func New(rawURL, version string, logger *slog.Logger) (*Route, error) {
	if rawURL == "" {
		return nil, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, errors.New("PENPOT_MCP_URL is not a valid URL")
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	r := &Route{endpoint: rawURL, version: version, log: logger, secrets: []string{rawURL}, timeout: CallTimeout}
	for _, vs := range u.Query() {
		for _, v := range vs {
			if len(v) >= 6 {
				r.secrets = append(r.secrets, v, url.QueryEscape(v))
			}
		}
	}
	r.connect = r.dial
	return r, nil
}

// uaTransport adds the User-Agent the official server requires (it answers
// 403 without one).
type uaTransport struct {
	ua   string
	base http.RoundTripper
}

func (t uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", t.ua)
	return t.base.RoundTrip(req)
}

func (r *Route) dial(ctx context.Context) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "penpot-mcp", Version: r.version}, nil)
	return client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             r.endpoint,
		HTTPClient:           &http.Client{Transport: uaTransport{"penpot-mcp/" + r.version, http.DefaultTransport}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
}

// scrub removes the URL and its key from an error's text.
func (r *Route) scrub(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, s := range r.secrets {
		msg = strings.ReplaceAll(msg, s, "<PENPOT_MCP_URL>")
	}
	return errors.New(msg)
}

func (r *Route) sessionFor(ctx context.Context) (*mcp.ClientSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session != nil {
		return r.session, nil
	}
	cs, err := r.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to the Penpot MCP server: %w", r.scrub(err))
	}
	if !r.checked {
		tools, err := cs.ListTools(ctx, nil)
		if err != nil {
			_ = cs.Close()
			return nil, fmt.Errorf("listing the Penpot MCP server's tools: %w", r.scrub(err))
		}
		found := false
		for _, t := range tools.Tools {
			found = found || t.Name == "execute_code"
		}
		if !found {
			_ = cs.Close()
			return nil, errors.New("the Penpot MCP server no longer offers execute_code; this server needs an update")
		}
		r.checked = true
	}
	r.session = cs
	return cs, nil
}

func (r *Route) drop(cs *mcp.ClientSession) {
	r.mu.Lock()
	if r.session == cs {
		r.session = nil
	}
	r.mu.Unlock()
	_ = cs.Close()
}

// Call runs one tool of the official server and returns its text. A session
// the server forgot (it answers 400 or 404 before running anything) is
// re-opened once; a timeout is never retried.
func (r *Route) Call(ctx context.Context, tool string, args map[string]any) (string, error) {
	for attempt := 0; ; attempt++ {
		cs, err := r.sessionFor(ctx)
		if err != nil {
			return "", err
		}
		cctx, cancel := context.WithTimeout(ctx, r.timeout)
		res, err := cs.CallTool(cctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "timed out") ||
				strings.Contains(err.Error(), "timeout") {
				return "", ErrMaybeApplied
			}
			r.drop(cs)
			if attempt == 0 && sessionGone(err) {
				r.log.Info("penpot mcp session expired, reconnecting")
				continue
			}
			return "", r.scrub(err)
		}
		text := textOf(res)
		if res.IsError || strings.HasPrefix(text, "Tool execution failed") {
			return "", mapError(text)
		}
		return text, nil
	}
}

func sessionGone(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "404") || strings.Contains(s, "400") || strings.Contains(s, "session")
}

func textOf(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, t.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// mapError turns the official server's failure text into an error the agent
// can act on.
func mapError(text string) error {
	low := strings.ToLower(text)
	switch {
	case strings.Contains(low, "heartbeat"), strings.Contains(low, "no penpot instance connected"):
		return ErrNoTab
	case strings.Contains(low, "timed out"), strings.Contains(low, "timeout"):
		return ErrMaybeApplied
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, "Tool execution failed:"))
	if len(text) > 2000 {
		text = text[:2000] + "..."
	}
	return fmt.Errorf("the Penpot tab reported: %s", text)
}

// Result is what execute_code returns: the code's return value and its
// console log.
type Result struct {
	Value json.RawMessage `json:"result"`
	Log   string          `json:"log,omitempty"`
}

// Exec runs a Plugin API function body (await allowed, return gives the
// result) in the connected tab.
func (r *Route) Exec(ctx context.Context, code string) (Result, error) {
	text, err := r.Call(ctx, "execute_code", map[string]any{"code": code})
	if err != nil {
		return Result{}, err
	}
	var res struct {
		Result json.RawMessage `json:"result"`
		Log    any             `json:"log"`
	}
	if err := json.Unmarshal([]byte(text), &res); err != nil {
		// Older servers answer with the bare value.
		return Result{Value: json.RawMessage(text)}, nil
	}
	out := Result{Value: res.Result}
	switch l := res.Log.(type) {
	case string:
		out.Log = l
	case nil:
	default:
		b, _ := json.Marshal(l)
		out.Log = string(b)
	}
	return out, nil
}

// Open is the file and page open in the connected tab.
type Open struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	PageID   string `json:"page_id"`
	PageName string `json:"page_name"`
}

// Current reads which file and page the tab has open.
func (r *Route) Current(ctx context.Context) (Open, error) {
	res, err := r.Exec(ctx, `const f = penpot.currentFile, p = penpot.currentPage;
return f ? {file_id: f.id, file_name: f.name, page_id: p ? p.id : "", page_name: p ? p.name : ""} : null;`)
	if err != nil {
		return Open{}, err
	}
	var o *Open
	if err := json.Unmarshal(res.Value, &o); err != nil || o == nil {
		return Open{}, ErrNoTab
	}
	return *o, nil
}

// Guard fails unless the tab has fileID open, so code never runs against
// another file (docs/bridge.md § 2, rule 3).
func (r *Route) Guard(ctx context.Context, fileID, fileName string) (Open, error) {
	o, err := r.Current(ctx)
	if err != nil {
		return Open{}, err
	}
	if o.FileID != fileID {
		return o, fmt.Errorf("the Penpot tab has %q open, not %q: open %q in Penpot and connect the MCP plugin there",
			o.FileName, fileName, fileName)
	}
	return o, nil
}

// Close ends the session.
func (r *Route) Close() {
	r.mu.Lock()
	cs := r.session
	r.session = nil
	r.mu.Unlock()
	if cs != nil {
		_ = cs.Close()
	}
}
