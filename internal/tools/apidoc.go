package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// methodRE matches a backend RPC method name, such as create-project.
var methodRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)+$`)

// maxDocText caps one api_info answer; a larger schema is shown one level
// deep, with member to open a part.
const maxDocText = 12000

type openAPI struct {
	Paths map[string]struct {
		Post struct {
			Description *string `json:"description"`
			Deprecated  bool    `json:"deprecated"`
			RequestBody struct {
				Content map[string]struct {
					Schema json.RawMessage `json:"schema"`
				} `json:"content"`
			} `json:"requestBody"`
			Responses map[string]struct {
				Content map[string]struct {
					Schema json.RawMessage `json:"schema"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"post"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

// backendInfo answers api_info from the backend's OpenAPI document: one
// method's parameters and reply, or one parameter (member) of it.
func backendInfo(ctx context.Context, d Deps, method, member string) (string, error) {
	raw, err := d.Client.APIDoc(ctx)
	if err != nil {
		return "", err
	}
	var doc openAPI
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("reading Penpot's API documentation: %w", err)
	}
	m, ok := doc.Paths[method]
	if !ok {
		var near []string
		for name := range doc.Paths {
			for _, part := range strings.Split(method, "-") {
				if len(part) > 2 && strings.Contains(name, part) {
					near = append(near, name)
					break
				}
			}
		}
		sort.Strings(near)
		if len(near) > 40 {
			near = near[:40]
		}
		if len(near) == 0 {
			return "", fmt.Errorf("the backend has no method %q", method)
		}
		return "", fmt.Errorf("the backend has no method %q; close names: %s", method, strings.Join(near, ", "))
	}
	params := m.Post.RequestBody.Content["application/json"].Schema
	var reply json.RawMessage
	if r, ok := m.Post.Responses["default"]; ok {
		reply = r.Content["application/json"].Schema
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (POST /api/main/methods/%s)\n", method, method)
	if m.Post.Deprecated {
		b.WriteString("Deprecated.\n")
	}
	if m.Post.Description != nil && *m.Post.Description != "" {
		b.WriteString("\n" + strings.TrimSpace(*m.Post.Description) + "\n")
	}
	if member != "" {
		for _, part := range []struct {
			label  string
			schema json.RawMessage
		}{{"Parameter", params}, {"Reply field", reply}} {
			if prop, ok := property(part.schema, member); ok {
				fmt.Fprintf(&b, "\n%s %s:\n%s\n", part.label, member, render(prop, doc.Components.Schemas))
				return capText(b.String()), nil
			}
		}
		return "", fmt.Errorf("%s has no parameter or reply field %q", method, member)
	}
	fmt.Fprintf(&b, "\nParameters:\n%s\n", render(params, doc.Components.Schemas))
	if len(reply) > 0 {
		fmt.Fprintf(&b, "\nReply:\n%s\n", render(reply, doc.Components.Schemas))
	}
	return capText(b.String()), nil
}

// property returns one property of an object schema.
func property(schema json.RawMessage, name string) (json.RawMessage, bool) {
	var s struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(schema, &s) != nil {
		return nil, false
	}
	p, ok := s.Properties[name]
	return p, ok
}

// render prints a schema as indented JSON, with the shared schemas it
// refers to appended. Too large, it prints each property's type instead.
func render(schema json.RawMessage, shared map[string]json.RawMessage) string {
	if len(schema) == 0 {
		return "(none)"
	}
	var v any
	if json.Unmarshal(schema, &v) != nil {
		return string(schema)
	}
	out, _ := json.MarshalIndent(v, "", "  ")
	text := string(out)
	refs := map[string]bool{}
	for _, m := range schemaRefRE.FindAllStringSubmatch(text, -1) {
		refs[m[1]] = true
	}
	names := make([]string, 0, len(refs))
	for n := range refs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if s, ok := shared[n]; ok {
			text += "\n" + n + ": " + string(s)
		}
	}
	if len(text) <= maxDocText {
		return text
	}
	return outline(v)
}

var schemaRefRE = regexp.MustCompile(`"#/components/schemas/([A-Za-z0-9_]+)"`)

// outline lists an object schema's properties one level deep.
func outline(v any) string {
	m, _ := v.(map[string]any)
	props, _ := m["properties"].(map[string]any)
	if len(props) == 0 {
		return "(too large to show; it is not an object with properties)"
	}
	required := map[string]bool{}
	if r, ok := m["required"].([]any); ok {
		for _, x := range r {
			required[fmt.Sprint(x)] = true
		}
	}
	names := make([]string, 0, len(props))
	for n := range props {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("(too large to show whole: give member for one of these)\n")
	for _, n := range names {
		p, _ := props[n].(map[string]any)
		typ := fmt.Sprint(p["type"])
		if ref, ok := p["$ref"].(string); ok {
			typ = strings.TrimPrefix(ref, "#/components/schemas/")
		} else if p["type"] == nil {
			typ = "(composite)"
		}
		req := ""
		if required[n] {
			req = ", required"
		}
		fmt.Fprintf(&b, "- %s: %s%s\n", n, typ, req)
	}
	return b.String()
}

func capText(s string) string {
	if len(s) > maxDocText {
		return s[:maxDocText] + "\n... (cut; give member for one part)"
	}
	return s
}
