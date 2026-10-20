package tools

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite skills/penpot-mcp/references/tools.md from the registered tools")

const toolsDoc = "../../skills/penpot-mcp/references/tools.md"

// hybrid tools do their work on the API and add a step in the tab when one
// is connected (docs/bridge.md).
var hybrid = map[string]bool{"draw": true, "create_text": true, "set_layout": true}

// exporterTools need the exporter's login session.
var exporterTools = map[string]bool{"export_shape": true, "export_boards": true}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}

// renderToolsDoc lists every tool the server registers with a tab
// connected, marking which ones need the tab.
func renderToolsDoc(t *testing.T) string {
	t.Helper()
	plain, _ := session(t)
	apiTools, err := plain.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	onAPI := map[string]bool{}
	for _, tool := range apiTools.Tools {
		onAPI[tool.Name] = true
	}
	tb := &tab{openFile: fileID}
	full, _ := sessionWith(t, withRoute(tb.serve(t)))
	all, err := full.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(all.Tools, func(i, j int) bool { return all.Tools[i].Name < all.Tools[j].Name })
	var b strings.Builder
	b.WriteString("# penpot-mcp tools\n\n")
	b.WriteString("Generated from the server by `go test ./internal/tools -run TestSkillListsEveryTool -update`;\n")
	b.WriteString("`TestSkillListsEveryTool` fails when this file and the server differ.\n\n")
	b.WriteString("Route: **API** needs only the access token; **API + login** also needs\n")
	b.WriteString("`PENPOT_EMAIL` and `PENPOT_PASSWORD`; **hybrid** runs on the API and adds a\n")
	b.WriteString("step in the tab when one is connected; **tab** needs `PENPOT_MCP_URL` and a\n")
	b.WriteString("connected Penpot tab with the file open.\n\n")
	b.WriteString("| Tool | Route | Does |\n| --- | --- | --- |\n")
	for _, tool := range all.Tools {
		route := "API"
		switch {
		case !onAPI[tool.Name]:
			route = "tab"
		case hybrid[tool.Name]:
			route = "hybrid"
		case exporterTools[tool.Name]:
			route = "API + login"
		}
		desc := strings.ReplaceAll(firstSentence(tool.Description), "|", "\\|")
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", tool.Name, route, desc)
	}
	return b.String()
}

// TestSkillListsEveryTool keeps the skill's tool list in step with the
// server: a tool added, renamed or reworded without regenerating fails.
func TestSkillListsEveryTool(t *testing.T) {
	got := renderToolsDoc(t)
	if *update {
		if err := os.WriteFile(toolsDoc, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(toolsDoc)
	if err != nil {
		t.Fatalf("%v; run go test ./internal/tools -run TestSkillListsEveryTool -update", err)
	}
	if string(want) != got {
		t.Errorf("%s is out of date; run go test ./internal/tools -run TestSkillListsEveryTool -update", toolsDoc)
	}
}
