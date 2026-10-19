package penpot

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

// Box returns the geometry of an unrotated box at absolute page coordinates.
// Points run top-left, top-right, bottom-right, bottom-left.
func Box(x, y, w, h float64) Geometry {
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
