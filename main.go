package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/jinzhu/copier"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// -- 2d geometry

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
func lineFromAngle(x, y, angle, length float64) Line {
	return Line{
		X1: x,
		Y1: y,
		X2: x + (length * math.Cos(angle)),
		Y2: y + (length * math.Sin(angle)),
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

// -- 3d geometry

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

// -- camera

const (
	maxConcurrent = 100
)

type Camera struct {
	pos                       *Vec2
	camZ                      float64
	posZ                      float64
	dir                       *Vec2
	headingAngle              float64
	plane                     *Vec2
	w                         int
	h                         int
	pitch                     int
	pitchAngle                float64
	fovAngle, fovDepth        float64
	mapObj                    *Map
	mapWidth                  int
	mapHeight                 int
	floor                     *ebiten.Image
	sky                       *ebiten.Image
	texSize                   int
	levels                    []*Level
	floorLvl                  *HorizontalLevel
	slices                    []*image.Rectangle
	zBuffer                   []float64
	sprites                   []*Sprite
	spriteLvls                []*Level
	spriteOrder               []int
	spriteDistance            []float64
	tex                       *TextureManager
	lightFalloff              float64
	globalIllumination        float64
	minLightRGB               color.NRGBA
	maxLightRGB               color.NRGBA
	renderDistance            float64
	convergenceDistance       float64
	convergencePoint          *Vec3
	convergenceSprite         *Sprite
	alwaysSetSpriteScreenRect bool
	semaphore                 chan struct{}
}

func NewCamera(width int, height int, texSize int, mapObj *Map, tex *TextureManager) *Camera {
	c := &Camera{}
	c.mapObj = mapObj
	firstLevel := mapObj.Level(0)
	c.mapWidth = len(firstLevel)
	c.mapHeight = len(firstLevel[0])
	c.pos = &Vec2{X: 1.0, Y: 1.0}
	c.camZ = 0.0
	c.SetHeadingAngle(0)
	c.SetPitchAngle(0)
	fovDegrees := 70.0
	fovDepth := 1.0
	c.SetFovAngle(fovDegrees, fovDepth)
	c.SetRenderDistance(-1)
	c.SetLightFalloff(-100)
	c.SetGlobalIllumination(300)
	c.SetLightRGB(color.NRGBA{R: 0, G: 0, B: 0}, color.NRGBA{R: 255, G: 255, B: 255})
	c.texSize = texSize
	c.tex = tex
	c.SetViewSize(width, height)
	c.sprites = []*Sprite{}
	c.updateSpriteLevels(16)
	c.semaphore = make(chan struct{}, maxConcurrent)
	c.convergenceDistance = -1
	c.convergencePoint = nil
	c.convergenceSprite = nil
	c.raycast()
	return c
}

// Draw the raycasted camera view to the screen.
func (c *Camera) Draw(screen *ebiten.Image) {
	//--draw basic sky and floor--//
	texRect := image.Rect(0, 0, c.texSize, c.texSize)
	lightingRGBA := &color.RGBA{R: c.maxLightRGB.R, G: c.maxLightRGB.G, B: c.maxLightRGB.B, A: 255}

	floorRect := image.Rect(0, int(float64(c.h)*0.5)+c.pitch,
		c.w, c.h)
	drawTexture(screen, c.floor, &floorRect, &texRect, lightingRGBA)

	skyRect := image.Rect(0, 0, c.w, int(float64(c.h)*0.5)+c.pitch)
	drawTexture(screen, c.sky, &skyRect, &texRect, lightingRGBA)

	//--draw walls--//
	for x := 0; x < c.w; x++ {
		for i := cap(c.levels) - 1; i >= 0; i-- {
			drawTexture(screen, c.levels[i].CurrTex[x], c.levels[i].Sv[x], c.levels[i].Cts[x], c.levels[i].St[x])
		}
	}

	// draw textured floor
	if c.floorLvl != nil && c.floorLvl.image != nil {
		c.floorLvl.image.WritePixels(c.floorLvl.horBuffer.Pix)

		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(c.floorLvl.image, op)
	}

	// draw sprites
	for x := 0; x < c.w; x++ {
		for i := 0; i < cap(c.spriteLvls); i++ {
			spriteLvl := c.spriteLvls[i]
			if spriteLvl == nil {
				continue
			}

			texture := spriteLvl.CurrTex[x]
			if texture != nil {
				drawTexture(screen, texture, spriteLvl.Sv[x], spriteLvl.Cts[x], spriteLvl.St[x])
			}
		}
	}
}

func (c *Camera) SetViewSize(width, height int) {
	c.w = width
	c.h = height
	c.levels = c.createLevels(c.mapObj.NumLevels())
	c.slices = makeSlices(c.texSize, c.texSize, 0, 0)
	c.floorLvl = c.createFloorLevel()
	c.zBuffer = make([]float64, width)
}
func (c *Camera) ViewSize() (int, int) {
	return c.w, c.h
}
func (c *Camera) SetFovAngle(fovDegrees, fovDepth float64) {
	c.fovAngle = radians(fovDegrees)
	c.fovDepth = fovDepth
	var headingAngle float64 = 0
	if c.dir != nil {
		headingAngle = c.getAngleFromVec(c.dir)
	}
	c.dir = c.getVecForAngle(headingAngle)
	c.plane = c.getVecForFov(c.dir)
}
func (c *Camera) FovRadians() float64 {
	return c.fovAngle
}
func (c *Camera) FovAngle() float64 {
	return degrees(c.fovAngle)
}
func (c *Camera) FovRadiansVertical() float64 {
	return 2 * math.Atan(math.Tan(c.fovAngle/2)*(float64(c.h)/float64(c.w)))
}
func (c *Camera) FovAngleVertical() float64 {
	return degrees(c.FovRadiansVertical())
}
func (c *Camera) FovDepth() float64 {
	return c.fovDepth
}
func (c *Camera) SetFloorTexture(floor *ebiten.Image) {
	c.floor = floor
}
func (c *Camera) SetSkyTexture(sky *ebiten.Image) {
	c.sky = sky
}
func (c *Camera) SetRenderDistance(distance float64) {
	if distance < 0 {
		c.renderDistance = math.MaxFloat64
	} else {
		c.renderDistance = distance
	}
}
func (c *Camera) SetLightFalloff(falloff float64) {
	c.lightFalloff = falloff
}
func (c *Camera) SetGlobalIllumination(illumination float64) {
	c.globalIllumination = illumination
}
func (c *Camera) SetLightRGB(min, max color.NRGBA) {
	c.minLightRGB = min
	c.maxLightRGB = max
}
func (c *Camera) SetAlwaysSetSpriteScreenRect(b bool) {
	c.alwaysSetSpriteScreenRect = b
}
func (c *Camera) Update(sprites []*Sprite) {
	c.floorLvl.initialize(c.w, c.h)
	c.convergenceDistance = -1
	c.convergencePoint = nil
	c.convergenceSprite = nil
	if len(sprites) != len(c.sprites) {
		c.updateSpriteLevels(len(sprites))
	} else {
		c.clearAllSpriteLevels()
	}
	c.sprites = sprites
	c.raycast()
}
func (c *Camera) raycast() {
	numLevels := c.mapObj.NumLevels()
	var wg sync.WaitGroup
	for i := 0; i < numLevels; i++ {
		wg.Add(1)
		go c.asyncCastLevel(i, &wg)
	}
	wg.Wait()
	numSprites := len(c.sprites)
	c.spriteOrder = make([]int, numSprites)
	c.spriteDistance = make([]float64, numSprites)
	for i := 0; i < numSprites; i++ {
		sprite := c.sprites[i]
		c.spriteOrder[i] = i
		c.spriteDistance[i] = math.Sqrt(math.Pow(c.pos.X-sprite.Pos().X, 2) + math.Pow(c.pos.Y-sprite.Pos().Y, 2))
	}
	combSort(c.spriteOrder, c.spriteDistance, numSprites)
	for i := 0; i < numSprites; i++ {
		wg.Add(1)
		go c.asyncCastSprite(i, &wg)
	}
	wg.Wait()
}
func (c *Camera) asyncCastLevel(levelNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	rMap := c.mapObj.Level(levelNum)
	for x := 0; x < c.w; x++ {
		c.castLevel(x, rMap, c.levels[levelNum], levelNum, wg)
	}
}
func (c *Camera) asyncCastSprite(spriteNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	c.semaphore <- struct{}{}
	defer func() {
		<-c.semaphore
	}()
	c.castSprite(spriteNum)
}
func (c *Camera) castLevel(x int, grid [][]int, lvl *Level, levelNum int, wg *sync.WaitGroup) {
	var _cts, _sv []*image.Rectangle
	var _st []*color.RGBA
	_cts = lvl.Cts
	_sv = lvl.Sv
	_st = lvl.St
	cameraX := 2.0*float64(x)/float64(c.w) - 1.0
	rayDirX := c.dir.X + c.plane.X*cameraX
	rayDirY := c.dir.Y + c.plane.Y*cameraX
	rayPosX := c.pos.X
	rayPosY := c.pos.Y
	mapX := int(rayPosX)
	mapY := int(rayPosY)
	var sideDistX float64
	var sideDistY float64
	deltaDistX := math.Abs(1 / rayDirX)
	deltaDistY := math.Abs(1 / rayDirY)
	var perpWallDist float64
	var stepX int
	var stepY int
	hit := 0
	side := -1
	if rayDirX < 0 {
		stepX = -1
		sideDistX = (rayPosX - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1.0 - rayPosX) * deltaDistX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (rayPosY - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1.0 - rayPosY) * deltaDistY
	}
	for hit == 0 {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}
		if side == 0 {
			perpWallDist = sideDistX - deltaDistX
		} else {
			perpWallDist = sideDistY - deltaDistY
		}
		if mapX >= 0 && mapY >= 0 && mapX < c.mapWidth && mapY < c.mapHeight {
			if perpWallDist > c.renderDistance {
				hit = 2
			} else if perpWallDist <= c.renderDistance && grid[mapX][mapY] > 0 {
				hit = 1
			}
		} else {
			hit = 2
		}
	}
	lineHeight := int(float64(c.h) / perpWallDist)
	drawStart := (-lineHeight/2 + c.h/2) + c.pitch + int(c.camZ/perpWallDist) - lineHeight*levelNum
	drawEnd := drawStart + lineHeight
	var wallX float64
	if side == 0 {
		wallX = rayPosY + perpWallDist*rayDirY
	} else {
		wallX = rayPosX + perpWallDist*rayDirX
	}
	wallX -= math.Floor(wallX)
	var texture *ebiten.Image
	if hit == 1 && mapX >= 0 && mapY >= 0 && mapX < c.mapWidth && mapY < c.mapHeight {
		texture = c.tex.TextureAt(mapX, mapY, levelNum, side)
	}
	c.levels[levelNum].CurrTex[x] = texture
	if texture != nil {
		texX := int(wallX * float64(c.texSize))
		if side == 0 && rayDirX > 0 {
			texX = c.texSize - texX - 1
		}
		if side == 1 && rayDirY < 0 {
			texX = c.texSize - texX - 1
		}
		_cts[x] = c.slices[texX]
		_sv[x].Min.Y = drawStart
		_sv[x].Max.Y = drawEnd
		shadowDepth := math.Sqrt(perpWallDist) * c.lightFalloff
		_st[x] = &color.RGBA{255, 255, 255, 255}
		_st[x].R = byte(clampInt(int(float64(_st[x].R)+shadowDepth+c.globalIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
		_st[x].G = byte(clampInt(int(float64(_st[x].G)+shadowDepth+c.globalIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
		_st[x].B = byte(clampInt(int(float64(_st[x].B)+shadowDepth+c.globalIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
		if side == 0 {
			wallDiff := 12
			_st[x].R = byte(clampInt(int(_st[x].R)-wallDiff, 0, 255))
			_st[x].G = byte(clampInt(int(_st[x].G)-wallDiff, 0, 255))
			_st[x].B = byte(clampInt(int(_st[x].B)-wallDiff, 0, 255))
		}
	}
	convergenceCol, convergenceRow := c.w/2-1, c.h/2-1
	if x == convergenceCol && drawStart <= convergenceRow && convergenceRow <= drawEnd {
		convergencePerpDist := perpWallDist * c.fovDepth
		convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
		convergenceDistance := convergenceLine3d.dist()
		if c.convergenceDistance == -1 || convergenceDistance < c.convergenceDistance {
			c.convergenceDistance = convergenceDistance
			c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
		}
	}
	if levelNum == 0 {
		c.zBuffer[x] = perpWallDist
	}
	if levelNum == 0 {
		if drawEnd < 0 {
			drawEnd = c.h
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			var floorXWall, floorYWall float64
			if side == 0 && rayDirX > 0 {
				floorXWall = float64(mapX)
				floorYWall = float64(mapY) + wallX
			} else if side == 0 && rayDirX < 0 {
				floorXWall = float64(mapX) + 1.0
				floorYWall = float64(mapY) + wallX
			} else if side == 1 && rayDirY > 0 {
				floorXWall = float64(mapX) + wallX
				floorYWall = float64(mapY)
			} else {
				floorXWall = float64(mapX) + wallX
				floorYWall = float64(mapY) + 1.0
			}
			var distWall, distPlayer, currentDist float64
			distWall = perpWallDist
			distPlayer = 0.0
			for y := drawEnd; y < c.h; y++ {
				currentDist = (float64(c.h) + (2.0 * c.camZ)) / (2.0*float64(y-c.pitch) - float64(c.h))
				if currentDist > c.renderDistance {
					continue
				}
				weight := (currentDist - distPlayer) / (distWall - distPlayer)
				currentFloorX := weight*floorXWall + (1.0-weight)*rayPosX
				currentFloorY := weight*floorYWall + (1.0-weight)*rayPosY
				if currentFloorX < 0 || currentFloorY < 0 || int(currentFloorX) >= c.mapWidth || int(currentFloorY) >= c.mapHeight {
					continue
				}
				if x == convergenceCol && y == convergenceRow {
					convergencePerpDist := currentDist * c.fovDepth
					convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
					convergenceDistance := convergenceLine3d.dist()
					if c.convergenceDistance == 0 || convergenceDistance < c.convergenceDistance {
						c.convergenceDistance = convergenceDistance
						c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
					}
				}
				floorTex := c.tex.FloorTextureAt(int(currentFloorX), int(currentFloorY))
				if floorTex == nil {
					continue
				}
				floorTexX := int(currentFloorX*float64(c.texSize)) % c.texSize
				floorTexY := int(currentFloorY*float64(c.texSize)) % c.texSize
				pxOffset := floorTex.PixOffset(floorTexX, floorTexY)
				if pxOffset < 0 {
					continue
				}
				pixel := color.RGBA{floorTex.Pix[pxOffset],
					floorTex.Pix[pxOffset+1],
					floorTex.Pix[pxOffset+2],
					floorTex.Pix[pxOffset+3]}
				pixelSt := &color.RGBA{R: 255, G: 255, B: 255}
				shadowDepth := math.Sqrt(currentDist) * c.lightFalloff
				pixelSt.R = byte(clampInt(int(float64(pixelSt.R)+shadowDepth+c.globalIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
				pixelSt.G = byte(clampInt(int(float64(pixelSt.G)+shadowDepth+c.globalIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
				pixelSt.B = byte(clampInt(int(float64(pixelSt.B)+shadowDepth+c.globalIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
				pixel.R = uint8(float64(pixel.R) * float64(pixelSt.R) / 256)
				pixel.G = uint8(float64(pixel.G) * float64(pixelSt.G) / 256)
				pixel.B = uint8(float64(pixel.B) * float64(pixelSt.B) / 256)
				pxOffset = c.floorLvl.horBuffer.PixOffset(x, y)
				c.floorLvl.horBuffer.Pix[pxOffset] = pixel.R
				c.floorLvl.horBuffer.Pix[pxOffset+1] = pixel.G
				c.floorLvl.horBuffer.Pix[pxOffset+2] = pixel.B
				c.floorLvl.horBuffer.Pix[pxOffset+3] = pixel.A
			}
		}()
	}
}
func (c *Camera) castSprite(spriteOrdIndex int) {
	sprite := c.sprites[c.spriteOrder[spriteOrdIndex]]
	spriteDist := c.spriteDistance[spriteOrdIndex]
	if spriteDist > c.renderDistance && !c.alwaysSetSpriteScreenRect {
		sprite.SetScreenRect(nil)
		return
	}
	renderSprite := false
	spriteX := sprite.Pos().X - c.pos.X
	spriteY := sprite.Pos().Y - c.pos.Y
	spriteTex := sprite.Texture()
	spriteTexRect := sprite.TextureRect()
	spriteTexWidth, spriteTexHeight := spriteTex.Bounds().Dx(), spriteTex.Bounds().Dy()
	spriteTexRatioWH := float64(spriteTexWidth) / float64(spriteTexHeight)
	spriteIllumination := sprite.Illumination()
	invDet := 1.0 / (c.plane.X*c.dir.Y - c.dir.X*c.plane.Y)
	transformX := invDet * (c.dir.Y*spriteX - c.dir.X*spriteY)
	transformY := invDet * (-c.plane.Y*spriteX + c.plane.X*spriteY)
	spriteScreenX := int(float64(c.w) / 2 * (1 + transformX/transformY))
	spriteScale := sprite.Scale()
	spriteAnchor := sprite.VerticalAnchor()
	var uDiv float64 = 1 / (spriteScale * spriteTexRatioWH)
	var vDiv float64 = 1 / spriteScale
	var vOffset float64 = getAnchorVerticalOffset(spriteAnchor, spriteScale, c.h)
	var vMove float64 = -sprite.PosZ()*float64(c.h) + vOffset
	vMoveScreen := int(vMove/transformY) + c.pitch + int(c.camZ/transformY)
	spriteHeight := int(math.Abs(float64(c.h)/transformY) / vDiv)
	drawStartY := -spriteHeight/2 + c.h/2 + vMoveScreen
	if drawStartY < 0 {
		drawStartY = 0
	}
	drawEndY := spriteHeight/2 + c.h/2 + vMoveScreen
	if drawEndY >= c.h {
		drawEndY = c.h - 1
	}
	spriteWidth := int(math.Abs(float64(c.h)/transformY) / uDiv)
	drawStartX := -spriteWidth/2 + spriteScreenX
	drawEndX := spriteWidth/2 + spriteScreenX
	if spriteWidth == 0 || spriteHeight == 0 || transformY <= 0 || drawStartX < -spriteWidth || drawEndX >= c.w+spriteWidth {
		sprite.SetScreenRect(nil)
		return
	}
	if drawStartX < 0 {
		drawStartX = 0
	}
	if drawEndX >= c.w {
		drawEndX = c.w - 1
	}
	canConverge := sprite.IsFocusable()
	convergenceCol, convergenceRow := c.w/2-1, c.h/2-1
	d := (drawStartY-vMoveScreen)*256 - c.h*128 + spriteHeight*128
	texStartY := ((d * spriteTexHeight) / spriteHeight) / 256
	d = (drawEndY-1-vMoveScreen)*256 - c.h*128 + spriteHeight*128
	texEndY := ((d * spriteTexHeight) / spriteHeight) / 256
	var spriteSlices []*image.Rectangle
	if !c.alwaysSetSpriteScreenRect || spriteDist <= c.renderDistance {
		for stripe := drawStartX; stripe < drawEndX; stripe++ {
			if transformY > 0 && stripe > 0 && stripe < c.w && transformY < c.zBuffer[stripe] {
				var spriteLvl *Level
				if !renderSprite {
					renderSprite = true
					spriteLvl = c.makeSpriteLevel(spriteOrdIndex)
					spriteSlices = makeSlices(spriteTexWidth, spriteTexHeight, spriteTexRect.Min.X, spriteTexRect.Min.Y)
				} else {
					spriteLvl = c.spriteLvls[spriteOrdIndex]
				}
				texX := int(256*(stripe-(-spriteWidth/2+spriteScreenX))*spriteTexWidth/spriteWidth) / 256
				if texX < 0 || texX >= cap(spriteSlices) {
					continue
				}
				if canConverge && stripe == convergenceCol && drawStartY <= convergenceRow && convergenceRow <= drawEndY {
					convergencePerpDist := spriteDist
					convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
					convergenceDistance := convergenceLine3d.dist()
					if c.convergenceDistance == -1 || convergenceDistance < c.convergenceDistance {
						c.convergenceDistance = convergenceDistance
						c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
						c.convergenceSprite = sprite
					}
				}
				spriteLvl.Cts[stripe] = spriteSlices[texX]
				spriteLvl.Cts[stripe].Min.Y = spriteTexRect.Min.Y + texStartY
				spriteLvl.Cts[stripe].Max.Y = spriteTexRect.Min.Y + texEndY + 1
				spriteLvl.CurrTex[stripe] = spriteTex
				spriteLvl.Sv[stripe].Min.Y = drawStartY
				spriteLvl.Sv[stripe].Max.Y = drawEndY
				shadowDepth := math.Sqrt(transformY) * c.lightFalloff
				spriteLvl.St[stripe] = &color.RGBA{255, 255, 255, 255}
				spriteLvl.St[stripe].R = byte(clampInt(int(float64(spriteLvl.St[stripe].R)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
				spriteLvl.St[stripe].G = byte(clampInt(int(float64(spriteLvl.St[stripe].G)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
				spriteLvl.St[stripe].B = byte(clampInt(int(float64(spriteLvl.St[stripe].B)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
			}
		}
	}
	if renderSprite || c.alwaysSetSpriteScreenRect {
		spriteCastRect := image.Rect(drawStartX, drawStartY, drawEndX, drawEndY)
		sprite.SetScreenRect(&spriteCastRect)
	} else {
		c.clearSpriteLevel(spriteOrdIndex)
		sprite.SetScreenRect(nil)
	}
}
func makeSlices(width, height, xOffset, yOffset int) []*image.Rectangle {
	newSlices := make([]*image.Rectangle, width)
	for x := 0; x < width; x++ {
		thisRect := image.Rect(xOffset+x, yOffset, xOffset+x+1, yOffset+height)
		newSlices[x] = &thisRect
	}
	return newSlices
}
func (c *Camera) createLevels(numLevels int) []*Level {
	levelArr := make([]*Level, numLevels)
	for i := 0; i < numLevels; i++ {
		levelArr[i] = new(Level)
		levelArr[i].Sv = sliceView(c.w, c.h)
		levelArr[i].Cts = make([]*image.Rectangle, c.w)
		levelArr[i].St = make([]*color.RGBA, c.w)
		levelArr[i].CurrTex = make([]*ebiten.Image, c.w)
	}
	return levelArr
}
func (c *Camera) createFloorLevel() *HorizontalLevel {
	horizontalLevel := new(HorizontalLevel)
	horizontalLevel.initialize(c.w, c.h)
	return horizontalLevel
}
func (c *Camera) updateSpriteLevels(spriteCapacity int) {
	if c.spriteLvls != nil {
		capacity := len(c.spriteLvls)
		if spriteCapacity <= capacity {
			c.clearAllSpriteLevels()
			return
		}
		for capacity <= spriteCapacity {
			capacity *= 2
		}
		spriteCapacity = capacity
	}
	c.spriteLvls = make([]*Level, spriteCapacity)
}
func (c *Camera) makeSpriteLevel(spriteOrdIndex int) *Level {
	spriteLvl := new(Level)
	spriteLvl.Sv = sliceView(c.w, c.h)
	spriteLvl.Cts = make([]*image.Rectangle, c.w)
	spriteLvl.St = make([]*color.RGBA, c.w)
	spriteLvl.CurrTex = make([]*ebiten.Image, c.w)
	c.spriteLvls[spriteOrdIndex] = spriteLvl
	return spriteLvl
}
func (c *Camera) clearAllSpriteLevels() {
	for i := 0; i < len(c.spriteLvls); i++ {
		c.clearSpriteLevel(i)
	}
}
func (c *Camera) clearSpriteLevel(spriteOrdIndex int) {
	c.spriteLvls[spriteOrdIndex] = nil
}
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
func (c *Camera) SetPosition(pos *Vec2) {
	c.pos = pos
}
func (c *Camera) GetPosition() *Vec2 {
	return c.pos
}
func (c *Camera) SetPositionZ(gridPosZ float64) {
	c.posZ = gridPosZ
	c.camZ = (gridPosZ - 0.5) * float64(c.h)
}
func (c *Camera) GetPositionZ() float64 {
	return c.posZ
}
func (c *Camera) SetHeadingAngle(headingAngle float64) {
	c.headingAngle = headingAngle
	cameraDir := c.getVecForAngle(headingAngle)
	c.dir = cameraDir
	c.plane = c.getVecForFov(cameraDir)
}
func (c *Camera) SetPitchAngle(pitchAngle float64) {
	c.pitchAngle = pitchAngle
	cameraPitch := getOppositeTriangleLeg(pitchAngle, float64(c.h)*c.fovDepth)
	c.pitch = clampInt(int(cameraPitch), -c.h/2, int(float64(c.h)*c.fovDepth))
}
func (c *Camera) getAngleFromVec(dir *Vec2) float64 {
	return math.Atan2(dir.Y, dir.X)
}
func (c *Camera) getVecForAngleLength(angle, length float64) *Vec2 {
	return &Vec2{X: length * math.Cos(angle), Y: length * math.Sin(angle)}
}
func (c *Camera) getVecForAngle(angle float64) *Vec2 {
	return &Vec2{X: c.fovDepth * math.Cos(angle), Y: c.fovDepth * math.Sin(angle)}
}
func (c *Camera) getVecForFov(dir *Vec2) *Vec2 {
	angle := c.getAngleFromVec(dir)
	length := math.Sqrt(math.Pow(dir.X, 2) + math.Pow(dir.Y, 2))
	hypotenuse := length / math.Cos(c.fovAngle/2)
	return dir.copy().sub(c.getVecForAngleLength(angle+c.fovAngle/2, hypotenuse))
}
func (c *Camera) GetConvergenceDistance() float64 {
	return c.convergenceDistance
}
func (c *Camera) GetConvergencePoint() *Vec3 {
	return c.convergencePoint
}
func (c *Camera) GetConvergenceSprite() *Sprite {
	return c.convergenceSprite
}

// -- collisions

type Collision struct {
	entity     *Entity
	collision  *Vec2
	collisionZ float64
}

// checks for valid move from current position, returns valid (x, y) position, whether a collision
// was encountered, and a list of entity collisions that may have been encountered
func (g *Game) getValidMove(entity *Entity, moveX, moveY, moveZ float64, checkAlternate bool) (*Vec2, bool, []*Collision) {
	posX, posY, posZ := entity.Position.X, entity.Position.Y, entity.PositionZ
	if posX == moveX && posY == moveY && posZ == moveZ {
		return &Vec2{X: posX, Y: posY}, false, []*Collision{}
	}

	newX, newY, newZ := moveX, moveY, moveZ
	moveLine := Line{X1: posX, Y1: posY, X2: newX, Y2: newY}

	intersectPoints := []Vec2{}
	collisionEntities := []*Collision{}

	// check wall collisions
	for _, borderLine := range g.collisionMap {
		// TODO: only check intersection of nearby wall cells instead of all of them
		if px, py, ok := lineIntersection(moveLine, borderLine); ok {
			intersectPoints = append(intersectPoints, Vec2{X: px, Y: py})
		}
	}

	// check sprite against player collision
	if entity != g.player.Entity && entity.Parent != g.player.Entity && entity.CollisionRadius > 0 {
		// TODO: only check for collision if player is somewhat nearby

		// quick check if intersects in Z-plane
		zIntersect := zEntityIntersection(newZ, entity, g.player.Entity)

		// check if movement line intersects with combined collision radii
		combinedCircle := Circle{X: g.player.Position.X, Y: g.player.Position.Y, Radius: g.player.CollisionRadius + entity.CollisionRadius}
		combinedIntersects := lineCircleIntersection(moveLine, combinedCircle, true)

		if zIntersect >= 0 && len(combinedIntersects) > 0 {
			playerCircle := Circle{X: g.player.Position.X, Y: g.player.Position.Y, Radius: g.player.CollisionRadius}
			for _, chkPoint := range combinedIntersects {
				// intersections from combined circle radius indicate center point to check intersection toward sprite collision circle
				chkLine := Line{X1: chkPoint.X, Y1: chkPoint.Y, X2: g.player.Position.X, Y2: g.player.Position.Y}
				intersectPoints = append(intersectPoints, lineCircleIntersection(chkLine, playerCircle, true)...)

				for _, intersect := range intersectPoints {
					collisionEntities = append(
						collisionEntities, &Collision{entity: g.player.Entity, collision: &intersect, collisionZ: zIntersect},
					)
				}
			}
		}
	}

	// check sprite collisions
	for sprite := range g.sprites {
		// TODO: only check intersection of nearby sprites instead of all of them
		if entity == sprite.Entity || entity.Parent == sprite.Entity || entity.CollisionRadius <= 0 || sprite.CollisionRadius <= 0 {
			continue
		}

		// quick check if intersects in Z-plane
		zIntersect := zEntityIntersection(newZ, entity, sprite.Entity)

		// check if movement line intersects with combined collision radii
		combinedCircle := Circle{X: sprite.Position.X, Y: sprite.Position.Y, Radius: sprite.CollisionRadius + entity.CollisionRadius}
		combinedIntersects := lineCircleIntersection(moveLine, combinedCircle, true)

		if zIntersect >= 0 && len(combinedIntersects) > 0 {
			spriteCircle := Circle{X: sprite.Position.X, Y: sprite.Position.Y, Radius: sprite.CollisionRadius}
			for _, chkPoint := range combinedIntersects {
				// intersections from combined circle radius indicate center point to check intersection toward sprite collision circle
				chkLine := Line{X1: chkPoint.X, Y1: chkPoint.Y, X2: sprite.Position.X, Y2: sprite.Position.Y}
				intersectPoints = append(intersectPoints, lineCircleIntersection(chkLine, spriteCircle, true)...)

				for _, intersect := range intersectPoints {
					collisionEntities = append(
						collisionEntities, &Collision{entity: sprite.Entity, collision: &intersect, collisionZ: zIntersect},
					)
				}
			}
		}
	}

	// sort collisions by distance to current entity position
	sort.Slice(collisionEntities, func(i, j int) bool {
		distI := distSquared(posX, posY, collisionEntities[i].collision.X, collisionEntities[i].collision.Y)
		distJ := distSquared(posX, posY, collisionEntities[j].collision.X, collisionEntities[j].collision.Y)
		return distI < distJ
	})

	isCollision := len(intersectPoints) > 0

	if isCollision {
		if checkAlternate {
			// find the point closest to the start position
			min := math.Inf(1)
			minI := -1
			for i, p := range intersectPoints {
				d2 := distSquared(posX, posY, p.X, p.Y)
				if d2 < min {
					min = d2
					minI = i
				}
			}

			// use the closest intersecting point to determine a safe distance to make the move
			moveLine = Line{X1: posX, Y1: posY, X2: intersectPoints[minI].X, Y2: intersectPoints[minI].Y}
			dist := math.Sqrt(min)
			angle := moveLine.angle()

			// generate new move line using calculated angle and safe distance from intersecting point
			moveLine = lineFromAngle(posX, posY, angle, dist-0.01)

			newX, newY = moveLine.X2, moveLine.Y2

			// if either X or Y direction was already intersecting, attempt move only in the adjacent direction
			xDiff := math.Abs(newX - posX)
			yDiff := math.Abs(newY - posY)
			if xDiff > 0.001 || yDiff > 0.001 {
				switch {
				case xDiff <= 0.001:
					// no more room to move in X, try to move only Y
					// fmt.Printf("\t[@%v,%v] move to (%v,%v) try adjacent move to {%v,%v}\n",
					// 	c.pos.X, c.pos.Y, moveX, moveY, posX, moveY)
					return g.getValidMove(entity, posX, moveY, posZ, false)
				case yDiff <= 0.001:
					// no more room to move in Y, try to move only X
					// fmt.Printf("\t[@%v,%v] move to (%v,%v) try adjacent move to {%v,%v}\n",
					// 	c.pos.X, c.pos.Y, moveX, moveY, moveX, posY)
					return g.getValidMove(entity, moveX, posY, posZ, false)
				default:
					// try the new position
					// TODO: need some way to try a potentially valid shorter move without checkAlternate while also avoiding infinite loop
					return g.getValidMove(entity, newX, newY, posZ, false)
				}
			} else {
				// looks like it cannot move
				return &Vec2{X: posX, Y: posY}, isCollision, collisionEntities
			}
		} else {
			// looks like it cannot move
			return &Vec2{X: posX, Y: posY}, isCollision, collisionEntities
		}
	}

	// prevent index out of bounds errors
	ix := int(newX)
	iy := int(newY)

	switch {
	case ix < 0 || newX < 0:
		newX = clipDistance
		ix = 0
	case ix >= g.mapWidth:
		newX = float64(g.mapWidth) - clipDistance
		ix = int(newX)
	}

	switch {
	case iy < 0 || newY < 0:
		newY = clipDistance
		iy = 0
	case iy >= g.mapHeight:
		newY = float64(g.mapHeight) - clipDistance
		iy = int(newY)
	}

	worldMap := g.mapObj.Level(0)
	if worldMap[ix][iy] <= 0 {
		posX = newX
		posY = newY
	} else {
		isCollision = true
	}

	return &Vec2{X: posX, Y: posY}, isCollision, collisionEntities
}

// zEntityIntersection returns the best positionZ intersection point on the target from the source (-1 if no intersection)
func zEntityIntersection(sourceZ float64, source, target *Entity) float64 {
	srcMinZ, srcMaxZ := zEntityMinMax(sourceZ, source)
	tgtMinZ, tgtMaxZ := zEntityMinMax(target.PositionZ, target)

	var intersectZ float64 = -1
	if srcMinZ > tgtMaxZ || tgtMinZ > srcMaxZ {
		// no intersection
		return intersectZ
	}

	// find best simple intersection within the target range
	midZ := srcMinZ + (srcMaxZ-srcMinZ)/2
	intersectZ = clamp(midZ, tgtMinZ, tgtMaxZ)

	return intersectZ
}

// zEntityMinMax calculates the minZ/maxZ used for basic collision checking in the Z-plane
func zEntityMinMax(positionZ float64, entity *Entity) (float64, float64) {
	var minZ, maxZ float64
	collisionHeight := entity.CollisionHeight

	switch entity.Anchor {
	case AnchorBottom:
		minZ, maxZ = positionZ, positionZ+collisionHeight
	case AnchorCenter:
		minZ, maxZ = positionZ-collisionHeight/2, positionZ+collisionHeight/2
	case AnchorTop:
		minZ, maxZ = positionZ-collisionHeight, positionZ
	}

	return minZ, maxZ
}

// -- crosshairs

type Crosshairs struct {
	*Sprite
	hitTimer     int
	HitIndicator *Sprite
}

func NewCrosshairs(
	x, y, scale float64, img *ebiten.Image, columns, rows, crosshairIndex, hitIndex int,
) *Crosshairs {
	mapColor := color.RGBA{0, 0, 0, 0}

	normalCrosshairs := &Crosshairs{
		Sprite: NewSpriteFromSheet(x, y, scale, img, mapColor, columns, rows, crosshairIndex, AnchorCenter, 0, 0),
	}

	hitCrosshairs := NewSpriteFromSheet(x, y, scale, img, mapColor, columns, rows, hitIndex, AnchorCenter, 0, 0)

	hitCrosshairs.SetAnimationFrame(hitIndex)
	normalCrosshairs.HitIndicator = hitCrosshairs

	return normalCrosshairs
}

func (c *Crosshairs) ActivateHitIndicator(hitTime int) {
	if c.HitIndicator != nil {
		c.hitTimer = hitTime
	}
}

func (c *Crosshairs) IsHitIndicatorActive() bool {
	return c.HitIndicator != nil && c.hitTimer > 0
}

func (c *Crosshairs) Update() {
	if c.HitIndicator != nil && c.hitTimer > 0 {
		// TODO: prefer to use timer rather than frame update counter?
		c.hitTimer -= 1
	}
}

// -- effect

type Effect struct {
	*Sprite
	loopCount int
}

func NewEffect(
	x, y, scale float64, animationRate int, img *ebiten.Image, columns, rows int, anchor SpriteAnchor, loopCount int,
) *Effect {
	mapColor := color.RGBA{0, 0, 0, 0}
	e := &Effect{
		Sprite:    NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, 0, 0),
		loopCount: loopCount,
	}

	// effects should not be convergence capable by player focal point
	e.Sprite.Focusable = false

	// effects self illuminate so they do not get dimmed in dark conditions
	e.illumination = 5000

	return e
}

// -- entity

type Entity struct {
	Position        *Vec2
	PositionZ       float64
	Scale           float64
	Anchor          SpriteAnchor
	Angle           float64
	Pitch           float64
	Velocity        float64
	CollisionRadius float64
	CollisionHeight float64
	MapColor        color.RGBA
	Parent          *Entity
}

func (e *Entity) Pos() *Vec2 {
	return e.Position
}

func (e *Entity) PosZ() float64 {
	return e.PositionZ
}

// -- input

func (g *Game) handleInput() {
	// p pauses the game
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.paused {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
			g.paused = false
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			g.paused = true
		}
	}

	// escape exits the game
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		exit(0)
	}

	if g.paused {
		// dont process input when paused
		return
	}

	forward := false
	backward := false
	strafeLeft := false
	strafeRight := false

	moveModifier := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		moveModifier = 1.5
	}

	x, y := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.fireWeapon()
	}

	if g.mouseX == math.MinInt32 && g.mouseY == math.MinInt32 {
		// initialize first position to establish delta
		if x != 0 && y != 0 {
			g.mouseX, g.mouseY = x, y
		}
	} else {
		dx, dy := g.mouseX-x, g.mouseY-y
		g.mouseX, g.mouseY = x, y

		if dx != 0 {
			g.player.rotate(float64(dx) * moveModifier)
		}

		if dy != 0 {
			g.player.updatePitch(float64(dy))
		}
	}

	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		g.player.NextWeapon(wheelY > 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit1) {
		g.player.SelectWeapon(0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit2) {
		g.player.SelectWeapon(1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyH) {
		// put away/holster weapon
		g.player.SelectWeapon(-1)
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		strafeLeft = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		strafeRight = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		forward = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		backward = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyC) {
		g.player.crouch()
	} else if ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.player.goProne()
	} else if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.player.Jump()
	} else {
		// Apply gravity when space is not pressed
		g.player.applyGravity()
	}

	if forward {
		g.move(moveModifier)
	} else if backward {
		g.move(-moveModifier)
	}

	if strafeLeft {
		g.strafe(-moveModifier)
	} else if strafeRight {
		g.strafe(moveModifier)
	}
}

// -- level

// Level --struct to represent rects and tints of vertical Level slices --//
type Level struct {
	// Sv --texture draw location
	Sv []*image.Rectangle

	// Cts --texture source location
	Cts []*image.Rectangle

	// St --current slice tint (for lighting/shading)--//
	St []*color.RGBA

	// CurrTex --the texture to use as source
	CurrTex []*ebiten.Image
}

// sliceView Creates rectangle slices for each x in width.
func sliceView(width, height int) []*image.Rectangle {
	arr := make([]*image.Rectangle, width)

	for x := 0; x < width; x++ {
		thisRect := image.Rect(x, 0, x+1, height)
		arr[x] = &thisRect
	}

	return arr
}

// HorizontalLevel is for handling horizontal renders that cannot use vertical slices (e.g. floor, ceiling)
type HorizontalLevel struct {
	// horBuffer is the image representing the pixels to render during the update
	horBuffer *image.RGBA
	// image is the ebitengine image object rendering the horBuffer during draw
	image *ebiten.Image
}

func (h *HorizontalLevel) initialize(width, height int) {
	h.horBuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	if h.image == nil {
		h.image = ebiten.NewImage(width, height)
	}
}

// -- minimap

func (g *Game) miniMap() *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, g.mapWidth, g.mapHeight))

	// wall/world positions
	worldMap := g.mapObj.Level(0)
	for x, row := range worldMap {
		for y := range row {
			c := g.getMapColor(x, y)
			if c.A == 255 {
				c.A = 142
			}
			m.Set(x, y, c)
		}
	}

	// sprite positions, sort by color to avoid random color getting chosen as last when using map keys
	sprites := make([]*Entity, 0, len(g.sprites))
	for s := range g.sprites {
		sprites = append(sprites, s.Entity)
	}
	sort.Slice(sprites, func(i, j int) bool {
		iComp := (sprites[i].MapColor.R + sprites[i].MapColor.G + sprites[i].MapColor.B)
		jComp := (sprites[j].MapColor.R + sprites[j].MapColor.G + sprites[j].MapColor.B)
		return iComp < jComp
	})

	for _, sprite := range sprites {
		if sprite.MapColor.A > 0 {

			m.Set(int(sprite.Position.X), int(sprite.Position.Y), sprite.MapColor)
		}
	}

	// projectile positions
	projectiles := make([]*Entity, 0, len(g.projectiles))
	for p := range g.projectiles {
		projectiles = append(projectiles, p.Entity)
	}
	sort.Slice(projectiles, func(i, j int) bool {
		iComp := (projectiles[i].MapColor.R + projectiles[i].MapColor.G + projectiles[i].MapColor.B)
		jComp := (projectiles[j].MapColor.R + projectiles[j].MapColor.G + projectiles[j].MapColor.B)
		return iComp < jComp
	})

	for _, projectile := range projectiles {
		if projectile.MapColor.A > 0 {

			m.Set(int(projectile.Position.X), int(projectile.Position.Y), projectile.MapColor)
		}
	}

	// player position
	m.Set(int(g.player.Position.X), int(g.player.Position.Y), g.player.MapColor)

	return m
}

func (g *Game) getMapColor(x, y int) color.RGBA {
	worldMap := g.mapObj.Level(0)
	switch worldMap[x][y] {
	case 0:
		return color.RGBA{43, 30, 24, 255}
	case 1:
		return color.RGBA{100, 89, 73, 255}
	case 2:
		return color.RGBA{51, 32, 0, 196}
	case 3:
		return color.RGBA{56, 36, 0, 196}
	case 6:
		// ebitengine splash logo color!
		return color.RGBA{219, 86, 32, 255}
	default:
		return color.RGBA{255, 194, 32, 255}
	}
}

// -- player

type Player struct {
	*Entity
	cameraZ      float64
	moved        bool
	weapon       *Weapon
	weaponSet    []*Weapon
	lastWeapon   *Weapon
	velocityZ    float64
	jumpHoldTime float64
	isJumping    bool
}

func NewPlayer(x, y, angle, pitch float64) *Player {
	p := &Player{
		Entity: &Entity{
			Position:  &Vec2{X: x, Y: y},
			PositionZ: 0,
			Angle:     angle,
			Pitch:     pitch,
			Velocity:  0,
			MapColor:  color.RGBA{255, 0, 0, 255},
		},
		cameraZ:      0.5,
		moved:        false,
		weaponSet:    []*Weapon{},
		jumpHoldTime: 0,
		isJumping:    false,
	}

	return p
}

func (p *Player) AddWeapon(w *Weapon) {
	p.weaponSet = append(p.weaponSet, w)
}

func (p *Player) SelectWeapon(weaponIndex int) *Weapon {
	// TODO: add some kind of sheath/unsheath animation
	if weaponIndex < 0 {
		// put away weapon
		if p.weapon != nil {
			// store as last weapon
			p.lastWeapon = p.weapon
		}
		p.weapon = nil
		return nil
	}
	newWeapon := p.weapon
	if weaponIndex < len(p.weaponSet) {
		newWeapon = p.weaponSet[weaponIndex]
	}
	if newWeapon != p.weapon {
		// store as last weapon
		p.lastWeapon = p.weapon
		p.weapon = newWeapon
	}
	return p.weapon
}

func (p *Player) NextWeapon(reverse bool) *Weapon {
	_, weaponIndex := p.getSelectedWeapon()
	if weaponIndex < 0 {
		// check last weapon in event of unsheathing previously sheathed weapon
		weaponIndex = p.getWeaponIndex(p.lastWeapon)
		if weaponIndex < 0 {
			weaponIndex = 0
		}
		return p.SelectWeapon(weaponIndex)
	}

	weaponIndex++
	if weaponIndex >= len(p.weaponSet) {
		weaponIndex = 0
	}
	return p.SelectWeapon(weaponIndex)
}

func (p *Player) getWeaponIndex(w *Weapon) int {
	if w == nil {
		return -1
	}
	for index, wCheck := range p.weaponSet {
		if wCheck == w {
			return index
		}
	}
	return -1
}

func (p *Player) getSelectedWeapon() (*Weapon, int) {
	if p.weapon == nil {
		return nil, -1
	}

	return p.weapon, p.getWeaponIndex(p.weapon)
}

func (p *Player) IsStanding() bool {
	return p.PositionZ == 0 && p.cameraZ == 0.5
}

const (
	jumpVelocity = 3.2
	gravity      = 9.8
)

func (p *Player) Jump() {
	if p.IsStanding() {
		p.velocityZ = jumpVelocity
		p.moved = true
	}

	p.velocityZ -= gravity * (1.0 / 60.0) // Assuming 60 FPS
	p.PositionZ += p.velocityZ * (1.0 / 60.0)

	if p.PositionZ <= 0 {
		p.PositionZ = 0
		p.velocityZ = 0
		p.Stand()
	} else {
		p.cameraZ = 0.5 + p.PositionZ
	}
	p.moved = true
}

func (p *Player) applyGravity() {
	if !p.IsStanding() {
		p.velocityZ -= gravity * (1.0 / 60.0)
		p.PositionZ += p.velocityZ * (1.0 / 60.0)

		if p.PositionZ <= 0 {
			p.PositionZ = 0
			p.velocityZ = 0
			p.Stand()
		} else {
			p.cameraZ = 0.5 + p.PositionZ
		}
		p.moved = true
	}
}

func (p *Player) Stand() {
	p.cameraZ = 0.5
	p.PositionZ = 0
	p.moved = true
}

func (p *Player) updatePitch(pModifier float64) {
	pSpeed := playerRotateSpeed * pModifier
	// current raycasting method can only allow up to 22.5 degrees down, 45 degrees up
	p.Pitch = clamp(pSpeed+p.Pitch, -math.Pi/8, math.Pi/4)
	p.moved = true
}

func (p *Player) crouch() {
	p.cameraZ = 0.3
	p.PositionZ = 0
	p.moved = true
}

func (p *Player) goProne() {
	p.cameraZ = 0.1
	p.PositionZ = 0
	p.moved = true
}

const (
	playerMoveSpeed   = 0.06
	playerStrafeSpeed = 0.05
	playerRotateSpeed = 0.005
)

// move player by move speed in the forward/backward direction
func (g *Game) move(moveModifier float64) {
	mSpeed := playerMoveSpeed * moveModifier
	moveLine := lineFromAngle(g.player.Position.X, g.player.Position.Y, g.player.Angle, mSpeed)

	newPos, _, _ := g.getValidMove(g.player.Entity, moveLine.X2, moveLine.Y2, g.player.PositionZ, true)
	if !newPos.eq(g.player.Pos()) {
		g.player.Position = newPos
		g.player.moved = true
	}
}

// Move player by strafe speed in the left/right direction
func (g *Game) strafe(moveModifier float64) {
	mSpeed := playerStrafeSpeed * moveModifier
	strafeAngle := HalfPi
	if mSpeed < 0 {
		strafeAngle = -strafeAngle
	}
	strafeLine := lineFromAngle(g.player.Position.X, g.player.Position.Y, g.player.Angle-strafeAngle, math.Abs(mSpeed))

	newPos, _, _ := g.getValidMove(g.player.Entity, strafeLine.X2, strafeLine.Y2, g.player.PositionZ, true)
	if !newPos.eq(g.player.Pos()) {
		g.player.Position = newPos
		g.player.moved = true
	}
}

// rotate player heading angle by rotation speed
func (p *Player) rotate(rModifier float64) {
	rSpeed := playerRotateSpeed * rModifier
	p.Angle += rSpeed

	for p.Angle > Pi {
		p.Angle = p.Angle - Pi2
	}
	for p.Angle <= -Pi {
		p.Angle = p.Angle + Pi2
	}

	p.moved = true
}

// -- projectile

type Projectile struct {
	*Sprite
	Ricochets    int
	Lifespan     float64
	ImpactEffect Effect
}

func NewProjectile(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewSprite(x, y, scale, img, mapColor, anchor, collisionRadius, collisionHeight),
		Ricochets:    0,
		Lifespan:     math.MaxFloat64,
		ImpactEffect: Effect{},
	}

	// projectiles should not be convergence capable by player focal point
	p.Focusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func NewAnimatedProjectile(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA, columns, rows int,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, collisionRadius, collisionHeight),
		Ricochets:    0,
		Lifespan:     math.MaxFloat64,
		ImpactEffect: Effect{},
	}

	// projectiles should not be convergence capable by player focal point
	p.Focusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func (p *Projectile) SpawnEffect(x, y, z, angle, pitch float64) *Effect {
	impactEffect := clone(&p.ImpactEffect)
	spriteInstance := clone(p.ImpactEffect.Sprite)

	impactEffect.Sprite = spriteInstance
	impactEffect.Position = &Vec2{X: x, Y: y}
	impactEffect.PositionZ = z
	impactEffect.Angle = angle
	impactEffect.Pitch = pitch

	// keep track of what spawned it
	impactEffect.Parent = p.Parent

	return impactEffect
}

// -- weapon

type Weapon struct {
	*Sprite
	firing             bool
	cooldown           int
	rateOfFire         float64
	projectileVelocity float64
	projectile         Projectile
}

func NewAnimatedWeapon(
	x, y, scale float64, animationRate int, img *ebiten.Image, columns, rows int, projectile Projectile, projectileVelocity, rateOfFire float64,
) *Weapon {
	mapColor := color.RGBA{0, 0, 0, 0}
	w := &Weapon{
		Sprite: NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, AnchorCenter, 0, 0),
	}
	w.projectile = projectile
	w.projectileVelocity = projectileVelocity
	w.rateOfFire = rateOfFire

	return w
}

func (w *Weapon) Fire() bool {
	if w.cooldown <= 0 {
		// TODO: handle rate of fire greater than 60 per second?
		w.cooldown = int(1 / w.rateOfFire * float64(ebiten.TPS()))

		if !w.firing {
			w.firing = true
			w.Sprite.ResetAnimation()
		}

		return true
	}
	return false
}

func (w *Weapon) SpawnProjectile(x, y, z, angle, pitch float64, spawnedBy *Entity) *Projectile {
	p := clone(&w.projectile)
	s := clone(w.projectile.Sprite)

	p.Sprite = s
	p.Position = &Vec2{X: x, Y: y}
	p.PositionZ = z
	p.Angle = angle
	p.Pitch = pitch

	// convert velocity from distance/second to distance per tick
	p.Velocity = w.projectileVelocity / float64(ebiten.TPS())

	// keep track of what spawned it
	p.Parent = spawnedBy

	return p
}

func (w *Weapon) OnCooldown() bool {
	return w.cooldown > 0
}

func (w *Weapon) ResetCooldown() {
	w.cooldown = 0
}

func (w *Weapon) Update() {
	if w.cooldown > 0 {
		w.cooldown -= 1
	}
	if w.firing && w.Sprite.LoopCounter() < 1 {
		w.Sprite.Update(nil)
	} else {
		w.firing = false
		w.Sprite.ResetAnimation()
	}
}

// -- rendering

func drawTexture(screen *ebiten.Image, texture *ebiten.Image, destinationRectangle *image.Rectangle, sourceRectangle *image.Rectangle, color *color.RGBA) {
	if texture == nil || destinationRectangle == nil || sourceRectangle == nil {
		return
	}

	// if destinationRectangle is not the same size as sourceRectangle, scale to fit
	var scaleX, scaleY float64 = 1.0, 1.0
	if !destinationRectangle.Eq(*sourceRectangle) {
		sSize := sourceRectangle.Size()
		dSize := destinationRectangle.Size()

		scaleX = float64(dSize.X) / float64(sSize.X)
		scaleY = float64(dSize.Y) / float64(sSize.Y)
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest

	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(float64(destinationRectangle.Min.X), float64(destinationRectangle.Min.Y))

	destTexture := texture.SubImage(*sourceRectangle).(*ebiten.Image)

	if color != nil {
		// color channel modulation/tinting
		op.ColorScale.Scale(float32(color.R)/255, float32(color.G)/255, float32(color.B)/255, float32(color.A)/255)
	}

	screen.DrawImage(destTexture, op)
}

// -- texture handler

type TextureManager struct {
	mapObj         *Map
	textures       []*ebiten.Image
	floorTex       *image.RGBA
	renderFloorTex bool
}

func NewTextureHandler(mapObj *Map, textureCapacity int) *TextureManager {
	t := &TextureManager{
		mapObj:         mapObj,
		textures:       make([]*ebiten.Image, textureCapacity),
		renderFloorTex: true,
	}
	return t
}

func (t *TextureManager) TextureAt(x, y, levelNum, side int) *ebiten.Image {
	texNum := -1

	mapLayer := t.mapObj.Level(levelNum)
	if mapLayer == nil {
		return nil
	}

	mapWidth := len(mapLayer)
	if mapWidth == 0 {
		return nil
	}
	mapHeight := len(mapLayer[0])
	if mapHeight == 0 {
		return nil
	}

	if x >= 0 && x < mapWidth && y >= 0 && y < mapHeight {
		texNum = mapLayer[x][y] - 1 // 1 subtracted from it so that texture 0 can be used
	}

	if side == 0 {
		//--some supid hacks to make the houses render correctly--//
		// this corrects textures on two sides of house since the textures are not symmetrical
		if texNum == 3 {
			texNum = 4
		} else if texNum == 4 {
			texNum = 3
		}

		if texNum == 1 {
			texNum = 4
		} else if texNum == 2 {
			texNum = 3
		}

		// make the ebitengine splash only show on one side
		if texNum == 5 {
			texNum = 0
		}
	}

	if texNum < 0 {
		return nil
	}
	return t.textures[texNum]
}

func (t *TextureManager) FloorTextureAt(x, y int) *image.RGBA {
	// x/y could be used to render different floor texture at given coords,
	// but for this demo we will just be rendering the same texture everywhere.
	if t.renderFloorTex {
		return t.floorTex
	}
	return nil
}

// -- sprite

type SpriteAnchor int

const (
	// AnchorBottom anchors the bottom of the sprite to its Z-position
	AnchorBottom SpriteAnchor = iota
	// AnchorCenter anchors the center of the sprite to its Z-position
	AnchorCenter
	// AnchorTop anchors the top of the sprite to its Z-position
	AnchorTop
)

func getAnchorVerticalOffset(anchor SpriteAnchor, spriteScale float64, cameraHeight int) float64 {
	halfHeight := float64(cameraHeight) / 2

	switch anchor {
	case AnchorBottom:
		return halfHeight - (spriteScale * halfHeight)
	case AnchorCenter:
		return halfHeight
	case AnchorTop:
		return halfHeight + (spriteScale * halfHeight)
	}

	return 0
}

type Sprite struct {
	*Entity
	W, H           int
	AnimationRate  int
	Focusable      bool
	illumination   float64
	animReversed   bool
	animCounter    int
	loopCounter    int
	columns, rows  int
	texNum, lenTex int
	texFacingMap   map[float64]int
	texFacingKeys  []float64
	texRects       []image.Rectangle
	textures       []*ebiten.Image
	screenRect     *image.Rectangle
}

func (s *Sprite) Scale() float64 {
	return s.Entity.Scale
}

func (s *Sprite) VerticalAnchor() SpriteAnchor {
	return s.Entity.Anchor
}

func (s *Sprite) Texture() *ebiten.Image {
	return s.textures[s.texNum]
}

func (s *Sprite) TextureRect() image.Rectangle {
	return s.texRects[s.texNum]
}

func (s *Sprite) Illumination() float64 {
	return s.illumination
}

func (s *Sprite) SetScreenRect(rect *image.Rectangle) {
	s.screenRect = rect
}

func (s *Sprite) IsFocusable() bool {
	return s.Focusable
}

func NewSprite(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			Position:        &Vec2{X: x, Y: y},
			PositionZ:       0,
			Scale:           scale,
			Anchor:          anchor,
			Angle:           0,
			Velocity:        0,
			CollisionRadius: collisionRadius,
			CollisionHeight: collisionHeight,
			MapColor:        mapColor,
		},
		Focusable: true,
	}

	s.texNum = 0
	s.lenTex = 1
	s.textures = make([]*ebiten.Image, s.lenTex)

	s.W, s.H = img.Bounds().Dx(), img.Bounds().Dy()
	s.texRects = []image.Rectangle{image.Rect(0, 0, s.W, s.H)}

	s.textures[0] = img

	return s
}

func NewSpriteFromSheet(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	columns, rows, spriteIndex int, anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			Position:        &Vec2{X: x, Y: y},
			PositionZ:       0,
			Scale:           scale,
			Anchor:          anchor,
			Angle:           0,
			Velocity:        0,
			CollisionRadius: collisionRadius,
			CollisionHeight: collisionHeight,
			MapColor:        mapColor,
		},
		Focusable: true,
	}

	s.texNum = spriteIndex
	s.columns, s.rows = columns, rows
	s.lenTex = columns * rows
	s.textures = make([]*ebiten.Image, s.lenTex)
	s.texRects = make([]image.Rectangle, s.lenTex)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	// crop sheet by given number of columns and rows into a single dimension array
	s.W = w / columns
	s.H = h / rows

	for r := 0; r < rows; r++ {
		y := r * s.H
		for c := 0; c < columns; c++ {
			x := c * s.W
			cellRect := image.Rect(x, y, x+s.W, y+s.H)
			cellImg := img.SubImage(cellRect).(*ebiten.Image)

			index := c + r*columns
			s.textures[index] = cellImg
			s.texRects[index] = cellRect
		}
	}

	return s
}

func NewAnimatedSprite(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA,
	columns, rows int, anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Sprite {
	s := &Sprite{
		Entity: &Entity{
			Position:        &Vec2{X: x, Y: y},
			PositionZ:       0,
			Scale:           scale,
			Anchor:          anchor,
			Angle:           0,
			Velocity:        0,
			CollisionRadius: collisionRadius,
			CollisionHeight: collisionHeight,
			MapColor:        mapColor,
		},
		Focusable: true,
	}

	s.AnimationRate = animationRate
	s.animCounter = 0
	s.loopCounter = 0

	s.texNum = 0
	s.columns, s.rows = columns, rows
	s.lenTex = columns * rows
	s.textures = make([]*ebiten.Image, s.lenTex)
	s.texRects = make([]image.Rectangle, s.lenTex)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()

	// crop sheet by given number of columns and rows into a single dimension array
	s.W = w / columns
	s.H = h / rows

	for r := 0; r < rows; r++ {
		y := r * s.H
		for c := 0; c < columns; c++ {
			x := c * s.W
			cellRect := image.Rect(x, y, x+s.W, y+s.H)
			cellImg := img.SubImage(cellRect).(*ebiten.Image)

			index := c + r*columns
			s.textures[index] = cellImg
			s.texRects[index] = cellRect
		}
	}

	return s
}

func (s *Sprite) SetTextureFacingMap(texFacingMap map[float64]int) {
	s.texFacingMap = texFacingMap

	// create pre-sorted list of keys used during facing determination
	s.texFacingKeys = make([]float64, len(texFacingMap))
	for k := range texFacingMap {
		s.texFacingKeys = append(s.texFacingKeys, k)
	}
	sort.Float64s(s.texFacingKeys)
}

func (s *Sprite) getTextureFacingKeyForAngle(facingAngle float64) float64 {
	var closestKeyAngle float64 = -1
	if s.texFacingMap == nil || len(s.texFacingMap) == 0 || s.texFacingKeys == nil || len(s.texFacingKeys) == 0 {
		return closestKeyAngle
	}

	closestKeyDiff := math.MaxFloat64
	for _, keyAngle := range s.texFacingKeys {
		keyDiff := math.Min(Pi2-math.Abs(float64(keyAngle)-facingAngle), math.Abs(float64(keyAngle)-facingAngle))
		if keyDiff < closestKeyDiff {
			closestKeyDiff = keyDiff
			closestKeyAngle = keyAngle
		}
	}

	return closestKeyAngle
}

func (s *Sprite) SetAnimationReversed(isReverse bool) {
	s.animReversed = isReverse
}

func (s *Sprite) SetAnimationFrame(texNum int) {
	s.texNum = texNum
}

func (s *Sprite) ResetAnimation() {
	s.animCounter = 0
	s.loopCounter = 0
	s.texNum = 0
}

func (s *Sprite) LoopCounter() int {
	return s.loopCounter
}

func (s *Sprite) ScreenRect() *image.Rectangle {
	return s.screenRect
}

func (s *Sprite) Update(camPos *Vec2) {
	if s.AnimationRate <= 0 {
		return
	}

	if s.animCounter >= s.AnimationRate {
		minTexNum := 0
		maxTexNum := s.lenTex - 1

		if len(s.texFacingMap) > 1 && camPos != nil {
			// TODO: may want to be able to change facing even between animation frame changes

			// use facing from camera position to determine min/max texNum in texFacingMap
			// to update facing of sprite relative to camera and sprite angle
			texRow := 0

			// calculate angle from sprite relative to camera position by getting angle of line between them
			lineToCam := Line{X1: s.Position.X, Y1: s.Position.Y, X2: camPos.X, Y2: camPos.Y}
			facingAngle := lineToCam.angle() - s.Angle
			if facingAngle < 0 {
				// convert to positive angle needed to determine facing index to use
				facingAngle += Pi2
			}
			facingKeyAngle := s.getTextureFacingKeyForAngle(facingAngle)
			if texFacingValue, ok := s.texFacingMap[facingKeyAngle]; ok {
				texRow = texFacingValue
			}

			minTexNum = texRow * s.columns
			maxTexNum = texRow*s.columns + s.columns - 1
		}

		s.animCounter = 0

		if s.animReversed {
			s.texNum -= 1
			if s.texNum > maxTexNum || s.texNum < minTexNum {
				s.texNum = maxTexNum
				s.loopCounter++
			}
		} else {
			s.texNum += 1
			if s.texNum > maxTexNum || s.texNum < minTexNum {
				s.texNum = minTexNum
				s.loopCounter++
			}
		}
	} else {
		s.animCounter++
	}
}

func (s *Sprite) AddDebugLines(lineWidth int, clr color.Color) {
	lW := float64(lineWidth)
	sW := float64(s.W)
	sH := float64(s.H)
	sCr := s.CollisionRadius * sW

	for i, img := range s.textures {
		imgRect := s.texRects[i]
		x, y := float64(imgRect.Min.X), float64(imgRect.Min.Y)

		// bounding box
		ebitenutil.DrawRect(img, x, y, lW, sH, clr)
		ebitenutil.DrawRect(img, x, y, sW, lW, clr)
		ebitenutil.DrawRect(img, x+sW-lW-1, y+sH-lW-1, lW, -sH, clr)
		ebitenutil.DrawRect(img, x+sW-lW-1, y+sH-lW-1, -sW, lW, clr)

		// center lines
		ebitenutil.DrawRect(img, x+sW/2-lW/2-1, y, lW, sH, clr)
		ebitenutil.DrawRect(img, x, y+sH/2-lW/2-1, sW, lW, clr)

		// collision markers
		if s.CollisionRadius > 0 {
			ebitenutil.DrawRect(img, x+sW/2-sCr-lW/2-1, y, lW, sH, color.White)
			ebitenutil.DrawRect(img, x+sW/2+sCr-lW/2-1, y, lW, sH, color.White)
		}
	}
}

// -- resources

//go:embed resources
var embedded embed.FS

// loadContent will be called once per game and is the place to load
// all of your content.
func (g *Game) loadContent() {

	// TODO: make resource management better

	// load wall textures
	g.tex.textures[0] = getTextureFromFile("stone.png")
	g.tex.textures[1] = getTextureFromFile("left_bot_house.png")
	g.tex.textures[2] = getTextureFromFile("right_bot_house.png")
	g.tex.textures[3] = getTextureFromFile("left_top_house.png")
	g.tex.textures[4] = getTextureFromFile("right_top_house.png")
	g.tex.textures[5] = getTextureFromFile("ebitengine_splash.png")

	// separating sprites out a bit from wall textures
	g.tex.textures[8] = getSpriteFromFile("large_rock.png")
	g.tex.textures[9] = getSpriteFromFile("tree_09.png")
	g.tex.textures[10] = getSpriteFromFile("tree_10.png")
	g.tex.textures[14] = getSpriteFromFile("tree_14.png")

	// load texture sheets
	g.tex.textures[15] = getSpriteFromFile("sorcerer_sheet.png")
	g.tex.textures[16] = getSpriteFromFile("crosshairs_sheet.png")
	g.tex.textures[17] = getSpriteFromFile("charged_bolt_sheet.png")
	g.tex.textures[18] = getSpriteFromFile("blue_explosion_sheet.png")
	g.tex.textures[19] = getSpriteFromFile("outleader_walking_sheet.png")
	g.tex.textures[20] = getSpriteFromFile("hand_spell.png")
	g.tex.textures[21] = getSpriteFromFile("hand_staff.png")
	g.tex.textures[22] = getSpriteFromFile("red_bolt.png")
	g.tex.textures[23] = getSpriteFromFile("red_explosion_sheet.png")
	g.tex.textures[24] = getSpriteFromFile("bat_sheet.png")

	// just setting the grass texture apart from the rest since it gets special handling
	if g.debug {
		g.tex.floorTex = getRGBAFromFile("grass_debug.png")
	} else {
		g.tex.floorTex = getRGBAFromFile("grass.png")
	}
}

func newImageFromFile(path string) (*ebiten.Image, image.Image, error) {
	f, err := embedded.Open(filepath.ToSlash(path))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	eb, im, err := ebitenutil.NewImageFromReader(f)
	return eb, im, err
}

func getRGBAFromFile(texFile string) *image.RGBA {
	var rgba *image.RGBA
	_, tex, err := newImageFromFile("resources/textures/" + texFile)
	if err != nil {
		log.Fatal(err)
	}
	if tex != nil {
		rgba = image.NewRGBA(image.Rect(0, 0, texWidth, texWidth))
		// convert into RGBA format
		for x := 0; x < texWidth; x++ {
			for y := 0; y < texWidth; y++ {
				clr := tex.At(x, y).(color.RGBA)
				rgba.SetRGBA(x, y, clr)
			}
		}
	}

	return rgba
}

func getTextureFromFile(texFile string) *ebiten.Image {
	eImg, _, err := newImageFromFile("resources/textures/" + texFile)
	if err != nil {
		log.Fatal(err)
	}
	return eImg
}

func getSpriteFromFile(sFile string) *ebiten.Image {
	eImg, _, err := newImageFromFile("resources/sprites/" + sFile)
	if err != nil {
		log.Fatal(err)
	}
	return eImg
}

func (g *Game) loadSprites() {
	g.projectiles = make(map[*Projectile]struct{}, 1024)
	g.effects = make(map[*Effect]struct{}, 1024)
	g.sprites = make(map[*Sprite]struct{}, 128)

	// colors for minimap representation
	blueish := color.RGBA{62, 62, 100, 96}
	reddish := color.RGBA{180, 62, 62, 96}
	brown := color.RGBA{47, 40, 30, 196}
	green := color.RGBA{27, 37, 7, 196}
	orange := color.RGBA{69, 30, 5, 196}
	yellow := color.RGBA{255, 200, 0, 196}

	// preload projectile sprites
	chargedBoltImg := g.tex.textures[17]
	chargedBoltWidth := chargedBoltImg.Bounds().Dx()
	chargedBoltCols, chargedBoltRows := 6, 1
	chargedBoltScale := 0.3
	// in pixels, radius to use for collision testing
	chargedBoltPxRadius := 50.0
	chargedBoltCollisionRadius := (chargedBoltScale * chargedBoltPxRadius) / (float64(chargedBoltWidth) / float64(chargedBoltCols))
	chargedBoltCollisionHeight := 2 * chargedBoltCollisionRadius
	chargedBoltProjectile := NewAnimatedProjectile(
		0, 0, chargedBoltScale, 1, chargedBoltImg, blueish,
		chargedBoltCols, chargedBoltRows, AnchorCenter, chargedBoltCollisionRadius, chargedBoltCollisionHeight,
	)

	redBoltImg := g.tex.textures[22]
	redBoltWidth := redBoltImg.Bounds().Dx()
	redBoltScale := 0.25
	// in pixels, radius to use for collision testing
	redBoltPxRadius := 4.0
	redBoltCollisionRadius := (redBoltScale * redBoltPxRadius) / float64(redBoltWidth)
	redBoltCollisionHeight := 2 * redBoltCollisionRadius
	redBoltProjectile := NewProjectile(
		0, 0, redBoltScale, redBoltImg, reddish,
		AnchorCenter, redBoltCollisionRadius, redBoltCollisionHeight,
	)

	// preload effect sprites
	blueExplosionEffect := NewEffect(
		0, 0, 0.75, 3, g.tex.textures[18], 5, 3, AnchorCenter, 1,
	)
	chargedBoltProjectile.ImpactEffect = *blueExplosionEffect

	redExplosionEffect := NewEffect(
		0, 0, 0.20, 1, g.tex.textures[23], 8, 3, AnchorCenter, 1,
	)
	redBoltProjectile.ImpactEffect = *redExplosionEffect

	// create weapons
	chargedBoltRoF := 2.5      // Rate of Fire (as RoF/second)
	chargedBoltVelocity := 6.0 // Velocity (as distance travelled/second)
	chargedBoltWeapon := NewAnimatedWeapon(1, 1, 1.0, 7, g.tex.textures[20], 3, 1, *chargedBoltProjectile, chargedBoltVelocity, chargedBoltRoF)
	g.player.AddWeapon(chargedBoltWeapon)

	staffBoltRoF := 6.0
	staffBoltVelocity := 24.0
	staffBoltWeapon := NewAnimatedWeapon(1, 1, 1.0, 7, g.tex.textures[21], 3, 1, *redBoltProjectile, staffBoltVelocity, staffBoltRoF)
	g.player.AddWeapon(staffBoltWeapon)

	// animated single facing sorcerer
	sorcImg := g.tex.textures[15]
	sorcWidth, sorcHeight := sorcImg.Bounds().Dx(), sorcImg.Bounds().Dy()
	sorcCols, sorcRows := 10, 1
	sorcScale := 1.25
	// in pixels, radius and height to use for collision testing
	sorcPxRadius, sorcPxHeight := 40.0, 120.0
	// convert pixel to grid using image pixel size
	sorcCollisionRadius := (sorcScale * sorcPxRadius) / (float64(sorcWidth) / float64(sorcCols))
	sorcCollisionHeight := (sorcScale * sorcPxHeight) / (float64(sorcHeight) / float64(sorcRows))
	sorc := NewAnimatedSprite(
		22.5, 11.75, sorcScale, 5, sorcImg, yellow, sorcCols, sorcRows, AnchorBottom, sorcCollisionRadius, sorcCollisionHeight,
	)
	// give sprite a sample velocity for movement
	sorc.Angle = radians(180)
	sorc.Velocity = 0.02
	g.addSprite(sorc)

	// animated walking 8-directional sprite character
	// [walkerTexFacingMap] player facing angle : texture row index
	var walkerTexFacingMap = map[float64]int{
		radians(315): 0,
		radians(270): 1,
		radians(225): 2,
		radians(180): 3,
		radians(135): 4,
		radians(90):  5,
		radians(45):  6,
		radians(0):   7,
	}
	walkerImg := g.tex.textures[19]
	walkerWidth, walkerHeight := walkerImg.Bounds().Dx(), walkerImg.Bounds().Dy()
	walkerCols, walkerRows := 4, 8
	walkerScale := 0.75
	// in pixels, radius and height to use for collision testing
	walkerPxRadius, walkerPxHeight := 30.0, 80.0
	// convert pixel to grid using image pixel size
	walkerCollisionRadius := (walkerScale * walkerPxRadius) / (float64(walkerWidth) / float64(walkerCols))
	walkerCollisionHeight := (walkerScale * walkerPxHeight) / (float64(walkerHeight) / float64(walkerRows))
	walker := NewAnimatedSprite(
		7.5, 6.0, walkerScale, 10, walkerImg, yellow, walkerCols, walkerRows, AnchorBottom, walkerCollisionRadius, walkerCollisionHeight,
	)
	walker.SetAnimationReversed(true) // this sprite sheet has reversed animation frame order
	walker.SetTextureFacingMap(walkerTexFacingMap)
	// give sprite a sample velocity for movement
	walker.Angle = radians(0)
	walker.Velocity = 0.02
	g.addSprite(walker)

	// animated flying 4-directional sprite creature
	// [batTexFacingMap] player facing angle : texture row index
	var batTexFacingMap = map[float64]int{
		radians(270): 1,
		radians(180): 2,
		radians(90):  3,
		radians(0):   0,
	}
	batImg := g.tex.textures[24]
	batWidth, batHeight := batImg.Bounds().Dx(), batImg.Bounds().Dy()
	batCols, batRows := 3, 4
	batScale := 0.25
	// in pixels, radius and height to use for collision testing
	batPxRadius, batPxHeight := 14.0, 25.0
	// convert pixel to grid using image pixel size
	batCollisionRadius := (batScale * batPxRadius) / (float64(batWidth) / float64(batCols))
	batCollisionHeight := (batScale * batPxHeight) / (float64(batHeight) / float64(batRows))
	batty := NewAnimatedSprite(
		10.0, 5.0, batScale, 10, batImg, yellow, batCols, batRows, AnchorTop, batCollisionRadius, batCollisionHeight,
	)
	batty.SetTextureFacingMap(batTexFacingMap)
	// raising Z-position of sprite model but using AnchorTop to show below that position
	batty.PositionZ = 1.0
	// give sprite a sample velocity for movement
	batty.Angle = radians(150)
	batty.Velocity = 0.03
	g.addSprite(batty)

	if g.debug {
		// just some debugging stuff
		sorc.AddDebugLines(2, color.RGBA{0, 255, 0, 255})
		walker.AddDebugLines(2, color.RGBA{0, 255, 0, 255})
		batty.AddDebugLines(2, color.RGBA{0, 255, 0, 255})
		chargedBoltProjectile.AddDebugLines(2, color.RGBA{0, 255, 0, 255})
		redBoltProjectile.AddDebugLines(2, color.RGBA{0, 255, 0, 255})
	}

	// rock that can be jumped over but not walked through
	rockImg := g.tex.textures[8]
	rockWidth, rockHeight := rockImg.Bounds().Dx(), rockImg.Bounds().Dy()
	rockScale := 0.4
	rockPxRadius, rockPxHeight := 24.0, 35.0
	rockCollisionRadius := (rockScale * rockPxRadius) / float64(rockWidth)
	rockCollisionHeight := (rockScale * rockPxHeight) / float64(rockHeight)
	rock := NewSprite(8.0, 5.5, rockScale, rockImg, brown, AnchorBottom, rockCollisionRadius, rockCollisionHeight)
	g.addSprite(rock)

	// testing sprite scaling
	testScale := 0.5
	g.addSprite(NewSprite(10.5, 2.5, testScale, g.tex.textures[9], green, AnchorBottom, 0, 0))

	// // line of trees for testing in front of initial view
	// Setting CollisionRadius=0 to disable collision against small trees
	g.addSprite(NewSprite(19.5, 11.5, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(17.5, 11.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(15.5, 11.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	// // // render a forest!
	g.addSprite(NewSprite(11.5, 1.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 1.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(132.5, 1.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 2, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 2, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 2, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 2.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.25, 2.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 2.25, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 3, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 3, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.25, 3, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(10.5, 3.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 3.25, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 3.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.25, 3.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(10.5, 4, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 4, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 4, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 4, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(10.5, 4.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.25, 4.5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 4.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 4.5, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.5, 4.25, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(10.5, 5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 5, 1.0, g.tex.textures[9], green, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.25, 5, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.5, 5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 5.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 5.25, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 5.25, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.5, 5.5, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(15.5, 5.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(11.5, 6, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 6, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.25, 6, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.25, 6, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(15.5, 6, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 6.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 6.25, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.5, 6.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(12.5, 7, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 7, 1.0, g.tex.textures[10], brown, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(14.5, 7, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 7.5, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
	g.addSprite(NewSprite(13.5, 8, 1.0, g.tex.textures[14], orange, AnchorBottom, 0, 0))
}

func (g *Game) addSprite(sprite *Sprite) {
	g.sprites[sprite] = struct{}{}
}

// func (g *Game) deleteSprite(sprite *Sprite) {
// 	delete(g.sprites, sprite)
// }

func (g *Game) addProjectile(projectile *Projectile) {
	g.projectiles[projectile] = struct{}{}
}

func (g *Game) deleteProjectile(projectile *Projectile) {
	delete(g.projectiles, projectile)
}

func (g *Game) addEffect(effect *Effect) {
	g.effects[effect] = struct{}{}
}

func (g *Game) deleteEffect(effect *Effect) {
	delete(g.effects, effect)
}

// -- map

// a multi-layered map
type Map struct {
	bottom [][]int
	middle [][]int
	top    [][]int
}

func (m *Map) NumLevels() int {
	return 4
}

func (m *Map) Level(level int) [][]int {
	if level == 0 {
		return m.bottom
	} else if level == 1 {
		return m.middle
	} else {
		return m.top // if above highest level just keep extending last one up
	}
}

func NewMap() *Map {
	m := &Map{}

	m.bottom = [][]int{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 2, 3, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 3, 2, 3, 2, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 6, 1, 1, 0, 0, 2, 3, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1},
		{1, 0, 1, 0, 1, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}

	m.middle = [][]int{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 5, 4, 3, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 4, 5, 2, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}

	m.top = [][]int{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}

	return m
}

func (m *Map) GetCollisionLines(clipDistance float64) []Line {
	if len(m.bottom) == 0 || len(m.bottom[0]) == 0 {
		return []Line{}
	}

	lines := rect(clipDistance, clipDistance,
		float64(len(m.bottom))-2*clipDistance, float64(len(m.bottom[0]))-2*clipDistance)

	for x, row := range m.bottom {
		for y, value := range row {
			if value > 0 {
				lines = append(lines, rect(float64(x)-clipDistance, float64(y)-clipDistance,
					1.0+(2*clipDistance), 1.0+(2*clipDistance))...)
			}
		}
	}

	return lines
}

// -- game

const (
	//--RaycastEngine constants
	//--set constant, texture size to be the wall (and sprite) texture size--//
	texWidth = 256

	// distance to keep away from walls and obstacles to avoid clipping
	// TODO: may want a smaller distance to test vs. sprites
	clipDistance = 0.1
)

// main game object
type Game struct {
	paused bool

	//--create slicer and declare slices--//
	tex                *TextureManager
	initRenderFloorTex bool

	// window resolution and scaling
	screenWidth  int
	screenHeight int
	renderScale  float64
	fullscreen   bool
	vsync        bool

	opengl     bool
	fovDegrees float64
	fovDepth   float64

	//--viewport width / height--//
	width  int
	height int

	player *Player

	//--define camera and render scene--//
	camera *Camera
	scene  *ebiten.Image

	mouseX, mouseY int

	crosshairs *Crosshairs

	// zoom settings
	zoomFovDepth float64

	renderDistance float64

	// lighting settings
	lightFalloff       float64
	globalIllumination float64
	minLightRGB        *color.NRGBA
	maxLightRGB        *color.NRGBA

	//--array of levels, levels refer to "floors" of the world--//
	mapObj       *Map
	collisionMap []Line

	sprites     map[*Sprite]struct{}
	projectiles map[*Projectile]struct{}
	effects     map[*Effect]struct{}

	mapWidth, mapHeight int

	showSpriteBoxes bool
	debug           bool
}

// NewGame - Allows the game to perform any initialization it needs to before starting to run.
// This is where it can query for any required services and load any non-graphic
// related content.  Calling base.Initialize will enumerate through any components
// and initialize them as well.
func NewGame() *Game {
	fmt.Printf("Initializing Game\n")

	// initialize Game object
	g := &Game{
		screenWidth:        1024,
		screenHeight:       768,
		fovDegrees:         68,
		renderScale:        1.0,
		fullscreen:         false,
		vsync:              true,
		opengl:             true,
		renderDistance:     -1,
		initRenderFloorTex: true,
		showSpriteBoxes:    false,
		debug:              false,
	}

	ebiten.SetWindowTitle("Office Escape")

	// default TPS is 60
	// ebiten.SetMaxTPS(60)

	if g.opengl {
		os.Setenv("EBITENGINE_GRAPHICS_LIBRARY", "opengl")
	}

	// use scale to keep the desired window width and height
	g.setResolution(g.screenWidth, g.screenHeight)
	g.setRenderScale(g.renderScale)
	g.setFullscreen(g.fullscreen)
	g.setVsyncEnabled(g.vsync)

	// load map
	g.mapObj = NewMap()

	// load texture handler
	g.tex = NewTextureHandler(g.mapObj, 32)
	g.tex.renderFloorTex = g.initRenderFloorTex

	g.collisionMap = g.mapObj.GetCollisionLines(clipDistance)
	worldMap := g.mapObj.Level(0)
	g.mapWidth = len(worldMap)
	g.mapHeight = len(worldMap[0])

	// load content once when first run
	g.loadContent()

	// create crosshairs and weapon
	g.crosshairs = NewCrosshairs(1, 1, 2.0, g.tex.textures[16], 8, 8, 55, 57)

	// init player model
	angleDegrees := 60.0
	g.player = NewPlayer(8.5, 3.5, radians(angleDegrees), 0)
	g.player.CollisionRadius = clipDistance
	g.player.CollisionHeight = 0.5

	// init the sprites
	g.loadSprites()

	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	g.mouseX, g.mouseY = math.MinInt32, math.MinInt32

	//--init camera and renderer--//
	g.camera = NewCamera(g.width, g.height, texWidth, g.mapObj, g.tex)
	g.setRenderDistance(g.renderDistance)

	g.camera.SetFloorTexture(getTextureFromFile("floor.png"))
	g.camera.SetSkyTexture(getTextureFromFile("sky.png"))

	// initialize camera to player position
	g.updatePlayerCamera(true)
	g.setFovAngle(g.fovDegrees)
	g.fovDepth = g.camera.FovDepth()

	g.zoomFovDepth = 2.0

	// set demo lighting settings
	g.setLightFalloff(-200)
	g.setGlobalIllumination(500)
	minLightRGB := &color.NRGBA{R: 76, G: 76, B: 76, A: 255}
	maxLightRGB := &color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	g.setLightRGB(minLightRGB, maxLightRGB)

	return g
}

// Run is the Ebiten Run loop caller
func (g *Game) Run() {
	g.paused = false

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// Layout takes the outside size (e.g., the window size) and returns the (logical) screen size.
// If you don't have to adjust the screen size with the outside size, just return a fixed size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.screenWidth, g.screenHeight
	return w, h
}

// Update - Allows the game to run logic such as updating the world, gathering input, and playing audio.
// Update is called every tick (1/60 [s] by default).
func (g *Game) Update() error {
	// handle input (when paused making sure only to allow input for closing menu so it can be unpaused)
	g.handleInput()

	if !g.paused {
		// Perform logical updates
		w := g.player.weapon
		if w != nil {
			w.Update()
		}
		g.updateProjectiles()
		g.updateSprites()

		// handle player camera movement
		g.updatePlayerCamera(false)
	}

	return nil
}

// Draw draws the game screen.
// Draw is called every frame (typically 1/60[s] for 60Hz display).
func (g *Game) Draw(screen *ebiten.Image) {
	// Put projectiles together with sprites for raycasting both as sprites
	numSprites, numProjectiles, numEffects := len(g.sprites), len(g.projectiles), len(g.effects)
	raycastSprites := make([]*Sprite, numSprites+numProjectiles+numEffects)
	index := 0
	for sprite := range g.sprites {
		raycastSprites[index] = sprite
		index += 1
	}
	for projectile := range g.projectiles {
		raycastSprites[index] = projectile.Sprite
		index += 1
	}
	for effect := range g.effects {
		raycastSprites[index] = effect.Sprite
		index += 1
	}

	// Update camera (calculate raycast)
	g.camera.Update(raycastSprites)

	// Render raycast scene
	g.camera.Draw(g.scene)

	// draw equipped weapon
	if g.player.weapon != nil {
		w := g.player.weapon
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest

		// determine base size of weapon based on window size compared to image size
		compSize := g.screenHeight
		if g.screenWidth < g.screenHeight {
			compSize = g.screenWidth
		}

		drawScale := 1.0
		if w.H != compSize/3 {
			// weapon should only take up 1/3rd of screen space
			drawScale = (float64(compSize) / 3) / float64(w.H)
		}

		weaponScale := w.Scale() * drawScale * g.renderScale
		op.GeoM.Scale(weaponScale, weaponScale)
		op.GeoM.Translate(
			float64(g.width)/2-float64(w.W)*weaponScale/2,
			float64(g.height)-float64(w.H)*weaponScale+1,
		)

		// apply lighting setting
		op.ColorScale.Scale(float32(g.maxLightRGB.R)/255, float32(g.maxLightRGB.G)/255, float32(g.maxLightRGB.B)/255, 1)

		g.scene.DrawImage(w.Texture(), op)
	}

	if g.showSpriteBoxes {
		// draw sprite screen indicators to show we know where it was raycasted (must occur after camera.Update)
		for sprite := range g.sprites {
			drawSpriteBox(g.scene, sprite)
		}

		for sprite := range g.projectiles {
			drawSpriteBox(g.scene, sprite.Sprite)
		}

		for sprite := range g.effects {
			drawSpriteBox(g.scene, sprite.Sprite)
		}
	}

	// draw sprite screen indicator only for sprite at point of convergence
	convergenceSprite := g.camera.GetConvergenceSprite()
	if convergenceSprite != nil {
		for sprite := range g.sprites {
			if convergenceSprite == sprite {
				drawSpriteIndicator(g.scene, sprite)
				break
			}
		}
	}

	// draw raycasted scene
	op := &ebiten.DrawImageOptions{}
	if g.renderScale != 1.0 {
		op.Filter = ebiten.FilterNearest
		op.GeoM.Scale(1/g.renderScale, 1/g.renderScale)
	}
	screen.DrawImage(g.scene, op)

	// draw minimap
	mm := g.miniMap()
	mmImg := ebiten.NewImageFromImage(mm)
	if mmImg != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest

		op.GeoM.Scale(5.0, 5.0)
		op.GeoM.Translate(0, 50)
		screen.DrawImage(mmImg, op)
	}

	// draw crosshairs
	if g.crosshairs != nil {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest

		crosshairScale := g.crosshairs.Scale()
		op.GeoM.Scale(crosshairScale, crosshairScale)
		op.GeoM.Translate(
			float64(g.screenWidth)/2-float64(g.crosshairs.W)*crosshairScale/2,
			float64(g.screenHeight)/2-float64(g.crosshairs.H)*crosshairScale/2,
		)
		screen.DrawImage(g.crosshairs.Texture(), op)

		if g.crosshairs.IsHitIndicatorActive() {
			screen.DrawImage(g.crosshairs.HitIndicator.Texture(), op)
			g.crosshairs.Update()
		}
	}

	// draw FPS/TPS counter debug display
	fps := fmt.Sprintf("FPS: %f\nTPS: %f/%v", ebiten.ActualFPS(), ebiten.ActualTPS(), ebiten.TPS())
	ebitenutil.DebugPrint(screen, fps)
}

func drawSpriteBox(screen *ebiten.Image, sprite *Sprite) {
	r := sprite.ScreenRect()
	if r == nil {
		return
	}

	minX, minY := float32(r.Min.X), float32(r.Min.Y)
	maxX, maxY := float32(r.Max.X), float32(r.Max.Y)

	vector.StrokeRect(screen, minX, minY, maxX-minX, maxY-minY, 1, color.RGBA{255, 0, 0, 255}, false)
}

func drawSpriteIndicator(screen *ebiten.Image, sprite *Sprite) {
	r := sprite.ScreenRect()
	if r == nil {
		return
	}

	dX, dY := float32(r.Dx())/8, float32(r.Dy())/8
	midX, minY := float32(r.Max.X)-float32(r.Dx())/2, float32(r.Min.Y)-dY

	vector.StrokeLine(screen, midX, minY+dY, midX-dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
	vector.StrokeLine(screen, midX, minY+dY, midX+dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
	vector.StrokeLine(screen, midX-dX, minY, midX+dX, minY, 1, color.RGBA{0, 255, 0, 255}, false)
}

func (g *Game) setFullscreen(fullscreen bool) {
	g.fullscreen = fullscreen
	ebiten.SetFullscreen(fullscreen)
}

func (g *Game) setResolution(screenWidth, screenHeight int) {
	g.screenWidth, g.screenHeight = screenWidth, screenHeight
	ebiten.SetWindowSize(screenWidth, screenHeight)
	g.setRenderScale(g.renderScale)
}

func (g *Game) setRenderScale(renderScale float64) {
	g.renderScale = renderScale
	g.width = int(math.Floor(float64(g.screenWidth) * g.renderScale))
	g.height = int(math.Floor(float64(g.screenHeight) * g.renderScale))
	if g.camera != nil {
		g.camera.SetViewSize(g.width, g.height)
	}
	g.scene = ebiten.NewImage(g.width, g.height)
}

func (g *Game) setRenderDistance(renderDistance float64) {
	g.renderDistance = renderDistance
	g.camera.SetRenderDistance(g.renderDistance)
}

func (g *Game) setLightFalloff(lightFalloff float64) {
	g.lightFalloff = lightFalloff
	g.camera.SetLightFalloff(g.lightFalloff)
}

func (g *Game) setGlobalIllumination(globalIllumination float64) {
	g.globalIllumination = globalIllumination
	g.camera.SetGlobalIllumination(g.globalIllumination)
}

func (g *Game) setLightRGB(minLightRGB, maxLightRGB *color.NRGBA) {
	g.minLightRGB = minLightRGB
	g.maxLightRGB = maxLightRGB
	g.camera.SetLightRGB(*g.minLightRGB, *g.maxLightRGB)
}

func (g *Game) setVsyncEnabled(enableVsync bool) {
	g.vsync = enableVsync
	ebiten.SetVsyncEnabled(enableVsync)
}

func (g *Game) setFovAngle(fovDegrees float64) {
	g.fovDegrees = fovDegrees
	g.camera.SetFovAngle(fovDegrees, 1.0)
}

func (g *Game) fireWeapon() {
	w := g.player.weapon
	if w == nil {
		g.player.NextWeapon(false)
		return
	}
	if w.OnCooldown() {
		return
	}

	// set weapon firing for animation to run
	w.Fire()

	// spawning projectile at player position just slightly below player's center point of view
	pX, pY, pZ := g.player.Position.X, g.player.Position.Y, clamp(g.player.cameraZ-0.1, 0.05, 0.95)
	// pitch, angle based on raycasted point at crosshairs
	var pAngle, pPitch float64
	convergenceDistance := g.camera.GetConvergenceDistance()
	convergencePoint := g.camera.GetConvergencePoint()
	if convergenceDistance <= 0 || convergencePoint == nil {
		pAngle, pPitch = g.player.Angle, g.player.Pitch
	} else {
		convergenceLine3d := &Line3d{
			X1: pX, Y1: pY, Z1: pZ,
			X2: convergencePoint.X, Y2: convergencePoint.Y, Z2: convergencePoint.Z,
		}
		pAngle, pPitch = convergenceLine3d.heading(), convergenceLine3d.pitch()
	}

	projectile := w.SpawnProjectile(pX, pY, pZ, pAngle, pPitch, g.player.Entity)
	if projectile != nil {
		g.addProjectile(projectile)
	}
}

// Update camera to match player position and orientation
func (g *Game) updatePlayerCamera(forceUpdate bool) {
	if !g.player.moved && !forceUpdate {
		// only update camera position if player moved or forceUpdate set
		return
	}

	// reset player moved flag to only update camera when necessary
	g.player.moved = false

	g.camera.SetPosition(g.player.Position.copy())
	g.camera.SetPositionZ(g.player.cameraZ)
	g.camera.SetHeadingAngle(g.player.Angle)
	g.camera.SetPitchAngle(g.player.Pitch)
}

func (g *Game) updateProjectiles() {
	// Testing animated projectile movement
	for p := range g.projectiles {
		if p.Velocity != 0 {

			trajectory := line3dFromAngle(p.Position.X, p.Position.Y, p.PositionZ, p.Angle, p.Pitch, p.Velocity)

			xCheck := trajectory.X2
			yCheck := trajectory.Y2
			zCheck := trajectory.Z2

			newPos, isCollision, collisions := g.getValidMove(p.Entity, xCheck, yCheck, zCheck, false)
			if isCollision || p.PositionZ <= 0 {
				// for testing purposes, projectiles instantly get deleted when collision occurs
				g.deleteProjectile(p)

				// make a sprite/wall getting hit by projectile cause some visual effect
				if p.ImpactEffect.Sprite != nil {
					if len(collisions) >= 1 {
						// use the first collision point to place effect at
						newPos = collisions[0].collision
					}

					// TODO: give impact effect optional ability to have some velocity based on the projectile movement upon impact if it didn't hit a wall
					effect := p.SpawnEffect(newPos.X, newPos.Y, p.PositionZ, p.Angle, p.Pitch)

					g.addEffect(effect)
				}

				for _, collisionEntity := range collisions {
					if collisionEntity.entity == g.player.Entity {
						println("ouch!")
					} else {
						// show crosshair hit effect
						g.crosshairs.ActivateHitIndicator(30)
					}
				}
			} else {
				p.Position = newPos
				p.PositionZ = zCheck
			}
		}
		p.Update(g.player.Position)
	}

	// Testing animated effects (explosions)
	for e := range g.effects {
		e.Update(g.player.Position)
		if e.LoopCounter() >= e.loopCount {
			g.deleteEffect(e)
		}
	}
}

func (g *Game) updateSprites() {
	// Testing animated sprite movement
	for s := range g.sprites {
		if s.Velocity != 0 {
			vLine := lineFromAngle(s.Position.X, s.Position.Y, s.Angle, s.Velocity)

			xCheck := vLine.X2
			yCheck := vLine.Y2
			zCheck := s.PositionZ

			newPos, isCollision, _ := g.getValidMove(s.Entity, xCheck, yCheck, zCheck, false)
			if isCollision {
				// for testing purposes, letting the sample sprite ping pong off walls in somewhat random direction
				s.Angle = randFloat(-math.Pi, math.Pi)
				s.Velocity = randFloat(0.01, 0.03)
			} else {
				s.Position = newPos
			}
		}
		s.Update(g.player.Position)
	}
}

func main() {
	// run the game
	g := NewGame()
	g.Run()
}
