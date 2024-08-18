// main.go
package main

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
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
	screenWidth                 int     = 1024
	screenHeight                int     = 768
	playerSpeedStanding         float64 = 0.08
	playerSpeedCrouching        float64 = 0.01
	playerRotateSpeed           float64 = 0.07
	playerStandingHeightOffset  float64 = 0.2
	playerCrouchingHeightOffset float64 = 0.6
	playerCrouchingSpeed        float64 = 0.03
	enemySpriteScale                    = 0.5
)

type Game struct {
	player      Player
	enemies     []Enemy
	minimap     *ebiten.Image
	level       Level
	gameOver    bool
	enemySprite *ebiten.Image
	zBuffer     []float64
}

type Enemy struct {
	x, y           float64
	watchingPlayer bool
	sprite         *ebiten.Image
	height         float64
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

	enemySprite, err := loadEnemySprite()
	if err != nil {
		log.Fatal(err)
	}
	g := &Game{
		player:      player,
		minimap:     ebiten.NewImage(level.Width()*4, level.Height()*4),
		level:       level,
		enemies:     enemies,
		gameOver:    false,
		enemySprite: enemySprite,
		zBuffer:     make([]float64, screenWidth),
	}

	for i := range g.enemies {
		g.enemies[i].sprite = enemySprite
		g.enemies[i].height = 1.0
	}

	g.generateStaticMinimap()

	return g
}

func loadEnemySprite() (*ebiten.Image, error) {
	f, err := assets.Open("assets/enemy_front.png")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
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
			// reset the game
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

	// check if player is in enemy's line of sight
	if g.isPlayerInEnemySight() {
		g.gameOver = false
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

	// check collision with walls and enemies
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
		// calculate direction from enemy center to player center
		enemyCenterX := enemy.x + 0.5
		enemyCenterY := enemy.y + 0.5
		playerCenterX := g.player.x + 0.5
		playerCenterY := g.player.y + 0.5

		dirX := playerCenterX - enemyCenterX
		dirY := playerCenterY - enemyCenterY
		distance := math.Sqrt(dirX*dirX + dirY*dirY)

		// normalize direction
		dirX /= distance
		dirY /= distance

		// cast a ray from the enemy center to the player center
		stepX, stepY := dirX*0.05, dirY*0.05 // use smaller steps for more precision
		curX, curY := enemyCenterX, enemyCenterY

		for i := 0; i < int(distance*20); i++ {
			curX += stepX
			curY += stepY

			mapX, mapY := int(curX), int(curY)

			// check if we've hit a wall or construct
			entity := g.level.GetEntityAt(mapX, mapY)
			if entity == LevelEntity_Wall {
				// wall blocks line of sight
				break
			} else if entity == LevelEntity_Construct {
				// if player is crouching behind a construct, they're safe
				if g.player.isCrouching && math.Abs(curX-playerCenterX) > 0.5 && math.Abs(curY-playerCenterY) > 0.5 {
					break
				}
			}

			// check if we've reached the player
			if math.Abs(curX-playerCenterX) < 0.5 && math.Abs(curY-playerCenterY) < 0.5 {
				// we've reached the player without hitting a wall or construct
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
	g.drawEntitiesAndSprites(screen) // Replace drawBlocks with this new method
	g.drawMinimap(screen)
	g.drawUI(screen)
}

func (g *Game) drawEntitiesAndSprites(screen *ebiten.Image) {
	// Prepare a slice to hold all renderable objects (walls, constructs, and sprites)
	type renderObject struct {
		isSprite bool
		entity   LevelEntity
		enemy    *Enemy
		dist     float64
		x        int
		side     int
	}
	var renderObjects []renderObject

	// Initialize zBuffer
	for i := range g.zBuffer {
		g.zBuffer[i] = math.Inf(1)
	}

	// Cast rays and collect wall and construct objects
	for x := 0; x < screenWidth; x++ {
		rayDirX, rayDirY := g.calculateRayDirection(x)
		entities := g.castRay(rayDirX, rayDirY)
		for _, entity := range entities {
			renderObjects = append(renderObjects, renderObject{
				isSprite: false,
				entity:   entity.entity,
				dist:     entity.dist,
				x:        x,
				side:     entity.side,
			})
		}
	}

	// Collect sprite objects
	for i := range g.enemies {
		enemy := &g.enemies[i]
		// Calculate sprite position relative to camera
		spriteX := enemy.x - g.player.x
		spriteY := enemy.y - g.player.y

		// Transform sprite with the inverse camera matrix
		invDet := 1.0 / (g.player.planeX*g.player.dirY - g.player.dirX*g.player.planeY)
		transformX := invDet * (g.player.dirY*spriteX - g.player.dirX*spriteY)
		transformY := invDet * (-g.player.planeY*spriteX + g.player.planeX*spriteY)

		spriteScreenX := int((float64(screenWidth) / 2) * (1 + transformX/transformY))

		renderObjects = append(renderObjects, renderObject{
			isSprite: true,
			enemy:    enemy,
			dist:     transformY,
			x:        spriteScreenX,
		})
	}

	// Sort all objects by distance (furthest first)
	sort.Slice(renderObjects, func(i, j int) bool {
		return renderObjects[i].dist > renderObjects[j].dist
	})

	// Render all objects
	for _, obj := range renderObjects {
		if obj.isSprite {
			g.drawSprite(screen, obj.enemy, obj.x, obj.dist)
		} else {
			g.drawEntity(screen, obj.x, obj.entity, obj.dist, obj.side)
			// Update zBuffer for walls and constructs
			if obj.x >= 0 && obj.x < screenWidth {
				g.zBuffer[obj.x] = obj.dist
			}
		}
	}
}

func (g *Game) drawSprite(screen *ebiten.Image, enemy *Enemy, spriteScreenX int, transformY float64) {
	spriteHeight := int(math.Abs(float64(screenHeight)/transformY) * enemySpriteScale)
	drawStartY := -spriteHeight/2 + screenHeight/2
	drawEndY := spriteHeight/2 + screenHeight/2

	spriteWidth := spriteHeight
	drawStartX := -spriteWidth/2 + spriteScreenX
	drawEndX := spriteWidth/2 + spriteScreenX

	// Adjust vertical position based on enemy height
	heightOffset := int((0.5 - enemy.height/2) * float64(screenHeight) / transformY)
	drawStartY += heightOffset
	drawEndY += heightOffset

	for stripe := drawStartX; stripe < drawEndX; stripe++ {
		if stripe < 0 || stripe >= screenWidth {
			continue
		}

		texX := int(256*(stripe-(-spriteWidth/2+spriteScreenX))*enemy.sprite.Bounds().Dx()/spriteWidth) / 256

		// Check if the sprite is behind a wall or construct
		if transformY > 0 && transformY < g.zBuffer[stripe] {
			for y := drawStartY; y < drawEndY; y++ {
				if y < 0 || y >= screenHeight {
					continue
				}

				texY := int(256*(y-drawStartY)*enemy.sprite.Bounds().Dy()/spriteHeight) / 256

				color := enemy.sprite.At(texX, texY)
				if _, _, _, a := color.RGBA(); a > 0 {
					screen.Set(stripe, y, color)
				}

			}
		}
	}
}

func (g *Game) drawEntity(screen *ebiten.Image, x int, entity LevelEntity, dist float64, side int) {
	_, drawStart, drawEnd := g.calculateLineParameters(dist, entity)
	wallColor := g.getEntityColor(entity, side)
	if entity == LevelEntity_Enemy {
		return
	}

	vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
}

func (g *Game) drawSprites(screen *ebiten.Image) {
	// sort enemies by distance from player (furthest first)
	sort.Slice(g.enemies, func(i, j int) bool {
		distI := math.Pow(g.enemies[i].x-g.player.x, 2) + math.Pow(g.enemies[i].y-g.player.y, 2)
		distJ := math.Pow(g.enemies[j].x-g.player.x, 2) + math.Pow(g.enemies[j].y-g.player.y, 2)
		return distI > distJ
	})

	for _, enemy := range g.enemies {
		// translate sprite position to relative to camera
		spriteX := enemy.x - g.player.x
		spriteY := enemy.y - g.player.y

		// transform sprite with the inverse camera matrix
		invDet := 1.0 / (g.player.planeX*g.player.dirY - g.player.dirX*g.player.planeY)
		transformX := invDet * (g.player.dirY*spriteX - g.player.dirX*spriteY)
		transformY := invDet * (-g.player.planeY*spriteX + g.player.planeX*spriteY)

		spriteScreenX := int((float64(screenWidth) / 2) * (1 + transformX/transformY))

		// calculate sprite dimensions on screen
		spriteHeight := int(math.Abs(float64(screenHeight)/transformY) * enemySpriteScale)
		drawStartY := -spriteHeight/2 + screenHeight/2
		if drawStartY < 0 {
			drawStartY = 0
		}
		drawEndY := spriteHeight/2 + screenHeight/2
		if drawEndY >= screenHeight {
			drawEndY = screenHeight - 1
		}

		spriteWidth := spriteHeight
		drawStartX := -spriteWidth/2 + spriteScreenX
		if drawStartX < 0 {
			drawStartX = 0
		}
		drawEndX := spriteWidth/2 + spriteScreenX
		if drawEndX >= screenWidth {
			drawEndX = screenWidth - 1
		}

		// draw the sprite
		for stripe := drawStartX; stripe < drawEndX; stripe++ {
			texX := int(256*(stripe-(-spriteWidth/2+spriteScreenX))*enemy.sprite.Bounds().Dx()/spriteWidth) / 256

			if transformY > 0 && stripe > 0 && stripe < screenWidth && transformY < g.zBuffer[stripe] {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(spriteWidth)/float64(enemy.sprite.Bounds().Dx()), float64(spriteHeight)/float64(enemy.sprite.Bounds().Dy()))
				op.GeoM.Translate(float64(stripe), float64(drawStartY))
				screen.DrawImage(enemy.sprite.SubImage(image.Rect(texX, 0, texX+1, enemy.sprite.Bounds().Dy())).(*ebiten.Image), op)
			}
		}
	}
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

		// update zbuffer with the distance to the wall
		if len(entities) > 0 {
			g.zBuffer[x] = entities[0].dist
		} else {
			g.zBuffer[x] = math.Inf(1)
		}
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
		if entity.entity == LevelEntity_Enemy {
			continue
		}
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
		// calculate direction from enemy to player
		dirX := g.player.x - enemy.x
		dirY := g.player.y - enemy.y
		distance := math.Sqrt(dirX*dirX + dirY*dirY)

		// normalize direction
		dirX /= distance
		dirY /= distance

		// draw line of sight
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
