package main

import (
	"image/color"
	"math"

	"github.com/harbdog/raycaster-go/geom"
)

type Player struct {
	*Entity
	CameraZ      float64
	Moved        bool
	Weapon       *Weapon
	WeaponSet    []*Weapon
	LastWeapon   *Weapon
	VelocityZ    float64
	JumpHoldTime float64
	IsJumping    bool
}

func NewPlayer(x, y, angle, pitch float64) *Player {
	p := &Player{
		Entity: &Entity{
			Position:  &geom.Vector2{X: x, Y: y},
			PositionZ: 0,
			Angle:     angle,
			Pitch:     pitch,
			Velocity:  0,
			MapColor:  color.RGBA{255, 0, 0, 255},
		},
		CameraZ:      0.5,
		Moved:        false,
		WeaponSet:    []*Weapon{},
		JumpHoldTime: 0,
		IsJumping:    false,
	}

	return p
}

func (p *Player) AddWeapon(w *Weapon) {
	p.WeaponSet = append(p.WeaponSet, w)
}

func (p *Player) SelectWeapon(weaponIndex int) *Weapon {
	// TODO: add some kind of sheath/unsheath animation
	if weaponIndex < 0 {
		// put away weapon
		if p.Weapon != nil {
			// store as last weapon
			p.LastWeapon = p.Weapon
		}
		p.Weapon = nil
		return nil
	}
	newWeapon := p.Weapon
	if weaponIndex < len(p.WeaponSet) {
		newWeapon = p.WeaponSet[weaponIndex]
	}
	if newWeapon != p.Weapon {
		// store as last weapon
		p.LastWeapon = p.Weapon
		p.Weapon = newWeapon
	}
	return p.Weapon
}

func (p *Player) NextWeapon(reverse bool) *Weapon {
	_, weaponIndex := p.getSelectedWeapon()
	if weaponIndex < 0 {
		// check last weapon in event of unsheathing previously sheathed weapon
		weaponIndex = p.getWeaponIndex(p.LastWeapon)
		if weaponIndex < 0 {
			weaponIndex = 0
		}
		return p.SelectWeapon(weaponIndex)
	}

	weaponIndex++
	if weaponIndex >= len(p.WeaponSet) {
		weaponIndex = 0
	}
	return p.SelectWeapon(weaponIndex)
}

func (p *Player) getWeaponIndex(w *Weapon) int {
	if w == nil {
		return -1
	}
	for index, wCheck := range p.WeaponSet {
		if wCheck == w {
			return index
		}
	}
	return -1
}

func (p *Player) getSelectedWeapon() (*Weapon, int) {
	if p.Weapon == nil {
		return nil, -1
	}

	return p.Weapon, p.getWeaponIndex(p.Weapon)
}

func (p *Player) IsStanding() bool {
	return p.PositionZ == 0 && p.CameraZ == 0.5
}

const (
	jumpVelocity = 3.2
	gravity      = 9.8
)

func (p *Player) Jump() {
	if p.IsStanding() {
		p.VelocityZ = jumpVelocity
		p.Moved = true
	}

	p.VelocityZ -= gravity * (1.0 / 60.0) // Assuming 60 FPS
	p.PositionZ += p.VelocityZ * (1.0 / 60.0)

	if p.PositionZ <= 0 {
		p.PositionZ = 0
		p.VelocityZ = 0
		p.Stand()
	} else {
		p.CameraZ = 0.5 + p.PositionZ
	}
	p.Moved = true
}

func (p *Player) applyGravity() {
	if !p.IsStanding() {
		p.VelocityZ -= gravity * (1.0 / 60.0)
		p.PositionZ += p.VelocityZ * (1.0 / 60.0)

		if p.PositionZ <= 0 {
			p.PositionZ = 0
			p.VelocityZ = 0
			p.Stand()
		} else {
			p.CameraZ = 0.5 + p.PositionZ
		}
		p.Moved = true
	}
}

func (p *Player) Stand() {
	p.CameraZ = 0.5
	p.PositionZ = 0
	p.Moved = true
}

func (p *Player) updatePitch(pSpeed float64) {
	// current raycasting method can only allow up to 22.5 degrees down, 45 degrees up
	p.Pitch = geom.Clamp(pSpeed+p.Pitch, -math.Pi/8, math.Pi/4)
	p.Moved = true
}

func (p *Player) crouch() {
	p.CameraZ = 0.3
	p.PositionZ = 0
	p.Moved = true
}

func (p *Player) goProne() {
	p.CameraZ = 0.1
	p.PositionZ = 0
	p.Moved = true
}

// move player by move speed in the forward/backward direction
func (g *Game) move(mSpeed float64) {
	moveLine := geom.LineFromAngle(g.player.Position.X, g.player.Position.Y, g.player.Angle, mSpeed)

	newPos, _, _ := g.getValidMove(g.player.Entity, moveLine.X2, moveLine.Y2, g.player.PositionZ, true)
	if !newPos.Equals(g.player.Pos()) {
		g.player.Position = newPos
		g.player.Moved = true
	}
}

// Move player by strafe speed in the left/right direction
func (g *Game) strafe(sSpeed float64) {
	strafeAngle := geom.HalfPi
	if sSpeed < 0 {
		strafeAngle = -strafeAngle
	}
	strafeLine := geom.LineFromAngle(g.player.Position.X, g.player.Position.Y, g.player.Angle-strafeAngle, math.Abs(sSpeed))

	newPos, _, _ := g.getValidMove(g.player.Entity, strafeLine.X2, strafeLine.Y2, g.player.PositionZ, true)
	if !newPos.Equals(g.player.Pos()) {
		g.player.Position = newPos
		g.player.Moved = true
	}
}

// rotate player heading angle by rotation speed
func (p *Player) rotate(rSpeed float64) {
	p.Angle += rSpeed

	for p.Angle > geom.Pi {
		p.Angle = p.Angle - geom.Pi2
	}
	for p.Angle <= -geom.Pi {
		p.Angle = p.Angle + geom.Pi2
	}

	p.Moved = true
}
