package tools

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileNameSanitises(t *testing.T) {
	for in, want := range map[string]string{
		"Login":                  "Login",
		"a/b\\c:d":               "a-b-c-d",
		"  ..odd..  ":            "odd",
		"???":                    "board",
		"":                       "board",
		strings.Repeat("x", 200): strings.Repeat("x", 120),
	} {
		if got := fileName(in); got != want {
			t.Errorf("fileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckScale(t *testing.T) {
	for _, ok := range []float64{0, 0.25, 1, 4} {
		if err := checkScale(ok); err != nil {
			t.Errorf("scale %v: %v", ok, err)
		}
	}
	for _, bad := range []float64{0.1, 5, -1} {
		if err := checkScale(bad); err == nil {
			t.Errorf("scale %v should fail", bad)
		}
	}
}

func TestExportBoardsThroughTab(t *testing.T) {
	tb := &tab{openFile: fileID, export: base64.StdEncoding.EncodeToString(pngOf(t, true))}
	cs, b := sessionWith(t, withRoute(tb.serve(t)))
	var out exportOut
	res := call(t, cs, "export_boards", map[string]any{"file_id": fileID, "dir": "shots", "prefix": "Log"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if len(out.Files) != 1 || out.Files[0].Name != "Login" {
		t.Fatalf("export = %+v", out)
	}
	if _, err := os.Stat(filepath.Join(b.root, "shots", "Login.png")); err != nil {
		t.Fatal(err)
	}

	// The file is there now: a second run needs overwrite.
	res = call(t, cs, "export_boards", map[string]any{"file_id": fileID, "dir": "shots"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "overwrite") {
		t.Errorf("second run: %q", errText(res))
	}
	res = call(t, cs, "export_boards", map[string]any{"file_id": fileID, "dir": "shots", "overwrite": true}, nil)
	if res.IsError {
		t.Errorf("overwrite: %q", errText(res))
	}

	for name, args := range map[string]map[string]any{
		"no match":   {"file_id": fileID, "dir": "shots", "prefix": "Nope"},
		"no dir":     {"file_id": fileID},
		"bad format": {"file_id": fileID, "dir": "shots", "format": "gif"},
		"bad scale":  {"file_id": fileID, "dir": "shots", "scale": 9},
		"outside":    {"file_id": fileID, "dir": "../elsewhere", "overwrite": true},
	} {
		if res := call(t, cs, "export_boards", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
}

func TestExportBoardsNeedsRenderer(t *testing.T) {
	cs, _ := session(t)
	res := call(t, cs, "export_boards", map[string]any{"file_id": fileID, "dir": "shots"}, nil)
	if !res.IsError || !strings.Contains(errText(res), "PENPOT_EMAIL") {
		t.Errorf("got %q", errText(res))
	}
}

func TestExportShapeInputChecks(t *testing.T) {
	tb := &tab{openFile: fileID, export: base64.StdEncoding.EncodeToString(pngOf(t, true))}
	cs, _ := sessionWith(t, withRoute(tb.serve(t)))
	for name, args := range map[string]map[string]any{
		"bad shape id": {"file_id": fileID, "shape_id": "nope", "path": "a.png"},
		"bad format":   {"file_id": fileID, "shape_id": boardID, "path": "a.png", "format": "gif"},
		"bad scale":    {"file_id": fileID, "shape_id": boardID, "path": "a.png", "scale": 10},
		"unknown id":   {"file_id": fileID, "shape_id": "d0000000-0000-4000-8000-0000000000ff", "path": "a.png"},
	} {
		if res := call(t, cs, "export_shape", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
	var out exportOut
	res := call(t, cs, "export_shape", map[string]any{"file_id": fileID, "shape_id": boardID, "path": "a.jpeg", "format": "jpeg"}, &out)
	if res.IsError {
		t.Logf("jpeg through a png fake: %s", errText(res))
	}
}

func TestExportFileAndImportFile(t *testing.T) {
	cs, b := session(t)
	var out exportOut
	res := call(t, cs, "export_file", map[string]any{"file_id": fileID, "path": "backup/design.penpot"}, &out)
	if res.IsError {
		t.Fatal(errText(res))
	}
	data, err := os.ReadFile(filepath.Join(b.root, "backup", "design.penpot"))
	if err != nil || string(data) != "PK-fake-penpot-archive" {
		t.Fatalf("file %q, %v", data, err)
	}
	if res := call(t, cs, "export_file", map[string]any{"file_id": fileID, "path": "backup/design.penpot"}, nil); !res.IsError {
		t.Error("an existing file needs overwrite")
	}
	if res := call(t, cs, "export_file", map[string]any{"file_id": fileID, "path": "backup/design.zip"}, nil); !res.IsError {
		t.Error("a wrong extension was accepted")
	}
	if res := call(t, cs, "export_file", map[string]any{"file_id": "x", "path": "a.penpot"}, nil); !res.IsError {
		t.Error("a bad id was accepted")
	}

	var imp fileImportOut
	res = call(t, cs, "import_file", map[string]any{"project_id": newProjectID, "path": "backup/design.penpot"}, &imp)
	if res.IsError {
		t.Fatal(errText(res))
	}
	if len(imp.FileIDs) != 1 || imp.FileIDs[0] != "f0000000-0000-4000-8000-000000000009" {
		t.Errorf("import = %+v", imp)
	}
	if got := b.bodies["import-binfile"]; got != nil {
		t.Logf("import body: %v", got)
	}
	for name, args := range map[string]map[string]any{
		"wrong ext": {"project_id": newProjectID, "path": "backup/design.zip"},
		"bad id":    {"project_id": "x", "path": "backup/design.penpot"},
		"missing":   {"project_id": newProjectID, "path": "backup/none.penpot"},
		"bad name":  {"project_id": newProjectID, "path": "backup/design.penpot", "name": strings.Repeat("n", 500)},
		"outside":   {"project_id": newProjectID, "path": "../x.penpot"},
	} {
		if res := call(t, cs, "import_file", args, nil); !res.IsError {
			t.Errorf("%s: was accepted", name)
		}
	}
}
