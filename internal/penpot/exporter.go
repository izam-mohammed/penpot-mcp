package penpot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

// ErrNoExporter is returned when PENPOT_EMAIL and PENPOT_PASSWORD are not set.
var ErrNoExporter = errors.New("exports need PENPOT_EMAIL and PENPOT_PASSWORD in the server's environment: " +
	"Penpot's exporter accepts only a login session, not an access token")

// MaxExport caps one exported file.
const MaxExport = 200 << 20

// Exporter renders shapes through Penpot's exporter service, which accepts
// only the session cookie from login-with-password and transit bodies. The
// password and the cookie are held only here, never logged or formatted
// into errors.
type Exporter struct {
	c        *Client
	email    string
	password string
	http     *http.Client
	log      *slog.Logger
	wait     func(context.Context, time.Duration) error

	mu      sync.Mutex
	cookie  string
	profile string
}

// NewExporter returns nil when email or password is empty.
func NewExporter(c *Client, email, password string) *Exporter {
	if email == "" || password == "" {
		return nil
	}
	return &Exporter{
		c: c, email: email, password: password, log: c.log, wait: c.sleep,
		http: &http.Client{Timeout: 5 * time.Minute},
	}
}

// Export is one shape to render.
type Export struct {
	FileID, PageID, ObjectID string
	Type                     string // png, jpeg, webp, svg or pdf
	Scale                    float64
	Name                     string
}

// login gets a session cookie, once per Exporter or after a 401.
func (e *Exporter) login(ctx context.Context) (cookie, profile string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cookie != "" {
		return e.cookie, e.profile, nil
	}
	body, _ := json.Marshal(map[string]string{"email": e.email, "password": e.password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		e.c.base.JoinPath("api", "main", "methods", "login-with-password").String(), bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "penpot-mcp/"+e.c.version)
	resp, err := e.http.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("login: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		// The reply names the problem; it never holds the password.
		return "", "", fmt.Errorf("login with PENPOT_EMAIL and PENPOT_PASSWORD failed: %w", decodeError(resp.StatusCode, data))
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "auth-token" {
			e.cookie = ck.Value
		}
	}
	if e.cookie == "" {
		return "", "", errors.New("login succeeded but Penpot set no auth-token cookie")
	}
	var p Profile
	_ = json.Unmarshal(data, &p)
	e.profile = p.ID
	e.log.Info("exporter session started")
	return e.cookie, e.profile, nil
}

func (e *Exporter) forget() {
	e.mu.Lock()
	e.cookie, e.profile = "", ""
	e.mu.Unlock()
}

// Render exports one shape and returns the file's bytes. It retries a busy
// exporter (429 queue-full, "ResourceRequest timed out") up to three times
// and logs in again once after a 401.
func (e *Exporter) Render(ctx context.Context, x Export) ([]byte, error) {
	if e == nil {
		return nil, ErrNoExporter
	}
	relogged := false
	for attempt := 0; ; attempt++ {
		data, status, err := e.render(ctx, x)
		switch {
		case err == nil:
			return data, nil
		case status == http.StatusUnauthorized && !relogged:
			e.forget()
			relogged = true
			continue
		case attempt < 3 && (status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable ||
			strings.Contains(err.Error(), "timed out")):
			e.log.Info("exporter busy, retrying", "attempt", attempt+1)
			if werr := e.wait(ctx, time.Duration(2<<attempt)*time.Second); werr != nil {
				return nil, werr
			}
			continue
		}
		return nil, err
	}
}

func (e *Exporter) render(ctx context.Context, x Export) ([]byte, int, error) {
	cookie, profile, err := e.login(ctx)
	if err != nil {
		return nil, 0, err
	}
	scale := x.Scale
	if scale == 0 {
		scale = 1
	}
	body, err := transit.Marshal(transit.Map{
		{K: "cmd", V: transit.Keyword("export-shapes")},
		{K: "profile-id", V: transit.UUID(profile)},
		{K: "wait", V: true},
		{K: "exports", V: []any{transit.Map{
			{K: "file-id", V: transit.UUID(x.FileID)},
			{K: "page-id", V: transit.UUID(x.PageID)},
			{K: "object-id", V: transit.UUID(x.ObjectID)},
			{K: "type", V: transit.Keyword(x.Type)},
			{K: "scale", V: scale},
			{K: "suffix", V: ""},
			{K: "name", V: x.Name},
		}}},
	})
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.c.base.JoinPath("api", "export").String(), bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/transit+json")
	req.Header.Set("Accept", "application/transit+json")
	req.Header.Set("User-Agent", "penpot-mcp/"+e.c.version)
	// A request cookie: Secure, HttpOnly and SameSite apply to Set-Cookie only.
	req.AddCookie(&http.Cookie{Name: "auth-token", Value: cookie}) // #nosec G124
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("export: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("export: HTTP %d: %s", resp.StatusCode, exporterError(data))
	}
	v, err := transit.Unmarshal(data)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("export: reading the reply: %w", err)
	}
	res, _ := v.(map[string]any)
	uri, _ := res["uri"].(string)
	if uri == "" {
		return nil, resp.StatusCode, fmt.Errorf("export: the reply has no download uri")
	}
	file, err := e.c.Download(ctx, uri, MaxExport)
	return file, 0, err
}

// exporterError picks the type, code and hint out of a transit error body.
func exporterError(data []byte) string {
	v, err := transit.Unmarshal(data)
	m, ok := v.(map[string]any)
	if err != nil || !ok {
		s := strings.TrimSpace(string(data))
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	}
	var parts []string
	for _, k := range []string{"type", "code", "hint"} {
		if s, ok := m[k].(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ": ")
}
