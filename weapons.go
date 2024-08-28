package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// -- projectile

type Projectile struct {
	*Sprite
	Ricochets    int
	Lifespan     float64
	ImpactEffect Effect
	Gravity      float64
}

func NewProjectile(
	x, y, scale float64, img *ebiten.Image, mapColor color.RGBA,
	anchor SpriteAnchor, collisionRadius, collisionHeight, gravity float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewSprite(x, y, scale, img, mapColor, anchor, collisionRadius, collisionHeight),
		Ricochets:    0,
		Lifespan:     math.MaxFloat64,
		ImpactEffect: Effect{},
		Gravity:      gravity,
	}

	// projectiles should not be convergence capable by player focal point
	p.isFocusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func NewAnimatedProjectile(
	x, y, scale float64, animationRate int, img *ebiten.Image, mapColor color.RGBA, columns, rows int,
	anchor SpriteAnchor, collisionRadius, collisionHeight float64,
) *Projectile {
	p := &Projectile{
		Sprite:       NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, anchor, collisionRadius, collisionHeight),
		Ricochets:    0,
		Lifespan:     math.MaxFloat64,
		ImpactEffect: Effect{},
	}

	// projectiles should not be convergence capable by player focal point
	p.isFocusable = false

	// projectiles self illuminate so they do not get dimmed in dark conditions
	p.illumination = 5000

	return p
}

func (p *Projectile) SpawnEffect(x, y, z, angle, pitch float64) *Effect {
	impactEffect := clone(&p.ImpactEffect)
	spriteInstance := clone(p.ImpactEffect.Sprite)

	impactEffect.Sprite = spriteInstance
	impactEffect.Position = &Vec2{X: x, Y: y}
	impactEffect.PositionZ = z
	impactEffect.Angle = angle
	impactEffect.Pitch = pitch

	// keep track of what spawned it
	impactEffect.Parent = p.Parent

	return impactEffect
}

// -- weapon

type Weapon struct {
	*Sprite
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
		Sprite: NewAnimatedSprite(x, y, scale, animationRate, img, mapColor, columns, rows, AnchorCenter, 0, 0),
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
			w.Sprite.ResetAnimation()
		}

		return true
	}
	return false
}

func (w *Weapon) SpawnProjectile(x, y, z, angle, pitch float64, spawnedBy *Entity) *Projectile {
	p := clone(&w.projectile)
	s := clone(w.projectile.Sprite)

	p.Sprite = s
	p.Position = &Vec2{X: x, Y: y}
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
	if w.firing && w.Sprite.LoopCounter() < 1 {
		w.Sprite.Update(nil)
	} else {
		w.firing = false
		w.Sprite.ResetAnimation()
	}
}

func (g *Game) fireWeapon() {
	w := g.player.weapon
	if w == nil {
		g.player.NextWeapon(false)
		return
	}
	if w.OnCooldown() {
		return
	}

	// set weapon firing for animation to run
	w.Fire()

	// spawning projectile at player position just slightly below player's center point of view
	pX, pY, pZ := g.player.Position.X, g.player.Position.Y, clamp(g.player.cameraZ-0.1, 0.05, 0.95)
	// pitch, angle based on raycasted point at crosshairs
	var pAngle, pPitch float64
	convergenceDistance := g.camera.GetConvergenceDistance()
	convergencePoint := g.camera.GetConvergencePoint()
	if convergenceDistance <= 0 || convergencePoint == nil {
		pAngle, pPitch = g.player.Angle, g.player.Pitch
	} else {
		convergenceLine3d := &Line3d{
			X1: pX, Y1: pY, Z1: pZ,
			X2: convergencePoint.X, Y2: convergencePoint.Y, Z2: convergencePoint.Z,
		}
		pAngle, pPitch = convergenceLine3d.heading(), convergenceLine3d.pitch()
	}

	projectile := w.SpawnProjectile(pX, pY, pZ, pAngle, pPitch, g.player.Entity)
	if projectile != nil {
		g.addProjectile(projectile)
	}
}

func (g *Game) updateProjectiles() {
	for p := range g.projectiles {
		if p.Velocity != 0 {

			trajectory := line3dFromAngle(p.Position.X, p.Position.Y, p.PositionZ, p.Angle, p.Pitch, p.Velocity)

			xCheck := trajectory.X2
			yCheck := trajectory.Y2
			zCheck := trajectory.Z2

			zCheck -= p.Gravity

			newPos, isCollision, collisions := g.getValidMove(p.Entity, xCheck, yCheck, zCheck, false)
			if isCollision || p.PositionZ <= 0 {
				// for testing purposes, projectiles instantly get deleted when collision occurs
				g.deleteProjectile(p)

				// make a sprite/wall getting hit by projectile cause some visual effect
				if p.ImpactEffect.Sprite != nil {
					if len(collisions) >= 1 {
						// use the first collision point to place effect at
						newPos = collisions[0].collision
					}

					// TODO: give impact effect optional ability to have some velocity based on the projectile movement upon impact if it didn't hit a wall
					effect := p.SpawnEffect(newPos.X, newPos.Y, p.PositionZ, p.Angle, p.Pitch)

					g.addEffect(effect)
				}

				for _, collisionEntity := range collisions {
					if collisionEntity.entity == g.player.Entity {
						println("ouch!")
					} else {
						// show crosshair hit effect
						g.crosshairs.ActivateHitIndicator(30)
					}
				}
			} else {
				p.Position = newPos
				p.PositionZ = zCheck

				// Update pitch due to gravity
				if p.Gravity != 0 {
					dx := p.Position.X - trajectory.X1
					dy := p.Position.Y - trajectory.Y1
					dz := p.PositionZ - trajectory.Z1
					p.Pitch = math.Atan2(dz, math.Sqrt(dx*dx+dy*dy))
				}
			}
		}
		p.Update(g.player.Position)
	}

	// Testing animated effects (explosions)
	for e := range g.effects {
		e.Update(g.player.Position)
		if e.LoopCounter() >= e.loopCount {
			g.deleteEffect(e)
		}
	}
}
