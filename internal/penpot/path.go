package penpot

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Path limits: an SVG d string longer than this, or with more segments, is
// refused.
const (
	MaxPathLen      = 200_000
	MaxPathSegments = 5_000
)

// Segment is one command of Penpot's path content: move-to, line-to,
// curve-to (cubic) or close-path, in absolute page coordinates.
type Segment struct {
	Command string             `json:"command"`
	Params  map[string]float64 `json:"params"`
}

// ParsePath turns an SVG path d string into Penpot segments. Every SVG
// command is accepted; quadratics, smooth curves and arcs become cubics.
func ParsePath(d string) ([]Segment, error) {
	if len(d) > MaxPathLen {
		return nil, fmt.Errorf("path is longer than %d bytes", MaxPathLen)
	}
	p := pathParser{s: d}
	var out []Segment
	var cur, start, lastCtrl Point
	var prev byte
	for {
		p.skipSep()
		if p.i >= len(p.s) {
			break
		}
		cmd := p.s[p.i]
		if !isCommand(cmd) {
			// A number after a command repeats it; after M it means L.
			if prev == 0 {
				return nil, errors.New("path must start with M")
			}
			cmd = prev
			switch cmd {
			case 'M':
				cmd = 'L'
			case 'm':
				cmd = 'l'
			case 'Z', 'z':
				return nil, fmt.Errorf("unexpected number at byte %d", p.i)
			}
		} else {
			p.i++
		}
		if len(out) == 0 && cmd != 'M' && cmd != 'm' {
			return nil, errors.New("path must start with M")
		}
		rel := cmd >= 'a'
		at := func(x, y float64) Point {
			if rel {
				return Point{X: cur.X + x, Y: cur.Y + y}
			}
			return Point{X: x, Y: y}
		}
		ctrl := Point{}
		hasCtrl := false
		switch cmd | 0x20 { // lower case
		case 'm':
			x, y, err := p.pair()
			if err != nil {
				return nil, err
			}
			cur = at(x, y)
			start = cur
			out = append(out, move(cur))
		case 'l':
			x, y, err := p.pair()
			if err != nil {
				return nil, err
			}
			cur = at(x, y)
			out = append(out, line(cur))
		case 'h':
			x, err := p.num()
			if err != nil {
				return nil, err
			}
			if rel {
				x += cur.X
			}
			cur = Point{X: x, Y: cur.Y}
			out = append(out, line(cur))
		case 'v':
			y, err := p.num()
			if err != nil {
				return nil, err
			}
			if rel {
				y += cur.Y
			}
			cur = Point{X: cur.X, Y: y}
			out = append(out, line(cur))
		case 'c', 's':
			var c1 Point
			if cmd|0x20 == 'c' {
				x, y, err := p.pair()
				if err != nil {
					return nil, err
				}
				c1 = at(x, y)
			} else {
				c1 = reflect(cur, lastCtrl, prev, "cCsS")
			}
			x2, y2, err := p.pair()
			if err != nil {
				return nil, err
			}
			c2 := at(x2, y2)
			x, y, err := p.pair()
			if err != nil {
				return nil, err
			}
			end := at(x, y)
			out = append(out, curve(c1, c2, end))
			cur, ctrl, hasCtrl = end, c2, true
		case 'q', 't':
			var q Point
			if cmd|0x20 == 'q' {
				x, y, err := p.pair()
				if err != nil {
					return nil, err
				}
				q = at(x, y)
			} else {
				q = reflect(cur, lastCtrl, prev, "qQtT")
			}
			x, y, err := p.pair()
			if err != nil {
				return nil, err
			}
			end := at(x, y)
			c1 := Point{X: cur.X + 2.0/3*(q.X-cur.X), Y: cur.Y + 2.0/3*(q.Y-cur.Y)}
			c2 := Point{X: end.X + 2.0/3*(q.X-end.X), Y: end.Y + 2.0/3*(q.Y-end.Y)}
			out = append(out, curve(c1, c2, end))
			cur, ctrl, hasCtrl = end, q, true
		case 'a':
			rx, ry, err := p.pair()
			if err != nil {
				return nil, err
			}
			rot, err := p.num()
			if err != nil {
				return nil, err
			}
			large, err := p.flag()
			if err != nil {
				return nil, err
			}
			sweep, err := p.flag()
			if err != nil {
				return nil, err
			}
			x, y, err := p.pair()
			if err != nil {
				return nil, err
			}
			end := at(x, y)
			out = append(out, arc(cur, end, rx, ry, rot, large, sweep)...)
			cur = end
		case 'z':
			out = append(out, Segment{Command: "close-path", Params: map[string]float64{}})
			cur = start
		default:
			return nil, fmt.Errorf("unknown path command %q", cmd)
		}
		if hasCtrl {
			lastCtrl = ctrl
		}
		prev = cmd
		if len(out) > MaxPathSegments {
			return nil, fmt.Errorf("path has more than %d segments", MaxPathSegments)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("path is empty")
	}
	return out, nil
}

// reflect is the implied first control point of S or T: the previous control
// point mirrored about the current point, or the current point itself.
func reflect(cur, last Point, prev byte, kinds string) Point {
	for i := 0; i < len(kinds); i++ {
		if prev == kinds[i] {
			return Point{X: 2*cur.X - last.X, Y: 2*cur.Y - last.Y}
		}
	}
	return cur
}

func move(p Point) Segment {
	return Segment{Command: "move-to", Params: map[string]float64{"x": p.X, "y": p.Y}}
}

func line(p Point) Segment {
	return Segment{Command: "line-to", Params: map[string]float64{"x": p.X, "y": p.Y}}
}

func curve(c1, c2, end Point) Segment {
	return Segment{Command: "curve-to", Params: map[string]float64{
		"x": end.X, "y": end.Y, "c1x": c1.X, "c1y": c1.Y, "c2x": c2.X, "c2y": c2.Y,
	}}
}

// arc converts an SVG elliptical arc to cubics, at most 90 degrees each
// (SVG 1.1 implementation notes, F.6.5).
func arc(from, to Point, rx, ry, rotDeg float64, large, sweep bool) []Segment {
	if from == to {
		return nil
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return []Segment{line(to)}
	}
	phi := rotDeg * math.Pi / 180
	cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)
	dx, dy := (from.X-to.X)/2, (from.Y-to.Y)/2
	x1 := cosPhi*dx + sinPhi*dy
	y1 := -sinPhi*dx + cosPhi*dy
	if l := x1*x1/(rx*rx) + y1*y1/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1*y1 - ry*ry*x1*x1
	den := rx*rx*y1*y1 + ry*ry*x1*x1
	k := 0.0
	if den != 0 && num > 0 {
		k = math.Sqrt(num / den)
	}
	if large == sweep {
		k = -k
	}
	cxp, cyp := k*rx*y1/ry, -k*ry*x1/rx
	cx := cosPhi*cxp - sinPhi*cyp + (from.X+to.X)/2
	cy := sinPhi*cxp + cosPhi*cyp + (from.Y+to.Y)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
	}
	t1 := angle(1, 0, (x1-cxp)/rx, (y1-cyp)/ry)
	dt := angle((x1-cxp)/rx, (y1-cyp)/ry, (-x1-cxp)/rx, (-y1-cyp)/ry)
	if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	} else if sweep && dt < 0 {
		dt += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dt) / (math.Pi / 2)))
	step := dt / float64(n)
	alpha := 4.0 / 3 * math.Tan(step/4)
	pt := func(t float64) (Point, Point) { // point and derivative
		ct, st := math.Cos(t), math.Sin(t)
		p := Point{X: cx + rx*ct*cosPhi - ry*st*sinPhi, Y: cy + rx*ct*sinPhi + ry*st*cosPhi}
		d := Point{X: -rx*st*cosPhi - ry*ct*sinPhi, Y: -rx*st*sinPhi + ry*ct*cosPhi}
		return p, d
	}
	segs := make([]Segment, 0, n)
	t := t1
	p0, d0 := pt(t)
	for i := 0; i < n; i++ {
		p1, d1 := pt(t + step)
		if i == n-1 {
			p1 = to
		}
		segs = append(segs, curve(
			Point{X: p0.X + alpha*d0.X, Y: p0.Y + alpha*d0.Y},
			Point{X: p1.X - alpha*d1.X, Y: p1.Y - alpha*d1.Y},
			p1,
		))
		t += step
		p0, d0 = p1, d1
	}
	return segs
}

// PathBounds is the exact bounding box of the segments, including curve
// extrema.
func PathBounds(segs []Segment) (x, y, w, h float64) {
	var pts []Point
	var cur Point
	for _, s := range segs {
		switch s.Command {
		case "move-to", "line-to":
			cur = Point{X: s.Params["x"], Y: s.Params["y"]}
			pts = append(pts, cur)
		case "curve-to":
			c1 := Point{X: s.Params["c1x"], Y: s.Params["c1y"]}
			c2 := Point{X: s.Params["c2x"], Y: s.Params["c2y"]}
			end := Point{X: s.Params["x"], Y: s.Params["y"]}
			pts = append(pts, end)
			for _, t := range cubicExtrema(cur.X, c1.X, c2.X, end.X) {
				pts = append(pts, cubicAt(cur, c1, c2, end, t))
			}
			for _, t := range cubicExtrema(cur.Y, c1.Y, c2.Y, end.Y) {
				pts = append(pts, cubicAt(cur, c1, c2, end, t))
			}
			cur = end
		}
	}
	return Bounds(pts)
}

// TranslatePath shifts every coordinate by dx, dy.
func TranslatePath(segs []Segment, dx, dy float64) []Segment {
	out := make([]Segment, len(segs))
	for i, s := range segs {
		params := make(map[string]float64, len(s.Params))
		for k, v := range s.Params {
			switch k {
			case "x", "c1x", "c2x":
				v += dx
			case "y", "c1y", "c2y":
				v += dy
			}
			params[k] = v
		}
		out[i] = Segment{Command: s.Command, Params: params}
	}
	return out
}

func cubicAt(p0, p1, p2, p3 Point, t float64) Point {
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return Point{X: a*p0.X + b*p1.X + c*p2.X + d*p3.X, Y: a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y}
}

// cubicExtrema returns the t in (0,1) where one coordinate of a cubic has a
// zero derivative.
func cubicExtrema(p0, p1, p2, p3 float64) []float64 {
	a := -p0 + 3*p1 - 3*p2 + p3
	b := 2 * (p0 - 2*p1 + p2)
	c := p1 - p0
	var ts []float64
	keep := func(t float64) {
		if t > 0 && t < 1 {
			ts = append(ts, t)
		}
	}
	if math.Abs(a) < 1e-12 {
		if math.Abs(b) > 1e-12 {
			keep(-c / b)
		}
		return ts
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return ts
	}
	sq := math.Sqrt(disc)
	keep((-b + sq) / (2 * a))
	keep((-b - sq) / (2 * a))
	return ts
}

type pathParser struct {
	s string
	i int
}

func isCommand(c byte) bool {
	switch c | 0x20 {
	case 'm', 'l', 'h', 'v', 'c', 's', 'q', 't', 'a', 'z':
		return true
	}
	return false
}

func (p *pathParser) skipSep() {
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case ' ', ',', '\t', '\n', '\r', '\f':
			p.i++
		default:
			return
		}
	}
}

func (p *pathParser) num() (float64, error) {
	p.skipSep()
	start := p.i
	if p.i < len(p.s) && (p.s[p.i] == '-' || p.s[p.i] == '+') {
		p.i++
	}
	digits, dot := false, false
	for ; p.i < len(p.s); p.i++ {
		c := p.s[p.i]
		if c >= '0' && c <= '9' {
			digits = true
		} else if c == '.' && !dot {
			dot = true
		} else {
			break
		}
	}
	if digits && p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		j := p.i + 1
		if j < len(p.s) && (p.s[j] == '-' || p.s[j] == '+') {
			j++
		}
		if j < len(p.s) && p.s[j] >= '0' && p.s[j] <= '9' {
			p.i = j
			for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
				p.i++
			}
		}
	}
	if !digits {
		return 0, fmt.Errorf("expected a number at byte %d", start)
	}
	v, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil || math.IsInf(v, 0) || math.Abs(v) > 1e7 {
		return 0, fmt.Errorf("bad number %q at byte %d", p.s[start:p.i], start)
	}
	return v, nil
}

func (p *pathParser) pair() (float64, float64, error) {
	x, err := p.num()
	if err != nil {
		return 0, 0, err
	}
	y, err := p.num()
	return x, y, err
}

// flag reads an arc flag, which may be written without a separator.
func (p *pathParser) flag() (bool, error) {
	p.skipSep()
	if p.i < len(p.s) && (p.s[p.i] == '0' || p.s[p.i] == '1') {
		p.i++
		return p.s[p.i-1] == '1', nil
	}
	return false, fmt.Errorf("expected an arc flag at byte %d", p.i)
}
