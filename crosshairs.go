package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

type Crosshairs struct {
	*SpriteInstance
	hitTimer     int
	HitIndicator *SpriteInstance
}

func NewCrosshairs(
	x, y, scale float64, img *ebiten.Image, columns, rows, crosshairIndex, hitIndex int,
) *Crosshairs {
	mapColor := color.RGBA{0, 0, 0, 0}

	normalCrosshairs := &Crosshairs{
		SpriteInstance: NewSpriteFromSheet(x, y, scale, img, mapColor, columns, rows, crosshairIndex, AnchorCenter, 0, 0),
	}

	hitCrosshairs := NewSpriteFromSheet(x, y, scale, img, mapColor, columns, rows, hitIndex, AnchorCenter, 0, 0)

	hitCrosshairs.SetAnimationFrame(hitIndex)
	normalCrosshairs.HitIndicator = hitCrosshairs

	return normalCrosshairs
}

func (c *Crosshairs) ActivateHitIndicator(hitTime int) {
	if c.HitIndicator != nil {
		c.hitTimer = hitTime
	}
}

func (c *Crosshairs) IsHitIndicatorActive() bool {
	return c.HitIndicator != nil && c.hitTimer > 0
}

func (c *Crosshairs) Update() {
	if c.HitIndicator != nil && c.hitTimer > 0 {
		// TODO: prefer to use timer rather than frame update counter?
		c.hitTimer -= 1
	}
}
