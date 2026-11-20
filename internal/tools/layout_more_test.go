package tools

import (
	"strings"
	"testing"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

func TestSetLayoutFlexAndGrid(t *testing.T) {
	do, b := richSession(t)
	for name, args := range map[string]map[string]any{
		"row":           {"board_id": boardID, "layout": "flex", "gap": 8, "padding": 4},
		"column center": {"board_id": boardID, "layout": "flex", "direction": "column", "align_items": "center", "justify": "center"},
		"column end":    {"board_id": boardID, "layout": "flex", "direction": "column", "align_items": "end", "justify": "end"},
		"space between": {"board_id": boardID, "layout": "flex", "justify": "space-between", "align_items": "end"},
		"space around":  {"board_id": boardID, "layout": "flex", "justify": "space-around"},
		"space evenly":  {"board_id": boardID, "layout": "flex", "justify": "space-evenly", "align_items": "center"},
		"grid":          {"board_id": boardID, "layout": "grid", "columns": 2, "gap": 6, "padding": 10},
		"grid default":  {"board_id": boardID, "layout": "grid"},
		"empty board":   {"board_id": board2, "layout": "flex"},
		"empty grid":    {"board_id": board2, "layout": "grid"},
		"with page":     {"board_id": boardID, "page_id": pageID, "layout": "flex"},
	} {
		before := len(b.raw)
		if got := do("set_layout", args); got != "" {
			t.Errorf("%s: %s", name, got)
		} else if len(b.raw) != before+1 {
			t.Errorf("%s: wrote %d times", name, len(b.raw)-before)
		}
	}
	if !strings.Contains(strings.Join(b.raw, "\n"), `"layoutGridCells"`) {
		t.Error("a grid should write its cells")
	}
}

func TestSetLayoutRefusals(t *testing.T) {
	do, b := richSession(t)
	for name, args := range map[string]map[string]any{
		"bad file":      {"file_id": "x", "board_id": boardID, "layout": "flex"},
		"bad page":      {"page_id": "x", "board_id": boardID, "layout": "flex"},
		"bad board id":  {"board_id": "x", "layout": "flex"},
		"no layout":     {"board_id": boardID},
		"bad layout":    {"board_id": boardID, "layout": "stack"},
		"bad direction": {"board_id": boardID, "layout": "flex", "direction": "diagonal"},
		"bad justify":   {"board_id": boardID, "layout": "flex", "justify": "left"},
		"bad align":     {"board_id": boardID, "layout": "flex", "align_items": "stretch"},
		"bad gap":       {"board_id": boardID, "layout": "flex", "gap": -1},
		"bad padding":   {"board_id": boardID, "layout": "flex", "padding": 1e9},
		"bad columns":   {"board_id": boardID, "layout": "grid", "columns": 500},
		"not a board":   {"board_id": rectID, "layout": "flex"},
		"root":          {"board_id": penpot.ZeroID, "layout": "flex"},
		"unknown":       {"board_id": "d0000000-0000-4000-8000-0000000000ff", "layout": "flex"},
	} {
		if got := do("set_layout", args); got == "" {
			t.Errorf("%s: was accepted", name)
		}
	}
	if len(b.raw) != 0 {
		t.Errorf("refused layouts reached update-file: %d", len(b.raw))
	}
}

func TestLayoutPlaceAndAttrs(t *testing.T) {
	kids := []item{{id: "a", w: 10, h: 20}, {id: "b", w: 30, h: 10}, {id: "c", w: 5, h: 5}}
	board := item{x: 100, y: 100, w: 200, h: 100}
	for _, l := range []layoutFields{
		{Layout: "flex"},
		{Layout: "flex", Direction: "column", Justify: "space-between", Gap: 4},
		{Layout: "flex", Justify: "space-around", AlignItems: "center"},
		{Layout: "flex", Justify: "space-evenly", AlignItems: "end", Padding: 5},
		{Layout: "flex", Justify: "end"},
		{Layout: "flex", Justify: "center"},
		{Layout: "grid", Columns: 2, Gap: 3},
	} {
		if len(l.place(board, kids)) != len(kids) {
			t.Errorf("%+v lost a child", l)
		}
		if got := l.attrs(kids); got["layout"] != l.Layout {
			t.Errorf("%+v attrs = %v", l, got)
		}
		if got := l.place(board, nil); len(got) != 0 {
			t.Errorf("%+v placed nothing into %v", l, got)
		}
	}
	grid := layoutFields{Layout: "grid", Columns: 2}
	if cols, rows := grid.grid(5); cols != 2 || rows != 3 {
		t.Errorf("grid(5) = %d x %d", cols, rows)
	}
	if cols, rows := (layoutFields{Layout: "grid"}).grid(0); cols != 2 || rows != 1 {
		t.Errorf("grid(0) = %d x %d", cols, rows)
	}
	if got := tracks(3); len(got) != 3 || got[0]["type"] != "flex" {
		t.Errorf("tracks = %v", got)
	}
}
