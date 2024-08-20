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

func (g *Game) updateEnemy(e *Enemy) {
	nearestToken := g.findNearestCoin(e)
	if nearestToken != nil && g.distanceBetween(e.x, e.y, nearestToken.x, nearestToken.y) <= coinAttractionDistance {
		g.enemyFollowCoin(e, nearestToken)
	} else {
		g.enemyPatrol(e)
	}
}

func (g *Game) findNearestCoin(e *Enemy) *Coin {
	var nearestToken *Coin
	nearestDist := math.Inf(1)

	for i := range coins {
		dist := g.distanceBetween(e.x, e.y, coins[i].x, coins[i].y)
		if dist < nearestDist {
			nearestDist = dist
			nearestToken = &coins[i]
		}
	}

	return nearestToken
}

func (g *Game) distanceBetween(x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	return math.Sqrt(dx*dx + dy*dy)
}

func (g *Game) enemyFollowCoin(e *Enemy, nearestToken *Coin) {
	// Calculate direction to token
	dx, dy := nearestToken.x-e.x, nearestToken.y-e.y
	dist := math.Sqrt(dx*dx + dy*dy)

	// Stop a little before the token
	stopDistance := 0.5 // Adjust this value to change how close the enemy gets to the token
	if dist > stopDistance {
		// Move towards the token
		e.x += (dx / dist) * e.speed
		e.y += (dy / dist) * e.speed

		// Update direction (facing) only while moving
		e.dirX, e.dirY = dx/dist, dy/dist
	}
	// If within stopDistance, the enemy stops moving and keeps its current direction
}

func (g *Game) enemyPatrol(e *Enemy) {
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
