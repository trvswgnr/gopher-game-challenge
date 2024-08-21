// game.go

package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	player          Player
	enemies         []Enemy
	minimap         *ebiten.Image
	level           Level
	gameOver        bool
	enemySprites    map[string]*ebiten.Image
	zBuffer         []float64
	prevMouseX      int
	prevMouseY      int
	discoveredAreas [][]float64
}

func NewGame() *Game {
	file, err := assets.Open("assets/level-1.png")
	if err != nil {
		log.Fatal(err)
	}

	level := NewLevel(file)

	playerX, playerY := level.getPlayer()
	player := NewPlayer(playerX, playerY)

	g := &Game{
		player:          player,
		minimap:         ebiten.NewImage(level.width()*minimapScale, level.height()*minimapScale),
		level:           level,
		enemies:         make([]Enemy, 0),
		gameOver:        false,
		enemySprites:    loadEnemySprites(),
		zBuffer:         make([]float64, screenWidth),
		prevMouseX:      0,
		prevMouseY:      0,
		discoveredAreas: make([][]float64, level.height()),
	}

	for i := range g.discoveredAreas {
		g.discoveredAreas[i] = make([]float64, level.width())
	}

	g.initializeEnemies()

	g.generateStaticMinimap()

	g.updateDiscoveredAreas()

	return g
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) Update() error {
	if g.gameOver {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			// reset the game
			*g = *NewGame()
		}
		return nil
	}

	g.handleInput()
	g.updateDiscoveredAreas()

	// update enemies
	for i := range g.enemies {
		g.updateEnemy(&g.enemies[i])
	}

	deltaTime := 1.0 / ebiten.ActualFPS()
	g.updateCoins(deltaTime)

	// check if player is in enemy's field of vision
	if g.isPlayerDetectedByEnemy() {
		g.gameOver = false // todo: set to true when not debugging
		isPlayerDetected = true
	} else {
		isPlayerDetected = false
	}

	return nil
}

func (g *Game) handleInput() {
	if g.gameOver {
		return
	}

	moveSpeed := g.player.speed

	strafeSpeed := g.player.speed * 0.75 // slightly slower strafing

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.movePlayer(moveSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.movePlayer(-moveSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.strafePlayer(-strafeSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.strafePlayer(strafeSpeed)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		g.dropCoin()
	}

	if ebiten.IsKeyPressed(ebiten.KeyControl) {
		g.player.speed = playerSpeedCrouching
		g.adjustPlayerHeightOffset(playerCrouchingTransitionSpeed)
	} else {
		g.player.speed = playerSpeedStanding
		g.adjustPlayerHeightOffset(-playerCrouchingTransitionSpeed)
	}

	g.handleMouseLook()

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}
}

func (g *Game) checkCollision(x, y float64) bool {
	// check position is within level bounds
	if x < 0 || y < 0 || int(x) >= g.level.width() || int(y) >= g.level.height() {
		return true
	}

	// check position is wall or construct
	entity := g.level.getEntityAt(int(x), int(y))
	if entity == LevelEntity_Wall || entity == LevelEntity_Construct {
		return true
	}

	// check enemy collision
	for _, enemy := range g.enemies {
		dx := x - enemy.x
		dy := y - enemy.y
		distSquared := dx*dx + dy*dy
		if distSquared < 0.25 { // collision radius of 0.5
			g.gameOver = true // running into an enemy probably alerts them lol
			return true
		}
	}

	return false
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-40, screenHeight/2-10)
	ebitenutil.DebugPrintAt(screen, "Press SPACE to restart", screenWidth/2-80, screenHeight/2+10)
}
