package main

import (
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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

type PatrolPoint struct {
	x, y float64
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

// normalize angle to [-π, π]
func getNormalizedAngle(enemyToPlayerY float64, enemyToPlayerX float64, enemy *Enemy) float64 {
	angle := math.Atan2(enemyToPlayerY, enemyToPlayerX) - math.Atan2(enemy.dirY, enemy.dirX)
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	return angle
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
