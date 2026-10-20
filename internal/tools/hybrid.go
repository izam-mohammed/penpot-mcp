package tools

import (
	"context"
	"encoding/json"
)

// tabCheck is what the connected tab saw after an API write: the hybrid
// step of docs/bridge.md for draw, create_text and set_layout.
type tabCheck struct {
	Reflowed bool                `json:"reflowed"`
	Text     map[string]textSize `json:"text_sizes,omitempty" jsonschema:"each text's size as Penpot laid it out"`
	Note     string              `json:"note,omitempty"`
}

type textSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// afterWrite lets the tab reflow and reads the real size of textIDs, when
// the MCP route is on and the tab has fileID open. It never fails the
// write it follows: without a tab the API result stands.
func afterWrite(ctx context.Context, d Deps, fileID string, textIDs []string) *tabCheck {
	if d.Route == nil {
		return nil
	}
	open, err := d.Route.Current(ctx)
	if err != nil {
		return &tabCheck{Note: "not reflowed: " + err.Error()}
	}
	if open.FileID != fileID {
		return &tabCheck{Note: "not reflowed: the tab has " + open.FileName + " open"}
	}
	code := `if (typeof penpot.waitForLayoutUpdate === 'function') await penpot.waitForLayoutUpdate();
const out = {};
for (const id of ` + js(textIDs) + `) {
  const s = penpot.currentPage.getShapeById(id);
  if (s) out[id] = {width: s.width, height: s.height};
}
return out;`
	var res struct{ sizes map[string]textSize }
	err = d.Writer.Exclusive(fileID, func() error {
		r, err := d.Route.Exec(ctx, code)
		if err != nil {
			return err
		}
		return json.Unmarshal(r.Value, &res.sizes)
	})
	if err != nil {
		return &tabCheck{Note: "not reflowed: " + err.Error()}
	}
	return &tabCheck{Reflowed: true, Text: res.sizes}
}

func textIDs(out drawOut) []string {
	var ids []string
	for _, s := range out.Shapes {
		if s.Type == "text" {
			ids = append(ids, s.ShapeID)
		}
	}
	return ids
}
