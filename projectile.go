package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// -- projectile

type Projectile struct {
	*Sprite
	ricochets int
	// lifespan is the time in seconds that the projectile will exist for
	lifespan     float64
	impactEffect Effect
	gravity      float64
}

func NewProjectile(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight, gravity float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewSprite(x, y, scale, img, mapColor, anchor, collisionRadius, collisionHeight),
		ricochets:    0,
		lifespan:     math.MaxFloat64,
		impactEffect: Effect{},
		gravity:      gravity,
	}

	// projectiles should not be convergence capable by player focal point
	p.isFocusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func NewAnimatedProjectile(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA, columns, rows int,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64, gravity float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, collisionRadius, collisionHeight),
		ricochets:    0,
		lifespan:     math.MaxFloat64,
		impactEffect: Effect{},
		gravity:      gravity,
	}

	// projectiles should not be convergence capable by player focal point
	p.isFocusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func (p *Projectile) spawnEffect(x, y, z, angle, pitch float64) *Effect {
	impactEffect := clone(&p.impactEffect)
	spriteInstance := clone(p.impactEffect.Sprite)

	impactEffect.Sprite = spriteInstance
	impactEffect.pos = &Vec2{X: x, Y: y}
	impactEffect.posZ = z
	impactEffect.angle = angle
	impactEffect.pitch = pitch

	// keep track of what spawned it
	impactEffect.parent = p.parent

	return impactEffect
}
