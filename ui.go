// ui.go
package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var isPlayerDetected = false

func (g *Game) drawUI(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()), 10, 10)
	ebitenutil.DebugPrintAt(screen, "move with WASD, look with mouse, ctrl to crouch", 10, screenHeight-40)
	ebitenutil.DebugPrintAt(screen, "ESC to exit", 10, screenHeight-20)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("height offset: %0.2f", g.player.heightOffset), 10, screenHeight-60)

	crouchStatus := "Standing"
	if g.player.isCrouching {
		crouchStatus = "Crouching"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", crouchStatus), 10, screenHeight-80)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Player Detected: %t", isPlayerDetected), 10, screenHeight-100)
}
