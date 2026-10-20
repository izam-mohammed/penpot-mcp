package tools

import (
	"strings"
	"testing"
)

func TestComments(t *testing.T) {
	cs, b := session(t)
	var threads commentsOut
	if res := call(t, cs, "list_comments", map[string]any{"file_id": fileID}, &threads); res.IsError {
		t.Fatal(errText(res))
	}
	if len(threads.Threads) != 1 || threads.Threads[0].Number != 3 || threads.Threads[0].Text != "make it bigger" || threads.Threads[0].X != 10 {
		t.Errorf("threads = %+v", threads.Threads)
	}
	// A thread's comments need only the thread id.
	var one commentsOut
	if res := call(t, cs, "list_comments", map[string]any{"thread_id": threadID}, &one); res.IsError {
		t.Fatal(errText(res))
	}
	if one.ThreadID != threadID || len(one.Comments) == 0 {
		t.Errorf("thread comments = %+v", one)
	}
	var th thread
	if res := call(t, cs, "add_comment", map[string]any{"file_id": fileID, "board_id": boardID, "x": 40, "y": 50, "text": "done"}, &th); res.IsError {
		t.Fatal(errText(res))
	}
	body := b.bodies["create-comment-thread"]
	if body["frameId"] != boardID || body["content"] != "done" {
		t.Errorf("create-comment-thread body = %v", body)
	}
	res := call(t, cs, "add_comment", map[string]any{"file_id": fileID, "board_id": rectID, "x": 0, "y": 0, "text": "x"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "pin to boards") {
		t.Errorf("comment on a rect: %q", errText(res))
	}
	if res := call(t, cs, "reply_comment", map[string]any{"thread_id": threadID, "text": strings.Repeat("x", 751)}, nil); !res.IsError {
		t.Error("a 751-character reply should be refused")
	}
	if res := call(t, cs, "resolve_thread", map[string]any{"thread_id": threadID}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if b.bodies["update-comment-thread"]["isResolved"] != true {
		t.Errorf("resolve body = %v", b.bodies["update-comment-thread"])
	}
}

func TestShareNeedsConfirm(t *testing.T) {
	cs, b := session(t)
	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"create_share_link", map[string]any{"file_id": fileID}},
		{"create_webhook", map[string]any{"url": "https://hooks.example.com/p"}},
		{"delete_webhook", map[string]any{"id": rectID}},
		{"delete_share_link", map[string]any{"id": rectID}},
		{"restore_snapshot", map[string]any{"file_id": fileID, "snapshot_id": snapID}},
	} {
		res := call(t, cs, c.tool, c.args, nil)
		if !res.IsError || !strings.Contains(errText(res), "confirm: true") {
			t.Errorf("%s: got %q", c.tool, errText(res))
		}
	}
	if res := call(t, cs, "create_webhook", map[string]any{"url": "http://hooks.example.com", "confirm": true}, nil); !res.IsError {
		t.Error("a plain http webhook should be refused")
	}
	if len(b.calls) != 0 {
		t.Errorf("unconfirmed sharing reached Penpot: %v", b.calls)
	}

	var link shareOut
	if res := call(t, cs, "create_share_link", map[string]any{"file_id": fileID, "who_comment": "all", "confirm": true}, &link); res.IsError {
		t.Fatal(errText(res))
	}
	if !strings.Contains(link.URL, "/#/view?") || !strings.Contains(link.URL, "share-id=5a000000") || len(link.Pages) != 1 {
		t.Errorf("link = %+v", link)
	}
	if b.bodies["create-share-link"]["whoComment"] != "all" || b.bodies["create-share-link"]["whoInspect"] != "team" {
		t.Errorf("share body = %v", b.bodies["create-share-link"])
	}
}

func TestRestoreSnapshotsFirst(t *testing.T) {
	cs, b := session(t)
	var out snapshotsOut
	if res := call(t, cs, "restore_snapshot", map[string]any{"file_id": fileID, "snapshot_id": snapID, "confirm": true}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	order := strings.Join(b.calls, " ")
	if !strings.Contains(order, "create-file-snapshot restore-file-snapshot") {
		t.Errorf("the current state must be saved before the restore: %s", order)
	}
	if out.Safety == nil || !strings.HasPrefix(b.bodies["create-file-snapshot"]["label"].(string), "before restore") {
		t.Errorf("safety snapshot = %+v", out.Safety)
	}
	res := call(t, cs, "restore_snapshot", map[string]any{"file_id": fileID, "snapshot_id": rectID, "confirm": true}, nil)
	if !res.IsError || !strings.Contains(errText(res), "not a version of this file") {
		t.Errorf("unknown snapshot: %q", errText(res))
	}
}

func TestPluginData(t *testing.T) {
	cs, b := session(t)
	v := "approved"
	var out pluginDataOut
	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "shape_id": rectID, "namespace": "review", "key": "state", "value": v}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	for _, want := range []string{`"type":"set-plugin-data"`, `"objectType":"shape"`, `"objectId":"` + rectID + `"`, `"value":"approved"`} {
		if !strings.Contains(b.raw[0], want) {
			t.Errorf("change lacks %s: %s", want, b.raw[0])
		}
	}
	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "namespace": "bad space", "key": "k"}, nil); !res.IsError {
		t.Error("a namespace with a space should be refused")
	}
}
