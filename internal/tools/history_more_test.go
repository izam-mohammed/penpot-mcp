package tools

import (
	"strings"
	"testing"
)

func TestPluginDataOnFileAndShape(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"pluginData":{"notes":{"owner":"ana"}}`
	var out pluginDataOut
	res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "namespace": "notes", "key": "owner"}, &out)
	if res.IsError || out.Value == nil || *out.Value != "ana" {
		t.Fatalf("read file data = %+v %q", out, errText(res))
	}
	var missing pluginDataOut
	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "namespace": "notes", "key": "none"}, &missing); res.IsError || missing.Value != nil {
		t.Errorf("an absent key = %+v %q", missing, errText(res))
	}

	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "namespace": "notes", "key": "owner", "value": "bo"}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if !strings.Contains(b.raw[0], `"objectType":"file"`) || !strings.Contains(b.raw[0], `"value":"bo"`) {
		t.Errorf("file write: %s", b.raw[0])
	}
	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "shape_id": rectID, "namespace": "notes", "key": "k", "value": ""}, nil); res.IsError {
		t.Fatal(errText(res))
	}
	if !strings.Contains(b.raw[1], `"objectType":"shape"`) || !strings.Contains(b.raw[1], `"value":null`) {
		t.Errorf("shape delete: %s", b.raw[1])
	}
	// A shape's own data is read from the page.
	if res := call(t, cs, "plugin_data", map[string]any{"file_id": fileID, "shape_id": rectID, "namespace": "notes", "key": "k"}, &out); res.IsError {
		t.Errorf("shape read: %q", errText(res))
	}
}

func TestPluginDataRefusals(t *testing.T) {
	cs, b := session(t)
	for name, args := range map[string]map[string]any{
		"bad namespace": {"file_id": fileID, "namespace": "has space", "key": "k"},
		"bad key":       {"file_id": fileID, "namespace": "n", "key": ""},
		"huge value":    {"file_id": fileID, "namespace": "n", "key": "k", "value": strings.Repeat("x", 10_001)},
		"bad shape":     {"file_id": fileID, "namespace": "n", "key": "k", "shape_id": "x"},
		"unknown shape": {"file_id": fileID, "namespace": "n", "key": "k", "shape_id": "d0000000-0000-4000-8000-0000000000ff"},
	} {
		if res := call(t, cs, "plugin_data", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if len(b.raw) != 0 {
		t.Errorf("refused writes reached update-file: %d", len(b.raw))
	}
}
