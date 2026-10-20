package penpot

import (
	"math"
	"testing"
)

func TestParsePath(t *testing.T) {
	segs, err := ParsePath("m10,10 20,0 v20 H10 z M0 0 q10-10 20 0 t20 0 A10 10 0 0 1 60 0")
	if err != nil {
		t.Fatal(err)
	}
	cmds := ""
	for _, s := range segs {
		cmds += s.Command[:1]
	}
	// m, implicit l, v, H, z, M, q, t, then an arc split into cubics.
	if cmds[:8] != "mlllcmcc" || len(segs) < 9 {
		t.Errorf("commands %s", cmds)
	}
	if p := segs[1].Params; p["x"] != 30 || p["y"] != 10 {
		t.Errorf("implicit relative line ends at %v,%v, want 30,10", p["x"], p["y"])
	}
	last := segs[len(segs)-1].Params
	if math.Abs(last["x"]-60) > 1e-9 || math.Abs(last["y"]) > 1e-9 {
		t.Errorf("arc ends at %v,%v, want 60,0", last["x"], last["y"])
	}
}

func TestPathBoundsIncludesCurves(t *testing.T) {
	segs, err := ParsePath("M0 0 C0 -40 100 -40 100 0")
	if err != nil {
		t.Fatal(err)
	}
	x, y, w, h := PathBounds(segs)
	if x != 0 || w != 100 || math.Abs(y+30) > 1e-9 || math.Abs(h-30) > 1e-9 {
		t.Errorf("bounds %v,%v %vx%v, want 0,-30 100x30", x, y, w, h)
	}
}

func TestRotatedGeometry(t *testing.T) {
	g := RotatedGeometry(0, 0, 100, 50, 90)
	if g.Selrect.Width != 100 || g.X != 0 {
		t.Errorf("selrect must stay the unrotated box: %+v", g.Selrect)
	}
	// 90 degrees clockwise about (50, 25): the top-left corner goes to (75, -25).
	if p := g.Points[0]; math.Abs(p.X-75) > 1e-9 || math.Abs(p.Y+25) > 1e-9 {
		t.Errorf("top-left corner at %v, want 75,-25", p)
	}
	m, inv := g.Transform, g.TransformInverse
	if math.Abs(m.A*inv.A+m.C*inv.B-1) > 1e-9 || math.Abs(m.B) != 1 {
		t.Errorf("transform %+v and inverse %+v do not match", m, inv)
	}
	if r := RotatedGeometry(0, 0, 10, 10, -90).Rotation; r != 270 {
		t.Errorf("rotation -90 stored as %v, want 270", r)
	}
	if g := RotatedGeometry(1, 2, 3, 4, 0); g.Transform != Identity {
		t.Error("no rotation should keep the identity transform")
	}
}
