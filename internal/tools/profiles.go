package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type profileKey struct{}

// pick returns the Deps of the profile the call named, or d itself.
func (d Deps) pick(ctx context.Context) Deps {
	name, _ := ctx.Value(profileKey{}).(string)
	if p, ok := d.Profiles[name]; ok && name != "" {
		return p
	}
	return d
}

func (d Deps) anyRoute() bool {
	if d.Route != nil {
		return true
	}
	for _, p := range d.Profiles {
		if p.Route != nil {
			return true
		}
	}
	return false
}

// profileMiddleware takes the "profile" argument out of every tool call and
// puts it in the context for pick, and adds that argument to every tool's
// input schema in tools/list. It is installed only when profiles are
// configured.
func profileMiddleware(names []string) mcp.Middleware {
	known := map[string]bool{}
	for _, n := range names {
		known[n] = true
	}
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			switch method {
			case "tools/call":
				call, ok := req.(*mcp.CallToolRequest)
				if !ok || call.Params == nil || len(call.Params.Arguments) == 0 {
					break
				}
				var args map[string]json.RawMessage
				if json.Unmarshal(call.Params.Arguments, &args) != nil {
					break
				}
				raw, ok := args["profile"]
				if !ok {
					break
				}
				var name string
				if err := json.Unmarshal(raw, &name); err != nil || !known[name] {
					return nil, fmt.Errorf("profile must be one of %v", names)
				}
				delete(args, "profile")
				data, err := json.Marshal(args)
				if err != nil {
					return nil, err
				}
				call.Params.Arguments = data
				ctx = context.WithValue(ctx, profileKey{}, name)
			case "tools/list":
				res, err := next(ctx, method, req)
				if list, ok := res.(*mcp.ListToolsResult); ok && err == nil {
					list.Tools = withProfileArg(list.Tools, names)
				}
				return res, err
			}
			return next(ctx, method, req)
		}
	}
}

// withProfileArg returns copies of tools whose input schema also takes
// "profile".
func withProfileArg(tools []*mcp.Tool, names []string) []*mcp.Tool {
	out := make([]*mcp.Tool, 0, len(tools))
	for _, t := range tools {
		var schema map[string]any
		data, err := json.Marshal(t.InputSchema)
		if err != nil || json.Unmarshal(data, &schema) != nil {
			out = append(out, t)
			continue
		}
		props, _ := schema["properties"].(map[string]any)
		if props == nil {
			props = map[string]any{}
		}
		props["profile"] = map[string]any{
			"type": "string", "enum": names,
			"description": "which Penpot account to act as; the default one when empty",
		}
		schema["properties"] = props
		c := *t
		c.InputSchema = schema
		out = append(out, &c)
	}
	return out
}

func profileNames(p map[string]Deps) []string {
	names := make([]string, 0, len(p))
	for n := range p {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
