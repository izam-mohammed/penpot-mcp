package tools

import (
	"context"
	"fmt"

	"github.com/izam-mohammed/penpot-mcp/internal/penpot"
)

// rawPage is a page with every shape kept as decoded JSON, for edits that
// must carry attributes this server does not model.
type rawPage struct {
	objects map[string]map[string]any
}

func loadRaw(ctx context.Context, d Deps, t penpot.Target) (rawPage, error) {
	var p struct {
		Objects map[string]map[string]any `json:"objects"`
	}
	if err := d.Client.Call(ctx, "get-page", map[string]string{"fileId": t.FileID, "pageId": t.PageID}, &p); err != nil {
		return rawPage{}, err
	}
	if p.Objects == nil {
		p.Objects = map[string]map[string]any{}
	}
	return rawPage{objects: p.Objects}, nil
}

// get returns a shape, or an error naming the page when it is not there.
func (p rawPage) get(id string, t penpot.Target) (map[string]any, error) {
	o, ok := p.objects[id]
	if !ok {
		return nil, fmt.Errorf("shape %s is not on page %q", id, t.PageName)
	}
	return o, nil
}

func str(o map[string]any, k string) string {
	s, _ := o[k].(string)
	return s
}

func num(o map[string]any, k string) float64 {
	f, _ := o[k].(float64)
	return f
}

func children(o map[string]any) []string {
	list, _ := o["shapes"].([]any)
	out := make([]string, 0, len(list))
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// descendants lists id's children, grandchildren and so on, parents first.
func (p rawPage) descendants(id string) []string {
	var out []string
	var walk func(string)
	walk = func(cur string) {
		for _, c := range children(p.objects[cur]) {
			if _, ok := p.objects[c]; !ok || c == cur {
				continue
			}
			out = append(out, c)
			walk(c)
		}
	}
	walk(id)
	return out
}

// ancestors lists id's parents up to (not including) the root, nearest
// first.
func (p rawPage) ancestors(id string) []string {
	var out []string
	seen := map[string]bool{id: true}
	for cur := str(p.objects[id], "parentId"); cur != "" && cur != penpot.ZeroID && !seen[cur]; cur = str(p.objects[cur], "parentId") {
		seen[cur] = true
		out = append(out, cur)
	}
	return out
}

func isContainer(typ string) bool {
	return typ == penpot.TypeBoard || typ == penpot.TypeGroup || typ == penpot.TypeBool
}

// shiftRaw returns the attributes that move a stored shape by dx, dy:
// box, selrect, points, path content and text position data.
func shiftRaw(o map[string]any, dx, dy float64) (map[string]any, error) {
	attrs := map[string]any{
		"x": num(o, "x") + dx,
		"y": num(o, "y") + dy,
	}
	if sr, ok := o["selrect"].(map[string]any); ok {
		n := map[string]any{}
		for k, v := range sr {
			f, _ := v.(float64)
			switch k {
			case "x", "x1", "x2":
				f += dx
			case "y", "y1", "y2":
				f += dy
			}
			n[k] = f
		}
		attrs["selrect"] = n
	}
	if pts, ok := o["points"].([]any); ok {
		n := make([]map[string]any, 0, len(pts))
		for _, v := range pts {
			pt, _ := v.(map[string]any)
			n = append(n, map[string]any{"x": num(pt, "x") + dx, "y": num(pt, "y") + dy})
		}
		attrs["points"] = n
	}
	if c, ok := o["content"]; ok && c != nil {
		switch t := str(o, "type"); t {
		case penpot.TypePath, penpot.TypeBool:
			segs, err := rawSegments(c)
			if err != nil {
				return nil, fmt.Errorf("%s %q: %w", t, str(o, "name"), err)
			}
			attrs["content"] = penpot.TranslatePath(segs, dx, dy)
		}
	}
	if pd, ok := o["positionData"].([]any); ok {
		n := make([]any, 0, len(pd))
		for _, v := range pd {
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			c := make(map[string]any, len(m))
			for k, val := range m {
				c[k] = val
			}
			c["x"] = num(m, "x") + dx
			c["y"] = num(m, "y") + dy
			n = append(n, c)
		}
		attrs["positionData"] = n
	}
	return attrs, nil
}

// rawSegments reads stored path content: an SVG string (what get-page
// returns) or a segment list.
func rawSegments(c any) ([]penpot.Segment, error) {
	switch v := c.(type) {
	case string:
		if v == "" {
			return nil, nil
		}
		return penpot.ParsePath(v)
	case []any:
		segs := make([]penpot.Segment, 0, len(v))
		for _, item := range v {
			m, _ := item.(map[string]any)
			params := map[string]float64{}
			if pm, ok := m["params"].(map[string]any); ok {
				for k, pv := range pm {
					if f, ok := pv.(float64); ok {
						params[k] = f
					}
				}
			}
			segs = append(segs, penpot.Segment{Command: str(m, "command"), Params: params})
		}
		return segs, nil
	}
	return nil, fmt.Errorf("cannot read its path content")
}

// moveTree returns mod-obj changes that move id and everything inside it by
// dx, dy.
func moveTree(p rawPage, pageID, id string, dx, dy float64) ([]map[string]any, error) {
	var changes []map[string]any
	for _, sid := range append([]string{id}, p.descendants(id)...) {
		attrs, err := shiftRaw(p.objects[sid], dx, dy)
		if err != nil {
			return nil, err
		}
		changes = append(changes, penpot.ModObj(pageID, sid, attrs))
	}
	return changes, nil
}

// refit lists the groups and bools among ids' ancestors, innermost first,
// for a reg-objects change after their children changed.
func refit(p rawPage, ids []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, id := range ids {
		for _, a := range p.ancestors(id) {
			t := str(p.objects[a], "type")
			if (t == penpot.TypeGroup || t == penpot.TypeBool) && !seen[a] {
				seen[a] = true
				out = append(out, a)
			}
		}
	}
	return out
}
