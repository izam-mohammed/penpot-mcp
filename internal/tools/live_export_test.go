//go:build live

package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLiveExport writes the first board of the live file to out/board.png
// (needs PENPOT_EMAIL and PENPOT_PASSWORD) and the whole file to
// out/live.penpot, in a temporary working directory.
func TestLiveExport(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)

	var file exportOut
	mustCall(t, cs, "export_file", map[string]any{"file_id": fileID, "path": "out/live.penpot"}, &file)
	t.Logf(".penpot: %d bytes", file.Files[0].Bytes)

	if os.Getenv("PENPOT_EMAIL") == "" || os.Getenv("PENPOT_PASSWORD") == "" {
		t.Skip("PENPOT_EMAIL and PENPOT_PASSWORD not set: board export skipped")
	}
	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID, "depth": 1}, &tree)
	board := ""
	for _, s := range tree.Shapes {
		if s.Type == "frame" {
			board = s.ID
			break
		}
	}
	var png exportOut
	mustCall(t, cs, "export_shape", map[string]any{"file_id": fileID, "shape_id": board, "path": "out/board.png", "scale": 2}, &png)
	if filepath.Base(png.Files[0].Path) != "board.png" || png.Files[0].Width == 0 {
		t.Errorf("export = %+v", png.Files)
	}
	t.Logf("board.png: %d x %d, %d bytes", png.Files[0].Width, png.Files[0].Height, png.Files[0].Bytes)
}
