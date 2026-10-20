package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

// openAPIFixture is a cut of Penpot's /api/main/doc/openapi.json: one small
// method and one whose schema is too large to show whole.
var openAPIFixture = func() []byte {
	big := map[string]any{}
	for i := 0; i < 400; i++ {
		big[strings.Repeat("x", 30)+string(rune('a'+i%26))+strings.Repeat("y", i%7)] = map[string]any{"type": "string", "description": strings.Repeat("long ", 20)}
	}
	doc := map[string]any{
		"openapi": "3.0.0",
		"paths": map[string]any{
			"create-project": map[string]any{"post": map[string]any{
				"description": "Create a project in a team.", "deprecated": false,
				"requestBody": map[string]any{"content": map[string]any{"application/json": map[string]any{
					"schema": map[string]any{"type": "object", "required": []string{"teamId", "name"}, "properties": map[string]any{
						"teamId": map[string]any{"$ref": "#/components/schemas/Uuid"},
						"name":   map[string]any{"type": "string", "maxLength": 250},
					}},
					"example": "{\"teamId\":\"never shown\"}",
				}}},
				"responses": map[string]any{"default": map[string]any{"content": map[string]any{"application/json": map[string]any{
					"schema": map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"$ref": "#/components/schemas/Uuid"}}},
				}}}},
			}},
			"update-file": map[string]any{"post": map[string]any{
				"requestBody": map[string]any{"content": map[string]any{"application/json": map[string]any{
					"schema": map[string]any{"type": "object", "required": []string{"changes"}, "properties": map[string]any{
						"changes": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": big}},
						"revn":    map[string]any{"type": "integer"},
					}},
				}}},
			}},
		},
		"components": map[string]any{"schemas": map[string]any{
			"Uuid": map[string]any{"type": "string", "format": "uuid"},
		}},
	}
	b, _ := json.Marshal(doc)
	return b
}()

func TestAPIInfoBackend(t *testing.T) {
	cs, b := session(t)
	var out textOut
	if res := call(t, cs, "api_info", map[string]any{"type": "create-project"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	for _, want := range []string{"POST /api/main/methods/create-project", "Create a project in a team.", `"teamId"`, `Uuid: {"format":"uuid"`, "Reply:"} {
		if !strings.Contains(out.Text, want) {
			t.Errorf("create-project lacks %q:\n%s", want, out.Text)
		}
	}
	if strings.Contains(out.Text, "never shown") {
		t.Error("the example payload should be left out")
	}

	if res := call(t, cs, "api_info", map[string]any{"type": "create-project", "member": "name"}, &out); res.IsError || !strings.Contains(out.Text, `"maxLength": 250`) {
		t.Errorf("member name: %q %q", out.Text, errText(res))
	}
	if res := call(t, cs, "api_info", map[string]any{"type": "update-file"}, &out); res.IsError ||
		!strings.Contains(out.Text, "give member") || !strings.Contains(out.Text, "- changes: array, required") || len(out.Text) > maxDocText+200 {
		t.Errorf("large schema should be outlined: %d chars %q", len(out.Text), out.Text[:min(len(out.Text), 300)])
	}
	res := call(t, cs, "api_info", map[string]any{"type": "create-projects-now"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "close names: create-project") {
		t.Errorf("unknown method: %q", errText(res))
	}
	res = call(t, cs, "api_info", map[string]any{"type": "Shape", "source": "plugin"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "PENPOT_MCP_URL") {
		t.Errorf("plugin source without a tab: %q", errText(res))
	}
	n := 0
	for _, c := range b.calls {
		if c == "openapi.json" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the document was fetched %d times, want once", n)
	}
}
