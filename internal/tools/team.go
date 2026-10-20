package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type membersIn struct {
	TeamID string `json:"team_id,omitempty" jsonschema:"the team; the account's default team when empty"`
}

// member is one row of get-team-members.
type member struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Fullname string `json:"fullname,omitempty"`
	Email    string `json:"email"`
	IsOwner  bool   `json:"isOwner"`
	IsAdmin  bool   `json:"isAdmin"`
	CanEdit  bool   `json:"canEdit"`
}

type memberOut struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type membersOut struct {
	TeamID  string      `json:"team_id"`
	Members []memberOut `json:"members"`
}

func registerTeam(s *mcp.Server, d Deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_members",
		Description: "List a team's members with their role: owner, admin, editor or viewer.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in membersIn) (*mcp.CallToolResult, membersOut, error) {
		d := d.pick(ctx)
		team, err := teamOrDefault(ctx, d, in.TeamID)
		if err != nil {
			return nil, membersOut{}, err
		}
		var ms []member
		if err := d.Client.Call(ctx, "get-team-members", map[string]string{"teamId": team}, &ms); err != nil {
			return nil, membersOut{}, err
		}
		out := membersOut{TeamID: team, Members: []memberOut{}}
		for _, m := range ms {
			name := m.Name
			if name == "" {
				name = m.Fullname
			}
			out.Members = append(out.Members, memberOut{ID: m.ID, Name: name, Email: m.Email, Role: role(m)})
		}
		return nil, out, nil
	})
}

func role(m member) string {
	switch {
	case m.IsOwner:
		return "owner"
	case m.IsAdmin:
		return "admin"
	case m.CanEdit:
		return "editor"
	}
	return "viewer"
}
