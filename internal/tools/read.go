package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

type empty struct{}

type whoamiOut struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	DefaultTeamID    string `json:"default_team_id"`
	DefaultProjectID string `json:"default_project_id"`
}

type teamsOut struct {
	Teams []penpot.Team `json:"teams"`
}

type projectsIn struct {
	TeamID string `json:"team_id,omitempty" jsonschema:"only projects of this team; all teams when empty"`
}

type projectsOut struct {
	Projects []penpot.Project `json:"projects"`
}

type filesIn struct {
	ProjectID string `json:"project_id" jsonschema:"the project to list files of"`
}

type filesOut struct {
	Files []penpot.FileSummary `json:"files"`
}

type pagesIn struct {
	FileID string `json:"file_id" jsonschema:"the file to list pages of"`
}

type pagesOut struct {
	FileName string            `json:"file_name"`
	Pages    []penpot.PageMeta `json:"pages"`
}

type treeIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	PageID string `json:"page_id,omitempty" jsonschema:"the page; the first page when empty"`
	Depth  int    `json:"depth,omitempty" jsonschema:"levels below the page root to include, 1 to 10, default 3"`
}

// Node is one shape in the page tree, listed flat in drawing order (top-most
// first, each parent before its children). The schema of a nested type would
// be recursive, which tool output schemas cannot hold.
type Node struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parent_id"`
	Depth    int      `json:"depth"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
	Width    *float64 `json:"width,omitempty"`
	Height   *float64 `json:"height,omitempty"`
	More     int      `json:"more_children,omitempty"`
}

type treeOut struct {
	FileName string `json:"file_name"`
	PageID   string `json:"page_id"`
	PageName string `json:"page_name"`
	Shapes   []Node `json:"shapes"`
}

func registerRead(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "whoami",
		Description: "The Penpot account this server acts as: id, name, email, default team and project.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, whoamiOut, error) {
		d := d.pick(ctx)
		var p penpot.Profile
		if err := d.Client.Call(ctx, "get-profile", empty{}, &p); err != nil {
			return nil, whoamiOut{}, err
		}
		return nil, whoamiOut{p.ID, p.Fullname, p.Email, p.DefaultTeamID, p.DefaultProjectID}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_teams",
		Description: "List the Penpot teams this account belongs to.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ empty) (*mcp.CallToolResult, teamsOut, error) {
		d := d.pick(ctx)
		var t []penpot.Team
		err := d.Client.Call(ctx, "get-teams", empty{}, &t)
		return nil, teamsOut{Teams: t}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_projects",
		Description: "List Penpot projects, across all teams or for one team.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in projectsIn) (*mcp.CallToolResult, projectsOut, error) {
		d := d.pick(ctx)
		var p []penpot.Project
		var err error
		if in.TeamID == "" {
			err = d.Client.Call(ctx, "get-all-projects", empty{}, &p)
		} else {
			if err = checkID("team_id", in.TeamID); err != nil {
				return nil, projectsOut{}, err
			}
			err = d.Client.Call(ctx, "get-projects", map[string]string{"teamId": in.TeamID}, &p)
		}
		return nil, projectsOut{Projects: p}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_files",
		Description: "List the files in a Penpot project.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in filesIn) (*mcp.CallToolResult, filesOut, error) {
		d := d.pick(ctx)
		if err := checkID("project_id", in.ProjectID); err != nil {
			return nil, filesOut{}, err
		}
		var f []penpot.FileSummary
		err := d.Client.Call(ctx, "get-project-files", map[string]string{"projectId": in.ProjectID}, &f)
		return nil, filesOut{Files: f}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_pages",
		Description: "List the pages of a Penpot file, in order.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in pagesIn) (*mcp.CallToolResult, pagesOut, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, pagesOut{}, err
		}
		name, pages, err := d.Writer.Pages(ctx, in.FileID)
		return nil, pagesOut{FileName: name, Pages: pages}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "get_page_tree",
		Description: "A page's shapes as a flat list in drawing order (top-most first, parents before children): " +
			"id, parent_id, depth, name, type, x, y, width, height. Coordinates are absolute page pixels.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in treeIn) (*mcp.CallToolResult, treeOut, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, treeOut{}, err
		}
		if in.PageID != "" {
			if err := checkID("page_id", in.PageID); err != nil {
				return nil, treeOut{}, err
			}
		}
		depth := in.Depth
		if depth == 0 {
			depth = 3
		}
		if depth < 1 || depth > 10 {
			return nil, treeOut{}, fmt.Errorf("depth must be 1 to 10")
		}
		t, err := d.Writer.Resolve(ctx, in.FileID, in.PageID)
		if err != nil {
			return nil, treeOut{}, err
		}
		var p penpot.Page
		if err := d.Client.Call(ctx, "get-page", map[string]string{"fileId": t.FileID, "pageId": t.PageID}, &p); err != nil {
			return nil, treeOut{}, err
		}
		root := p.Objects[penpot.ZeroID]
		return nil, treeOut{
			FileName: t.FileName, PageID: t.PageID, PageName: t.PageName,
			Shapes: flatten(p.Objects, penpot.ZeroID, root.Shapes, 1, depth, nil),
		}, nil
	})
}

// maxNodes caps the reply size.
const maxNodes = 2000

func flatten(objs map[string]penpot.Object, parent string, ids []string, level, depth int, out []Node) []Node {
	// Penpot lists children bottom to top; show top-most first.
	for i := len(ids) - 1; i >= 0 && len(out) < maxNodes; i-- {
		o, ok := objs[ids[i]]
		if !ok {
			continue
		}
		n := Node{
			ID: o.ID, ParentID: parent, Depth: level, Name: o.Name, Type: o.Type,
			X: o.X, Y: o.Y, Width: o.Width, Height: o.Height,
		}
		if level == depth {
			n.More = len(o.Shapes)
		}
		out = append(out, n)
		if level < depth {
			out = flatten(objs, o.ID, o.Shapes, level+1, depth, out)
		}
	}
	return out
}
