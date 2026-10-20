package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type shareIn struct {
	FileID     string   `json:"file_id" jsonschema:"the file"`
	PageIDs    []string `json:"page_ids,omitempty" jsonschema:"the pages the link shows; every page when empty"`
	WhoComment string   `json:"who_comment,omitempty" jsonschema:"team (default) or all: who may comment through the link"`
	WhoInspect string   `json:"who_inspect,omitempty" jsonschema:"team (default) or all: who may inspect code through the link"`
	Confirm    bool     `json:"confirm,omitempty" jsonschema:"must be true; anyone with the link can view the pages, so say so only after the user asked for a link"`
}

type shareOut struct {
	ID     string   `json:"id"`
	FileID string   `json:"file_id"`
	URL    string   `json:"url"`
	Pages  []string `json:"page_ids"`
}

type idConfirmIn struct {
	ID      string `json:"id" jsonschema:"the share link or webhook"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true; say so only after the user asked for this to be deleted"`
}

type teamIn struct {
	TeamID string `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
}

type webhookIn struct {
	TeamID  string `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
	URL     string `json:"url" jsonschema:"the https endpoint Penpot posts events to"`
	Transit bool   `json:"transit,omitempty" jsonschema:"send transit+json instead of JSON"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true; Penpot will send this team's file events to the URL, so say so only after the user asked for it"`
}

type webhook struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MType    string `json:"mtype,omitempty"`
	IsActive bool   `json:"is_active"`
	Errors   int    `json:"error_count,omitempty"`
}

type webhooksOut struct {
	TeamID   string    `json:"team_id"`
	Webhooks []webhook `json:"webhooks"`
}

func registerSharing(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name: "create_share_link",
		Description: "Make a view link to a file's pages. Anyone with the link can see them. " +
			"Needs confirm: true, given only after the user asked for a link.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: ptr(true)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in shareIn) (*mcp.CallToolResult, shareOut, error) {
		d := d.pick(ctx)
		out, err := createShare(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_share_link",
		Description: "Delete a share link: it stops working for everyone. Needs confirm: true.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in idConfirmIn) (*mcp.CallToolResult, idConfirmIn, error) {
		d := d.pick(ctx)
		if err := checkID("id", in.ID); err != nil {
			return nil, idConfirmIn{}, err
		}
		if err := checkConfirm(in.Confirm, "a share link"); err != nil {
			return nil, idConfirmIn{}, err
		}
		if err := d.Client.Call(ctx, "delete-share-link", map[string]string{"id": in.ID}, nil); err != nil {
			return nil, idConfirmIn{}, err
		}
		d.log().Info("deleted share link", "id", in.ID)
		return nil, in, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_webhooks",
		Description: "A team's webhooks: URL, format, active, recent errors.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in teamIn) (*mcp.CallToolResult, webhooksOut, error) {
		d := d.pick(ctx)
		team, err := teamOrDefault(ctx, d, in.TeamID)
		if err != nil {
			return nil, webhooksOut{}, err
		}
		var rows []map[string]any
		if err := d.Client.Call(ctx, "get-webhooks", map[string]string{"teamId": team}, &rows); err != nil {
			return nil, webhooksOut{}, err
		}
		out := webhooksOut{TeamID: team, Webhooks: []webhook{}}
		for _, r := range rows {
			out.Webhooks = append(out.Webhooks, webhook{
				ID: str(r, "id"), URL: str(r, "uri"), MType: str(r, "mtype"),
				IsActive: r["isActive"] == true, Errors: int(num(r, "errorCount")),
			})
		}
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "create_webhook",
		Description: "Make Penpot post a team's file events (edits, comments, new files) to a URL. " +
			"That sends design data outside Penpot: needs confirm: true, given only after the user asked for it.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), OpenWorldHint: ptr(true)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in webhookIn) (*mcp.CallToolResult, webhook, error) {
		d := d.pick(ctx)
		u, err := url.Parse(in.URL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return nil, webhook{}, fmt.Errorf("url must be an https URL without credentials in it")
		}
		if !in.Confirm {
			return nil, webhook{}, fmt.Errorf("a webhook sends this team's events to %s: needs confirm: true; ask the user first", u.Host)
		}
		team, err := teamOrDefault(ctx, d, in.TeamID)
		if err != nil {
			return nil, webhook{}, err
		}
		mtype := "application/json"
		if in.Transit {
			mtype = "application/transit+json"
		}
		var r map[string]any
		if err := d.Client.Call(ctx, "create-webhook", map[string]string{"teamId": team, "uri": u.String(), "mtype": mtype}, &r); err != nil {
			return nil, webhook{}, err
		}
		d.log().Info("created webhook", "team_id", team, "host", u.Host)
		return nil, webhook{ID: str(r, "id"), URL: u.String(), MType: mtype, IsActive: true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_webhook",
		Description: "Delete a webhook. Needs confirm: true.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in idConfirmIn) (*mcp.CallToolResult, idConfirmIn, error) {
		d := d.pick(ctx)
		if err := checkID("id", in.ID); err != nil {
			return nil, idConfirmIn{}, err
		}
		if err := checkConfirm(in.Confirm, "a webhook"); err != nil {
			return nil, idConfirmIn{}, err
		}
		if err := d.Client.Call(ctx, "delete-webhook", map[string]string{"id": in.ID}, nil); err != nil {
			return nil, idConfirmIn{}, err
		}
		d.log().Info("deleted webhook", "id", in.ID)
		return nil, in, nil
	})
}

func createShare(ctx context.Context, d Deps, in shareIn) (shareOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return shareOut{}, err
	}
	who := func(label, v string) (string, error) {
		switch v {
		case "", "team":
			return "team", nil
		case "all":
			return "all", nil
		}
		return "", fmt.Errorf("%s must be team or all", label)
	}
	comment, err := who("who_comment", in.WhoComment)
	if err != nil {
		return shareOut{}, err
	}
	inspect, err := who("who_inspect", in.WhoInspect)
	if err != nil {
		return shareOut{}, err
	}
	for _, id := range in.PageIDs {
		if err := checkID("page_ids", id); err != nil {
			return shareOut{}, err
		}
	}
	if !in.Confirm {
		return shareOut{}, fmt.Errorf("anyone with a share link can view the file: needs confirm: true; ask the user first")
	}
	_, pages, err := d.Writer.Pages(ctx, in.FileID)
	if err != nil {
		return shareOut{}, err
	}
	ids := in.PageIDs
	if len(ids) == 0 {
		for _, p := range pages {
			ids = append(ids, p.ID)
		}
	}
	for _, id := range ids {
		if !pageListed(pages, id) {
			return shareOut{}, fmt.Errorf("page %s is not in the file", id)
		}
	}
	var r map[string]any
	if err := d.Client.Call(ctx, "create-share-link", map[string]any{
		"fileId": in.FileID, "whoComment": comment, "whoInspect": inspect, "pages": ids,
	}, &r); err != nil {
		return shareOut{}, err
	}
	id := str(r, "id")
	q := url.Values{"file-id": {in.FileID}, "page-id": {ids[0]}, "share-id": {id}, "section": {"interactions"}, "index": {"0"}}
	link := d.Client.BaseURL() + "/#/view?" + q.Encode()
	d.log().Info("created share link", "file_id", in.FileID, "id", id)
	return shareOut{ID: id, FileID: in.FileID, URL: link, Pages: ids}, nil
}
