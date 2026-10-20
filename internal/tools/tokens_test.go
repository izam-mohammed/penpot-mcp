package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportPaletteIdempotent(t *testing.T) {
	cs, b := session(t)
	if err := os.WriteFile(filepath.Join(b.root, "theme.css"), []byte(":root{--primary:#2563EB;--ink:#101828}"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.lib = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"primary","path":"Brand","color":"#000000"}}`
	var out importOut
	if res := call(t, cs, "import_palette", map[string]any{"file_id": fileID, "path": "theme.css", "group": "Brand"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Added != 1 || out.Updated != 1 || len(b.updates) != 1 {
		t.Errorf("want one write updating primary and adding ink: %+v, %d writes", out, len(b.updates))
	}
	if !strings.Contains(b.raw[0], `"id":"`+colorID+`"`) || strings.Count(b.raw[0], `"type":"add-color"`) != 1 {
		t.Errorf("primary should keep its id: %s", b.raw[0])
	}
}

func TestImportPaletteRefusesOutside(t *testing.T) {
	cs, b := session(t)
	for _, path := range []string{"../secret.txt", "/etc/hosts", ".env"} {
		res := call(t, cs, "import_palette", map[string]any{"file_id": fileID, "path": path}, nil)
		if !res.IsError {
			t.Errorf("%s should be refused", path)
		}
	}
	if len(b.calls) != 0 {
		t.Errorf("a refused path reached Penpot: %v", b.calls)
	}
}

func TestSyncTokens(t *testing.T) {
	cs, b := session(t)
	var out syncOut
	tokens := map[string]any{"color": map[string]any{"primary": map[string]any{"$value": "#2563EB", "$type": "color"}}}
	if res := call(t, cs, "sync_tokens", map[string]any{"file_id": fileID, "tokens": tokens, "set_name": "Light"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Tokens != 1 || len(out.Sets) != 1 || out.Sets[0] != "Light" {
		t.Errorf("sync = %+v", out)
	}
	if !strings.Contains(b.raw[0], `"type":"set-tokens-lib"`) || !strings.Contains(b.raw[0], `"tokensLib":"{\"Light\":{\"color\"`) {
		t.Errorf("want the DTCG lib with the set wrapped: %s", b.raw[0])
	}
	for _, bad := range []map[string]any{
		{"file_id": fileID, "tokens": map[string]any{"Light": map[string]any{"x": map[string]any{"$value": "1"}}}},
		{"file_id": fileID, "tokens": map[string]any{"primary": map[string]any{"$value": "#2563EB", "$type": "color"}}},
		{"file_id": fileID},
	} {
		if res := call(t, cs, "sync_tokens", bad, nil); !res.IsError {
			t.Errorf("%v should be refused", bad)
		}
	}
	if len(b.updates) != 1 {
		t.Errorf("refused syncs reached Penpot: %d writes", len(b.updates))
	}
}

func TestApplyToken(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"tokensLib":{"Light":{"color":{"brand":{"$value":"#E53935","$type":"color"}},"radius":{"md":{"$value":"8","$type":"borderRadius"}}}}`
	if res := call(t, cs, "apply_token", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "token": "color.brand", "attribute": "fill"}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	for _, want := range []string{`"fillColor":"#E53935"`, `"appliedTokens":{"fill":"color.brand"}`} {
		if !strings.Contains(b.raw[0], want) {
			t.Errorf("fill change lacks %s: %s", want, b.raw[0])
		}
	}
	if res := call(t, cs, "apply_token", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "token": "radius.md", "attribute": "radius"}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if !strings.Contains(b.raw[1], `"r1":8`) || !strings.Contains(b.raw[1], `"r4":"radius.md"`) {
		t.Errorf("radius change: %s", b.raw[1])
	}
	res := call(t, cs, "apply_token", map[string]any{"file_id": fileID, "shape_ids": []string{rectID}, "token": "color.nope", "attribute": "fill"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "not in the file's tokens") {
		t.Errorf("unknown token: got %q", errText(res))
	}
}

func TestDeleteAssets(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"Old","color":"#000000"}},` +
		`"components":{"` + componentID + `":{"id":"` + componentID + `","name":"Card","mainInstanceId":"` + boardID + `","mainInstancePage":"` + pageID + `"}}`
	if res := call(t, cs, "delete_color", map[string]any{"file_id": fileID, "id": colorID}, nil); !res.IsError || !strings.Contains(errText(res), "confirm: true") {
		t.Errorf("unconfirmed delete: got %q", errText(res))
	}
	if len(b.calls) != 0 {
		t.Errorf("an unconfirmed delete reached Penpot: %v", b.calls)
	}
	if res := call(t, cs, "delete_component", map[string]any{"file_id": fileID, "id": componentID, "confirm": true}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if !strings.Contains(b.raw[0], `{"id":"`+boardID+`","pageId":"`+pageID+`","type":"del-obj"}`) || !strings.Contains(b.raw[0], `"type":"del-component"`) {
		t.Errorf("want the main instance and the component deleted: %s", b.raw[0])
	}
	if !strings.Contains(b.logs.String(), "Card") {
		t.Error("the delete was not logged")
	}
}
