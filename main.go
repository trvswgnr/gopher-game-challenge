// main.go
package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"os"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/*
var assets embed.FS

const (
	screenWidth                    int     = 1024
	screenHeight                   int     = 768
	playerSpeedStanding            float64 = 0.08
	playerSpeedCrouching           float64 = 0.01
	playerStandingHeightOffset     float64 = 0.2
	playerCrouchingHeightOffset    float64 = 0.6
	playerCrouchingTransitionSpeed float64 = 0.03
	mouseSensitivity               float64 = 0.002
	minimapScale                   int     = 8
)

type Game struct {
	player                 Player
	enemies                []Enemy
	minimap                *ebiten.Image
	level                  Level
	gameOver               bool
	enemySprites           map[string]*ebiten.Image
	zBuffer                []float64
	prevMouseX, prevMouseY int
	discoveredAreas        [][]float64
}

type Direction int

const (
	North Direction = iota
	East
	South
	West
)

type Enemy struct {
	x, y         float64
	dirX, dirY   float64
	patrolPoints []PatrolPoint
	currentPoint int
	speed        float64
	fovAngle     float64
	fovDistance  float64
}

type PatrolPoint struct {
	x, y float64
}

type Player struct {
	x, y           float64
	dirX, dirY     float64
	planeX, planeY float64
	heightOffset   float64
	isCrouching    bool
	speed          float64
	verticalAngle  float64
}

func NewPlayer(x, y float64) Player {
	offset := 0.5 // offset to center player in tile

	return Player{
		x:             x + offset,
		y:             y + offset,
		dirX:          -1,
		dirY:          0,
		planeX:        0,
		planeY:        0.66,
		heightOffset:  playerStandingHeightOffset,
		isCrouching:   false,
		speed:         playerSpeedStanding,
		verticalAngle: 0,
	}
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

func (g *Game) updateDiscoveredAreas() {
	const discoveryRadius float64 = 5.0 // changes the discovery radius
	const fadeRadius float64 = 2.0      // changes the fade effect radius
	playerX, playerY := int(g.player.x), int(g.player.y)

	for y := playerY - int(discoveryRadius) - int(fadeRadius); y <= playerY+int(discoveryRadius)+int(fadeRadius); y++ {
		for x := playerX - int(discoveryRadius) - int(fadeRadius); x <= playerX+int(discoveryRadius)+int(fadeRadius); x++ {
			if x >= 0 && x < g.level.width() && y >= 0 && y < g.level.height() {
				dx, dy := float64(x-playerX), float64(y-playerY)
				distance := math.Sqrt(dx*dx + dy*dy)

				if distance <= discoveryRadius {
					g.discoveredAreas[y][x] = 1.0
				} else if distance <= discoveryRadius+fadeRadius {
					fade := 1.0 - (distance-discoveryRadius)/fadeRadius
					g.discoveredAreas[y][x] = math.Max(g.discoveredAreas[y][x], fade)
				}
			}
		}
	}
}

func loadEnemySprites() map[string]*ebiten.Image {
	enemySprites := make(map[string]*ebiten.Image)
	spriteNames := []string{"front", "front-left", "front-right", "back", "back-left", "back-right"}

	for _, name := range spriteNames {
		asset, err := assets.Open(fmt.Sprintf("assets/enemy-%s.png", name))
		if err != nil {
			log.Fatalf("failed to load enemy sprite %s: %v", name, err)
		}
		sprite, _, err := ebitenutil.NewImageFromReader(asset)
		if err != nil {
			log.Fatalf("failed to read enemy sprite %s: %v", name, err)
		}
		enemySprites[name] = sprite
	}

	return enemySprites
}

func (g *Game) initializeEnemies() {
	for _, enemyPos := range g.level.getEnemies() {
		enemy := Enemy{
			x:            enemyPos.x,
			y:            enemyPos.y,
			dirX:         1,
			dirY:         0,
			patrolPoints: generatePatrolPoints(g.level, enemyPos.x, enemyPos.y),
			currentPoint: 0,
			speed:        0.01,
			fovAngle:     math.Pi / 3, // 60 degrees
			fovDistance:  5,
		}
		g.enemies = append(g.enemies, enemy)
	}
}

func (g *Game) generateStaticMinimap() {
	g.minimap = ebiten.NewImage(g.level.width()*minimapScale, g.level.height()*minimapScale)
	for y := 0; y < g.level.height(); y++ {
		for x := 0; x < g.level.width(); x++ {
			switch g.level.getEntityAt(x, y) {
			case LevelEntity_Wall:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{50, 50, 50, 255}, false)
			case LevelEntity_Construct:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{140, 140, 140, 255}, false)
			default:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{140, 140, 140, 255}, false)
			}
		}
	}
}

func generatePatrolPoints(level Level, startX, startY float64) []PatrolPoint {
	// todo: do something more interesting here
	points := []PatrolPoint{
		{startX, startY},
		{startX + 1, startY},
		{startX + 2, startY + 2},
		{startX, startY + 2},
	}

	// validate points (make sure they're not walls)
	validPoints := make([]PatrolPoint, 0)
	for _, p := range points {
		if level.getEntityAt(int(p.x), int(p.y)) != LevelEntity_Wall {
			validPoints = append(validPoints, p)
		}
	}

	return validPoints
}

func (g *Game) isPlayerDetectedByEnemy() bool {
	for _, enemy := range g.enemies {
		if g.canEnemySeePlayer(&enemy) {
			return true
		}
	}
	return false
}

func (g *Game) canEnemySeePlayer(enemy *Enemy) bool {
	// calculate angle and distance between enemy and player
	dx := g.player.x - enemy.x
	dy := g.player.y - enemy.y
	distToPlayer := math.Sqrt(dx*dx + dy*dy)
	angleToPlayer := math.Atan2(dy, dx)

	// check if player is within enemy's fov and range
	enemyAngle := math.Atan2(enemy.dirY, enemy.dirX)
	angleDiff := math.Abs(angleToPlayer - enemyAngle)
	if angleDiff > math.Pi {
		angleDiff = 2*math.Pi - angleDiff
	}

	if distToPlayer <= enemy.fovDistance && angleDiff <= enemy.fovAngle/2 {
		// check if there's a clear line of sight
		steps := int(distToPlayer * 100) // change to adjust precision
		lastConstructHeight := 0.0

		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			checkX := enemy.x + t*dx
			checkY := enemy.y + t*dy
			checkTileX, checkTileY := int(checkX), int(checkY)

			// check for out of bounds
			if checkTileX < 0 || checkTileX >= g.level.width() || checkTileY < 0 || checkTileY >= g.level.height() {
				return false
			}

			entity := g.level.getEntityAt(checkTileX, checkTileY)

			// if we hit a wall, enemy can't see player
			if entity == LevelEntity_Wall {
				return false
			}

			// if we hit a construct
			if entity == LevelEntity_Construct {
				constructHeight := 0.5 // might change this if construct heights change
				lastConstructHeight = constructHeight

				// if this is the last step (player's position) and player is crouching
				if i == steps && g.player.isCrouching {
					return false // player is hidden behind the construct
				}
			}

			// we've reached the player's position
			if checkTileX == int(g.player.x) && checkTileY == int(g.player.y) {
				if g.player.isCrouching && lastConstructHeight > 0 {
					return false // player is crouching and there was a construct in the line of sight
				}
				return true // player can be seen
			}
		}
	}
	return false
}

func (g *Game) updateEnemy(e *Enemy) {
	// move towards the current patrol point
	targetX, targetY := e.patrolPoints[e.currentPoint].x, e.patrolPoints[e.currentPoint].y
	dx, dy := targetX-e.x, targetY-e.y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < e.speed {
		// reached the current patrol point, move to the next one
		e.currentPoint = (e.currentPoint + 1) % len(e.patrolPoints)
	} else {
		// move towards the current patrol point
		e.x += (dx / dist) * e.speed
		e.y += (dy / dist) * e.speed
	}

	// update direction
	e.dirX, e.dirY = dx/dist, dy/dist
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

	if ebiten.IsKeyPressed(ebiten.KeyShift) {
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

func (g *Game) movePlayer(forwardSpeed, strafeSpeed float64) {
	nextX := g.player.x + g.player.dirX*forwardSpeed + g.player.planeX*strafeSpeed
	nextY := g.player.y + g.player.dirY*forwardSpeed + g.player.planeY*strafeSpeed

	if !g.playerCollision(nextX, g.player.y) {
		g.player.x = nextX
	}
	if !g.playerCollision(g.player.x, nextY) {
		g.player.y = nextY
	}
}

func (g *Game) strafePlayer(speed float64) {
	g.movePlayer(0, speed)
}

func (g *Game) handleMouseLook() {
	cx, cy := ebiten.CursorPosition()

	if g.prevMouseX == 0 && g.prevMouseY == 0 {
		g.prevMouseX, g.prevMouseY = cx, cy
		return
	}

	sensitivityX := mouseSensitivity
	sensitivityY := mouseSensitivity

	dx := float64(cx - g.prevMouseX)
	dy := float64(cy - g.prevMouseY)

	g.rotatePlayer(-dx * sensitivityX)

	// handle vertical look
	g.player.verticalAngle -= dy * sensitivityY

	// clamp vertical angle to prevent looking too far up or down
	maxVerticalAngle := math.Pi / 3 // 60 degrees
	g.player.verticalAngle = math.Max(-maxVerticalAngle, math.Min(maxVerticalAngle, g.player.verticalAngle))

	g.prevMouseX, g.prevMouseY = cx, cy
}

func (g *Game) playerCollision(x, y float64) bool {
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

func (g *Game) rotatePlayer(angle float64) {
	oldDirX := g.player.dirX
	g.player.dirX = g.player.dirX*math.Cos(angle) - g.player.dirY*math.Sin(angle)
	g.player.dirY = oldDirX*math.Sin(angle) + g.player.dirY*math.Cos(angle)
	oldPlaneX := g.player.planeX
	g.player.planeX = g.player.planeX*math.Cos(angle) - g.player.planeY*math.Sin(angle)
	g.player.planeY = oldPlaneX*math.Sin(angle) + g.player.planeY*math.Cos(angle)
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-40, screenHeight/2-10)
	ebitenutil.DebugPrintAt(screen, "Press SPACE to restart", screenWidth/2-80, screenHeight/2+10)
}

func (g *Game) calculateRayDirection(x int) (float64, float64) {
	cameraX := 2*float64(x)/float64(screenWidth) - 1
	rayDirX := g.player.dirX + g.player.planeX*cameraX
	rayDirY := g.player.dirY + g.player.planeY*cameraX
	return rayDirX, rayDirY
}

func (g *Game) castRay(x int, rayDirX, rayDirY float64) []struct {
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
		hitEntity := g.level.getEntityAt(mapX, mapY)
		if hitEntity != LevelEntity_Empty {
			var dist float64
			if side == 0 {
				dist = (float64(mapX) - g.player.x + (1-float64(stepX))/2) / rayDirX
			} else {
				dist = (float64(mapY) - g.player.y + (1-float64(stepY))/2) / rayDirY
			}

			// update zbuffer
			g.zBuffer[x] = dist

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

func (g *Game) calculateLineParameters(dist float64, entity LevelEntity) (int, int, int) {
	lineHeight := int(float64(screenHeight) / dist)

	// adjust the vertical position based on player height and vertical angle
	verticalOffset := int(float64(screenHeight) * math.Tan(g.player.verticalAngle))
	heightOffset := int((0.5-g.player.heightOffset)*float64(screenHeight)/dist) + verticalOffset

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
	g.player.isCrouching = g.player.heightOffset == playerCrouchingHeightOffset
}

func (g *Game) getEntityColor(entity LevelEntity, side int) color.RGBA {
	var entityColor color.RGBA
	switch entity {
	case LevelEntity_Wall:
		entityColor = color.RGBA{100, 100, 100, 255}
	case LevelEntity_Enemy:
		entityColor = color.RGBA{198, 54, 54, 255}
	case LevelEntity_Exit:
		entityColor = color.RGBA{255, 255, 0, 255}
	case LevelEntity_Player:
		entityColor = color.RGBA{0, 255, 0, 255}
	case LevelEntity_Construct:
		entityColor = color.RGBA{150, 50, 200, 255}
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

func (g *Game) drawDynamicMinimap(screen *ebiten.Image) {
	minimapImage := ebiten.NewImage(g.level.width()*minimapScale, g.level.height()*minimapScale)

	for y := 0; y < g.level.height(); y++ {
		for x := 0; x < g.level.width(); x++ {
			visibility := g.discoveredAreas[y][x]
			if visibility > 0 {
				var tileColor color.RGBA
				switch g.level.getEntityAt(x, y) {
				case LevelEntity_Wall:
					tileColor = color.RGBA{50, 50, 50, 255}
				case LevelEntity_Construct:
					tileColor = color.RGBA{140, 140, 140, 255}
				default:
					tileColor = color.RGBA{200, 200, 200, 255}
				}

				// apply fog effect
				tileColor.R = uint8(float64(tileColor.R) * visibility)
				tileColor.G = uint8(float64(tileColor.G) * visibility)
				tileColor.B = uint8(float64(tileColor.B) * visibility)

				vector.DrawFilledRect(minimapImage, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), tileColor, false)
			} else {
				vector.DrawFilledRect(minimapImage, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{20, 20, 20, 255}, false)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.width()*minimapScale-10), 10)
	screen.DrawImage(minimapImage, op)

	g.drawMinimapPlayer(screen)
	g.drawMinimapEnemies(screen)
}

func (g *Game) drawMinimapPlayer(screen *ebiten.Image) {
	// calculate player position on minimap
	playerX := float32(screenWidth - g.level.width()*minimapScale - 10 + int(g.player.x*float64(minimapScale)))
	playerY := float32(10 + int(g.player.y*float64(minimapScale)))

	// calculate triangle points
	triangleSize := float32(minimapScale)
	angle := math.Atan2(g.player.dirY, g.player.dirX)

	x1 := playerX + triangleSize*float32(math.Cos(angle))
	y1 := playerY + triangleSize*float32(math.Sin(angle))

	x2 := playerX + triangleSize*float32(math.Cos(angle+2.5))
	y2 := playerY + triangleSize*float32(math.Sin(angle+2.5))

	x3 := playerX + triangleSize*float32(math.Cos(angle-2.5))
	y3 := playerY + triangleSize*float32(math.Sin(angle-2.5))

	// define triangle vertices
	vertices := []ebiten.Vertex{
		{DstX: x1, DstY: y1, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: x3, DstY: y3, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}

	// define triangle indices
	indices := []uint16{0, 1, 2}

	// choose color based on crouching state
	var playerColor color.RGBA
	if g.player.isCrouching {
		playerColor = color.RGBA{0, 255, 0, 255} // green when crouching
	} else {
		playerColor = color.RGBA{0, 255, 255, 255} // teal when standing
	}

	// create a 1x1 image with the player color
	playerColorImage := ebiten.NewImage(1, 1)
	playerColorImage.Fill(playerColor)

	// draw the triangle
	screen.DrawTriangles(vertices, indices, playerColorImage, nil)
}

func (g *Game) drawMinimapEnemies(screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		enemyX, enemyY := int(enemy.x), int(enemy.y)

		if g.discoveredAreas[enemyY][enemyX] > 0 {
			screenX := float32(screenWidth - g.level.width()*minimapScale - 10 + int(enemy.x*float64(minimapScale)))
			screenY := float32(10 + int(enemy.y*float64(minimapScale)))

			// draw enemy (red)
			vector.DrawFilledCircle(screen, screenX, screenY, float32(minimapScale)/2, color.RGBA{255, 0, 0, 255}, false)

			// draw field of vision
			centerAngle := math.Atan2(enemy.dirY, enemy.dirX)
			leftAngle := centerAngle - enemy.fovAngle/2
			rightAngle := centerAngle + enemy.fovAngle/2

			// create vertices for the fov arc
			const segments = 20
			vertices := make([]ebiten.Vertex, segments+2)
			indices := make([]uint16, (segments+1)*3)

			// center vertex
			vertices[0] = ebiten.Vertex{
				DstX:   screenX,
				DstY:   screenY,
				SrcX:   0,
				SrcY:   0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 0,
				ColorA: 0.25,
			}

			// arc vertices
			for i := 0; i <= segments; i++ {
				angle := leftAngle + (rightAngle-leftAngle)*float64(i)/float64(segments)
				x := screenX + float32(math.Cos(angle)*enemy.fovDistance*float64(minimapScale))
				y := screenY + float32(math.Sin(angle)*enemy.fovDistance*float64(minimapScale))
				vertices[i+1] = ebiten.Vertex{
					DstX:   x,
					DstY:   y,
					SrcX:   0,
					SrcY:   0,
					ColorR: 1,
					ColorG: 1,
					ColorB: 0,
					ColorA: 0.25,
				}

				if i < segments {
					indices[i*3] = 0
					indices[i*3+1] = uint16(i + 1)
					indices[i*3+2] = uint16(i + 2)
				}
			}

			// draw the filled fov arc
			screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
				CompositeMode: ebiten.CompositeModeLighter,
			})

			// draw the fov outline
			for i := 0; i <= segments; i++ {
				if i < segments {
					vector.StrokeLine(screen, vertices[i+1].DstX, vertices[i+1].DstY, vertices[i+2].DstX, vertices[i+2].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
				}
			}
			vector.StrokeLine(screen, screenX, screenY, vertices[1].DstX, vertices[1].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
			vector.StrokeLine(screen, screenX, screenY, vertices[segments+1].DstX, vertices[segments+1].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
		}
	}
}

var emptySubImage = ebiten.NewImage(3, 3).SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

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

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Player Detected: %t", isPlayerDetected), 10, screenHeight-100)
}

var isPlayerDetected = false

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

	// check if player is in enemy's field of vision
	if g.isPlayerDetectedByEnemy() {
		g.gameOver = false // todo: set to true when not debugging
		isPlayerDetected = true
	} else {
		isPlayerDetected = false
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.gameOver {
		g.drawGameOver(screen)
		return
	}

	// reset zbuffer
	for i := range g.zBuffer {
		g.zBuffer[i] = math.Inf(1)
	}

	// draw floor and ceiling
	floorColor := color.RGBA{30, 30, 30, 255}
	ceilingColor := color.RGBA{160, 227, 254, 255}
	horizon := screenHeight/2 + int(float64(screenHeight)*math.Tan(g.player.verticalAngle))
	for y := 0; y < screenHeight; y++ {
		if y < horizon {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, ceilingColor, false)
		} else {
			vector.DrawFilledRect(screen, 0, float32(y), float32(screenWidth), 1, floorColor, false)
		}
	}

	var drawables []Drawable // all drawable entities

	// collect walls and constructs
	for x := 0; x < screenWidth; x++ {
		rayDirX, rayDirY := g.calculateRayDirection(x)
		entities := g.castRay(x, rayDirX, rayDirY)
		for _, entity := range entities {
			drawables = append(drawables, Drawable{
				entityType: entityTypeWallOrConstruct,
				x:          x,
				dist:       entity.dist,
				entity:     entity.entity,
				side:       entity.side,
			})
		}
	}

	// collect enemies
	for i := range g.enemies {
		enemy := &g.enemies[i]
		spriteX := enemy.x - g.player.x
		spriteY := enemy.y - g.player.y
		invDet := 1.0 / (g.player.planeX*g.player.dirY - g.player.dirX*g.player.planeY)
		transformX := invDet * (g.player.dirY*spriteX - g.player.dirX*spriteY)
		transformY := invDet * (-g.player.planeY*spriteX + g.player.planeX*spriteY)

		spriteScreenX := int((float64(screenWidth) / 2) * (1 + transformX/transformY))

		drawables = append(drawables, Drawable{
			entityType:    entityTypeEnemy,
			x:             spriteScreenX,
			dist:          transformY,
			enemy:         enemy,
			spriteScreenX: spriteScreenX,
			transformY:    transformY,
		})
	}

	// sort drawables by distance (furthest first)
	sort.Slice(drawables, func(i, j int) bool {
		return drawables[i].dist > drawables[j].dist
	})

	// draw all entities in order
	for _, d := range drawables {
		switch d.entityType {
		case entityTypeWallOrConstruct:
			g.drawWallOrConstruct(screen, d.x, d.dist, d.entity, d.side)
		case entityTypeEnemy:
			g.drawEnemy(screen, d)
		}
	}

	g.drawDynamicMinimap(screen)
	g.drawUI(screen)
}

type EntityType int

const (
	entityTypeWallOrConstruct EntityType = iota
	entityTypeEnemy
)

type Drawable struct {
	entityType    EntityType
	x             int
	dist          float64
	entity        LevelEntity
	side          int
	enemy         *Enemy
	spriteScreenX int
	transformY    float64
}

func (g *Game) drawWallOrConstruct(screen *ebiten.Image, x int, dist float64, entity LevelEntity, side int) {
	_, drawStart, drawEnd := g.calculateLineParameters(dist, entity)
	wallColor := g.getEntityColor(entity, side)
	vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
}

func (g *Game) drawEnemy(screen *ebiten.Image, d Drawable) {
	enemy := d.enemy
	spriteScreenX := d.spriteScreenX
	transformY := d.transformY

	// calculate sprite dimensions on screen
	spriteHeight := int(math.Abs(float64(screenHeight) / transformY))
	spriteWidth := int(math.Abs(float64(screenHeight) / transformY))

	// adjust vertical position based on player height
	vMoveScreen := int(float64(spriteHeight) * (0.5 - g.player.heightOffset))
	drawStartY := -spriteHeight/2 + screenHeight/2 + vMoveScreen
	drawEndY := spriteHeight/2 + screenHeight/2 + vMoveScreen
	drawStartX := -spriteWidth/2 + spriteScreenX
	drawEndX := spriteWidth/2 + spriteScreenX

	// adjust for player's vertical look angle
	verticalAngleOffset := int(float64(screenHeight) * math.Tan(g.player.verticalAngle))
	drawStartY += verticalAngleOffset
	drawEndY += verticalAngleOffset

	// determine which sprite to use based on enemy's orientation relative to player
	enemyToPlayerX := g.player.x - enemy.x
	enemyToPlayerY := g.player.y - enemy.y
	angle := math.Atan2(enemyToPlayerY, enemyToPlayerX) - math.Atan2(enemy.dirY, enemy.dirX)

	// normalize angle to [-π, π]
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}

	// select appropriate sprite based on angle
	var spriteName string
	if math.Abs(angle) < math.Pi/6 {
		spriteName = "front"
	} else if angle >= math.Pi/6 && angle < math.Pi/2 {
		spriteName = "front-left"
	} else if angle >= math.Pi/2 && angle < 5*math.Pi/6 {
		spriteName = "back-left"
	} else if angle >= 5*math.Pi/6 || angle < -5*math.Pi/6 {
		spriteName = "back"
	} else if angle >= -5*math.Pi/6 && angle < -math.Pi/2 {
		spriteName = "back-right"
	} else {
		spriteName = "front-right"
	}
	enemySprite := g.enemySprites[spriteName]

	// calculate visible portion of sprite
	visibleStartY := 0
	visibleEndY := enemySprite.Bounds().Dy()
	if drawStartY < 0 {
		visibleStartY = -drawStartY * enemySprite.Bounds().Dy() / spriteHeight
		drawStartY = 0
	}
	if drawEndY >= screenHeight {
		visibleEndY = (screenHeight - drawStartY) * enemySprite.Bounds().Dy() / spriteHeight
		drawEndY = screenHeight - 1
	}
	if drawStartX < 0 {
		drawStartX = 0
	}
	if drawEndX >= screenWidth {
		drawEndX = screenWidth - 1
	}

	// draw sprite column by column
	for stripe := drawStartX; stripe < drawEndX; stripe++ {
		// check if stripe is visible and in front of walls
		if transformY > 0 && stripe > 0 && stripe < screenWidth && transformY < g.zBuffer[stripe] {
			// calculate texture x coordinate
			texX := int((float64(stripe-(-spriteWidth/2+spriteScreenX)) * float64(enemySprite.Bounds().Dx())) / float64(spriteWidth))

			// create sub-image for the current stripe
			subImg := enemySprite.SubImage(image.Rect(texX, visibleStartY, texX+1, visibleEndY)).(*ebiten.Image)

			// set up drawing options
			op := &ebiten.DrawImageOptions{}
			scaleY := float64(drawEndY-drawStartY) / float64(visibleEndY-visibleStartY)
			op.GeoM.Scale(1, scaleY)
			op.GeoM.Translate(float64(stripe), float64(drawStartY))

			// draw the sprite stripe
			screen.DrawImage(subImg, op)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("maze 3d raycasting")
	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

type LevelEntity int

const (
	LevelEntity_Empty LevelEntity = iota
	LevelEntity_Wall
	LevelEntity_Enemy
	LevelEntity_Exit
	LevelEntity_Player
	LevelEntity_Construct
)

type LevelEntityColor = color.RGBA

var (
	LevelEntityColor_Empty     = color.RGBA{255, 255, 255, 255}
	LevelEntityColor_Wall      = color.RGBA{0, 0, 0, 255}
	LevelEntityColor_Enemy     = color.RGBA{255, 0, 0, 255}
	LevelEntityColor_Exit      = color.RGBA{0, 255, 0, 255}
	LevelEntityColor_Player    = color.RGBA{0, 0, 255, 255}
	LevelEntityColor_Construct = color.RGBA{255, 255, 0, 255}
)

type Level [][]LevelEntity

func NewLevel(file fs.File) Level {
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	matrix := make(Level, height)
	for i := range matrix {
		matrix[i] = make([]LevelEntity, width)
	}

	// fill matrix based on pixel colors
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x, y)

			switch {
			case c == LevelEntityColor_Empty:
				matrix[y][x] = LevelEntity_Empty
			case c == LevelEntityColor_Wall:
				matrix[y][x] = LevelEntity_Wall
			case c == LevelEntityColor_Enemy:
				matrix[y][x] = LevelEntity_Enemy
			case c == LevelEntityColor_Exit:
				matrix[y][x] = LevelEntity_Exit
			case c == LevelEntityColor_Player:
				matrix[y][x] = LevelEntity_Player
			case c == LevelEntityColor_Construct:
				matrix[y][x] = LevelEntity_Construct
			}
		}
	}

	return matrix
}

func (level Level) getPlayer() (float64, float64) {
	playerX := 0
	playerY := 0
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				playerX = x
				playerY = y
				// remove player block from level so it doesn't render or collide
				level[y][x] = LevelEntity_Empty
				break
			}
		}
	}

	return float64(playerX), float64(playerY)
}

func (level Level) getEnemies() []Enemy {
	enemies := []Enemy{}
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Enemy {
				enemies = append(enemies, Enemy{x: float64(x), y: float64(y)})
				// remove enemy block from level so it doesn't render or collide
				level[y][x] = LevelEntity_Empty
			}
		}
	}
	return enemies
}

func (l Level) width() int                       { return len(l[0]) }
func (l Level) height() int                      { return len(l) }
func (l Level) getEntityAt(x, y int) LevelEntity { return l[y][x] }
