//go:build live

package tools

import (
	"fmt"
	"testing"
	"time"
)

// liveProject is the project and file TestLiveDashboard makes. They are
// reused on later runs: deletes come with the guarded delete tools.
const liveProject = "penpot-mcp live"

// TestLiveDashboard finds or creates the "penpot-mcp live" project and file,
// adds a page, renames it, and renames the file away and back.
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
	t.Logf("file %s now has %d pages", fileID, len(pages.Pages))
}
