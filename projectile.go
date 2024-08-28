package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Projectile struct {
	*SpriteInstance
	Ricochets    int
	Lifespan     float64
	ImpactEffect Effect
}

func NewProjectile(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Projectile {
	p := &Projectile{
		SpriteInstance: NewSprite(x, y, scale, img, mapColor, anchor, collisionRadius, collisionHeight),
		Ricochets:      0,
		Lifespan:       math.MaxFloat64,
		ImpactEffect:   Effect{},
	}

	// projectiles should not be convergence capable by player focal point
	p.Focusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func NewAnimatedProjectile(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA, columns, rows int,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Projectile {
	p := &Projectile{
		SpriteInstance: NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, collisionRadius, collisionHeight),
		Ricochets:      0,
		Lifespan:       math.MaxFloat64,
		ImpactEffect:   Effect{},
	}

	// projectiles should not be convergence capable by player focal point
	p.Focusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func (p *Projectile) SpawnEffect(x, y, z, angle, pitch float64) *Effect {
	impactEffect := Clone(&p.ImpactEffect)
	spriteInstance := Clone(p.ImpactEffect.SpriteInstance)

	impactEffect.SpriteInstance = spriteInstance
	impactEffect.Position = &Vector2{X: x, Y: y}
	impactEffect.PositionZ = z
	impactEffect.Angle = angle
	impactEffect.Pitch = pitch

	// keep track of what spawned it
	impactEffect.Parent = p.Parent

	return impactEffect
}
