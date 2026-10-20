// Package tools registers the MCP tools that call Penpot.
package tools

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// Deps is what every tool needs.
type Deps struct {
	Client *penpot.Client
	Writer *penpot.Writer
	Fonts  *fonts.Measurer
	Log    *slog.Logger // destructive calls are logged here; nil discards
}

func (d Deps) log() *slog.Logger {
	if d.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return d.Log
}

// Register adds every tool to the server.
func Register(s *mcp.Server, d Deps) {
	registerRead(s, d)
	registerDraw(s, d)
	registerLayout(s, d)
	registerEdit(s, d)
	registerDashboard(s, d)
	registerPages(s, d)
	registerTeam(s, d)
}
