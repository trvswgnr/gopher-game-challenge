package model

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
	jumpVelocity = 4.0
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

func (p *Player) ApplyGravity() {
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

func (p *Player) SetPitch(pSpeed float64) {
	// current raycasting method can only allow up to 22.5 degrees down, 45 degrees up
	p.Pitch = geom.Clamp(pSpeed+p.Pitch, -math.Pi/8, math.Pi/4)
	p.Moved = true
}

func (p *Player) Crouch() {
	p.CameraZ = 0.3
	p.PositionZ = 0
	p.Moved = true
}

func (p *Player) Prone() {
	p.CameraZ = 0.1
	p.PositionZ = 0
	p.Moved = true
}
