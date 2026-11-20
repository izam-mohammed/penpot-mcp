package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// placeholder is a value of the schema's type, with ids that are valid.
func placeholder(name string, schema map[string]any) any {
	switch schema["type"] {
	case "string":
		if name == "id" || strings.HasSuffix(name, "_id") {
			return fileID
		}
		return "x"
	case "integer", "number":
		return 1
	case "boolean":
		return false
	case "array":
		item, _ := schema["items"].(map[string]any)
		if item["type"] == "string" {
			return []string{rectID}
		}
		return []any{}
	case "object":
		return map[string]any{}
	}
	return "x"
}

// TestEveryIDParameterIsChecked sends each tool a malformed id in turn, with
// the rest of its arguments filled in. Each must be refused before any write
// reaches Penpot.
func TestEveryIDParameterIsChecked(t *testing.T) {
	tb := &tab{openFile: fileID}
	cs, b := sessionWith(t, withRoute(tb.serve(t)))
	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, tool := range list.Tools {
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]map[string]any `json:"properties"`
			Required   []string                  `json:"required"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, bad := range names {
			prop := schema.Properties[bad]
			isIDs := strings.HasSuffix(bad, "_ids") && prop["type"] == "array"
			isID := (bad == "id" || strings.HasSuffix(bad, "_id")) && prop["type"] == "string"
			if !isID && !isIDs {
				continue
			}
			args := map[string]any{}
			for _, req := range schema.Required {
				args[req] = placeholder(req, schema.Properties[req])
			}
			if _, ok := schema.Properties["confirm"]; ok {
				args["confirm"] = true
			}
			if isIDs {
				args[bad] = []string{"not-a-uuid"}
			} else {
				args[bad] = "not-a-uuid"
			}
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool.Name, Arguments: args})
			if err != nil {
				// A schema-level refusal (a missing required field this
				// generator cannot fill) is still a refusal.
				continue
			}
			checked++
			if !res.IsError {
				t.Errorf("%s accepted %s = %q", tool.Name, bad, "not-a-uuid")
			}
		}
	}
	t.Logf("%d id parameters exercised", checked)
	if checked < 40 {
		t.Errorf("only %d id parameters were exercised", checked)
	}
	for _, c := range b.calls {
		if c == "update-file" {
			t.Error("a malformed id reached update-file")
		}
	}
}
