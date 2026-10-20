package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProfilesParallel(t *testing.T) {
	work, workDeps := newBackend(t)
	work.block = make(chan struct{})
	cs, home := sessionWith(t, func(d *Deps) { d.Profiles = map[string]Deps{"work": workDeps} })

	rect := func(profile string) map[string]any {
		args := map[string]any{"file_id": fileID, "name": "r", "x": 0, "y": 0, "width": 10, "height": 10}
		if profile != "" {
			args["profile"] = profile
		}
		return args
	}
	done := make(chan string, 1)
	go func() {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_rect", Arguments: rect("work")})
		if err != nil {
			done <- err.Error()
			return
		}
		if res.IsError {
			done <- errText(res)
			return
		}
		done <- ""
	}()
	// The work profile's write is stuck in Penpot; the default profile's
	// write must still go through.
	for work.count("update-file") == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if res := call(t, cs, "create_rect", rect(""), nil); res.IsError {
		t.Fatal(errText(res))
	}
	if home.count("update-file") != 1 {
		t.Errorf("the default profile's write went to %d update-files", home.count("update-file"))
	}
	close(work.block)
	if msg := <-done; msg != "" {
		t.Fatalf("work profile: %s", msg)
	}
	if work.count("update-file") != 1 {
		t.Error("profile: work did not reach its own Penpot")
	}

	if _, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "create_rect", Arguments: rect("nope")}); err == nil ||
		!strings.Contains(err.Error(), "profile must be one of [work]") {
		t.Errorf("an unknown profile should be refused: %v", err)
	}
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(tools.Tools[0].InputSchema)
	if !strings.Contains(string(data), `"profile"`) || !strings.Contains(string(data), `"work"`) {
		t.Errorf("tools should take a profile argument: %s", data)
	}
}
