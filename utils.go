package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"

	"github.com/jinzhu/copier"
)

// -- utils

func clone[T any](obj *T) *T {
	var newObj *T = new(T)
	copier.Copy(newObj, obj)
	return newObj
}

func randFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func exit(rc int) {
	os.Exit(rc)
}

// --- 2d geometry

const (
	Pi     = 3.14159
	Pi2    = Pi * 2
	HalfPi = Pi / 2
	eps    = 1e-14
)

func square(x float64) float64 { return x * x }

func degrees(radians float64) float64 {
	return radians * 180 / Pi
}

func radians(degrees float64) float64 {
	return degrees * Pi / 180
}

func maxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Restricts a value to be within a specified range.
func clamp(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}

	return value
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	}

	return value
}

// 2D vector
type Vec2 struct {
	X, Y float64
}

func (v *Vec2) String() string {
	return fmt.Sprintf("{%0.3f,%0.3f}", v.X, v.Y)
}

func (v *Vec2) add(v2 *Vec2) *Vec2 {
	v.X += v2.X
	v.Y += v2.Y
	return v
}

func (v *Vec2) sub(v2 *Vec2) *Vec2 {
	v.X -= v2.X
	v.Y -= v2.Y
	return v
}

func (v *Vec2) copy() *Vec2 {
	return &Vec2{X: v.X, Y: v.Y}
}

func (v *Vec2) eq(v2 *Vec2) bool {
	return v.X == v2.X && v.Y == v2.Y
}

// Line implementation for Geometry applications
type Line struct {
	X1, Y1, X2, Y2 float64
}

func (l *Line) String() string {
	return fmt.Sprintf("{%0.3f,%0.3f->%0.3f,%0.3f}", l.X1, l.Y1, l.X2, l.Y2)
}

// angle gets the angle of the line
func (l *Line) angle() float64 {
	return math.Atan2(l.Y2-l.Y1, l.X2-l.X1)
}

func getAngleFromVec(dir *Vec2) float64 {
	return math.Atan2(dir.Y, dir.X)
}

// dist gets the distance between the two endpoints of the line
func (l *Line) dist() float64 {
	return getDistance(l.X1, l.Y1, l.X2, l.Y2)
}

// getDistance returns the distance between two points
func getDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(distSquared(x1, y1, x2, y2))
}

// distSquared returns the d^2 of the distance between two points
func distSquared(x1, y1, x2, y2 float64) float64 {
	return square(x2-x1) + square(y2-y1)
}

// lineFromAngle creates a line from a starting point at a given angle and length
func lineFromAngle(x, y, angleRadians, length float64) Line {
	return Line{
		X1: x,
		Y1: y,
		X2: x + (length * math.Cos(angleRadians)),
		Y2: y + (length * math.Sin(angleRadians)),
	}
}

// rect implementation for Geometry applications
func rect(x, y, w, h float64) []Line {
	return []Line{
		{x, y, x, y + h},
		{x, y + h, x + w, y + h},
		{x + w, y + h, x + w, y},
		{x + w, y, x, y},
	}
}

// lineIntersection calculates the intersection of two lines.
func lineIntersection(l1, l2 Line) (float64, float64, bool) {
	// https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection#Given_two_points_on_each_line
	denom := (l1.X1-l1.X2)*(l2.Y1-l2.Y2) - (l1.Y1-l1.Y2)*(l2.X1-l2.X2)
	tNum := (l1.X1-l2.X1)*(l2.Y1-l2.Y2) - (l1.Y1-l2.Y1)*(l2.X1-l2.X2)
	uNum := -((l1.X1-l1.X2)*(l1.Y1-l2.Y1) - (l1.Y1-l1.Y2)*(l1.X1-l2.X1))

	if denom == 0 {
		return 0, 0, false
	}

	t := tNum / denom
	if t > 1 || t < 0 {
		return 0, 0, false
	}

	u := uNum / denom
	if u > 1 || u < 0 {
		return 0, 0, false
	}

	x := l1.X1 + t*(l1.X2-l1.X1)
	y := l1.Y1 + t*(l1.Y2-l1.Y1)
	return x, y, true
}

type Circle struct {
	X, Y   float64
	Radius float64
}

// lineCircleIntersection gets the intersection points (if any) of a circle,
// and either an infinite line or a line segment.
func lineCircleIntersection(li Line, ci Circle, isSegment bool) []Vec2 {
	// https://rosettacode.org/wiki/Line_circle_intersection#Go
	var res []Vec2
	x0, y0 := ci.X, ci.Y
	x1, y1 := li.X1, li.Y1
	x2, y2 := li.X2, li.Y2
	A := y2 - y1
	B := x1 - x2
	C := x2*y1 - x1*y2
	a := square(A) + square(B)
	var b, c float64
	var bnz = true
	if math.Abs(B) >= eps { // if B isn't zero or close to it
		b = 2 * (A*C + A*B*y0 - square(B)*x0)
		c = square(C) + 2*B*C*y0 - square(B)*(square(ci.Radius)-square(x0)-square(y0))
	} else {
		b = 2 * (B*C + A*B*x0 - square(A)*y0)
		c = square(C) + 2*A*C*x0 - square(A)*(square(ci.Radius)-square(x0)-square(y0))
		bnz = false
	}
	d := square(b) - 4*a*c // discriminant
	if d < 0 {
		// line & circle don't intersect
		return res
	}

	// checks whether a point is within a segment
	within := func(x, y float64) bool {
		d1 := math.Sqrt(square(x2-x1) + square(y2-y1)) // distance between end-points
		d2 := math.Sqrt(square(x-x1) + square(y-y1))   // distance from point to one end
		d3 := math.Sqrt(square(x2-x) + square(y2-y))   // distance from point to other end
		delta := d1 - d2 - d3
		return math.Abs(delta) < eps // true if delta is less than a small tolerance
	}

	var x, y float64
	fx := func() float64 { return -(A*x + C) / B }
	fy := func() float64 { return -(B*y + C) / A }
	rxy := func() {
		if !isSegment || within(x, y) {
			res = append(res, Vec2{X: x, Y: y})
		}
	}

	if d == 0 {
		// line is tangent to circle, so just one intersect at most
		if bnz {
			x = -b / (2 * a)
			y = fx()
			rxy()
		} else {
			y = -b / (2 * a)
			x = fy()
			rxy()
		}
	} else {
		// two intersects at most
		d = math.Sqrt(d)
		if bnz {
			x = (-b + d) / (2 * a)
			y = fx()
			rxy()
			x = (-b - d) / (2 * a)
			y = fx()
			rxy()
		} else {
			y = (-b + d) / (2 * a)
			x = fy()
			rxy()
			y = (-b - d) / (2 * a)
			x = fy()
			rxy()
		}
	}
	return res
}

// circleCollision checks for collision against another circle
// and returns distance between their center points
func (c *Circle) circleCollision(c2 *Circle) (float64, bool) {
	dx := (c.X + c.Radius) - (c2.X + c2.Radius)
	dy := (c.Y + c.Radius) - (c2.Y + c2.Radius)
	distance := math.Sqrt(dx*dx + dy*dy)

	collision := false
	if distance < c.Radius+c2.Radius {
		collision = true
	}
	return distance, collision
}

// getOppositeTriangleBase gets the base length opposite the non-hypotenuse leg in a right triangle
func getOppositeTriangleBase(angle, oppositeLength float64) float64 {
	base := oppositeLength / math.Tan(angle)
	return base
}

// getOppositeTriangleLeg gets the leg length opposite the non-hypotenuse base in a right triangle
func getOppositeTriangleLeg(angle, baseLength float64) float64 {
	opposite := baseLength * math.Tan(angle)
	return opposite
}

// getAdjacentHypotenuseTriangleLeg gets the leg length adjacent the hypotenuse for angle in a right triangle
func getAdjacentHypotenuseTriangleLeg(angle, hypotenuseLength float64) float64 {
	adjacent := hypotenuseLength * math.Cos(angle)
	return adjacent
}

// --- 3d geometry

// 3D vector
type Vec3 struct {
	X, Y, Z float64
}

func (v *Vec3) String() string {
	return fmt.Sprintf("{%0.3f,%0.3f,%0.3f}", v.X, v.Y, v.Z)
}

func (v *Vec3) add(v3 *Vec3) *Vec3 {
	v.X += v3.X
	v.Y += v3.Y
	v.Z += v3.Z
	return v
}

func (v *Vec3) sub(v3 *Vec3) *Vec3 {
	v.X -= v3.X
	v.Y -= v3.Y
	v.Z -= v3.Z
	return v
}

func (v *Vec3) copy() *Vec3 {
	return &Vec3{X: v.X, Y: v.Y, Z: v.Z}
}

func (v *Vec3) eq(v3 *Vec3) bool {
	return v.X == v3.X && v.Y == v3.Y && v.Z == v3.Z
}

// Line implementation for 3-Dimensional Geometry applications
type Line3d struct {
	X1, Y1, Z1, X2, Y2, Z2 float64
}

func (l *Line3d) String() string {
	return fmt.Sprintf("{%0.3f,%0.3f,%0.3f->%0.3f,%0.3f,%0.3f}", l.X1, l.Y1, l.Z1, l.X2, l.Y2, l.Z2)
}

// heading gets the XY axis angle of the 3-dimensional line
func (l *Line3d) heading() float64 {
	return math.Atan2(l.Y2-l.Y1, l.X2-l.X1)
}

// pitch gets the Z axis angle of the 3-dimensional line
func (l *Line3d) pitch() float64 {
	return math.Atan2(l.Z2-l.Z1, math.Sqrt(square(l.X2-l.X1)+square(l.Y2-l.Y1)))
}

// dist gets the distance between the two endpoints of the 3-dimensional line
func (l *Line3d) dist() float64 {
	return math.Sqrt(square(l.X2-l.X1) + square(l.Y2-l.Y1) + square(l.Z2-l.Z1))
}

// line3dFromAngle creates a 3-Dimensional line from starting point at a heading and pitch angle, and hypotenuse length
// based on answer from https://stackoverflow.com/questions/52781607/3d-point-from-two-angles-and-a-distance
func line3dFromAngle(x, y, z, heading, pitch, length float64) Line3d {
	return Line3d{
		X1: x,
		Y1: y,
		Z1: z,
		X2: x + (length * math.Cos(heading) * math.Cos(pitch)),
		Y2: y + (length * math.Sin(heading) * math.Cos(pitch)),
		Z2: z + (length * math.Sin(pitch)),
	}
}

// line3dFromBaseAngle creates a 3-Dimensional line from starting point at a heading and pitch angle, and XY axis length
func line3dFromBaseAngle(x, y, z, heading, pitch, xyLength float64) Line3d {
	return Line3d{
		X1: x,
		Y1: y,
		Z1: z,
		X2: x + (xyLength * math.Cos(heading)),
		Y2: y + (xyLength * math.Sin(heading)),
		Z2: z + (xyLength * math.Tan(pitch)),
	}
}

// comb sort
func combSort(order []int, dist []float64, amount int) {
	gap := amount
	swapped := false
	for gap > 1 || swapped {
		gap = (gap * 10) / 13
		if gap == 9 || gap == 10 {
			gap = 11
		}
		if gap < 1 {
			gap = 1
		}
		swapped = false
		for i := 0; i < amount-gap; i++ {
			j := i + gap
			if dist[i] < dist[j] {
				dist[i], dist[j] = dist[j], dist[i]
				order[i], order[j] = order[j], order[i]
				swapped = true
			}
		}
	}
}
