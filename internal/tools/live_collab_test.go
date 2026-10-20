//go:build live

package tools

import "testing"

// TestLiveCollab adds a comment to the live file, replies and resolves it,
// saves a snapshot and lists it, and makes a share link then deletes it.
func TestLiveCollab(t *testing.T) {
	cs := liveSession(t)
	fileID := liveFileID(t, cs)

	var th thread
	mustCall(t, cs, "add_comment", map[string]any{"file_id": fileID, "x": 100, "y": 1200, "text": "live check: please ignore"}, &th)
	var reply comment
	mustCall(t, cs, "reply_comment", map[string]any{"thread_id": th.ID, "text": "live check: reply"}, &reply)
	var resolved thread
	mustCall(t, cs, "resolve_thread", map[string]any{"thread_id": th.ID}, &resolved)
	var threads commentsOut
	mustCall(t, cs, "list_comments", map[string]any{"thread_id": th.ID}, &threads)
	if len(threads.Comments) != 2 {
		t.Errorf("thread has %d comments, want 2", len(threads.Comments))
	}

	var snap snapshot
	mustCall(t, cs, "create_snapshot", map[string]any{"file_id": fileID, "label": "live check"}, &snap)
	var snaps snapshotsOut
	mustCall(t, cs, "list_snapshots", map[string]any{"file_id": fileID}, &snaps)
	found := false
	for _, s := range snaps.Snapshots {
		found = found || s.ID == snap.ID
	}
	if !found {
		t.Errorf("snapshot %s not listed", snap.ID)
	}

	var link shareOut
	mustCall(t, cs, "create_share_link", map[string]any{"file_id": fileID, "confirm": true}, &link)
	t.Logf("share link %s", link.URL)
	var gone idConfirmIn
	mustCall(t, cs, "delete_share_link", map[string]any{"id": link.ID, "confirm": true}, &gone)
}
