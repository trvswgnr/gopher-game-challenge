package main

import (
	"image/color"
	"math"
)

// -- player

type Player struct {
	*Entity
	cameraZ      float64
	moved        bool
	weapon       *Weapon
	weaponSet    []*Weapon
	lastWeapon   *Weapon
	velocityZ    float64
	jumpHoldTime float64
	isJumping    bool
}

func NewPlayer(x, y, angle, pitch float64) *Player {
	p := &Player{
		Entity: &Entity{
			pos:      &Vec2{X: x, Y: y},
			posZ:     0,
			angle:    angle,
			pitch:    pitch,
			velocity: 0,
			mapColor: color.RGBA{255, 0, 0, 255},
		},
		cameraZ:      0.5,
		moved:        false,
		weaponSet:    []*Weapon{},
		jumpHoldTime: 0,
		isJumping:    false,
	}

	return p
}

func (p *Player) AddWeapon(w *Weapon) {
	p.weaponSet = append(p.weaponSet, w)
}

func (p *Player) SelectWeapon(weaponIndex int) *Weapon {
	// TODO: add some kind of sheath/unsheath animation
	if weaponIndex < 0 {
		// put away weapon
		if p.weapon != nil {
			// store as last weapon
			p.lastWeapon = p.weapon
		}
		p.weapon = nil
		return nil
	}
	newWeapon := p.weapon
	if weaponIndex < len(p.weaponSet) {
		newWeapon = p.weaponSet[weaponIndex]
	}
	if newWeapon != p.weapon {
		// store as last weapon
		p.lastWeapon = p.weapon
		p.weapon = newWeapon
	}
	return p.weapon
}

func (p *Player) NextWeapon(reverse bool) *Weapon {
	_, weaponIndex := p.getSelectedWeapon()
	if weaponIndex < 0 {
		// check last weapon in event of unsheathing previously sheathed weapon
		weaponIndex = p.getWeaponIndex(p.lastWeapon)
		if weaponIndex < 0 {
			weaponIndex = 0
		}
		return p.SelectWeapon(weaponIndex)
	}

	weaponIndex++
	if weaponIndex >= len(p.weaponSet) {
		weaponIndex = 0
	}
	return p.SelectWeapon(weaponIndex)
}

func (p *Player) getWeaponIndex(w *Weapon) int {
	if w == nil {
		return -1
	}
	for index, wCheck := range p.weaponSet {
		if wCheck == w {
			return index
		}
	}
	return -1
}

func (p *Player) getSelectedWeapon() (*Weapon, int) {
	if p.weapon == nil {
		return nil, -1
	}

	return p.weapon, p.getWeaponIndex(p.weapon)
}

func (p *Player) IsStanding() bool {
	return p.posZ == 0 && p.cameraZ == 0.5
}

const (
	jumpVelocity = 3.2
	gravity      = 9.8
)

func (p *Player) Jump() {
	if p.IsStanding() {
		p.velocityZ = jumpVelocity
		p.moved = true
	}

	p.velocityZ -= gravity * (1.0 / 60.0) // Assuming 60 FPS
	p.posZ += p.velocityZ * (1.0 / 60.0)

	if p.posZ <= 0 {
		p.posZ = 0
		p.velocityZ = 0
		p.Stand()
	} else {
		p.cameraZ = 0.5 + p.posZ
	}
	p.moved = true
}

func (p *Player) applyGravity() {
	if !p.IsStanding() {
		p.velocityZ -= gravity * (1.0 / 60.0)
		p.posZ += p.velocityZ * (1.0 / 60.0)

		if p.posZ <= 0 {
			p.posZ = 0
			p.velocityZ = 0
			p.Stand()
		} else {
			p.cameraZ = 0.5 + p.posZ
		}
		p.moved = true
	}
}

func (p *Player) Stand() {
	p.cameraZ = 0.5
	p.posZ = 0
	p.moved = true
}

func (p *Player) updatePitch(pModifier float64) {
	pSpeed := playerRotateSpeed * pModifier
	// current raycasting method can only allow up to 22.5 degrees down, 45 degrees up
	p.pitch = clamp(pSpeed+p.pitch, -math.Pi/8, math.Pi/4)
	p.moved = true
}

func (p *Player) crouch() {
	p.cameraZ = 0.3
	p.posZ = 0
	p.moved = true
}

func (p *Player) goProne() {
	p.cameraZ = 0.1
	p.posZ = 0
	p.moved = true
}

const (
	playerMoveSpeed   = 0.06
	playerStrafeSpeed = 0.05
	playerRotateSpeed = 0.005
)

// move player by move speed in the forward/backward direction
func (g *Game) move(moveModifier float64) {
	mSpeed := playerMoveSpeed * moveModifier
	moveLine := lineFromAngle(g.player.pos.X, g.player.pos.Y, g.player.angle, mSpeed)

	newPos, _, _ := g.getValidMove(g.player.Entity, moveLine.X2, moveLine.Y2, g.player.posZ, true)
	if !newPos.eq(g.player.getPos()) {
		g.player.pos = newPos
		g.player.moved = true
	}
}

// Move player by strafe speed in the left/right direction
func (g *Game) strafe(moveModifier float64) {
	mSpeed := playerStrafeSpeed * moveModifier
	strafeAngle := HalfPi
	if mSpeed < 0 {
		strafeAngle = -strafeAngle
	}
	strafeLine := lineFromAngle(g.player.pos.X, g.player.pos.Y, g.player.angle-strafeAngle, math.Abs(mSpeed))

	newPos, _, _ := g.getValidMove(g.player.Entity, strafeLine.X2, strafeLine.Y2, g.player.posZ, true)
	if !newPos.eq(g.player.getPos()) {
		g.player.pos = newPos
		g.player.moved = true
	}
}

// rotate player heading angle by rotation speed
func (p *Player) rotate(rModifier float64) {
	rSpeed := playerRotateSpeed * rModifier
	p.angle += rSpeed

	for p.angle > Pi {
		p.angle = p.angle - Pi2
	}
	for p.angle <= -Pi {
		p.angle = p.angle + Pi2
	}

	p.moved = true
}
