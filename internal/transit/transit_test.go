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
