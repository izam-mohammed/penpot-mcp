package tools

import (
	"strings"
	"testing"
)

func TestReadTools(t *testing.T) {
	cs, _ := session(t)
	var who whoamiOut
	if res := call(t, cs, "whoami", map[string]any{}, &who); res.IsError || who.Name != "Test" {
		t.Errorf("whoami = %+v %q", who, errText(res))
	}
	var teams teamsOut
	if res := call(t, cs, "list_teams", map[string]any{}, &teams); res.IsError || len(teams.Teams) != 1 {
		t.Errorf("teams = %+v %q", teams, errText(res))
	}
	var all, one projectsOut
	call(t, cs, "list_projects", map[string]any{}, &all)
	call(t, cs, "list_projects", map[string]any{"team_id": teamID}, &one)
	if len(all.Projects) != 2 || len(one.Projects) != 1 {
		t.Errorf("projects: all %d, team %d", len(all.Projects), len(one.Projects))
	}
	var files filesOut
	if res := call(t, cs, "list_files", map[string]any{"project_id": draftsID}, &files); res.IsError || len(files.Files) != 1 {
		t.Errorf("files = %+v %q", files, errText(res))
	}
	var pages pagesOut
	if res := call(t, cs, "list_pages", map[string]any{"file_id": fileID}, &pages); res.IsError || len(pages.Pages) != 1 || pages.FileName != "test file" {
		t.Errorf("pages = %+v %q", pages, errText(res))
	}
	for name, tc := range map[string]struct {
		tool string
		args map[string]any
	}{
		"bad team":     {"list_projects", map[string]any{"team_id": "x"}},
		"bad project":  {"list_files", map[string]any{"project_id": "x"}},
		"bad depth":    {"get_page_tree", map[string]any{"file_id": fileID, "depth": 11}},
		"bad page id":  {"get_page_tree", map[string]any{"file_id": fileID, "page_id": "x"}},
		"unknown page": {"get_page_tree", map[string]any{"file_id": fileID, "page_id": "a0000000-0000-4000-8000-0000000000ff"}},
	} {
		if res := call(t, cs, tc.tool, tc.args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
}

func TestSharingRules(t *testing.T) {
	cs, b := session(t)
	var out shareOut
	res := call(t, cs, "create_share_link", map[string]any{"file_id": fileID, "who_comment": "all", "who_inspect": "team", "confirm": true}, &out)
	if res.IsError || !strings.Contains(out.URL, "share-id=") || len(out.Pages) != 1 {
		t.Fatalf("share = %+v %q", out, errText(res))
	}
	for name, args := range map[string]map[string]any{
		"no confirm":   {"file_id": fileID},
		"bad who":      {"file_id": fileID, "who_comment": "everyone", "confirm": true},
		"bad inspect":  {"file_id": fileID, "who_inspect": "everyone", "confirm": true},
		"bad page id":  {"file_id": fileID, "page_ids": []string{"x"}, "confirm": true},
		"unknown page": {"file_id": fileID, "page_ids": []string{"a0000000-0000-4000-8000-0000000000ff"}, "confirm": true},
	} {
		if res := call(t, cs, "create_share_link", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	before := b.count("create-share-link")
	if before != 1 {
		t.Errorf("share links created = %d, want 1", before)
	}
	if res := call(t, cs, "delete_share_link", map[string]any{"id": "5a000000-0000-4000-8000-000000000001"}, nil); !res.IsError {
		t.Error("an unconfirmed delete was accepted")
	}
	if res := call(t, cs, "delete_share_link", map[string]any{"id": "5a000000-0000-4000-8000-000000000001", "confirm": true}, nil); res.IsError {
		t.Error(errText(res))
	}
}

func TestWebhookRules(t *testing.T) {
	cs, _ := session(t)
	var list webhooksOut
	if res := call(t, cs, "list_webhooks", map[string]any{}, &list); res.IsError || len(list.Webhooks) != 1 {
		t.Fatalf("webhooks = %+v %q", list, errText(res))
	}
	var made webhook
	if res := call(t, cs, "create_webhook", map[string]any{"url": "https://hooks.example.com/p", "transit": true, "confirm": true}, &made); res.IsError ||
		made.MType != "application/transit+json" {
		t.Errorf("webhook = %+v %q", made, errText(res))
	}
	for name, args := range map[string]map[string]any{
		"http":        {"url": "http://hooks.example.com/p", "confirm": true},
		"credentials": {"url": "https://user:pw@hooks.example.com/p", "confirm": true},
		"no host":     {"url": "https:///p", "confirm": true},
		"no confirm":  {"url": "https://hooks.example.com/p"},
		"bad team":    {"url": "https://hooks.example.com/p", "team_id": "x", "confirm": true},
	} {
		if res := call(t, cs, "create_webhook", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if res := call(t, cs, "delete_webhook", map[string]any{"id": "w1"}, nil); !res.IsError {
		t.Error("a malformed id was accepted")
	}
	if res := call(t, cs, "delete_webhook", map[string]any{"id": "5a000000-0000-4000-8000-000000000002", "confirm": true}, nil); res.IsError {
		t.Error(errText(res))
	}
}
