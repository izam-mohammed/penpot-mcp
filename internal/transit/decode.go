package transit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Unmarshal decodes transit+json into plain Go values: maps become
// map[string]any (keyword keys lose their "~:"), keywords, uuids and URIs
// become strings, arrays []any. Tagged values other than those are kept as
// their value. It follows the write cache ("^0", "^1", ...), so keys repeated
// in a reply decode right.
func Unmarshal(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("transit: %w", err)
	}
	r := &reader{}
	return r.value(raw, false)
}

type reader struct{ cache []string }

// cacheBase is the transit cache's radix: codes are "^" and one or two
// characters from '0'.
const cacheBase = 44

func cacheIndex(code string) (int, bool) {
	s := code[1:]
	switch len(s) {
	case 1:
		return int(s[0]) - 48, true
	case 2:
		return (int(s[0])-48)*cacheBase + int(s[1]) - 48, true
	}
	return 0, false
}

// str decodes one string, reading or filling the cache.
func (r *reader) str(s string, key bool) (string, error) {
	if strings.HasPrefix(s, "^") && s != "^ " {
		i, ok := cacheIndex(s)
		if !ok || i < 0 || i >= len(r.cache) {
			return "", fmt.Errorf("transit: unknown cache code %q", s)
		}
		return r.cache[i], nil
	}
	if len(s) > 3 && (key || strings.HasPrefix(s, "~:") || strings.HasPrefix(s, "~$") || strings.HasPrefix(s, "~#")) {
		r.cache = append(r.cache, s)
	}
	return s, nil
}

func scalar(s string) any {
	if len(s) < 2 || s[0] != '~' {
		return s
	}
	switch s[1] {
	case '~', '^', '`':
		return s[1:]
	case ':', '$', 'u', 'r':
		return s[2:]
	case '_':
		return nil
	case '?':
		return s[2:] == "t"
	}
	return s[2:]
}

func (r *reader) value(v any, key bool) (any, error) {
	switch x := v.(type) {
	case string:
		s, err := r.str(x, key)
		if err != nil {
			return nil, err
		}
		return scalar(s), nil
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, nil
		}
		return x.Float64()
	case []any:
		if len(x) > 0 {
			if head, ok := x[0].(string); ok {
				if head == "^ " {
					return r.mapOf(x[1:])
				}
				s, err := r.str(head, false)
				if err != nil {
					return nil, err
				}
				if strings.HasPrefix(s, "~#") && len(x) == 2 {
					return r.value(x[1], false)
				}
				out := []any{scalar(s)}
				for _, item := range x[1:] {
					d, err := r.value(item, false)
					if err != nil {
						return nil, err
					}
					out = append(out, d)
				}
				return out, nil
			}
		}
		out := make([]any, 0, len(x))
		for _, item := range x {
			d, err := r.value(item, false)
			if err != nil {
				return nil, err
			}
			out = append(out, d)
		}
		return out, nil
	case map[string]any:
		// Verbose transit writes maps as JSON objects, and a tagged value
		// (a quoted top-level scalar, a set) as a one-key {"~#tag": rep}.
		if len(x) == 1 {
			for k, item := range x {
				if strings.HasPrefix(k, "~#") {
					return r.value(item, false)
				}
			}
		}
		out := make(map[string]any, len(x))
		for k, item := range x {
			d, err := r.value(item, false)
			if err != nil {
				return nil, err
			}
			out[fmt.Sprint(scalar(k))] = d
		}
		return out, nil
	}
	return v, nil
}

func (r *reader) mapOf(kv []any) (map[string]any, error) {
	if len(kv)%2 != 0 {
		return nil, fmt.Errorf("transit: map with an odd number of items")
	}
	out := make(map[string]any, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		k, err := r.value(kv[i], true)
		if err != nil {
			return nil, err
		}
		v, err := r.value(kv[i+1], false)
		if err != nil {
			return nil, err
		}
		out[fmt.Sprint(k)] = v
	}
	return out, nil
}
