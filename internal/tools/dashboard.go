package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

type createProjectIn struct {
	TeamID string `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
	Name   string `json:"name" jsonschema:"project name"`
}

type createFileIn struct {
	ProjectID string `json:"project_id" jsonschema:"the project to create the file in"`
	Name      string `json:"name" jsonschema:"file name"`
}

type renameFileIn struct {
	FileID string `json:"file_id" jsonschema:"the file to rename"`
	Name   string `json:"name" jsonschema:"new file name"`
}

type fileOut struct {
	FileID    string            `json:"file_id"`
	FileName  string            `json:"file_name"`
	ProjectID string            `json:"project_id,omitempty"`
	Pages     []penpot.PageMeta `json:"pages,omitempty"`
}

func registerDashboard(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a project in a team. Returns the new project's id, made by Penpot.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createProjectIn) (*mcp.CallToolResult, penpot.Project, error) {
		p, err := createProject(ctx, d, in)
		return nil, p, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_file",
		Description: "Create a file in a project. It starts with one page; returns the file's id and its pages.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createFileIn) (*mcp.CallToolResult, fileOut, error) {
		out, err := createFile(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_file",
		Description: "Rename a file.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in renameFileIn) (*mcp.CallToolResult, fileOut, error) {
		out, err := renameFile(ctx, d, in)
		return nil, out, err
	})
}

func createProject(ctx context.Context, d Deps, in createProjectIn) (penpot.Project, error) {
	if err := checkName(in.Name); err != nil {
		return penpot.Project{}, err
	}
	team := in.TeamID
	if team == "" {
		var p penpot.Profile
		if err := d.Client.Call(ctx, "get-profile", empty{}, &p); err != nil {
			return penpot.Project{}, err
		}
		team = p.DefaultTeamID
	}
	if err := checkID("team_id", team); err != nil {
		return penpot.Project{}, err
	}
	// Penpot makes the id: since 2.19 it rejects one sent by the client.
	var p penpot.Project
	err := d.Client.Call(ctx, "create-project", map[string]string{"teamId": team, "name": in.Name}, &p)
	return p, err
}

func createFile(ctx context.Context, d Deps, in createFileIn) (fileOut, error) {
	if err := checkID("project_id", in.ProjectID); err != nil {
		return fileOut{}, err
	}
	if err := checkName(in.Name); err != nil {
		return fileOut{}, err
	}
	var f penpot.FileSummary
	if err := d.Client.Call(ctx, "create-file", map[string]string{"projectId": in.ProjectID, "name": in.Name}, &f); err != nil {
		return fileOut{}, err
	}
	name, pages, err := d.Writer.Pages(ctx, f.ID)
	if err != nil {
		return fileOut{}, err
	}
	return fileOut{FileID: f.ID, FileName: name, ProjectID: f.ProjectID, Pages: pages}, nil
}

func renameFile(ctx context.Context, d Deps, in renameFileIn) (fileOut, error) {
	if err := checkID("file_id", in.FileID); err != nil {
		return fileOut{}, err
	}
	if err := checkName(in.Name); err != nil {
		return fileOut{}, err
	}
	var f penpot.FileSummary
	if err := d.Client.Call(ctx, "rename-file", map[string]string{"id": in.FileID, "name": in.Name}, &f); err != nil {
		return fileOut{}, err
	}
	d.Writer.Forget(in.FileID)
	return fileOut{FileID: in.FileID, FileName: f.Name}, nil
}
