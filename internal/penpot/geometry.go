package penpot

import "math"

// Point is {x, y}.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rect is a selrect: the unrotated bounding box with its corners.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	X1     float64 `json:"x1"`
	Y1     float64 `json:"y1"`
	X2     float64 `json:"x2"`
	Y2     float64 `json:"y2"`
}

// Matrix is an affine transform {a, b, c, d, e, f}.
type Matrix struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
	C float64 `json:"c"`
	D float64 `json:"d"`
	E float64 `json:"e"`
	F float64 `json:"f"`
}

// Identity is the matrix with no rotation, scale or shift.
var Identity = Matrix{A: 1, D: 1}

// Geometry holds every geometry attribute Penpot stores for a shape. The
// backend recomputes none of them, so they must agree with each other.
type Geometry struct {
	X                float64 `json:"x"`
	Y                float64 `json:"y"`
	Width            float64 `json:"width"`
	Height           float64 `json:"height"`
	Rotation         float64 `json:"rotation"`
	Selrect          Rect    `json:"selrect"`
	Points           []Point `json:"points"`
	Transform        Matrix  `json:"transform"`
	TransformInverse Matrix  `json:"transformInverse"`
}

// BoxGeometry returns the geometry of an unrotated box at absolute page coordinates.
// Points run top-left, top-right, bottom-right, bottom-left.
func BoxGeometry(x, y, w, h float64) Geometry {
	return Geometry{
		X: x, Y: y, Width: w, Height: h,
		Selrect: Rect{X: x, Y: y, Width: w, Height: h, X1: x, Y1: y, X2: x + w, Y2: y + h},
		Points: []Point{
			{X: x, Y: y}, {X: x + w, Y: y}, {X: x + w, Y: y + h}, {X: x, Y: y + h},
		},
		Transform:        Identity,
		TransformInverse: Identity,
	}
}

// RotatedGeometry is BoxGeometry turned deg degrees clockwise about the box
// centre. The selrect stays the unrotated box; points are the turned corners.
func RotatedGeometry(x, y, w, h, deg float64) Geometry {
	g := BoxGeometry(x, y, w, h)
	deg = math.Mod(deg, 360)
	if deg < 0 {
		deg += 360
	}
	if deg == 0 {
		return g
	}
	rad := deg * math.Pi / 180
	c, s := math.Cos(rad), math.Sin(rad)
	cx, cy := x+w/2, y+h/2
	for i, p := range g.Points {
		dx, dy := p.X-cx, p.Y-cy
		g.Points[i] = Point{X: cx + c*dx - s*dy, Y: cy + s*dx + c*dy}
	}
	g.Rotation = deg
	g.Transform = Matrix{A: c, B: s, C: -s, D: c}
	g.TransformInverse = Matrix{A: c, B: -s, C: s, D: c}
	return g
}

// Bounds is the smallest unrotated box holding every point.
func Bounds(points []Point) (x, y, w, h float64) {
	if len(points) == 0 {
		return 0, 0, 0, 0
	}
	x1, y1, x2, y2 := points[0].X, points[0].Y, points[0].X, points[0].Y
	for _, p := range points[1:] {
		x1, y1 = math.Min(x1, p.X), math.Min(y1, p.Y)
		x2, y2 = math.Max(x2, p.X), math.Max(y2, p.Y)
	}
	return x1, y1, x2 - x1, y2 - y1
}
