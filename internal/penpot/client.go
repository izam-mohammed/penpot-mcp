// Package penpot talks to the Penpot backend RPC API with a personal access
// token. See docs/external/penpot-backend-api.md.
package penpot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ZeroID is the id of every page's root frame, and of the anonymous profile.
const ZeroID = "00000000-0000-0000-0000-000000000000"

// DefaultBaseURL is the hosted Penpot instance.
const DefaultBaseURL = "https://design.penpot.app"

const maxRetryAfter = 10 * time.Second

// Error is a Penpot RPC error body: {type, code, hint, explain?}.
type Error struct {
	Status  int    `json:"-"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Hint    string `json:"hint"`
	Explain string `json:"explain,omitempty"`
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("penpot: HTTP %d %s/%s", e.Status, e.Type, e.Code)
	if e.Hint != "" {
		msg += ": " + e.Hint
	}
	return msg
}

// IsCode reports whether err is a Penpot error with the given code.
func IsCode(err error, code string) bool {
	var pe *Error
	return errors.As(err, &pe) && pe.Code == code
}

// Client calls Penpot RPC commands. The token is held only here and is never
// formatted into logs or errors.
type Client struct {
	base    *url.URL
	token   string
	client  *http.Client
	log     *slog.Logger
	version string
	sleep   func(context.Context, time.Duration) error
}

// NewClient checks the base URL and returns a client. The URL must be https
// unless the host is a loopback address.
func NewClient(baseURL, token, version string, logger *slog.Logger) (*Client, error) {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("PENPOT_BASE_URL is not a valid URL")
	}
	plainLocal := u.Scheme == "http" && isLoopback(u.Hostname())
	if u.Scheme != "https" && !plainLocal {
		return nil, fmt.Errorf("PENPOT_BASE_URL must use https (plain http only for localhost)")
	}
	if token == "" {
		return nil, fmt.Errorf("PENPOT_ACCESS_TOKEN is not set")
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Client{
		base:    u,
		token:   token,
		client:  &http.Client{Timeout: 60 * time.Second},
		log:     logger,
		version: version,
		sleep:   sleepCtx,
	}, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Call posts params to /api/main/methods/{command} and decodes the reply into
// out (which may be nil). A 429 is retried once after Retry-After. An out of
// type *[]byte gets the raw reply instead, for streamed (SSE) replies.
func (c *Client) Call(ctx context.Context, command string, params, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encode %s params: %w", command, err)
	}
	return c.send(ctx, command, "application/json", body, out)
}

// CallTransit posts a body already encoded as transit+json. The reply is
// still read as JSON.
func (c *Client) CallTransit(ctx context.Context, command string, body []byte, out any) error {
	return c.send(ctx, command, "application/transit+json", body, out)
}

// Upload posts a multipart form: fields first, then the file part named
// fileField.
func (c *Client) Upload(ctx context.Context, command string, fields map[string]string,
	fileField, fileName, contentType string, data []byte, out any,
) error {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if err := w.WriteField(k, fields[k]); err != nil {
			return fmt.Errorf("encode %s form: %w", command, err)
		}
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, fileField, fileName))
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		return fmt.Errorf("encode %s form: %w", command, err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("encode %s form: %w", command, err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("encode %s form: %w", command, err)
	}
	return c.send(ctx, command, w.FormDataContentType(), b.Bytes(), out)
}

func (c *Client) send(ctx context.Context, command, contentType string, body []byte, out any) error {
	for attempt := 0; ; attempt++ {
		wait, err := c.do(ctx, command, contentType, body, out)
		if wait > 0 && attempt == 0 {
			c.log.Info("rate limited, retrying", "command", command, "after", wait)
			if serr := c.sleep(ctx, wait); serr != nil {
				return serr
			}
			continue
		}
		return err
	}
}

func (c *Client) do(ctx context.Context, command, contentType string, body []byte, out any) (time.Duration, error) {
	endpoint := c.base.JoinPath("api", "main", "methods", command).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build %s request: %w", command, err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("Accept", "application/json")
	if _, raw := out.(*[]byte); raw {
		// Streaming commands (export-binfile, import-binfile) answer with
		// server-sent events.
		req.Header.Set("Accept", "text/event-stream, application/json")
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Client", "penpot-mcp/"+c.version)
	req.Header.Set("User-Agent", "penpot-mcp/"+c.version)

	resp, err := c.client.Do(req)
	if err != nil {
		// url.Error carries the URL only, never headers.
		return 0, fmt.Errorf("call %s: %w", command, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return 0, fmt.Errorf("read %s reply: %w", command, err)
	}
	c.log.Debug("rpc", "command", command, "status", resp.StatusCode)

	if resp.StatusCode == http.StatusTooManyRequests {
		wait := retryAfter(resp.Header.Get("Retry-After"))
		return wait, decodeError(resp.StatusCode, data)
	}
	if resp.StatusCode >= 300 {
		return 0, decodeError(resp.StatusCode, data)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent || len(data) == 0 {
		return 0, nil
	}
	if raw, ok := out.(*[]byte); ok {
		*raw = data
		return 0, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return 0, fmt.Errorf("decode %s reply: %w", command, err)
	}
	return 0, nil
}

func retryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs < 1 {
		return time.Second
	}
	d := time.Duration(secs) * time.Second
	if d > maxRetryAfter {
		d = maxRetryAfter
	}
	return d
}

func decodeError(status int, data []byte) error {
	pe := &Error{Status: status}
	if err := json.Unmarshal(data, pe); err != nil || pe.Type == "" {
		pe.Type = "http"
		pe.Code = strings.ToLower(strings.ReplaceAll(http.StatusText(status), " ", "-"))
	}
	return pe
}

// BaseURL is the Penpot instance, without a trailing slash.
func (c *Client) BaseURL() string { return c.base.String() }

// Download fetches an asset URL the server handed back (an export or a
// .penpot file). The token goes only to the Penpot host itself; any other
// host gets a plain GET.
func (c *Client) Download(ctx context.Context, uri string, limit int64) ([]byte, error) {
	u, err := url.Parse(uri)
	if err != nil || (u.Scheme != "https" && (u.Scheme != "http" || !isLoopback(u.Hostname()))) {
		return nil, fmt.Errorf("the server returned an asset URL that is not https")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if u.Host == c.base.Host {
		req.Header.Set("Authorization", "Token "+c.token)
	}
	req.Header.Set("User-Agent", "penpot-mcp/"+c.version)
	return fetch(c.client, req, limit)
}

// fetch runs req and reads at most limit bytes of a 2xx reply.
func fetch(hc *http.Client, req *http.Request, limit int64) ([]byte, error) {
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download: larger than %d bytes", limit)
	}
	return data, nil
}
