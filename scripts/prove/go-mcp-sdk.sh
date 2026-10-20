#!/bin/sh
# Prove the Go MCP SDK works here: build a throwaway one-tool server against
# go-sdk v1.8.0, send initialize and tools/list over stdio, and expect the tool
# in the reply, per docs/external/go-mcp-sdk.md.
set -eu

command -v go >/dev/null 2>&1 || { echo "FAIL: go is not installed" >&2; exit 2; }

dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

cat >"$dir/main.go" <<'EOF'
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type in struct {
	Name string `json:"name"`
}

func main() {
	s := mcp.NewServer(&mcp.Implementation{Name: "prove", Version: "0.0.0"}, nil)
	mcp.AddTool(s, &mcp.Tool{Name: "prove_echo", Description: "echo"},
		func(_ context.Context, _ *mcp.CallToolRequest, a in) (*mcp.CallToolResult, any, error) {
			return nil, map[string]string{"hello": a.Name}, nil
		})
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
EOF

(
	cd "$dir"
	go mod init prove >/dev/null 2>&1
	go get github.com/modelcontextprotocol/go-sdk@v1.8.0 >/dev/null 2>&1
	go mod tidy >/dev/null 2>&1
	go build -o prove .
) || { echo "FAIL: could not build against go-sdk v1.8.0"; exit 1; }

# Keep stdin open briefly: the server stops on EOF before it answers.
reply=$({
	printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"prove","version":"0"}}}'
	sleep 1
	printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}'
	printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
	sleep 1
} | "$dir/prove" 2>"$dir/stderr" || true)

if printf '%s' "$reply" | grep -q '"prove_echo"'; then
	echo "PASS: go-sdk v1.8.0 server listed its tool over stdio"
else
	echo "FAIL: tool not listed: $(printf '%s' "$reply" | head -c 200) $(head -c 200 "$dir/stderr")"
	exit 1
fi
