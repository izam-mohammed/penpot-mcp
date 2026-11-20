package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// docsDir holds one generated page per tool group, for the book in
// docs/book (plans/14-docs.md).
const docsDir = "../../docs/book/src/reference/tools"

type docGroup struct {
	file, title, intro string
	tools              []string
}

// docGroups sorts every tool into the page a person looks for it on. A tool
// that is on no list fails the test, so a new tool must pick a page.
var docGroups = []docGroup{
	{"files", "Files, projects and pages", "Finding what to work on, and managing teams, projects, files and pages. Deletes ask for `confirm: true` first.", []string{
		"whoami", "list_teams", "list_members", "list_projects", "list_files", "search_files", "list_pages", "get_page_tree",
		"create_project", "rename_project", "move_project", "pin_project", "delete_project",
		"create_file", "rename_file", "duplicate_file", "move_files", "delete_file", "restore_file",
		"add_page", "rename_page", "reorder_pages", "delete_page", "plugin_data",
	}},
	{"drawing", "Drawing and editing", "Whole screens in one `draw` call, single shapes, edits, and the tab-only helpers for what a drawing looks like.", []string{
		"draw", "create_board", "create_rect", "create_ellipse", "create_text", "create_path", "create_image", "create_group", "create_bool",
		"set_layout", "update_shape", "move_shape", "reparent_shape", "delete_shapes",
		"reflow", "viewport", "get_selection", "current_file", "import_svg", "generate_markup", "generate_style", "generate_font_faces",
	}},
	{"library", "Library, tokens and components", "Colours, type styles, components, design tokens and shared libraries.", []string{
		"list_library", "add_color", "update_color", "delete_color", "add_typography", "update_typography", "delete_typography", "import_palette",
		"sync_tokens", "apply_token",
		"create_component", "instantiate_component", "delete_component", "set_file_shared", "link_library", "unlink_library",
		"detach", "reset_overrides", "swap_component", "switch_variant", "combine_as_variants",
	}},
	{"export", "Export and import", "Boards and shapes to image files, whole files to `.penpot` and back.", []string{
		"export_shape", "export_boards", "export_file", "import_file",
	}},
	{"collaboration", "Comments, versions and sharing", "Comment threads, saved versions, view links and webhooks. Share links, webhooks and restores need `confirm: true`.", []string{
		"list_comments", "add_comment", "reply_comment", "resolve_thread",
		"create_snapshot", "list_snapshots", "restore_snapshot",
		"create_share_link", "delete_share_link", "create_webhook", "list_webhooks", "delete_webhook", "active_users",
	}},
	{"advanced", "Last resorts and documentation", "For what no other tool does, and the way to look up Penpot's own API.", []string{
		"apply_changes", "execute_code", "api_info",
	}},
}

const docsHeader = "<!-- Generated from the server by `go test ./internal/tools -run TestDocsListEveryTool -update`. Do not edit. -->\n\n"

func cell(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return strings.ReplaceAll(s, "|", "\\|")
}

// schemaType names a JSON Schema node's type in a few words.
func schemaType(n map[string]any) string {
	var t string
	switch v := n["type"].(type) {
	case string:
		t = v
	case []any:
		var parts []string
		for _, p := range v {
			if s, ok := p.(string); ok && s != "null" {
				parts = append(parts, s)
			}
		}
		t = strings.Join(parts, " or ")
	}
	if t == "array" {
		if items, ok := n["items"].(map[string]any); ok {
			if it := schemaType(items); it != "" {
				t = "array of " + it
			}
		}
	}
	if enum, ok := n["enum"].([]any); ok {
		var vals []string
		for _, e := range enum {
			vals = append(vals, fmt.Sprintf("`%v`", e))
		}
		t += ": " + strings.Join(vals, ", ")
	}
	return t
}

func paramTable(t *testing.T, tool *mcp.Tool) string {
	t.Helper()
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
		Required   []string                  `json:"required"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Properties) == 0 {
		return "Takes no parameters.\n"
	}
	req := map[string]bool{}
	for _, r := range schema.Required {
		req[r] = true
	}
	names := make([]string, 0, len(schema.Properties))
	for n := range schema.Properties {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if req[names[i]] != req[names[j]] {
			return req[names[i]]
		}
		return names[i] < names[j]
	})
	var b strings.Builder
	b.WriteString("| Parameter | Type | Required | Description |\n| --- | --- | --- | --- |\n")
	for _, n := range names {
		p := schema.Properties[n]
		desc, _ := p["description"].(string)
		required := "no"
		if req[n] {
			required = "yes"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", n, cell(schemaType(p)), required, cell(desc))
	}
	return b.String()
}

var routeText = map[string]string{
	"API":          "API: needs only the access token.",
	"login or tab": "login or tab: needs `PENPOT_EMAIL` and `PENPOT_PASSWORD`, or a connected tab with the shape's page open.",
	"hybrid":       "hybrid: runs on the API and adds a step in the tab when one is connected.",
	"tab":          "tab: needs `PENPOT_MCP_URL` and a connected Penpot tab with the file open.",
}

// renderDocs returns every generated page by file name.
func renderDocs(t *testing.T) map[string]string {
	t.Helper()
	all, routes := registeredTools(t)
	byName := map[string]*mcp.Tool{}
	for _, tool := range all {
		byName[tool.Name] = tool
	}
	placed := map[string]bool{}
	pages := map[string]string{}
	for _, g := range docGroups {
		var b strings.Builder
		b.WriteString(docsHeader)
		fmt.Fprintf(&b, "# %s\n\n%s\n\n", g.title, g.intro)
		names := append([]string(nil), g.tools...)
		sort.Strings(names)
		for _, n := range names {
			tool, ok := byName[n]
			if !ok {
				t.Errorf("docGroups lists %q, which the server does not register", n)
				continue
			}
			if placed[n] {
				t.Errorf("docGroups lists %q twice", n)
			}
			placed[n] = true
			fmt.Fprintf(&b, "## `%s`\n\nRoute: %s\n\n%s\n\n%s\n", n, routeText[routes[n]], strings.TrimSpace(tool.Description), paramTable(t, tool))
		}
		pages[g.file+".md"] = b.String()
	}
	for _, tool := range all {
		if !placed[tool.Name] {
			t.Errorf("tool %q is on no docs page; add it to docGroups in internal/tools/docs_test.go", tool.Name)
		}
	}
	return pages
}

// TestDocsListEveryTool keeps the book's tool pages in step with the
// server: a tool added, renamed or reworded without regenerating fails.
func TestDocsListEveryTool(t *testing.T) {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "PENPOT_") {
			t.Setenv(name, "")
		}
	}
	got := renderDocs(t)
	if *update {
		if err := os.MkdirAll(docsDir, 0o750); err != nil {
			t.Fatal(err)
		}
		for name, body := range got {
			if err := os.WriteFile(filepath.Join(docsDir, name), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	const hint = "run go test ./internal/tools -run TestDocsListEveryTool -update"
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("%v; %s", err, hint)
	}
	for _, e := range entries {
		if _, ok := got[e.Name()]; !ok {
			t.Errorf("%s/%s is not a generated page; delete it or %s", docsDir, e.Name(), hint)
		}
	}
	for name, body := range got {
		want, err := os.ReadFile(filepath.Join(docsDir, name))
		if err != nil {
			t.Errorf("%v; %s", err, hint)
			continue
		}
		if string(want) != body {
			t.Errorf("%s/%s is out of date; %s", docsDir, name, hint)
		}
	}
}
