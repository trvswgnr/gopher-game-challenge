package main

import (
	"image/color"
)

type Entity struct {
	Position        *Vector2
	PositionZ       float64
	Scale           float64
	Anchor          SpriteAnchor
	Angle           float64
	Pitch           float64
	Velocity        float64
	CollisionRadius float64
	CollisionHeight float64
	MapColor        color.RGBA
	Parent          *Entity
}

func (e *Entity) Pos() *Vector2 {
	return e.Position
}

func (e *Entity) PosZ() float64 {
	return e.PositionZ
}
