package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuiltinVariantAndFile(t *testing.T) {
	cases := []struct {
		weight  int
		italic  bool
		variant string
		file    string
	}{
		{400, true, "italic", "sourcesanspro-italic.ttf"},
		{700, true, "bolditalic", "sourcesanspro-bolditalic.ttf"},
		{200, false, "200", "sourcesanspro-extralight.ttf"},
		{100, false, "200", "sourcesanspro-extralight.ttf"},
		{900, false, "black", "sourcesanspro-black.ttf"},
	}
	for _, c := range cases {
		if got := builtinVariant(c.weight, c.italic); got != c.variant {
			t.Errorf("builtinVariant(%d, %v) = %q, want %q", c.weight, c.italic, got, c.variant)
		}
		if got := builtinFile(c.weight, c.italic); got != c.file {
			t.Errorf("builtinFile(%d, %v) = %q, want %q", c.weight, c.italic, got, c.file)
		}
	}
}

func TestMeasureWrapped(t *testing.T) {
	srv, _ := fontServer(t)
	m := NewMeasurer(srv.URL, t.TempDir(), nil)
	f, _ := Resolve("Source Sans Pro", 400, false)
	ctx := context.Background()

	narrow := m.MeasureWrapped(ctx, f, "one two three four five six seven", 16, 1.5, 0, 60)
	if narrow.Width != 60 {
		t.Errorf("width = %v, want the box width 60", narrow.Width)
	}
	if narrow.Height <= 24 {
		t.Errorf("height = %v, want more than one line (24)", narrow.Height)
	}
	wide := m.MeasureWrapped(ctx, f, "one two\n\nthree", 16, 1.5, 0, 10000)
	if wide.Height != 72 { // three lines: two words, a blank, one word
		t.Errorf("height = %v, want 72", wide.Height)
	}
}

func TestSourceURLFailures(t *testing.T) {
	status := http.StatusOK
	body := "no url here"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	m := NewMeasurer("", "", nil)
	m.googleCSS = srv.URL + "/css"
	f, _ := Resolve("Inter", 400, true)

	if _, err := m.sourceURL(context.Background(), f); err == nil || !strings.Contains(err.Error(), "no ttf url") {
		t.Errorf("got %v", err)
	}
	status = http.StatusNotFound
	if _, err := m.sourceURL(context.Background(), f); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("got %v", err)
	}
	body = "@font-face { src: url(https://fonts.gstatic.com/s/x/v1/a.ttf); }"
	status = http.StatusOK
	got, err := m.sourceURL(context.Background(), f)
	if err != nil || got != "https://fonts.gstatic.com/s/x/v1/a.ttf" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestGetRejectsOversizeAndBadRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, maxFontBytes+10))
	}))
	t.Cleanup(srv.Close)
	m := NewMeasurer("", "", nil)
	if _, err := m.get(context.Background(), srv.URL, "font/ttf"); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("got %v", err)
	}
	if _, err := m.get(context.Background(), "://bad", "font/ttf"); err == nil {
		t.Error("a bad URL should fail")
	}
	if _, err := m.get(context.Background(), "http://127.0.0.1:1", "font/ttf"); err == nil {
		t.Error("a refused connection should fail")
	}
}

func TestBrokenFontFileIsRememberedAsFailed(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("this is not a font"))
	}))
	t.Cleanup(srv.Close)
	m := NewMeasurer(srv.URL, "", nil)
	f, _ := Resolve("Source Sans Pro", 400, false)
	for range 2 {
		if s := m.Measure(context.Background(), f, "abc", 10, 1, 0); s.Measured {
			t.Fatal("a broken font should fall back to estimates")
		}
	}
	if hits != 1 {
		t.Errorf("a failed font was fetched %d times, want once", hits)
	}
}
