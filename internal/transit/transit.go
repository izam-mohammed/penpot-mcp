// Package transit writes the small subset of transit+json that Penpot needs
// for changes it does not decode from plain JSON (mov-objects). It encodes
// only; replies are read as JSON. See docs/penpot-api-reference.md § 1.4.
package transit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Keyword is a Clojure keyword, written "~:name".
type Keyword string

// UUID is a uuid, written "~u<uuid>".
type UUID string

// Pair is one key and value of a Map.
type Pair struct {
	K Keyword
	V any
}

// Map is a map with keyword keys, written in key order as ["^ ", k, v, ...].
type Map []Pair

// Marshal encodes v. Supported values: Map, Keyword, UUID, []UUID, []any,
// string, bool, int, int64, float64 and nil.
func Marshal(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := encode(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encode(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case Keyword:
		return str(b, "~:"+string(x))
	case UUID:
		return str(b, "~u"+string(x))
	case string:
		// A string that starts with a transit marker is escaped with "~".
		if strings.HasPrefix(x, "~") || strings.HasPrefix(x, "^") || strings.HasPrefix(x, "`") {
			x = "~" + x
		}
		return str(b, x)
	case bool, int, int64, float64:
		data, err := json.Marshal(x)
		if err != nil {
			return err
		}
		b.Write(data)
	case []UUID:
		items := make([]any, len(x))
		for i, u := range x {
			items[i] = u
		}
		return encode(b, items)
	case []any:
		b.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := encode(b, item); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case Map:
		b.WriteString(`["^ "`)
		for _, p := range x {
			b.WriteByte(',')
			if err := encode(b, p.K); err != nil {
				return err
			}
			b.WriteByte(',')
			if err := encode(b, p.V); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	default:
		return fmt.Errorf("transit: cannot encode %T", v)
	}
	return nil
}

func str(b *bytes.Buffer, s string) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	b.Write(data)
	return nil
}
