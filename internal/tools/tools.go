// Package tools registers the MCP tools that call Penpot.
package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// Deps is what every tool needs.
type Deps struct {
	Client *penpot.Client
	Writer *penpot.Writer
}

// Register adds every tool to the server.
func Register(s *mcp.Server, d Deps) {
	registerRead(s, d)
	registerDraw(s, d)
}
