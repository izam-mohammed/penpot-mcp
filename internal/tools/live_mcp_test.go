//go:build live

package tools

import (
	"os"
	"strings"
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
// connected: it reads which file is open, reflows it, and applies a token
// that refers to another one, which only the tab can resolve.
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

	var synced syncOut
	mustCall(t, cs, "sync_tokens", map[string]any{"file_id": fileID, "set_name": "live", "tokens": map[string]any{
		"color": map[string]any{
			"accent": map[string]any{"$value": "#E11D48", "$type": "color"},
			"link":   map[string]any{"$value": "{color.accent}", "$type": "color"},
		},
	}}, &synced)
	var tree treeOut
	mustCall(t, cs, "get_page_tree", map[string]any{"file_id": fileID, "page_id": open.PageID, "depth": 4}, &tree)
	rect := ""
	for _, s := range tree.Shapes {
		if s.Type == "rect" {
			rect = s.ID
			break
		}
	}
	if rect == "" {
		t.Skip("no rect on the open page to apply a token to")
	}
	var applied changedOut
	mustCall(t, cs, "apply_token", map[string]any{
		"file_id": fileID, "page_id": open.PageID, "shape_ids": []string{rect},
		"token": "color.link", "attribute": "fill",
	}, &applied)
	t.Logf("applied color.link to %s through the tab", rect)
}

// TestLiveAPIInfo reads one backend method's documentation from Penpot's
// OpenAPI document; no tab needed.
func TestLiveAPIInfo(t *testing.T) {
	cs := liveSession(t)
	var out textOut
	mustCall(t, cs, "api_info", map[string]any{"type": "create-project", "source": "backend"}, &out)
	if !strings.Contains(out.Text, "teamId") {
		t.Errorf("create-project docs lack teamId:\n%s", out.Text)
	}
	mustCall(t, cs, "api_info", map[string]any{"type": "update-file", "source": "backend", "member": "revn"}, &out)
	t.Logf("update-file revn: %s", strings.ReplaceAll(out.Text, "\n", " "))
}
