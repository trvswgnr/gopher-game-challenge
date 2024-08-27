package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) handleInput() {
	// if p, pause game
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.paused {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
			g.paused = false
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			g.paused = true
		}
	}

	// if escape, exit game
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		exit(0)
	}

	if g.paused {
		return
	}

	forward := false
	backward := false
	strafeLeft := false
	strafeRight := false

	moveModifier := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		moveModifier = 2.0
	}

	if ebiten.CursorMode() != ebiten.CursorModeCaptured {
		ebiten.SetCursorMode(ebiten.CursorModeCaptured)

		// reset initial mouse capture position
		g.mouseX, g.mouseY = math.MinInt32, math.MinInt32
	}

	x, y := ebiten.CursorPosition()

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.fireWeapon()
	}

	if g.mouseX == math.MinInt32 && g.mouseY == math.MinInt32 {
		// initialize first position to establish delta
		if x != 0 && y != 0 {
			g.mouseX, g.mouseY = x, y
		}

	} else {
		dx, dy := g.mouseX-x, g.mouseY-y
		g.mouseX, g.mouseY = x, y

		if dx != 0 {
			g.player.rotate(0.005 * float64(dx) * moveModifier)
		}

		if dy != 0 {
			g.player.updatePitch(0.005 * float64(dy))
		}
	}

	_, wheelY := ebiten.Wheel()
	if wheelY != 0 {
		g.player.NextWeapon(wheelY > 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit1) {
		g.player.SelectWeapon(0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyDigit2) {
		g.player.SelectWeapon(1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyH) {
		// put away/holster weapon
		g.player.SelectWeapon(-1)
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		strafeLeft = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		strafeRight = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		forward = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		backward = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyC) {
		g.player.crouch()
	} else if ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.player.goProne()
	} else if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.player.Jump()
	} else {
		// Apply gravity when space is not pressed
		g.player.applyGravity()
	}

	if forward {
		g.move(0.06 * moveModifier)
	} else if backward {
		g.move(-0.06 * moveModifier)
	}

	if strafeLeft {
		g.strafe(-0.05 * moveModifier)
	} else if strafeRight {
		g.strafe(0.05 * moveModifier)
	}
}
