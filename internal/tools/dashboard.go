package tools

import (
	"context"
	"fmt"

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
		d := d.pick(ctx)
		p, err := createProject(ctx, d, in)
		return nil, p, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_file",
		Description: "Create a file in a project. It starts with one page; returns the file's id and its pages.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createFileIn) (*mcp.CallToolResult, fileOut, error) {
		d := d.pick(ctx)
		out, err := createFile(ctx, d, in)
		return nil, out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_file",
		Description: "Rename a file.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in renameFileIn) (*mcp.CallToolResult, fileOut, error) {
		d := d.pick(ctx)
		out, err := renameFile(ctx, d, in)
		return nil, out, err
	})

	registerDashboardRest(s, d)
}

func createProject(ctx context.Context, d Deps, in createProjectIn) (penpot.Project, error) {
	if err := checkName(in.Name); err != nil {
		return penpot.Project{}, err
	}
	team, err := teamOrDefault(ctx, d, in.TeamID)
	if err != nil {
		return penpot.Project{}, err
	}
	// Penpot makes the id: since 2.19 it rejects one sent by the client.
	var p penpot.Project
	err = d.Client.Call(ctx, "create-project", map[string]string{"teamId": team, "name": in.Name}, &p)
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

type projectIn struct {
	ProjectID string `json:"project_id" jsonschema:"the project"`
}

type renameProjectIn struct {
	projectIn
	Name string `json:"name" jsonschema:"new project name"`
}

type moveProjectIn struct {
	projectIn
	TeamID string `json:"team_id" jsonschema:"the team to move the project to"`
}

type pinProjectIn struct {
	projectIn
	Pinned bool `json:"pinned" jsonschema:"true pins the project on the dashboard, false unpins it"`
}

type deleteProjectIn struct {
	projectIn
	Confirm bool `json:"confirm,omitempty" jsonschema:"must be true; say so only after the user asked for this project to be deleted"`
}

type duplicateFileIn struct {
	FileID string `json:"file_id" jsonschema:"the file to copy"`
	Name   string `json:"name,omitempty" jsonschema:"the copy's name; Penpot names it when empty"`
}

type moveFilesIn struct {
	FileIDs   []string `json:"file_ids" jsonschema:"the files to move"`
	ProjectID string   `json:"project_id" jsonschema:"the project to move them into"`
}

type deleteFileIn struct {
	FileID  string `json:"file_id" jsonschema:"the file to delete"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true; say so only after the user asked for this file to be deleted"`
}

type restoreFileIn struct {
	TeamID  string   `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
	FileIDs []string `json:"file_ids,omitempty" jsonschema:"the deleted files to restore; when empty, lists the team's deleted files instead"`
}

type sharedIn struct {
	FileID string `json:"file_id" jsonschema:"the file"`
	Shared bool   `json:"shared" jsonschema:"true publishes the file as a shared library, false unpublishes it"`
}

type libraryIn struct {
	FileID    string `json:"file_id" jsonschema:"the file that uses the library"`
	LibraryID string `json:"library_id" jsonschema:"the shared library file"`
}

type searchIn struct {
	TeamID string `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
	Query  string `json:"query" jsonschema:"text to look for in file names"`
}

type projectOut struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name,omitempty"`
	TeamID    string `json:"team_id,omitempty"`
	Pinned    *bool  `json:"pinned,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

type filesChangedOut struct {
	Files []penpot.FileSummary `json:"files"`
}

// deletedFile is one row of get-team-deleted-files.
type deletedFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName,omitempty"`
	DeletedAt   string `json:"deletedAt,omitempty"`
}

type restoreOut struct {
	Restored []string      `json:"restored,omitempty"`
	Deleted  []deletedFile `json:"deleted,omitempty"`
}

var destructive = &mcp.ToolAnnotations{DestructiveHint: ptr(true)}

func ptr[T any](v T) *T { return &v }

// checkConfirm stops a destructive call before any request is sent.
func checkConfirm(confirm bool, what string) error {
	if !confirm {
		return fmt.Errorf("deleting %s needs confirm: true; ask the user first", what)
	}
	return nil
}

func registerDashboardRest(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "rename_project",
		Description: "Rename a project.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in renameProjectIn) (*mcp.CallToolResult, projectOut, error) {
		d := d.pick(ctx)
		if err := checkID("project_id", in.ProjectID); err != nil {
			return nil, projectOut{}, err
		}
		if err := checkName(in.Name); err != nil {
			return nil, projectOut{}, err
		}
		err := d.Client.Call(ctx, "rename-project", map[string]string{"id": in.ProjectID, "name": in.Name}, nil)
		return nil, projectOut{ProjectID: in.ProjectID, Name: in.Name}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "move_project",
		Description: "Move a project, with its files, to another team.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in moveProjectIn) (*mcp.CallToolResult, projectOut, error) {
		d := d.pick(ctx)
		if err := checkID("project_id", in.ProjectID); err != nil {
			return nil, projectOut{}, err
		}
		if err := checkID("team_id", in.TeamID); err != nil {
			return nil, projectOut{}, err
		}
		err := d.Client.Call(ctx, "move-project", map[string]string{"projectId": in.ProjectID, "teamId": in.TeamID}, nil)
		return nil, projectOut{ProjectID: in.ProjectID, TeamID: in.TeamID}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pin_project",
		Description: "Pin or unpin a project on the dashboard.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in pinProjectIn) (*mcp.CallToolResult, projectOut, error) {
		d := d.pick(ctx)
		p, err := findProject(ctx, d, in.ProjectID)
		if err != nil {
			return nil, projectOut{}, err
		}
		err = d.Client.Call(ctx, "update-project-pin", map[string]any{"id": p.ID, "teamId": p.TeamID, "isPinned": in.Pinned}, nil)
		return nil, projectOut{ProjectID: p.ID, Name: p.Name, TeamID: p.TeamID, Pinned: &in.Pinned}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_project",
		Description: "Delete a project and every file in it. Needs confirm: true, given only after the user asked. " +
			"Penpot keeps deleted files for a while; restore_file brings them back.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteProjectIn) (*mcp.CallToolResult, projectOut, error) {
		d := d.pick(ctx)
		if err := checkID("project_id", in.ProjectID); err != nil {
			return nil, projectOut{}, err
		}
		if err := checkConfirm(in.Confirm, "a project"); err != nil {
			return nil, projectOut{}, err
		}
		p, err := findProject(ctx, d, in.ProjectID)
		if err != nil {
			return nil, projectOut{}, err
		}
		if p.IsDefault {
			return nil, projectOut{}, fmt.Errorf("%q is the team's default project; Penpot does not delete it", p.Name)
		}
		if err := d.Client.Call(ctx, "delete-project", map[string]string{"id": p.ID}, nil); err != nil {
			return nil, projectOut{}, err
		}
		d.log().Info("deleted project", "project_id", p.ID, "name", p.Name, "team_id", p.TeamID)
		return nil, projectOut{ProjectID: p.ID, Name: p.Name, TeamID: p.TeamID, Deleted: true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "duplicate_file",
		Description: "Copy a file within its project. Returns the copy's id, made by Penpot.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in duplicateFileIn) (*mcp.CallToolResult, fileOut, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, fileOut{}, err
		}
		params := map[string]string{"fileId": in.FileID}
		if in.Name != "" {
			if err := checkName(in.Name); err != nil {
				return nil, fileOut{}, err
			}
			params["name"] = in.Name
		}
		var f penpot.FileSummary
		if err := d.Client.Call(ctx, "duplicate-file", params, &f); err != nil {
			return nil, fileOut{}, err
		}
		return nil, fileOut{FileID: f.ID, FileName: f.Name, ProjectID: f.ProjectID}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "move_files",
		Description: "Move files into another project of the same team.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in moveFilesIn) (*mcp.CallToolResult, filesChangedOut, error) {
		d := d.pick(ctx)
		if err := checkIDList("file_ids", in.FileIDs); err != nil {
			return nil, filesChangedOut{}, err
		}
		if err := checkID("project_id", in.ProjectID); err != nil {
			return nil, filesChangedOut{}, err
		}
		if err := d.Client.Call(ctx, "move-files", map[string]any{"ids": in.FileIDs, "projectId": in.ProjectID}, nil); err != nil {
			return nil, filesChangedOut{}, err
		}
		out := filesChangedOut{}
		for _, id := range in.FileIDs {
			out.Files = append(out.Files, penpot.FileSummary{ID: id, ProjectID: in.ProjectID})
		}
		return nil, out, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "delete_file",
		Description: "Delete a file. Needs confirm: true, given only after the user asked. " +
			"Penpot keeps deleted files for a while; restore_file brings it back.",
		Annotations: destructive,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in deleteFileIn) (*mcp.CallToolResult, fileOut, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, fileOut{}, err
		}
		if err := checkConfirm(in.Confirm, "a file"); err != nil {
			return nil, fileOut{}, err
		}
		name, _, err := d.Writer.Pages(ctx, in.FileID)
		if err != nil {
			return nil, fileOut{}, err
		}
		if err := d.Client.Call(ctx, "delete-file", map[string]string{"id": in.FileID}, nil); err != nil {
			return nil, fileOut{}, err
		}
		d.Writer.Forget(in.FileID)
		d.log().Info("deleted file", "file_id", in.FileID, "name", name)
		return nil, fileOut{FileID: in.FileID, FileName: name}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name: "restore_file",
		Description: "Bring back deleted files (and their projects) before Penpot purges them. " +
			"Without file_ids, lists the team's deleted files.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in restoreFileIn) (*mcp.CallToolResult, restoreOut, error) {
		d := d.pick(ctx)
		team, err := teamOrDefault(ctx, d, in.TeamID)
		if err != nil {
			return nil, restoreOut{}, err
		}
		if len(in.FileIDs) == 0 {
			var files []deletedFile
			err := d.Client.Call(ctx, "get-team-deleted-files", map[string]string{"teamId": team}, &files)
			return nil, restoreOut{Deleted: files}, err
		}
		if err := checkIDList("file_ids", in.FileIDs); err != nil {
			return nil, restoreOut{}, err
		}
		err = d.Client.Call(ctx, "restore-deleted-team-files", map[string]any{"teamId": team, "ids": in.FileIDs}, nil)
		return nil, restoreOut{Restored: in.FileIDs}, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_file_shared",
		Description: "Publish a file as a shared library, or unpublish it.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sharedIn) (*mcp.CallToolResult, penpot.FileSummary, error) {
		d := d.pick(ctx)
		if err := checkID("file_id", in.FileID); err != nil {
			return nil, penpot.FileSummary{}, err
		}
		err := d.Client.Call(ctx, "set-file-shared", map[string]any{"id": in.FileID, "isShared": in.Shared}, nil)
		return nil, penpot.FileSummary{ID: in.FileID, IsShared: in.Shared}, err
	})

	for _, l := range []struct{ name, command, desc string }{
		{"link_library", "link-file-to-library", "Link a shared library to a file, so its colours, type styles and components can be used there."},
		{"unlink_library", "unlink-file-from-library", "Unlink a shared library from a file."},
	} {
		mcp.AddTool(s, &mcp.Tool{
			Name:        l.name,
			Description: l.desc,
			Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
		}, func(ctx context.Context, _ *mcp.CallToolRequest, in libraryIn) (*mcp.CallToolResult, libraryIn, error) {
			d := d.pick(ctx)
			if err := checkID("file_id", in.FileID); err != nil {
				return nil, libraryIn{}, err
			}
			if err := checkID("library_id", in.LibraryID); err != nil {
				return nil, libraryIn{}, err
			}
			if in.FileID == in.LibraryID {
				return nil, libraryIn{}, fmt.Errorf("a file cannot use itself as a library")
			}
			err := d.Client.Call(ctx, l.command, map[string]string{"fileId": in.FileID, "libraryId": in.LibraryID}, nil)
			return nil, in, err
		})
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_files",
		Description: "Find files in a team by name.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, filesOut, error) {
		d := d.pick(ctx)
		if err := checkName(in.Query); err != nil {
			return nil, filesOut{}, fmt.Errorf("query: %w", err)
		}
		team, err := teamOrDefault(ctx, d, in.TeamID)
		if err != nil {
			return nil, filesOut{}, err
		}
		var f []penpot.FileSummary
		err = d.Client.Call(ctx, "search-files", map[string]string{"teamId": team, "searchTerm": in.Query}, &f)
		return nil, filesOut{Files: f}, err
	})
}

// teamOrDefault checks teamID, or returns the account's default team.
func teamOrDefault(ctx context.Context, d Deps, teamID string) (string, error) {
	if teamID == "" {
		var p penpot.Profile
		if err := d.Client.Call(ctx, "get-profile", empty{}, &p); err != nil {
			return "", err
		}
		teamID = p.DefaultTeamID
	}
	return teamID, checkID("team_id", teamID)
}

// findProject looks a project up by id, for its team and name.
func findProject(ctx context.Context, d Deps, id string) (penpot.Project, error) {
	if err := checkID("project_id", id); err != nil {
		return penpot.Project{}, err
	}
	var all []penpot.Project
	if err := d.Client.Call(ctx, "get-all-projects", empty{}, &all); err != nil {
		return penpot.Project{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return penpot.Project{}, fmt.Errorf("project %s is not in any of this account's teams", id)
}

// checkIDList checks a list of 1 to maxEditIDs distinct ids.
func checkIDList(label string, ids []string) error {
	if len(ids) == 0 || len(ids) > maxEditIDs {
		return fmt.Errorf("%s must hold 1 to %d ids", label, maxEditIDs)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if err := checkID(label, id); err != nil {
			return err
		}
		if seen[id] {
			return fmt.Errorf("%s lists %s twice", label, id)
		}
		seen[id] = true
	}
	return nil
}
