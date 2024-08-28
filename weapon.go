package main

import (
	"image/color"

	"github.com/harbdog/raycaster-go"
	"github.com/harbdog/raycaster-go/geom"

	"github.com/hajimehoshi/ebiten/v2"
)

type Weapon struct {
	*SpriteInstance
	firing             bool
	cooldown           int
	rateOfFire         float64
	projectileVelocity float64
	projectile         Projectile
}

func NewAnimatedWeapon(
	x, y, scale float64, animationRate int, img *ebiten.Image, columns, rows int, projectile Projectile, projectileVelocity, rateOfFire float64,
) *Weapon {
	mapColor := color.RGBA{0, 0, 0, 0}
	w := &Weapon{
		SpriteInstance: NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, raycaster.AnchorCenter, 0, 0),
	}
	w.projectile = projectile
	w.projectileVelocity = projectileVelocity
	w.rateOfFire = rateOfFire

	return w
}

func (w *Weapon) Fire() bool {
	if w.cooldown <= 0 {
		// TODO: handle rate of fire greater than 60 per second?
		w.cooldown = int(1 / w.rateOfFire * float64(ebiten.TPS()))

		if !w.firing {
			w.firing = true
			w.SpriteInstance.ResetAnimation()
		}

		return true
	}
	return false
}

func (w *Weapon) SpawnProjectile(x, y, z, angle, pitch float64, spawnedBy *Entity) *Projectile {
	p := Clone(&w.projectile)
	s := Clone(w.projectile.SpriteInstance)

	p.SpriteInstance = s
	p.Position = &geom.Vector2{X: x, Y: y}
	p.PositionZ = z
	p.Angle = angle
	p.Pitch = pitch

	// convert velocity from distance/second to distance per tick
	p.Velocity = w.projectileVelocity / float64(ebiten.TPS())

	// keep track of what spawned it
	p.Parent = spawnedBy

	return p
}

func (w *Weapon) OnCooldown() bool {
	return w.cooldown > 0
}

func (w *Weapon) ResetCooldown() {
	w.cooldown = 0
}

func (w *Weapon) Update() {
	if w.cooldown > 0 {
		w.cooldown -= 1
	}
	if w.firing && w.SpriteInstance.LoopCounter() < 1 {
		w.SpriteInstance.Update(nil)
	} else {
		w.firing = false
		w.SpriteInstance.ResetAnimation()
	}
}
