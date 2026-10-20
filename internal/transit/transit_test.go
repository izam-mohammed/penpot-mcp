package transit

import "testing"

// TestMovObjects matches the body scripts/prove/penpot-move.sh sent and
// Penpot accepted.
func TestMovObjects(t *testing.T) {
	got, err := Marshal(Map{
		{"type", Keyword("mov-objects")},
		{"page-id", UUID("p")},
		{"parent-id", UUID("c")},
		{"shapes", []UUID{"r"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `["^ ","~:type","~:mov-objects","~:page-id","~up","~:parent-id","~uc","~:shapes",["~ur"]]`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestScalars(t *testing.T) {
	got, err := Marshal([]any{"~x", "^y", "plain \"q\"", 3, int64(4), 1.5, true, nil})
	if err != nil {
		t.Fatal(err)
	}
	want := `["~~x","~^y","plain \"q\"",3,4,1.5,true,null]`
	if string(got) != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestRejectsUnknown(t *testing.T) {
	if _, err := Marshal(map[string]int{}); err == nil {
		t.Error("a Go map encoded; want an error")
	}
}

// TestTransitRoundTrip decodes an exporter reply shaped like the one in
// docs/penpot-api-reference.md § 10.12, with a repeated key from the cache.
func TestTransitRoundTrip(t *testing.T) {
	body, err := Marshal(Map{
		{"cmd", Keyword("export-shapes")},
		{"profile-id", UUID("e1000000-0000-4000-8000-0000000000aa")},
		{"wait", true},
		{"exports", []any{Map{{"file-id", UUID("f")}, {"type", Keyword("png")}, {"scale", 2}, {"name", "hero"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unmarshal(body)
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	ex := m["exports"].([]any)[0].(map[string]any)
	if m["cmd"] != "export-shapes" || m["wait"] != true || ex["type"] != "png" || ex["scale"] != int64(2) || ex["name"] != "hero" {
		t.Errorf("round trip = %v", m)
	}

	reply := `[["^ ","~:id","~u0d6f","~:name","a","~:uri","~rhttps://x/assets/by-id/1"],["^ ","^0","~u2","^1","b","^2","~rhttps://x/2"]]`
	v, err := Unmarshal([]byte(reply))
	if err != nil {
		t.Fatal(err)
	}
	second := v.([]any)[1].(map[string]any)
	if second["id"] != "2" || second["name"] != "b" || second["uri"] != "https://x/2" {
		t.Errorf("cached keys decoded wrong: %v", second)
	}
	if _, err := Unmarshal([]byte(`["^ ","^9",1]`)); err == nil {
		t.Error("an unknown cache code should fail")
	}
}
