//go:build live

package tools

import (
	"os"
	"testing"

	"github.com/izam-mohammed/penpot-mcp/internal/mcproute"
)

func liveRoute(t *testing.T) *mcproute.Route {
	t.Helper()
	r, err := mcproute.New(os.Getenv("PENPOT_MCP_URL"), "live", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Cleanup(r.Close)
	}
	return r
}

// TestTabLive (make live-mcp, not make live) needs the live file open in a tab with the MCP plugin
// connected: it reads which file is open, then reflows it.
func TestTabLive(t *testing.T) {
	if os.Getenv("PENPOT_MCP_URL") == "" {
		t.Skip("PENPOT_MCP_URL not set")
	}
	cs := liveSession(t)
	fileID := liveFileID(t, cs)
	var open mcproute.Open
	mustCall(t, cs, "current_file", nil, &open)
	t.Logf("tab has %q, page %q", open.FileName, open.PageName)
	if open.FileID != fileID {
		t.Fatalf("open %q in the tab first (it has %q)", liveFile, open.FileName)
	}
	var out tabOut
	mustCall(t, cs, "reflow", map[string]any{"file_id": fileID}, &out)
	if out.Result != true {
		t.Errorf("reflow = %+v", out)
	}
}
