# Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`)

Fetched: 19/10/2026 from https://github.com/modelcontextprotocol/go-sdk
(tag `v1.8.0`, source files `mcp/server.go`, `mcp/tool.go`,
`mcp/transport.go`, `mcp/streamable.go`, `go.mod`) and
https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp.
Version: v1.8.0, released 2026-09-14. Requires `go 1.25.0` (from its
`go.mod`). Earlier releases: v1.7.0 added the 2026-07-28 protocol revision,
with backward compatibility for 2025-11-25 and earlier kept.

## Authentication

None in the SDK for this project's use. The server runs over stdio for a
local client. The MCP-route client passes the official Penpot MCP key in the
endpoint URL (`PENPOT_MCP_URL`), never logged.

## What this project calls

### Server

```go
func NewServer(impl *Implementation, options *ServerOptions) *Server
// ServerOptions.Instructions: optional instructions for connected clients.

func AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out])
type ToolHandlerFor[In, Out any] func(_ context.Context, request *CallToolRequest,
    input In) (result *CallToolResult, output Out, _ error)
```

- `AddTool` infers the input JSON schema from `In` and panics on an invalid
  tool definition at start-up.
- With `ToolHandlerFor`, a returned error becomes a tool result with
  `IsError` set (the model sees it), not a protocol error.

Usage from the package overview:

```go
server := mcp.NewServer(&mcp.Implementation{Name: "greeter"}, nil)
err := server.Run(context.Background(), &mcp.StdioTransport{})
```

`StdioTransport.MaxLineLength` bounds one inbound JSON-RPC frame (0 picks
`DefaultMaxLineLength`).

### Streamable HTTP server (later plan)

```go
func NewStreamableHTTPHandler(getServer func(*http.Request) *Server,
    opts *StreamableHTTPOptions) *StreamableHTTPHandler
```

`StreamableHTTPOptions.Stateless`: no `Mcp-Session-Id`; the 2026-07-28
protocol is served over HTTP only in stateless mode. `MaxRequestBodyBytes`
defaults to `DefaultMaxRequestBodyBytes`.

### Client (for the MCP route)

```go
func NewClient(impl *Implementation, options *ClientOptions) *Client
func (c *Client) Connect(ctx context.Context, t Transport,
    opts *ClientSessionOptions) (cs *ClientSession, err error)
func (cs *ClientSession) CallTool(ctx context.Context,
    params *CallToolParams) (*CallToolResult, error)
func (cs *ClientSession) ListTools(ctx context.Context,
    params *ListToolsParams) (*ListToolsResult, error)

type StreamableClientTransport struct {
    Endpoint             string
    HTTPClient           *http.Client // custom headers via a RoundTripper
    MaxRetries           int          // default 5, negative disables
    DisableStandaloneSSE bool
}
```

The official Penpot MCP answers `403` without a `User-Agent`
(`../existing-mcp-reference.md`), so the `HTTPClient` carries a
RoundTripper that sets one.

## Errors

- A handler error from `ToolHandlerFor` is returned to the model as
  `IsError: true` content.
- The official Penpot MCP returns its failures as text results starting
  `Tool execution failed:` rather than JSON-RPC errors; the client must match
  on the text (`../existing-mcp-reference.md`).

## Limits

- Request body cap on the HTTP handler: `DefaultMaxRequestBodyBytes` (value
  not stated in the fetched docs).
- Stdio frame cap: `DefaultMaxLineLength` (value not stated).

## Gotchas

- The SDK's dependencies (from `go.mod`): `golang-jwt/jwt/v5`,
  `google/jsonschema-go`, `segmentio/encoding`, `yosida95/uritemplate/v3`,
  `golang.org/x/oauth2`, `golang.org/x/time`, `golang.org/x/tools`. All are
  pulled into this binary; `govulncheck` covers them.
- Roots, sampling and logging are deprecated as of protocol 2026-07-28; do
  not build on them.
