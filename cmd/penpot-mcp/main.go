// Command penpot-mcp is an MCP server that drives Penpot through its backend
// HTTP API. It speaks MCP over stdio.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
	"github.com/izam-mohammed/penpot-mcp/internal/tools"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const instructions = "Drives Penpot through its backend API: no browser tab needed. " +
	"Coordinates are absolute page pixels, also for shapes inside boards. " +
	"Start with list_projects, list_files and get_page_tree to find ids."

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "penpot-mcp:", err)
		os.Exit(1)
	}
}

func run() error {
	// stdout carries MCP; logs go to stderr.
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	client, err := penpot.NewClient(os.Getenv("PENPOT_BASE_URL"), os.Getenv("PENPOT_ACCESS_TOKEN"), version, logger)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var p penpot.Profile
	if err := client.Call(checkCtx, "get-profile", struct{}{}, &p); err != nil {
		return fmt.Errorf("checking the token: %w", err)
	}
	if p.ID == "" || p.ID == penpot.ZeroID {
		return errors.New("PENPOT_ACCESS_TOKEN was not accepted")
	}
	logger.Info("connected", "profile", p.Fullname)

	server := mcp.NewServer(&mcp.Implementation{Name: "penpot-mcp", Version: version},
		&mcp.ServerOptions{Instructions: instructions})
	tools.Register(server, tools.Deps{Client: client, Writer: penpot.NewWriter(client)})
	return server.Run(ctx, &mcp.StdioTransport{})
}
