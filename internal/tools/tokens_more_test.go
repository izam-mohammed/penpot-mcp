package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

const typoID = "d1000000-0000-4000-8000-000000000001"

const libWithStyles = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"Primary","path":"Brand","color":"#2563eb","opacity":0.5}},` +
	`"typographies":{"` + typoID + `":{"id":"` + typoID + `","name":"Body","path":"Text","fontId":"gfont-inter","fontFamily":"Inter",` +
	`"fontVariantId":"regular","fontSize":"16","fontWeight":"400","fontStyle":"normal","lineHeight":"1.4","letterSpacing":"0","modifiedAt":"x"}}`

func TestUpdateColor(t *testing.T) {
	cs, b := session(t)
	b.lib = libWithStyles
	var out assetOut
	res := call(t, cs, "update_color", map[string]any{
		"file_id": fileID, "color_id": colorID, "name": "Accent / Hot", "hex": "#FF0000", "opacity": 0.8,
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.Name != "Hot" || out.Path != "Accent" {
		t.Errorf("out = %+v", out)
	}
	raw := b.raw[0]
	for _, want := range []string{`"type":"mod-color"`, `"color":"#ff0000"`, `"opacity":0.8`} {
		if !strings.Contains(raw, want) {
			t.Errorf("change lacks %s: %s", want, raw)
		}
	}
	empty := ""
	_ = empty
	var out2 assetOut
	res = call(t, cs, "update_color", map[string]any{"file_id": fileID, "color_id": colorID, "path": ""}, &out2)
	if res.IsError || out2.Path != "" || out2.Name != "Primary" {
		t.Errorf("moving out of the group: %+v %q", out2, errText(res))
	}
}

func TestUpdateColorRefusals(t *testing.T) {
	cs, b := session(t)
	b.lib = libWithStyles + ``
	for name, args := range map[string]map[string]any{
		"bad color id": {"file_id": fileID, "color_id": "x"},
		"bad hex":      {"file_id": fileID, "color_id": colorID, "hex": "red"},
		"bad opacity":  {"file_id": fileID, "color_id": colorID, "opacity": 3},
		"unknown":      {"file_id": fileID, "color_id": "c1000000-0000-4000-8000-0000000000ff"},
		"bad name":     {"file_id": fileID, "color_id": colorID, "name": strings.Repeat("n", 500)},
	} {
		if res := call(t, cs, "update_color", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if b.count("update-file") != 0 {
		t.Error("a refused update reached update-file")
	}
}

func TestUpdateColorRefusesGradient(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"colors":{"` + colorID + `":{"id":"` + colorID + `","name":"Fade","gradient":{"type":"linear"}}}`
	res := call(t, cs, "update_color", map[string]any{"file_id": fileID, "color_id": colorID, "name": "Fade 2"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "gradient or image") {
		t.Errorf("got %q", errText(res))
	}
}

func TestUpdateTypography(t *testing.T) {
	cs, b := session(t)
	b.lib = libWithStyles
	var out assetOut
	res := call(t, cs, "update_typography", map[string]any{
		"file_id": fileID, "typography_id": typoID, "name": "Heading / H1", "font_family": "Roboto",
		"font_weight": 700, "italic": true, "font_size": 32, "line_height": 1.2, "letter_spacing": 0.5,
	}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if out.Name != "H1" || out.Path != "Heading" {
		t.Errorf("out = %+v", out)
	}
	raw := b.raw[0]
	for _, want := range []string{
		`"type":"mod-typography"`, `"fontFamily":"Roboto"`, `"fontId":"gfont-roboto"`, `"fontSize":"32"`,
		`"lineHeight":"1.2"`, `"letterSpacing":"0.5"`, `"fontStyle":"italic"`, `"fontVariantId":"700italic"`,
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("change lacks %s: %s", want, raw)
		}
	}
	if strings.Contains(raw, "modifiedAt") {
		t.Error("modifiedAt should not be sent back")
	}
	// Only the weight changes: the family and style come from the old style.
	res = call(t, cs, "update_typography", map[string]any{"file_id": fileID, "typography_id": typoID, "font_weight": 600}, nil)
	if res.IsError || !strings.Contains(b.raw[1], `"fontFamily":"Inter"`) {
		t.Errorf("weight only: %q %s", errText(res), b.raw[1])
	}
}

func TestUpdateTypographyRefusals(t *testing.T) {
	cs, b := session(t)
	b.lib = libWithStyles
	for name, args := range map[string]map[string]any{
		"bad id":     {"file_id": fileID, "typography_id": "x"},
		"unknown":    {"file_id": fileID, "typography_id": "d1000000-0000-4000-8000-0000000000ff"},
		"bad size":   {"file_id": fileID, "typography_id": typoID, "font_size": 2},
		"bad height": {"file_id": fileID, "typography_id": typoID, "line_height": 11},
		"bad weight": {"file_id": fileID, "typography_id": typoID, "font_weight": 450},
		"bad family": {"file_id": fileID, "typography_id": typoID, "font_family": "<b>"},
		"bad name":   {"file_id": fileID, "typography_id": typoID, "name": strings.Repeat("n", 500)},
	} {
		if res := call(t, cs, "update_typography", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	if b.count("update-file") != 0 {
		t.Error("a refused update reached update-file")
	}
}

func TestApplyTokenValues(t *testing.T) {
	cs, b := session(t)
	b.lib = `,"tokensLib":{"Light":{"opacity":{"half":{"$value":"50%","$type":"opacity"}},"num":{"o":{"$value":0.4,"$type":"opacity"}},` +
		`"color":{"bad":{"$value":"blue","$type":"color"}},"radius":{"huge":{"$value":"9999999","$type":"borderRadius"}}}}`
	ok := func(args map[string]any) {
		t.Helper()
		args["file_id"], args["shape_ids"] = fileID, []string{rectID}
		if res := call(t, cs, "apply_token", args, nil); res.IsError {
			t.Fatalf("%v: %s", args, errText(res))
		}
	}
	ok(map[string]any{"token": "opacity.half", "attribute": "opacity"})
	ok(map[string]any{"token": "num.o", "attribute": "opacity"})
	ok(map[string]any{"token": "x.y", "attribute": "radius", "value": "12px"})
	if !strings.Contains(b.raw[0], `"opacity":0.5`) || !strings.Contains(b.raw[1], `"opacity":0.4`) || !strings.Contains(b.raw[2], `"r1":12`) {
		t.Errorf("values: %v", b.raw)
	}

	for name, args := range map[string]map[string]any{
		"not a colour":   {"token": "color.bad", "attribute": "fill"},
		"huge radius":    {"token": "radius.huge", "attribute": "radius"},
		"bad attribute":  {"token": "opacity.half", "attribute": "blur"},
		"not a number":   {"token": "x.y", "attribute": "radius", "value": "wide"},
		"bad opacity":    {"token": "x.y", "attribute": "opacity", "value": "-1"},
		"bad token name": {"token": "no spaces allowed", "attribute": "fill", "value": "#fff"},
		"no stroke":      {"token": "x.y", "attribute": "stroke_color", "value": "#ffffff"},
		"radius on text": {"token": "x.y", "attribute": "radius", "value": "4", "shape_ids": []string{"nope"}},
	} {
		if _, has := args["shape_ids"]; !has {
			args["shape_ids"] = []string{rectID}
		}
		args["file_id"] = fileID
		if res := call(t, cs, "apply_token", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
}

func TestSyncTokensFromFileAndRefusals(t *testing.T) {
	cs, b := session(t)
	writeFile(t, b.root, "tokens.json", `{"Light":{"color":{"brand":{"$value":"#E53935","$type":"color"}}},"$metadata":{"tokenSetOrder":["Light"]}}`)
	var out syncOut
	if res := call(t, cs, "sync_tokens", map[string]any{"file_id": fileID, "path": "tokens.json"}, &out); res.IsError {
		t.Fatal(errText(res))
	}
	if out.Tokens != 1 || len(out.Sets) != 1 {
		t.Errorf("out = %+v", out)
	}
	writeFile(t, b.root, "list.json", `[1,2]`)
	good := map[string]any{"Light": map[string]any{"c": map[string]any{"$value": "#fff", "$type": "color"}}}
	for name, args := range map[string]map[string]any{
		"both":          {"file_id": fileID, "path": "tokens.json", "tokens": good},
		"neither":       {"file_id": fileID},
		"not an object": {"file_id": fileID, "path": "list.json"},
		"missing file":  {"file_id": fileID, "path": "none.json"},
		"bad set name":  {"file_id": fileID, "tokens": good, "set_name": strings.Repeat("s", 500)},
		"set not map":   {"file_id": fileID, "tokens": map[string]any{"Light": "x"}},
		"only metadata": {"file_id": fileID, "tokens": map[string]any{"$metadata": map[string]any{}}},
		"padded set":    {"file_id": fileID, "tokens": map[string]any{" Light": map[string]any{}}},
		"token as set":  {"file_id": fileID, "tokens": map[string]any{"c": map[string]any{"$value": "#fff", "$type": "color"}}},
		"no type":       {"file_id": fileID, "tokens": map[string]any{"Light": map[string]any{"c": map[string]any{"$value": "#fff"}}}},
		"bad token":     {"file_id": fileID, "tokens": map[string]any{"Light": map[string]any{"a b": map[string]any{"$value": "1", "$type": "color"}}}},
		"leaf not map":  {"file_id": fileID, "tokens": map[string]any{"Light": map[string]any{"a": "x"}}},
	} {
		if res := call(t, cs, "sync_tokens", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
}
