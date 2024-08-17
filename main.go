// main.go
package main

import (
	"embed"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/*
var assets embed.FS

const (
	screenWidth                 int     = 1024
	screenHeight                int     = 768
	playerSpeedStanding         float64 = 0.08
	playerSpeedCrouching        float64 = 0.01
	playerRotateSpeed           float64 = 0.07
	playerStandingHeightOffset  float64 = 0.2
	playerCrouchingHeightOffset float64 = 0.6
	playerCrouchingSpeed        float64 = 0.03
)

type Game struct {
	player   Player
	enemies  []Enemy
	minimap  *ebiten.Image
	level    Level
	gameOver bool
}

type Enemy struct {
	x, y           float64
	watchingPlayer bool
}

type Player struct {
	x, y           float64
	dirX, dirY     float64
	planeX, planeY float64
	heightOffset   float64
	isCrouching    bool
	speed          float64
}

func NewPlayer(x, y float64) Player {
	offsetX, offsetY := 0.5, 0.5 // offset to center the player in the tile
	return Player{
		x:            x + offsetX,
		y:            y + offsetY,
		dirX:         -1,
		dirY:         0,
		planeX:       0,
		planeY:       0.66,
		heightOffset: playerStandingHeightOffset,
		isCrouching:  false,
		speed:        playerSpeedStanding,
	}
}

func NewGame() *Game {
	file, err := assets.Open("assets/level-1.png")
	if err != nil {
		log.Fatal(err)
	}
	level := NewLevel(file)
	playerX, playerY := level.GetPlayer()
	player := NewPlayer(playerX, playerY)
	enemies := level.GetEnemies()
	g := &Game{
		player:   player,
		minimap:  ebiten.NewImage(level.Width()*4, level.Height()*4),
		level:    level,
		enemies:  enemies,
		gameOver: false,
	}

	g.generateStaticMinimap()

	return g
}

func (g *Game) generateStaticMinimap() {
	for y := 0; y < g.level.Height(); y++ {
		for x := 0; x < g.level.Width(); x++ {
			if g.level.GetEntityAt(x, y) == LevelEntity_Wall {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{50, 50, 50, 255}, false)
			} else {
				vector.DrawFilledRect(g.minimap, float32(x*4), float32(y*4), 4, 4, color.RGBA{140, 140, 140, 255}, false)
			}
		}
	}
}

var watchCounter int = 0
var watchTimer int = 0

func (g *Game) Update() error {
	if g.gameOver {
		if ebiten.IsKeyPressed(ebiten.KeySpace) {
			// Reset the game
			*g = *NewGame()
		}
		return nil
	}

	watchCounter++

	if watchCounter > 200 {
		watchTimer = 100
		watchCounter = 0
	}

	if watchTimer > 0 {
		watchTimer--
	}

	for i, _ := range g.enemies {
		g.enemies[i].watchingPlayer = watchTimer > 0
	}

	g.handleInput()

	// Check if player is in enemy's line of sight
	if g.isPlayerInEnemySight() {
		g.gameOver = true
	}

	return nil
}
func (g *Game) handleInput() {
	if g.gameOver {
		return
	}

	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.movePlayer(g.player.speed)
	} else if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.movePlayer(-g.player.speed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.rotatePlayer(-playerRotateSpeed)
	} else if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.rotatePlayer(playerRotateSpeed)
	}

	g.player.isCrouching = ebiten.IsKeyPressed(ebiten.KeyC)
	if g.player.isCrouching {
		g.player.speed = playerSpeedCrouching
		g.adjustPlayerHeightOffset(playerCrouchingSpeed)
	} else {
		g.player.speed = playerSpeedStanding
		g.adjustPlayerHeightOffset(-playerCrouchingSpeed)
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		os.Exit(0)
	}
}

func (g *Game) movePlayer(speed float64) {
	nextX := g.player.x + g.player.dirX*speed
	nextY := g.player.y + g.player.dirY*speed

	// Check collision with walls and enemies
	if !g.playerCollision(nextX, g.player.y) {
		g.player.x = nextX
	}
	if !g.playerCollision(g.player.x, nextY) {
		g.player.y = nextY
	}
}

func (g *Game) playerCollision(x, y float64) bool {
	// check if the position is within the level bounds
	if x < 0 || y < 0 || int(x) >= g.level.Width() || int(y) >= g.level.Height() {
		return true
	}

	// check if the position is a wall
	if g.level.GetEntityAt(int(x), int(y)) == LevelEntity_Wall {
		return true
	}

	// check if the position is an enemy
	if g.level.GetEntityAt(int(x), int(y)) == LevelEntity_Enemy {
		return true
	}

	return false
}

func (g *Game) rotatePlayer(angle float64) {
	oldDirX := g.player.dirX
	g.player.dirX = g.player.dirX*math.Cos(angle) - g.player.dirY*math.Sin(angle)
	g.player.dirY = oldDirX*math.Sin(angle) + g.player.dirY*math.Cos(angle)
	oldPlaneX := g.player.planeX
	g.player.planeX = g.player.planeX*math.Cos(angle) - g.player.planeY*math.Sin(angle)
	g.player.planeY = oldPlaneX*math.Sin(angle) + g.player.planeY*math.Cos(angle)
}

func (g *Game) isPlayerInEnemySight() bool {
	for i, enemy := range g.enemies {
		if !g.enemies[i].watchingPlayer {
			continue
		}
		// Calculate direction from enemy center to player center
		enemyCenterX := enemy.x + 0.5
		enemyCenterY := enemy.y + 0.5
		playerCenterX := g.player.x + 0.5
		playerCenterY := g.player.y + 0.5

		dirX := playerCenterX - enemyCenterX
		dirY := playerCenterY - enemyCenterY
		distance := math.Sqrt(dirX*dirX + dirY*dirY)

		// Normalize direction
		dirX /= distance
		dirY /= distance

		// Cast a ray from the enemy center to the player center
		stepX, stepY := dirX*0.05, dirY*0.05 // Use smaller steps for more precision
		curX, curY := enemyCenterX, enemyCenterY

		for i := 0; i < int(distance*20); i++ {
			curX += stepX
			curY += stepY

			mapX, mapY := int(curX), int(curY)

			// Check if we've hit a wall or construct
			entity := g.level.GetEntityAt(mapX, mapY)
			if entity == LevelEntity_Wall {
				// Wall blocks line of sight
				break
			} else if entity == LevelEntity_Construct {
				// If player is crouching behind a construct, they're safe
				if g.player.isCrouching && math.Abs(curX-playerCenterX) > 0.5 && math.Abs(curY-playerCenterY) > 0.5 {
					break
				}
			}

			// Check if we've reached the player
			if math.Abs(curX-playerCenterX) < 0.5 && math.Abs(curY-playerCenterY) < 0.5 {
				// We've reached the player without hitting a wall or construct
				return true
			}
		}
	}
	return false
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.gameOver {
		g.drawGameOver(screen)
		return
	}

	g.drawFloorAndCeiling(screen)
	g.drawBlocks(screen)
	g.drawMinimap(screen)
	g.drawUI(screen)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-40, screenHeight/2-10)
	ebitenutil.DebugPrintAt(screen, "Press SPACE to restart", screenWidth/2-80, screenHeight/2+10)
}

func (g *Game) drawFloorAndCeiling(screen *ebiten.Image) {
	floorColor := color.RGBA{30, 30, 30, 255}
	ceilingColor := color.RGBA{160, 227, 254, 255}

	for y := 0; y < screenHeight; y++ {
		if y < screenHeight/2 {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, ceilingColor, false)
		} else {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, floorColor, false)
		}
	}
}

func (g *Game) drawBlocks(screen *ebiten.Image) {
	for x := 0; x < screenWidth; x++ {
		rayDirX, rayDirY := g.calculateRayDirection(x)
		entities := g.castRay(rayDirX, rayDirY)
		g.drawEntities(screen, x, entities)
	}
}

func (g *Game) calculateRayDirection(x int) (float64, float64) {
	cameraX := 2*float64(x)/float64(screenWidth) - 1
	rayDirX := g.player.dirX + g.player.planeX*cameraX
	rayDirY := g.player.dirY + g.player.planeY*cameraX
	return rayDirX, rayDirY
}

func (g *Game) castRay(rayDirX, rayDirY float64) []struct {
	entity LevelEntity
	dist   float64
	side   int
} {
	mapX, mapY := int(g.player.x), int(g.player.y)
	var sideDistX, sideDistY float64
	deltaDistX := math.Abs(1 / rayDirX)
	deltaDistY := math.Abs(1 / rayDirY)
	var stepX, stepY int
	var side int

	if rayDirX < 0 {
		stepX = -1
		sideDistX = (g.player.x - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1.0 - g.player.x) * deltaDistX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (g.player.y - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1.0 - g.player.y) * deltaDistY
	}

	var hitWall bool
	var entities []struct {
		entity LevelEntity
		dist   float64
		side   int
	}

	for !hitWall {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}
		hitEntity := g.level.GetEntityAt(mapX, mapY)
		if hitEntity != LevelEntity_Empty {
			var dist float64
			if side == 0 {
				dist = (float64(mapX) - g.player.x + (1-float64(stepX))/2) / rayDirX
			} else {
				dist = (float64(mapY) - g.player.y + (1-float64(stepY))/2) / rayDirY
			}
			entities = append(entities, struct {
				entity LevelEntity
				dist   float64
				side   int
			}{hitEntity, dist, side})

			if hitEntity == LevelEntity_Wall {
				hitWall = true
			}
		}
	}

	return entities
}

func (g *Game) drawEntities(screen *ebiten.Image, x int, entities []struct {
	entity LevelEntity
	dist   float64
	side   int
}) {
	for i := len(entities) - 1; i >= 0; i-- {
		entity := entities[i]
		_, drawStart, drawEnd := g.calculateLineParameters(entity.dist, entity.entity)
		wallColor := g.getEntityColor(entity.entity, entity.side)
		vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
	}
}

func (g *Game) calculateLineParameters(dist float64, entity LevelEntity) (int, int, int) {
	lineHeight := int(float64(screenHeight) / dist)

	// adjust the vertical position based on player height
	heightOffset := int((0.5 - g.player.heightOffset) * float64(screenHeight) / dist)

	drawStart := -lineHeight/2 + screenHeight/2 + heightOffset
	drawEnd := lineHeight/2 + screenHeight/2 + heightOffset

	// make walls taller
	if entity == LevelEntity_Wall {
		factor := 2.0
		lineHeight = int(float64(lineHeight) * factor)
		drawStart = drawEnd - lineHeight
	}

	// make constructs shorter
	if entity == LevelEntity_Construct {
		factor := 0.8
		lineHeight = int(float64(lineHeight) * factor)
		drawStart = drawEnd - lineHeight
	}

	if drawStart < 0 {
		drawStart = 0
	}
	if drawEnd >= screenHeight {
		drawEnd = screenHeight - 1
	}

	return lineHeight, drawStart, drawEnd
}

func (g *Game) adjustPlayerHeightOffset(delta float64) {
	g.player.heightOffset += delta
	// clamp the height
	if g.player.heightOffset > playerCrouchingHeightOffset {
		g.player.heightOffset = playerCrouchingHeightOffset
	} else if g.player.heightOffset < playerStandingHeightOffset {
		g.player.heightOffset = playerStandingHeightOffset
	}
}

func (g *Game) getEntityColor(entity LevelEntity, side int) color.RGBA {
	var entityColor color.RGBA
	switch entity {
	case LevelEntity_Wall:
		entityColor = color.RGBA{100, 100, 100, 255}
	case LevelEntity_Enemy:
		entityColor = color.RGBA{198, 54, 54, 255}
	case LevelEntity_Exit:
		entityColor = LevelEntityColor_Exit
	case LevelEntity_Player:
		entityColor = LevelEntityColor_Player
	case LevelEntity_Construct:
		entityColor = LevelEntityColor_Construct
	default:
		entityColor = color.RGBA{200, 200, 200, 255}
	}

	if side == 1 {
		entityColor.R = entityColor.R / 2
		entityColor.G = entityColor.G / 2
		entityColor.B = entityColor.B / 2
	}

	return entityColor
}

func (g *Game) drawMinimap(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.Width()*4-10), 10)
	screen.DrawImage(g.minimap, op)

	g.drawPlayerOnMinimap(screen)
	g.drawEnemiesOnMinimap(screen)

	g.drawEnemyLinesOfSight(screen)
}

func (g *Game) drawPlayerOnMinimap(screen *ebiten.Image) {
	vector.DrawFilledCircle(
		screen,
		float32(screenWidth-g.level.Width()*4-10+int(g.player.x*4)),
		float32(10+int(g.player.y*4)),
		2,
		color.RGBA{255, 0, 0, 255},
		false,
	)
}

func (g *Game) drawEnemiesOnMinimap(screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		vector.DrawFilledCircle(
			screen,
			float32(screenWidth-g.level.Width()*4-10+int(enemy.x*4)),
			float32(10+int(enemy.y*4)),
			2,
			color.RGBA{0, 255, 0, 255},
			false,
		)
	}
}

func (g *Game) drawEnemyLinesOfSight(screen *ebiten.Image) {
	for i, enemy := range g.enemies {
		if !g.enemies[i].watchingPlayer {
			continue
		}
		// Calculate direction from enemy to player
		dirX := g.player.x - enemy.x
		dirY := g.player.y - enemy.y
		distance := math.Sqrt(dirX*dirX + dirY*dirY)

		// Normalize direction
		dirX /= distance
		dirY /= distance

		// Draw line of sight
		startX := float32(screenWidth - g.level.Width()*4 - 10 + int(enemy.x*4))
		startY := float32(10 + int(enemy.y*4))
		endX := float32(screenWidth - g.level.Width()*4 - 10 + int((enemy.x+dirX*distance)*4))
		endY := float32(10 + int((enemy.y+dirY*distance)*4))

		vector.StrokeLine(screen, startX, startY, endX, endY, 1, color.RGBA{255, 255, 0, 128}, false)
	}
}

func (g *Game) drawUI(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()), 10, 10)
	ebitenutil.DebugPrintAt(screen, "move with arrow keys", 10, screenHeight-40)
	ebitenutil.DebugPrintAt(screen, "ESC to exit", 10, screenHeight-20)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("height offset: %0.2f", g.player.heightOffset), 10, screenHeight-60)

	crouchStatus := "Standing"
	if g.player.isCrouching {
		crouchStatus = "Crouching"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", crouchStatus), 10, screenHeight-80)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("watch timer: %d", watchTimer), 10, screenHeight-100)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("maze 3d raycasting")

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
