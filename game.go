// game.go

package main

import (
	"image/color"
	"log"
	"math"
	"os"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

func (g *Game) drawGameOver(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "GAME OVER", screenWidth/2-40, screenHeight/2-10)
	ebitenutil.DebugPrintAt(screen, "Press SPACE to restart", screenWidth/2-80, screenHeight/2+10)
}
