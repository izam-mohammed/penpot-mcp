package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestResolve(t *testing.T) {
	cases := []struct {
		family         string
		weight         int
		italic         bool
		id, fam, varid string
	}{
		{"Inter", 400, false, "gfont-inter", "Inter", "regular"},
		{"Inter", 700, false, "gfont-inter", "Inter", "700"},
		{"Open Sans", 400, true, "gfont-open-sans", "Open Sans", "italic"},
		{"Open Sans", 600, true, "gfont-open-sans", "Open Sans", "600italic"},
		{"", 0, false, "sourcesanspro", "sourcesanspro", "regular"},
		{"Source Sans Pro", 700, false, "sourcesanspro", "sourcesanspro", "bold"},
		{"sourcesanspro", 500, false, "sourcesanspro", "sourcesanspro", "600"},
	}
	for _, c := range cases {
		f, err := Resolve(c.family, c.weight, c.italic)
		if err != nil {
			t.Fatal(err)
		}
		if f.FontID != c.id || f.Family != c.fam || f.VariantID != c.varid {
			t.Errorf("Resolve(%q, %d, %v) = %s / %s / %s, want %s / %s / %s",
				c.family, c.weight, c.italic, f.FontID, f.Family, f.VariantID, c.id, c.fam, c.varid)
		}
	}
	for _, bad := range []struct {
		family string
		weight int
	}{{"Inter", 450}, {"Inter", 1000}, {"<script>", 400}, {"a/../b", 400}} {
		if _, err := Resolve(bad.family, bad.weight, false); err == nil {
			t.Errorf("Resolve(%q, %d) should fail", bad.family, bad.weight)
		}
	}
}

// fontServer serves a Google-Fonts-like CSS reply pointing at a TTF, and the
// TTF itself (the Go Regular font, which ships with x/image).
func fontServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	hits := 0
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/css", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte("@font-face { src: url(https://fonts.gstatic.com/s/test/v1/x.ttf) format('truetype'); }"))
	})
	mux.HandleFunc("/fonts/", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write(goregular.TTF)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestTextMeasure(t *testing.T) {
	srv, _ := fontServer(t)
	m := NewMeasurer(srv.URL, t.TempDir(), nil)
	f, _ := Resolve("Source Sans Pro", 400, false)

	one := m.Measure(context.Background(), f, "Hello", 16, 1.2, 0)
	if !one.Measured {
		t.Fatal("want real metrics from the served font")
	}
	// Go Regular at 16 px: "Hello" advances about 36 px.
	if one.Width < 30 || one.Width > 45 {
		t.Errorf("width = %v, want about 37", one.Width)
	}
	if one.Height != 20 {
		t.Errorf("height = %v, want ceil(16 * 1.2) = 20", one.Height)
	}
	two := m.Measure(context.Background(), f, "Hello\nHello world", 16, 1.5, 0)
	if two.Height != 48 || two.Width <= one.Width {
		t.Errorf("two lines = %+v, want height 48 and the wider line's width", two)
	}
	spaced := m.Measure(context.Background(), f, "Hello", 16, 1.2, 2)
	if spaced.Width != one.Width+8 {
		t.Errorf("letter spacing 2 on 5 glyphs = %v, want %v", spaced.Width, one.Width+8)
	}
}

func TestMeasureFallsBack(t *testing.T) {
	m := NewMeasurer("http://127.0.0.1:1", "", nil)
	m.googleCSS = "http://127.0.0.1:1/css"
	f, _ := Resolve("Inter", 400, false)
	s := m.Measure(context.Background(), f, "abcd", 10, 1.2, 0)
	if s.Measured || s.Width != 23 { // ceil(4 * 10 * 0.55) + 1
		t.Errorf("fallback = %+v, want unmeasured width 23", s)
	}
}

func TestFontCachedOnDisk(t *testing.T) {
	srv, hits := fontServer(t)
	dir := t.TempDir()
	f, _ := Resolve("Source Sans Pro", 700, false)
	NewMeasurer(srv.URL, dir, nil).Measure(context.Background(), f, "x", 12, 1, 0)
	first := *hits
	if _, err := os.Stat(filepath.Join(dir, "source-sans-pro-700normal.ttf")); err != nil {
		t.Fatal("font not cached:", err)
	}
	NewMeasurer(srv.URL, dir, nil).Measure(context.Background(), f, "x", 12, 1, 0)
	if *hits != first {
		t.Errorf("second measurer fetched again (%d hits, want %d)", *hits, first)
	}
}
