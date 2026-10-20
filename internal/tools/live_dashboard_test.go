//go:build live

package tools

import (
	"fmt"
	"testing"
	"time"
)

// liveProject is the project and file TestLiveDashboard makes, reused on
// later runs.
const liveProject = "penpot-mcp live"

// TestLiveDashboard finds or creates the "penpot-mcp live" project and file,
// adds a page, renames, reorders and deletes it, renames the file away and
// back, then duplicates it, deletes the copy and finds it among the deleted.
func TestLiveDashboard(t *testing.T) {
	cs := liveSession(t)

	var projects projectsOut
	mustCall(t, cs, "list_projects", nil, &projects)
	projectID := ""
	for _, p := range projects.Projects {
		if p.Name == liveProject {
			projectID = p.ID
		}
	}
	if projectID == "" {
		var p struct{ ID, Name string }
		mustCall(t, cs, "create_project", map[string]any{"name": liveProject}, &p)
		projectID = p.ID
		t.Logf("created project %s", p.ID)
	}

	var files filesOut
	mustCall(t, cs, "list_files", map[string]any{"project_id": projectID}, &files)
	fileID := ""
	for _, f := range files.Files {
		if f.Name == liveProject {
			fileID = f.ID
		}
	}
	if fileID == "" {
		var f fileOut
		mustCall(t, cs, "create_file", map[string]any{"project_id": projectID, "name": liveProject}, &f)
		if len(f.Pages) == 0 {
			t.Fatalf("new file has no pages: %+v", f)
		}
		fileID = f.FileID
		t.Logf("created file %s with page %q", f.FileID, f.Pages[0].Name)
	}

	stamp := time.Now().Format("0102-150405")
	var added pageOut
	mustCall(t, cs, "add_page", map[string]any{"file_id": fileID, "name": "added " + stamp}, &added)
	if added.Pages[len(added.Pages)-1].ID != added.PageID {
		t.Errorf("the new page should be last: %+v", added.Pages)
	}
	var renamed pageOut
	mustCall(t, cs, "rename_page", map[string]any{"file_id": fileID, "page_id": added.PageID, "name": "renamed " + stamp}, &renamed)
	if renamed.PageName != "renamed "+stamp {
		t.Errorf("page name = %q", renamed.PageName)
	}

	var f fileOut
	away := fmt.Sprintf("%s (renamed %s)", liveProject, stamp)
	mustCall(t, cs, "rename_file", map[string]any{"file_id": fileID, "name": away}, &f)
	if f.FileName != away {
		t.Errorf("file name = %q", f.FileName)
	}
	var pages pagesOut
	mustCall(t, cs, "list_pages", map[string]any{"file_id": fileID}, &pages)
	if pages.FileName != away {
		t.Errorf("list_pages still says %q after the rename", pages.FileName)
	}
	mustCall(t, cs, "rename_file", map[string]any{"file_id": fileID, "name": liveProject}, &f)

	var moved pageOut
	mustCall(t, cs, "reorder_pages", map[string]any{"file_id": fileID, "page_id": added.PageID, "index": 0}, &moved)
	if moved.Pages[0].ID != added.PageID {
		t.Errorf("the page should be first after reorder: %+v", moved.Pages)
	}
	var gone pageOut
	mustCall(t, cs, "delete_page", map[string]any{"file_id": fileID, "page_id": added.PageID, "confirm": true}, &gone)
	if len(gone.Pages) != len(moved.Pages)-1 {
		t.Errorf("delete_page left %d pages, want %d", len(gone.Pages), len(moved.Pages)-1)
	}

	var copied fileOut
	mustCall(t, cs, "duplicate_file", map[string]any{"file_id": fileID, "name": liveProject + " copy " + stamp}, &copied)
	var found filesOut
	mustCall(t, cs, "search_files", map[string]any{"query": "copy " + stamp}, &found)
	if len(found.Files) != 1 || found.Files[0].ID != copied.FileID {
		t.Errorf("search_files found %+v, want the copy %s", found.Files, copied.FileID)
	}
	var deleted fileOut
	mustCall(t, cs, "delete_file", map[string]any{"file_id": copied.FileID, "confirm": true}, &deleted)
	var bin restoreOut
	mustCall(t, cs, "restore_file", map[string]any{}, &bin)
	inBin := false
	for _, d := range bin.Deleted {
		inBin = inBin || d.ID == copied.FileID
	}
	if !inBin {
		t.Errorf("the deleted copy %s is not among the deleted files", copied.FileID)
	}

	var members membersOut
	mustCall(t, cs, "list_members", nil, &members)
	t.Logf("file %s has %d pages; team has %d members", fileID, len(gone.Pages), len(members.Members))
}
