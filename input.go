package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) handleInput() {
	// p pauses the game
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		if g.paused {
			ebiten.SetCursorMode(ebiten.CursorModeCaptured)
			g.paused = false
		} else {
			ebiten.SetCursorMode(ebiten.CursorModeVisible)
			g.paused = true
		}
	}

	// escape exits the game
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		exit(0)
	}

	if g.paused {
		// dont process input when paused
		return
	}

	forward := false
	backward := false
	strafeLeft := false
	strafeRight := false

	moveModifier := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		moveModifier = 1.5
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
			g.player.rotate(float64(dx) * moveModifier)
		}

		if dy != 0 {
			g.player.updatePitch(float64(dy))
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

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		strafeLeft = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		strafeRight = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) {
		forward = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
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
		g.move(moveModifier)
	} else if backward {
		g.move(-moveModifier)
	}

	if strafeLeft {
		g.strafe(-moveModifier)
	} else if strafeRight {
		g.strafe(moveModifier)
	}
}
