package penpot

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/izam-mohammed/penpot-mcp/internal/transit"
)

func TestSleepCtx(t *testing.T) {
	if err := sleepCtx(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestCallRawReply(t *testing.T) {
	f, srv := newFake(t)
	f.on("export-binfile", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		_, _ = w.Write([]byte("event: end\ndata: x\n\n"))
	})
	var raw []byte
	if err := testClient(t, srv).Call(context.Background(), "export-binfile", map[string]string{"a": "b"}, &raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("event: end")) {
		t.Errorf("raw = %q", raw)
	}
}

func TestCallNoContentAndBadJSON(t *testing.T) {
	f, srv := newFake(t)
	f.on("empty", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	f.on("bad", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not json")) })
	c := testClient(t, srv)
	var out map[string]any
	if err := c.Call(context.Background(), "empty", struct{}{}, &out); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(context.Background(), "bad", struct{}{}, &out); err == nil || !strings.Contains(err.Error(), "decode bad reply") {
		t.Fatalf("got %v", err)
	}
	if err := c.Call(context.Background(), "x", func() {}, nil); err == nil {
		t.Fatal("want an encode error for an unmarshalable param")
	}
}

func TestCallTransitSendsContentType(t *testing.T) {
	f, srv := newFake(t)
	f.on("update-file", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/transit+json" {
			t.Errorf("Content-Type = %q", got)
		}
		reply(map[string]any{"revn": 1})(w, r)
	})
	var res UpdateResult
	if err := testClient(t, srv).CallTransit(context.Background(), "update-file", []byte(`["^ "]`), &res); err != nil {
		t.Fatal(err)
	}
	if res.Revn != 1 {
		t.Errorf("revn = %d", res.Revn)
	}
}

func TestUploadSendsMultipart(t *testing.T) {
	f, srv := newFake(t)
	var gotType string
	f.on("upload-file-media-object", func(w http.ResponseWriter, r *http.Request) {
		gotType = r.Header.Get("Content-Type")
		reply(map[string]any{"id": "x"})(w, r)
	})
	err := testClient(t, srv).Upload(context.Background(), "upload-file-media-object",
		map[string]string{"file-id": testFile, "name": "a"}, "content", "a.png", "image/png", []byte("png"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotType, "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q", gotType)
	}
}

func TestBaseURLAndAPIDoc(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/main/doc/openapi.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		hits++
		_, _ = w.Write([]byte(`{"openapi":"3.0.0"}`))
	}))
	t.Cleanup(srv.Close)
	c := testClient(t, srv)
	if c.BaseURL() != srv.URL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL(), srv.URL)
	}
	for range 2 {
		doc, err := c.APIDoc(context.Background())
		if err != nil || !strings.Contains(string(doc), "openapi") {
			t.Fatalf("doc %q, err %v", doc, err)
		}
	}
	if hits != 1 {
		t.Errorf("document fetched %d times, want once", hits)
	}
}

func TestAPIDocError(t *testing.T) {
	_, srv := newFake(t)
	if _, err := testClient(t, srv).APIDoc(context.Background()); err == nil {
		t.Fatal("want an error when the document is missing")
	}
}

func TestDownloadRejectsPlainHTTPRemote(t *testing.T) {
	_, srv := newFake(t)
	c := testClient(t, srv)
	for _, uri := range []string{"http://example.com/a", "ftp://example.com/a", "://bad"} {
		if _, err := c.Download(context.Background(), uri, 10); err == nil {
			t.Errorf("Download(%q) should fail", uri)
		}
	}
}

func TestDownloadLimitAndStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(bytes.Repeat([]byte("a"), 100))
	}))
	t.Cleanup(srv.Close)
	c := testClient(t, srv)
	if _, err := c.Download(context.Background(), srv.URL+"/big", 10); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("got %v", err)
	}
	if _, err := c.Download(context.Background(), srv.URL+"/missing", 10); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("got %v", err)
	}
}

func TestWriterResolveAndPages(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(3, 0, true)))
	w := NewWriter(testClient(t, srv))
	ctx := context.Background()

	tg, err := w.Resolve(ctx, testFile, "")
	if err != nil || tg.PageID != pageA || tg.PageName != "Page 1" || tg.FileName != "test file" {
		t.Fatalf("target %+v, err %v", tg, err)
	}
	if _, err := w.Resolve(ctx, testFile, "f0000000-0000-4000-8000-0000000000ff"); err == nil {
		t.Error("want an error for a page not in the file")
	}
	name, pages, err := w.Pages(ctx, testFile)
	if err != nil || name != "test file" || len(pages) != 1 {
		t.Fatalf("pages %v %v %v", name, pages, err)
	}
	if got := f.callCount("get-file"); got != 2 {
		t.Errorf("get-file called %d times, want 2 (cached resolve, refreshed pages)", got)
	}
	if got, err := w.FileName(ctx, testFile); err != nil || got != "test file" {
		t.Errorf("FileName = %q, %v", got, err)
	}
	w.Forget(testFile)
	if _, err := w.Resolve(ctx, testFile, ""); err != nil {
		t.Fatal(err)
	}
	if got := f.callCount("get-file"); got != 3 {
		t.Errorf("get-file called %d times after Forget, want 3", got)
	}
}

func TestWriterNoPagesAndMissingFile(t *testing.T) {
	f, srv := newFake(t)
	empty := fileReply(0, 0, true)
	empty["data"] = map[string]any{"pages": []string{}, "pagesIndex": map[string]any{}}
	f.on("get-file", reply(empty))
	w := NewWriter(testClient(t, srv))
	if _, err := w.Resolve(context.Background(), testFile, ""); err == nil || !strings.Contains(err.Error(), "no pages") {
		t.Errorf("got %v", err)
	}
	if _, err := w.Resolve(context.Background(), "f0000000-0000-4000-8000-0000000000aa", ""); err != nil {
		t.Logf("same handler serves every id: %v", err)
	}
	if _, _, err := NewWriter(testClient(t, srv)).Pages(context.Background(), testFile); err != nil {
		t.Fatal(err)
	}
}

func TestWriterReadOnlyToken(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, false)))
	w := NewWriter(testClient(t, srv))
	err := w.Apply(context.Background(), testFile, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot edit") {
		t.Fatalf("got %v", err)
	}
}

func TestWriterRetriesOnceOnRevnConflict(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, true)))
	calls := 0
	f.on("update-file", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"validation","code":"revn-conflict"}`))
			return
		}
		reply(map[string]any{"revn": 5})(w, r)
	})
	w := NewWriter(testClient(t, srv))
	if err := w.Apply(context.Background(), testFile, []map[string]any{{"type": "noop"}}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || f.callCount("get-file") != 2 {
		t.Errorf("update-file %d, get-file %d; want 2 and 2", calls, f.callCount("get-file"))
	}
}

func TestWriterGivesUpAfterSecondConflict(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, true)))
	f.on("update-file", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"validation","code":"vern-conflict"}`))
	})
	w := NewWriter(testClient(t, srv))
	if err := w.Apply(context.Background(), testFile, nil); !IsCode(err, "vern-conflict") {
		t.Fatalf("got %v", err)
	}
}

func TestWriterApplyTransit(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, true)))
	f.on("update-file", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/transit+json" {
			t.Errorf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		reply(map[string]any{"revn": 0})(w, r)
	})
	w := NewWriter(testClient(t, srv))
	err := w.ApplyTransit(context.Background(), testFile, []transit.Map{RegObjectsTransit(pageA, []string{shape})})
	if err != nil {
		t.Fatal(err)
	}
	if f.callCount("update-file") != 1 {
		t.Errorf("update-file called %d times, want 1", f.callCount("update-file"))
	}
}

func TestWriterExclusiveDropsCache(t *testing.T) {
	f, srv := newFake(t)
	f.on("get-file", reply(fileReply(0, 0, true)))
	w := NewWriter(testClient(t, srv))
	ctx := context.Background()
	if _, err := w.Resolve(ctx, testFile, ""); err != nil {
		t.Fatal(err)
	}
	ran := false
	if err := w.Exclusive(testFile, func() error { ran = true; return nil }); err != nil || !ran {
		t.Fatalf("ran %v, err %v", ran, err)
	}
	if _, err := w.Resolve(ctx, testFile, ""); err != nil {
		t.Fatal(err)
	}
	if got := f.callCount("get-file"); got != 2 {
		t.Errorf("get-file called %d times, want 2", got)
	}
}

// exporterServer logs in, then answers /api/export with the given status.
func exporterServer(t *testing.T, status int, body string) (*Exporter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/main/methods/login-with-password":
			http.SetCookie(w, &http.Cookie{Name: "auth-token", Value: "test-cookie"})
			_, _ = w.Write([]byte(`{"id":"e1000000-0000-4000-8000-0000000000aa"}`))
		case "/api/export":
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	e := NewExporter(testClient(t, srv), "me@example.invalid", "test-password")
	e.wait = func(context.Context, time.Duration) error { return nil }
	return e, srv
}

func TestExporterNilAndNoEmail(t *testing.T) {
	var e *Exporter
	if _, err := e.Render(context.Background(), Export{}); !errors.Is(err, ErrNoExporter) {
		t.Fatalf("got %v", err)
	}
	if NewExporter(&Client{}, "", "x") != nil || NewExporter(&Client{}, "x", "") != nil {
		t.Error("an exporter needs both email and password")
	}
}

func TestExporterErrorShapes(t *testing.T) {
	cases := []struct {
		name, body, want string
		status           int
	}{
		{"transit error", `["^ ","~:type","~:validation","~:code","~:bad-shape","~:hint","nope"]`, "validation: bad-shape: nope", 400},
		{"plain error", "boom", "boom", 500},
		{"long plain error", strings.Repeat("x", 300), strings.Repeat("x", 200), 500},
		{"no uri", `["^ ","~:id","~u1"]`, "no download uri", 200},
		{"bad transit", `not transit`, "reading the reply", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := exporterServer(t, tc.status, tc.body)
			_, err := e.Render(context.Background(), Export{FileID: testFile, PageID: pageA, ObjectID: shape, Type: "png"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestExporterBusyGivesUp(t *testing.T) {
	e, _ := exporterServer(t, http.StatusTooManyRequests, `["^ ","~:type","~:restriction","~:code","~:queue-full"]`)
	_, err := e.Render(context.Background(), Export{FileID: testFile, PageID: pageA, ObjectID: shape, Type: "png"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("got %v", err)
	}
}

func TestExporterWaitCancelled(t *testing.T) {
	e, _ := exporterServer(t, http.StatusServiceUnavailable, "busy")
	e.wait = func(context.Context, time.Duration) error { return context.Canceled }
	_, err := e.Render(context.Background(), Export{FileID: testFile, PageID: pageA, ObjectID: shape, Type: "png"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestExporterReloginAfter401(t *testing.T) {
	e, srv := exporterServer(t, http.StatusUnauthorized, "no")
	_, err := e.Render(context.Background(), Export{FileID: testFile, PageID: pageA, ObjectID: shape, Type: "png"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 401") {
		t.Fatalf("got %v", err)
	}
	_ = srv
}

func TestExporterLoginFailures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"rejected": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"validation","code":"wrong-credentials"}`))
		},
		"no cookie": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			t.Cleanup(srv.Close)
			e := NewExporter(testClient(t, srv), "me@example.invalid", "test-password")
			_, err := e.Render(context.Background(), Export{Type: "png"})
			if err == nil {
				t.Fatal("want a login error")
			}
			if strings.Contains(err.Error(), "test-password") {
				t.Errorf("the password leaked into %q", err)
			}
		})
	}
}
