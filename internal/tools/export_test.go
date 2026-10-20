package tools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportPathGuard(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "taken.png"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if p, err := outPath(root, "design/new/board.png", []string{".png"}, false); err != nil || !strings.HasSuffix(p, filepath.Join("design", "new", "board.png")) {
		t.Errorf("a new file in a new folder: %q, %v", p, err)
	}
	if _, err := outPath(root, "taken.png", []string{".png"}, true); err != nil {
		t.Errorf("overwrite: true should allow an existing file: %v", err)
	}
	for _, c := range []struct{ path, want string }{
		{"taken.png", "overwrite: true"},
		{"../x.png", "outside the working directory"},
		{filepath.Join(outside, "x.png"), "outside the working directory"},
		{"escape/x.png", "outside the working directory"},
		{"notes.txt", "must end in .png"},
		{".git/x.png", "inside .git"},
		{"", "path is required"},
	} {
		if _, err := outPath(root, c.path, []string{".png"}, false); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("outPath(%q) = %v, want %q", c.path, err, c.want)
		}
	}
}

func pngOf(t *testing.T, two bool) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.White)
		}
	}
	if two {
		img.Set(3, 3, color.Black)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestBlankPngFails(t *testing.T) {
	if w, h, err := checkPNG(pngOf(t, true)); err != nil || w != 8 || h != 8 {
		t.Errorf("a drawn PNG: %d x %d, %v", w, h, err)
	}
	if _, _, err := checkPNG(pngOf(t, false)); err == nil || !strings.Contains(err.Error(), "blank") {
		t.Errorf("a flat PNG should fail as blank: %v", err)
	}
	full := pngOf(t, true)
	if _, _, err := checkPNG(full[:len(full)/2]); err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Errorf("a cut PNG should fail as truncated: %v", err)
	}
}

func TestExportNeedsLogin(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "export_shape", map[string]any{"file_id": fileID, "shape_id": boardID, "path": "out/board.png"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "PENPOT_EMAIL") {
		t.Errorf("no exporter: got %q", errText(res))
	}
	res = call(t, cs, "export_shape", map[string]any{"file_id": fileID, "shape_id": boardID, "path": "out/board.pdf"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "must end in .png") {
		t.Errorf("wrong extension: got %q", errText(res))
	}
	if len(b.calls) != 0 {
		t.Errorf("a refused export reached Penpot: %v", b.calls)
	}
}

func TestSSEEnd(t *testing.T) {
	got, err := sseEnd([]byte("event: progress\ndata: {\"~:done\":1}\n\nevent: end\ndata: {\"~#'\":\"~rhttps://x/assets/by-id/9\"}\n\n"))
	if err != nil || got != `{"~#'":"~rhttps://x/assets/by-id/9"}` {
		t.Errorf("end = %q, %v", got, err)
	}
	if _, err := sseEnd([]byte("event: error\ndata: boom\n")); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("error event: %v", err)
	}
	if _, err := sseEnd([]byte("event: progress\ndata: 1\n")); err == nil {
		t.Error("a stream without end should fail")
	}
}
