package transit

import (
	"reflect"
	"strings"
	"testing"
)

func TestUnmarshalScalarTags(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{`"~~tilde"`, "~tilde"},
		{`"~^caret"`, "^caret"},
		{`"~_"`, nil},
		{`"~?t"`, true},
		{`"~?f"`, false},
		{`"~ia-tag"`, "a-tag"},
		{`"x"`, "x"},
		{`"~"`, "~"},
		{`12`, int64(12)},
		{`1.5`, 1.5},
		{`true`, true},
		{`null`, nil},
	}
	for _, tc := range cases {
		got, err := Unmarshal([]byte(tc.in))
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Unmarshal(%s) = %#v, %v; want %#v", tc.in, got, err, tc.want)
		}
	}
}

func TestUnmarshalCache(t *testing.T) {
	// "~:name" is cached as ^0; the map key "~:id" as ^1, then both are reused.
	in := `[["^ ","~:name","a","~:id","b"],["^ ","^0","c","^1","d"]]`
	got, err := Unmarshal([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []any{
		map[string]any{"name": "a", "id": "b"},
		map[string]any{"name": "c", "id": "d"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestUnmarshalTwoCharCache(t *testing.T) {
	var keys, vals []string
	for i := range 50 {
		k := "~:key-" + strings.Repeat("k", i%7) + string(rune('a'+i%26)) + string(rune('A'+i/26))
		keys = append(keys, k)
		vals = append(vals, `"v"`)
	}
	var pairs []string
	for i, k := range keys {
		pairs = append(pairs, `"`+k+`"`, vals[i])
	}
	// ^ then two characters addresses an index past the first 44 codes.
	in := `[["^ ",` + strings.Join(pairs, ",") + `],["^ ","^10","x"]]`
	if _, err := Unmarshal([]byte(in)); err != nil {
		t.Fatal(err)
	}
}

func TestUnmarshalErrors(t *testing.T) {
	for name, in := range map[string]string{
		"not json":       `{`,
		"unknown cache":  `["^ ","^5","x"]`,
		"bad cache code": `"^abc"`,
		"odd map":        `["^ ","~:a"]`,
		"cache in array": `["^3",1]`,
	} {
		if _, err := Unmarshal([]byte(in)); err == nil {
			t.Errorf("%s: Unmarshal(%s) should fail", name, in)
		}
	}
}

func TestUnmarshalShapes(t *testing.T) {
	// A tagged two-element array decodes to its representation, a plain
	// array keeps a non-string head, and a verbose object decodes its keys.
	cases := []struct {
		in   string
		want any
	}{
		{`["~#set",["a","b"]]`, []any{"a", "b"}},
		{`["~#point",1,2]`, []any{"point", int64(1), int64(2)}},
		{`[1,2]`, []any{int64(1), int64(2)}},
		{`[]`, []any{}},
		{`{"~#set":["a"]}`, []any{"a"}},
		{`{"~:a":1,"b":"~:c"}`, map[string]any{"a": int64(1), "b": "c"}},
	}
	for _, tc := range cases {
		got, err := Unmarshal([]byte(tc.in))
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Unmarshal(%s) = %#v, %v; want %#v", tc.in, got, err, tc.want)
		}
	}
}

func TestMarshalEverySupportedType(t *testing.T) {
	m := Map{
		{K: "n", V: nil},
		{K: "b", V: true},
		{K: "i", V: 3},
		{K: "i64", V: int64(4)},
		{K: "f", V: 1.5},
		{K: "escaped", V: "~tilde"},
		{K: "caret", V: "^x"},
		{K: "tick", V: "`x"},
		{K: "plain", V: "text"},
		{K: "list", V: []any{Keyword("k"), UUID("u")}},
		{K: "uuids", V: []UUID{"a", "b"}},
	}
	data, err := Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	got := back.(map[string]any)
	if got["escaped"] != "~tilde" || got["caret"] != "^x" || got["tick"] != "`x" || got["plain"] != "text" || got["i"] != int64(3) {
		t.Errorf("round trip lost a value: %#v", got)
	}
}

func TestMarshalRejectsUnknown(t *testing.T) {
	if _, err := Marshal(struct{}{}); err == nil {
		t.Error("an unsupported type should fail")
	}
	if _, err := Marshal([]any{make(chan int)}); err == nil {
		t.Error("an unsupported item should fail")
	}
	if _, err := Marshal(Map{{K: "k", V: make(chan int)}}); err == nil {
		t.Error("an unsupported map value should fail")
	}
}
