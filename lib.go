package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	e.Sprite.isFocusable = false

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

// -- settings

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
