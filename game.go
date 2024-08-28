package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

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
		if w.h != compSize/3 {
			// weapon should only take up 1/3rd of screen space
			drawScale = (float64(compSize) / 3) / float64(w.h)
		}

		weaponScale := w.Scale() * drawScale * g.renderScale
		op.GeoM.Scale(weaponScale, weaponScale)
		op.GeoM.Translate(
			float64(g.width)/2-float64(w.w)*weaponScale/2,
			float64(g.height)-float64(w.h)*weaponScale+1,
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
			float64(g.screenWidth)/2-float64(g.crosshairs.w)*crosshairScale/2,
			float64(g.screenHeight)/2-float64(g.crosshairs.h)*crosshairScale/2,
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
				s.Angle = randFloat(-math.Pi, math.Pi)
				s.Velocity = randFloat(0.01, 0.03)
			} else {
				s.Position = newPos
			}
		}
		s.Update(g.player.Position)
	}
}
