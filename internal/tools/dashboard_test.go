package tools

import (
	"context"
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

const page2ID = "a0000000-0000-4000-8000-000000000002"

func TestDeleteNeedsConfirm(t *testing.T) {
	cs, b := session(t)
	for _, c := range []struct {
		tool string
		args map[string]any
	}{
		{"delete_project", map[string]any{"project_id": newProjectID}},
		{"delete_project", map[string]any{"project_id": newProjectID, "confirm": false}},
		{"delete_file", map[string]any{"file_id": fileID}},
		{"delete_page", map[string]any{"file_id": fileID, "page_id": pageID}},
	} {
		res := call(t, cs, c.tool, c.args, nil)
		if !res.IsError || !strings.Contains(errText(res), "confirm: true") {
			t.Errorf("%s %v: got %q, want a confirm error", c.tool, c.args, errText(res))
		}
	}
	if len(b.calls) != 0 {
		t.Errorf("an unconfirmed delete reached Penpot: %v", b.calls)
	}

	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools.Tools {
		if strings.HasPrefix(tool.Name, "delete_") && tool.Name != "delete_shapes" {
			if tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
				t.Errorf("%s should carry destructiveHint: true", tool.Name)
			}
		}
	}
}

func TestDeleteLogged(t *testing.T) {
	cs, b := session(t)
	b.pages = [][2]string{{pageID, "Page 1"}, {page2ID, "Old ideas"}}

	var p projectOut
	if res := call(t, cs, "delete_project", map[string]any{"project_id": newProjectID, "confirm": true}, &p); res.IsError {
		t.Fatal(errText(res))
	}
	if !p.Deleted || p.Name != "Money Logs" {
		t.Errorf("reply = %+v, want the deleted project's name", p)
	}
	var f fileOut
	if res := call(t, cs, "delete_file", map[string]any{"file_id": fileID, "confirm": true}, &f); res.IsError {
		t.Fatal(errText(res))
	}
	if f.FileName != "test file" {
		t.Errorf("reply = %+v, want the deleted file's name", f)
	}
	var pg pageOut
	if res := call(t, cs, "delete_page", map[string]any{"file_id": fileID, "page_id": page2ID, "confirm": true}, &pg); res.IsError {
		t.Fatal(errText(res))
	}
	if pg.PageName != "Old ideas" || len(pg.Pages) != 1 {
		t.Errorf("reply = %+v, want the deleted page's name and one page left", pg)
	}

	logs := b.logs.String()
	for _, want := range []string{
		"deleted project", newProjectID, "Money Logs",
		"deleted file", fileID, "test file",
		"deleted page", page2ID, "Old ideas",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("log lacks %q:\n%s", want, logs)
		}
	}
	if strings.Contains(logs, "test-token") {
		t.Error("the token reached the log")
	}
}

func TestDeleteRefusals(t *testing.T) {
	cs, b := session(t)
	res := call(t, cs, "delete_project", map[string]any{"project_id": draftsID, "confirm": true}, nil)
	if !res.IsError || !strings.Contains(errText(res), "default project") {
		t.Errorf("default project: got %q", errText(res))
	}
	res = call(t, cs, "delete_page", map[string]any{"file_id": fileID, "page_id": pageID, "confirm": true}, nil)
	if !res.IsError || !strings.Contains(errText(res), "last page") {
		t.Errorf("last page: got %q", errText(res))
	}
	if b.count("delete-project") != 0 || b.count("update-file") != 0 {
		t.Errorf("a refused delete reached Penpot: %v", b.calls)
	}
}

func TestReorderPages(t *testing.T) {
	cs, b := session(t)
	b.pages = [][2]string{{pageID, "Page 1"}, {page2ID, "Tokens"}}
	var out pageOut
	if res := call(t, cs, "reorder_pages", map[string]any{"file_id": fileID, "page_id": page2ID, "index": 0}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Pages[0].ID != page2ID || out.Pages[1].ID != pageID {
		t.Errorf("pages = %+v, want Tokens first", out.Pages)
	}
	res := call(t, cs, "reorder_pages", map[string]any{"file_id": fileID, "page_id": page2ID, "index": 2}, nil)
	if !res.IsError || !strings.Contains(errText(res), "index must be 0 to 1") {
		t.Errorf("index past the end: got %q", errText(res))
	}
}

func TestPinUsesProjectTeam(t *testing.T) {
	cs, b := session(t)
	var out projectOut
	if res := call(t, cs, "pin_project", map[string]any{"project_id": newProjectID, "pinned": true}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	body := b.bodies["update-project-pin"]
	if body["teamId"] != teamID || body["isPinned"] != true || body["id"] != newProjectID {
		t.Errorf("update-project-pin body = %v", body)
	}
}

func TestRestoreFile(t *testing.T) {
	cs, b := session(t)
	var listed restoreOut
	if res := call(t, cs, "restore_file", map[string]any{}, &listed); res.IsError {
		t.Fatal(errText(res))
	}
	if len(listed.Deleted) != 1 || listed.Deleted[0].ID != fileID {
		t.Errorf("no file_ids should list deleted files: %+v", listed)
	}
	var restored restoreOut
	if res := call(t, cs, "restore_file", map[string]any{"file_ids": []string{fileID}}, &restored); res.IsError {
		t.Fatal(errText(res))
	}
	body := b.bodies["restore-deleted-team-files"]
	if body["teamId"] != teamID || len(restored.Restored) != 1 {
		t.Errorf("restore body = %v, reply = %+v", body, restored)
	}
}

func TestListMembersRoles(t *testing.T) {
	cs, _ := session(t)
	var out membersOut
	if res := call(t, cs, "list_members", map[string]any{}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	got := []string{}
	for _, m := range out.Members {
		got = append(got, m.Name+":"+m.Role)
	}
	if strings.Join(got, " ") != "Ana:owner Bo:editor Cy:viewer" {
		t.Errorf("members = %v", got)
	}
}

func TestFileToolsSendIDs(t *testing.T) {
	cs, b := session(t)
	var dup fileOut
	if res := call(t, cs, "duplicate_file", map[string]any{"file_id": fileID, "name": "copy"}, &dup); res.IsError {
		t.Fatal(errText(res))
	}
	if dup.FileID == fileID || b.bodies["duplicate-file"]["fileId"] != fileID {
		t.Errorf("duplicate = %+v, body = %v", dup, b.bodies["duplicate-file"])
	}
	res := call(t, cs, "move_files", map[string]any{"file_ids": []string{fileID, fileID}, "project_id": newProjectID}, nil)
	if !res.IsError || !strings.Contains(errText(res), "twice") {
		t.Errorf("duplicate ids: got %q", errText(res))
	}
	res = call(t, cs, "link_library", map[string]any{"file_id": fileID, "library_id": fileID}, nil)
	if !res.IsError || !strings.Contains(errText(res), "itself") {
		t.Errorf("self link: got %q", errText(res))
	}
	if res := call(t, cs, "link_library", map[string]any{"file_id": fileID, "library_id": rectID}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if b.bodies["link-file-to-library"]["libraryId"] != rectID {
		t.Errorf("link body = %v", b.bodies["link-file-to-library"])
	}
	var found filesOut
	if res := call(t, cs, "search_files", map[string]any{"query": "test"}, &found); res.IsError {
		t.Fatal(errText(res))
	}
	if len(found.Files) != 1 || b.bodies["search-files"]["searchTerm"] != "test" {
		t.Errorf("search = %+v", found)
	}
}
