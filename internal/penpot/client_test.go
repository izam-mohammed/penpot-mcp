package penpot

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestClientHeaders(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-profile", reply(map[string]any{"id": "p1"}))
	c := testClient(t, srv)

	var p Profile
	if err := c.Call(context.Background(), "get-profile", struct{}{}, &p); err != nil {
		t.Fatal(err)
	}
	h := f.headers[0]
	for k, want := range map[string]string{
		"Authorization": "Token " + testToken,
		"Accept":        "application/json",
		"Content-Type":  "application/json",
		"X-Client":      "penpot-mcp/test",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if p.ID != "p1" {
		t.Errorf("profile id = %q", p.ID)
	}
}

func TestErrorBody(t *testing.T) {
	f, srv := newFake(t)
	f.on("update-file", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"type":"validation","code":"vern-conflict","hint":"restored"}`))
	})
	err := testClient(t, srv).Call(context.Background(), "update-file", struct{}{}, nil)
	if !IsCode(err, "vern-conflict") {
		t.Fatalf("err = %v, want vern-conflict", err)
	}
	if !strings.Contains(err.Error(), "HTTP 409") {
		t.Errorf("err = %q, want the status", err)
	}
}

func TestRetryAfter(t *testing.T) {
	f, srv := newFake(t)
	n := 0
	f.on("get-teams", func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"rate-limit","code":"rate-limit"}`))
			return
		}
		reply([]any{})(w, r)
	})
	c := testClient(t, srv)
	var slept time.Duration
	c.sleep = func(_ context.Context, d time.Duration) error { slept = d; return nil }

	var teams []Team
	if err := c.Call(context.Background(), "get-teams", struct{}{}, &teams); err != nil {
		t.Fatal(err)
	}
	if n != 2 || slept != 3*time.Second {
		t.Errorf("calls = %d, slept = %v; want 2 calls after 3s", n, slept)
	}
}

func TestRetryAfterOnlyOnce(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-teams", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c := testClient(t, srv)
	c.sleep = func(context.Context, time.Duration) error { return nil }
	err := c.Call(context.Background(), "get-teams", struct{}{}, nil)
	if err == nil || f.callCount("get-teams") != 2 {
		t.Fatalf("err = %v, calls = %d; want an error after 2 calls", err, f.callCount("get-teams"))
	}
}

func TestTokenNeverLogged(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"type":"authentication","code":"authentication-required"}`))
	})
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	c, err := NewClient(srv.URL, testToken, "test", logger)
	if err != nil {
		t.Fatal(err)
	}
	callErr := c.Call(context.Background(), "get-file", map[string]string{"id": "x"}, nil)
	if callErr == nil {
		t.Fatal("want an error")
	}
	// A transport failure must not leak it either.
	bad, _ := NewClient("http://127.0.0.1:1", testToken, "test", logger)
	netErr := bad.Call(context.Background(), "get-file", struct{}{}, nil)

	for _, s := range []string{logs.String(), callErr.Error(), netErr.Error()} {
		if strings.Contains(s, testToken) {
			t.Fatalf("token leaked in %q", s)
		}
	}
}

func TestBaseURLMustBeHTTPS(t *testing.T) {
	for url, ok := range map[string]bool{
		"https://design.penpot.app":  true,
		"":                           true,
		"http://localhost:9001":      true,
		"http://127.0.0.1:9001":      true,
		"http://design.penpot.app":   false,
		"ftp://design.penpot.app":    false,
		"not a url":                  false,
		"http://penpot.example.test": false,
	} {
		_, err := NewClient(url, testToken, "test", nil)
		if (err == nil) != ok {
			t.Errorf("NewClient(%q) err = %v, want ok=%v", url, err, ok)
		}
	}
	if _, err := NewClient("", "", "test", nil); err == nil {
		t.Error("want an error for an empty token")
	}
}
