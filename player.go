// player.go
package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerSpeedStanding            float64 = 0.05
	playerSpeedCrouching           float64 = 0.01
	playerStandingHeightOffset     float64 = 0.2
	playerCrouchingHeightOffset    float64 = 0.6
	playerCrouchingTransitionSpeed float64 = 0.03
	mouseSensitivity               float64 = 0.002
)

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

func (g *Game) rotatePlayer(angle float64) {
	oldDirX := g.player.dirX
	g.player.dirX = g.player.dirX*math.Cos(angle) - g.player.dirY*math.Sin(angle)
	g.player.dirY = oldDirX*math.Sin(angle) + g.player.dirY*math.Cos(angle)
	oldPlaneX := g.player.planeX
	g.player.planeX = g.player.planeX*math.Cos(angle) - g.player.planeY*math.Sin(angle)
	g.player.planeY = oldPlaneX*math.Sin(angle) + g.player.planeY*math.Cos(angle)
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
