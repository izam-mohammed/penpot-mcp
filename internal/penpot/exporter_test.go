package penpot

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPasswordNeverLogged(t *testing.T) {
	const password, cookie = "hunter2-very-secret", "cookie-very-secret"
	var logins, exports atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/main/methods/login-with-password":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), password) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			logins.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "auth-token", Value: cookie})
			_, _ = w.Write([]byte(`{"id":"e1000000-0000-4000-8000-0000000000aa"}`))
		case "/api/export":
			if c, err := r.Cookie("auth-token"); err != nil || c.Value != cookie {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			body, _ := io.ReadAll(r.Body)
			if r.Header.Get("Content-Type") != "application/transit+json" || !strings.Contains(string(body), `"~:cmd","~:export-shapes"`) ||
				!strings.Contains(string(body), `"~:profile-id","~ue1000000-0000-4000-8000-0000000000aa"`) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if exports.Add(1) == 1 {
				// The first render finds the exporter busy.
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`["^ ","~:type","~:restriction","~:code","~:queue-full"]`))
				return
			}
			_, _ = w.Write([]byte(`["^ ","~:id","~u1","~:uri","~r` + srv.URL + `/assets/by-id/1"]`))
		case "/assets/by-id/1":
			_, _ = w.Write([]byte("\x89PNG-bytes"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var logs bytes.Buffer
	c, err := NewClient(srv.URL, "token-very-secret", "test", slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	if err != nil {
		t.Fatal(err)
	}
	e := NewExporter(c, "me@example.com", password)
	e.wait = func(context.Context, time.Duration) error { return nil }
	data, err := e.Render(context.Background(), Export{FileID: "f", PageID: "p", ObjectID: "o", Type: "png", Name: "hero"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "\x89PNG-bytes" || logins.Load() != 1 || exports.Load() != 2 {
		t.Errorf("data %q, %d logins, %d exports", data, logins.Load(), exports.Load())
	}
	for _, secret := range []string{password, cookie, "token-very-secret"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("a secret reached the log:\n%s", logs.String())
		}
	}

	bad := NewExporter(c, "me@example.com", "wrong")
	_, err = bad.Render(context.Background(), Export{FileID: "f", PageID: "p", ObjectID: "o", Type: "png"})
	if err == nil || strings.Contains(err.Error(), "wrong") {
		t.Errorf("a failed login should fail without echoing the password: %v", err)
	}
	if NewExporter(c, "", "") != nil {
		t.Error("no email or password should mean no exporter")
	}
	var none *Exporter
	if _, err := none.Render(context.Background(), Export{}); !errors.Is(err, ErrNoExporter) {
		t.Errorf("nil exporter: %v", err)
	}
}

func TestDownloadKeepsTokenHome(t *testing.T) {
	var sawToken atomic.Bool
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawToken.Store(r.Header.Get("Authorization") != "")
		_, _ = w.Write([]byte("x"))
	}))
	defer other.Close()
	home := httptest.NewServer(http.NotFoundHandler())
	defer home.Close()
	c, err := NewClient(home.URL, "token", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Download(context.Background(), other.URL+"/a", 10); err != nil {
		t.Fatal(err)
	}
	if sawToken.Load() {
		t.Error("the token was sent to a host other than Penpot's")
	}
	if _, err := c.Download(context.Background(), "ftp://x/y", 10); err == nil {
		t.Error("a non-https asset URL should be refused")
	}
}
