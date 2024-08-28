package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/harbdog/raycaster-go"
)

type Effect struct {
	*SpriteInstance
	loopCount int
}

func NewEffect(
	x, y, scale float64, animationRate int, img *ebiten.Image, columns, rows int, anchor raycaster.SpriteAnchor, loopCount int,
) *Effect {
	mapColor := color.RGBA{0, 0, 0, 0}
	e := &Effect{
		SpriteInstance: NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, 0, 0),
		loopCount:      loopCount,
	}

	// effects should not be convergence capable by player focal point
	e.SpriteInstance.Focusable = false

	// effects self illuminate so they do not get dimmed in dark conditions
	e.illumination = 5000

	return e
}
