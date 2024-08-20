// rendering.go
package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	minimapScale int = 8
)

type EntityType int

const (
	entityTypeWallOrConstruct EntityType = iota
	entityTypeEnemy
	entityTypeSprite
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

type SpriteParameters struct {
	spriteScreenX int
	transformY    float64
	spriteHeight  int
	spriteWidth   int
	drawStartY    int
	drawEndY      int
	drawStartX    int
	drawEndX      int
}

type SpriteVisiblePortion struct {
	visibleStartY int
	visibleEndY   int
	drawStartY    int
	drawEndY      int
	drawStartX    int
	drawEndX      int
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

func (g *Game) drawWallOrConstruct(screen *ebiten.Image, x int, dist float64, entity LevelEntity, side int) {
	_, drawStart, drawEnd := g.calculateLineParameters(dist, entity)
	wallColor := g.getEntityColor(entity, side)
	vector.DrawFilledRect(screen, float32(x), float32(drawStart), 1, float32(drawEnd-drawStart), wallColor, false)
}

func (g *Game) drawEnemy(screen *ebiten.Image, d Drawable) {
	enemy := d.enemy
	params := g.calculateSpriteParameters(d)

	// determine which sprite to use based on enemy's orientation relative to player
	enemyToPlayerX := g.player.x - enemy.x
	enemyToPlayerY := g.player.y - enemy.y

	angle := getNormalizedAngle(enemyToPlayerY, enemyToPlayerX, enemy)

	enemySprite := g.getEnemySpriteForAngle(angle)

	visiblePortion := getVisiblePortionOfSprite(enemySprite, params)

	g.drawSprite(screen, enemySprite, params, visiblePortion)
}

func (g *Game) getEnemySpriteForAngle(angle float64) *ebiten.Image {
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
	return enemySprite
}

func (g *Game) calculateSpriteParameters(d Drawable) SpriteParameters {
	params := SpriteParameters{
		spriteScreenX: d.spriteScreenX,
		transformY:    d.transformY,
	}

	params.spriteHeight = int(math.Abs(float64(screenHeight) / params.transformY))
	params.spriteWidth = int(math.Abs(float64(screenHeight) / params.transformY))

	vMoveScreen := int(float64(params.spriteHeight) * (0.5 - g.player.heightOffset))

	params.drawStartY = -params.spriteHeight/2 + screenHeight/2 + vMoveScreen
	params.drawEndY = params.spriteHeight/2 + screenHeight/2 + vMoveScreen

	if d.entityType == entityTypeSprite {
		// Coins don't need vertical movement or angle adjustments
		params.drawStartY = -params.spriteHeight/2 + screenHeight/2
		params.drawEndY = params.spriteHeight/2 + screenHeight/2
	}

	params.drawStartX = -params.spriteWidth/2 + params.spriteScreenX
	params.drawEndX = params.spriteWidth/2 + params.spriteScreenX

	verticalAngleOffset := int(float64(screenHeight) * math.Tan(g.player.verticalAngle))

	params.drawStartY += verticalAngleOffset
	params.drawEndY += verticalAngleOffset

	return params
}

func getVisiblePortionOfSprite(enemySprite *ebiten.Image, params SpriteParameters) SpriteVisiblePortion {
	visibleStartY := 0
	visibleEndY := enemySprite.Bounds().Dy()
	if params.drawStartY < 0 {
		visibleStartY = -params.drawStartY * enemySprite.Bounds().Dy() / params.spriteHeight
		params.drawStartY = 0
	}
	if params.drawEndY >= screenHeight {
		visibleEndY = (screenHeight - params.drawStartY) * enemySprite.Bounds().Dy() / params.spriteHeight
		params.drawEndY = screenHeight - 1
	}
	if params.drawStartX < 0 {
		params.drawStartX = 0
	}
	if params.drawEndX >= screenWidth {
		params.drawEndX = screenWidth - 1
	}
	return SpriteVisiblePortion{
		visibleStartY: visibleStartY,
		visibleEndY:   visibleEndY,
		drawStartY:    params.drawStartY,
		drawEndY:      params.drawEndY,
		drawStartX:    params.drawStartX,
		drawEndX:      params.drawEndX,
	}
}

// draw sprite column by column
func (g *Game) drawSprite(screen *ebiten.Image, enemySprite *ebiten.Image, params SpriteParameters, visiblePortion SpriteVisiblePortion) {
	for stripe := visiblePortion.drawStartX; stripe < visiblePortion.drawEndX; stripe++ {
		if params.transformY > 0 && stripe > 0 && stripe < screenWidth && params.transformY < g.zBuffer[stripe] {
			op := &ebiten.DrawImageOptions{}
			texX := int((float64(stripe-(-params.spriteWidth/2+params.spriteScreenX)) * float64(enemySprite.Bounds().Dx())) / float64(params.spriteWidth))
			subImg := enemySprite.SubImage(image.Rect(texX, visiblePortion.visibleStartY, texX+1, visiblePortion.visibleEndY)).(*ebiten.Image)
			scaleY := float64(visiblePortion.drawEndY-visiblePortion.drawStartY) / float64(visiblePortion.visibleEndY-visiblePortion.visibleStartY)
			op.GeoM.Scale(1, scaleY)
			op.GeoM.Translate(float64(stripe), float64(visiblePortion.drawStartY))
			screen.DrawImage(subImg, op)
		}
	}
}
