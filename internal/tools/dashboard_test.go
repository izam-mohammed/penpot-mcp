package tools

import (
	"strings"
	"testing"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

func TestCreateProjectUsesPenpotID(t *testing.T) {
	cs, b := session(t)
	var out penpot.Project
	res := call(t, cs, "create_project", map[string]any{"name": "Money Logs"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	body := b.bodies["create-project"]
	if _, sent := body["id"]; sent {
		t.Error("create-project must not send an id: Penpot 2.19 rejects it")
	}
	if body["teamId"] != teamID || body["name"] != "Money Logs" {
		t.Errorf("create-project body = %v, want the default team and the name", body)
	}
	if out.ID != newProjectID || out.Name != "Money Logs" {
		t.Errorf("project = %+v, want Penpot's id", out)
	}
}

func TestCreateFileReturnsPages(t *testing.T) {
	cs, b := session(t)
	var out fileOut
	res := call(t, cs, "create_file", map[string]any{"project_id": newProjectID, "name": "Money Logs: app design"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if _, sent := b.bodies["create-file"]["id"]; sent {
		t.Error("create-file must not send an id: Penpot 2.19 rejects it")
	}
	if out.FileID != fileID || out.ProjectID != newProjectID {
		t.Errorf("file = %+v, want Penpot's id", out)
	}
	if len(out.Pages) != 1 || out.Pages[0].ID != pageID {
		t.Errorf("pages = %+v, want the file's first page", out.Pages)
	}
}

func TestRenameFileDropsCache(t *testing.T) {
	cs, b := session(t)
	call(t, cs, "list_pages", map[string]any{"file_id": fileID}, nil)
	var out fileOut
	res := call(t, cs, "rename_file", map[string]any{"file_id": fileID, "name": "renamed"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.FileName != "renamed" {
		t.Errorf("file name = %q", out.FileName)
	}
	before := b.count("get-file")
	call(t, cs, "get_page_tree", map[string]any{"file_id": fileID}, nil)
	if b.count("get-file") != before+1 {
		t.Error("a rename should drop the cached file so the next write sees the new name")
	}
}

func TestAddAndRenamePage(t *testing.T) {
	cs, b := session(t)
	var added pageOut
	res := call(t, cs, "add_page", map[string]any{"file_id": fileID, "name": "Screens"}, &added)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if added.PageName != "Screens" || len(added.Pages) != 2 || added.Pages[1].ID != added.PageID {
		t.Fatalf("add_page = %+v, want the new page last", added)
	}
	if !uuidRE.MatchString(added.PageID) {
		t.Errorf("page id %q is not a UUID", added.PageID)
	}
	if !strings.Contains(b.raw[0], `"type":"add-page"`) || strings.Contains(b.raw[0], `"page":`) {
		t.Errorf("add-page should send the {id, name} form only: %s", b.raw[0])
	}

	var renamed pageOut
	res = call(t, cs, "rename_page", map[string]any{"file_id": fileID, "page_id": pageID, "name": "Tokens"}, &renamed)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if renamed.PageName != "Tokens" || renamed.Pages[0].Name != "Tokens" {
		t.Errorf("rename_page = %+v", renamed)
	}
}

func TestDashboardRejectsBadInput(t *testing.T) {
	cs, b := session(t)
	for _, c := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"create_project", map[string]any{"name": " "}, "name must be"},
		{"create_project", map[string]any{"name": "x", "team_id": "nope"}, "team_id"},
		{"create_file", map[string]any{"project_id": "nope", "name": "x"}, "project_id"},
		{"create_file", map[string]any{"project_id": newProjectID, "name": strings.Repeat("x", 201)}, "name must be"},
		{"rename_file", map[string]any{"file_id": fileID, "name": ""}, "name must be"},
		{"add_page", map[string]any{"file_id": "nope", "name": "x"}, "file_id"},
		{"rename_page", map[string]any{"file_id": fileID, "page_id": "nope", "name": "x"}, "page_id"},
	} {
		res := call(t, cs, c.tool, c.args, nil)
		if !res.IsError || !strings.Contains(errText(res), c.want) {
			t.Errorf("%s %v: got %q, want an error with %q", c.tool, c.args, errText(res), c.want)
		}
	}
	if len(b.calls) != 0 {
		t.Errorf("bad input reached Penpot: %v", b.calls)
	}
}

func TestRenamePageNotInFile(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "rename_page", map[string]any{"file_id": fileID, "page_id": rectID, "name": "x"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "is not in file") {
		t.Errorf("got %q", errText(res))
	}
	if b.count("update-file") != 0 {
		t.Error("a page not in the file must not reach update-file")
	}
}
