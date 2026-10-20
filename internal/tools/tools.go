// Package tools registers the MCP tools that call Penpot.
package tools

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/fonts"
	"github.com/izam-mohammed/penpot-mcp/internal/mcproute"
	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// Deps is what every tool needs.
type Deps struct {
	Client *penpot.Client
	Writer *penpot.Writer
	Fonts  *fonts.Measurer
	Log    *slog.Logger // destructive calls are logged here; nil discards
	Root   string       // the directory palette, token and export files are read from and written to

	// Exporter renders shapes; nil when PENPOT_EMAIL and PENPOT_PASSWORD
	// are not set.
	Exporter *penpot.Exporter

	// Route is the MCP route to a Penpot tab; nil when PENPOT_MCP_URL is
	// not set, and then its tools are not listed.
	Route *mcproute.Route

	// Profiles are other accounts, chosen per call with a "profile"
	// argument; each has its own client and write queue.
	Profiles map[string]Deps
}

func (d Deps) log() *slog.Logger {
	if d.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return d.Log
}

// Register adds every tool to the server.
func Register(s *mcp.Server, d Deps) {
	if len(d.Profiles) > 0 {
		s.AddReceivingMiddleware(profileMiddleware(profileNames(d.Profiles)))
	}
	registerRead(s, d)
	registerDraw(s, d)
	registerLayout(s, d)
	registerEdit(s, d)
	registerDashboard(s, d)
	registerPages(s, d)
	registerTeam(s, d)
	registerLibrary(s, d)
	registerTokens(s, d)
	registerExport(s, d)
	registerPlugin(s, d)
	registerComments(s, d)
	registerSharing(s, d)
	registerHistory(s, d)
}
