package penpot

import (
	"strings"
	"testing"
)

func TestParsePathAcceptsEveryCommand(t *testing.T) {
	for _, d := range []string{
		"M1,2 l3 4 h5 v6 H1 V1 Z",
		"M0 0 Q 5 5 10 0 T 20 0",
		"M0 0 C 1 1 2 2 3 3 S 5 5 6 6 s 1 1 2 2",
		"M0 0 A 5 5 0 1 1 10 0 a 5 5 0 0 0 10 0",
		"M0 0 A5 5 0 1110 0",
		"m0 0 1 1 2 2",
		"M1e1 2.5e+1 L-.5 .5",
	} {
		if _, err := ParsePath(d); err != nil {
			t.Errorf("ParsePath(%q): %v", d, err)
		}
	}
}

func TestParsePathRejects(t *testing.T) {
	for name, d := range map[string]string{
		"empty":            "",
		"no moveto":        "L 1 1",
		"number first":     "1 2",
		"number after Z":   "M0 0 Z 1",
		"missing number":   "M0",
		"letter as number": "M0 x",
		"huge number":      "M0 99999999999 L 1 1",
		"bad arc flag":     "M0 0 A 5 5 0 2 0 1 1",
		"too long":         "M0 0" + strings.Repeat(" L1 1", MaxPathLen),
	} {
		if _, err := ParsePath(d); err == nil {
			t.Errorf("%s: ParsePath(%q) should fail", name, d)
		}
	}
}
