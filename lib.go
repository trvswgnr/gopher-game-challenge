package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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

func (c *Crosshairs) activateHitIndicator(hitTime int) {
	if c.HitIndicator != nil {
		c.hitTimer = hitTime
	}
}

func (c *Crosshairs) isHitIndicatorActive() bool {
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
	e.Sprite.isFocusable = false

	// effects self illuminate so they do not get dimmed in dark conditions
	e.illumination = 5000

	return e
}

// -- entity

type Entity struct {
	pos            *Vec2
	posZ           float64
	scale          float64
	verticalAnchor SpriteAnchor
	// angle is in radians
	angle float64
	// pitch is in degrees
	pitch           float64
	velocity        float64
	collisionRadius float64
	collisionHeight float64
	mapColor        color.RGBA
	parent          *Entity
}

func (e *Entity) getPos() *Vec2 {
	return e.pos
}

func (e *Entity) getPosZ() float64 {
	return e.posZ
}

// -- level

// MapLayer represents rects and tints of vertical slices --//
type MapLayer struct {
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

// HorizontalMapLayer is for handling horizontal renders that cannot use vertical slices (e.g. floor, ceiling)
type HorizontalMapLayer struct {
	// horBuffer is the image representing the pixels to render during the update
	horBuffer *image.RGBA
	// image is the ebitengine image object rendering the horBuffer during draw
	image *ebiten.Image
}

func (h *HorizontalMapLayer) init(width, height int) {
	h.horBuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	if h.image == nil {
		h.image = ebiten.NewImage(width, height)
	}
}

// -- texture manager

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

func (t *TextureManager) getTextureAt(x, y, levelNum, side int) *ebiten.Image {
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

func (t *TextureManager) getFloorTextureAt(x, y int) *image.RGBA {
	// x/y could be used to render different floor texture at given coords,
	// but for this demo we will just be rendering the same texture everywhere.
	if t.renderFloorTex {
		return t.floorTex
	}
	return nil
}
