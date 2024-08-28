package main

import (
	"image"
	"image/color"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

// -- rendering

func drawTexture(screen *ebiten.Image, texture *ebiten.Image, destinationRectangle *image.Rectangle, sourceRectangle *image.Rectangle, color *color.RGBA) {
	if texture == nil || destinationRectangle == nil || sourceRectangle == nil {
		return
	}

	// if destinationRectangle is not the same size as sourceRectangle, scale to fit
	var scaleX, scaleY float64 = 1.0, 1.0
	if !destinationRectangle.Eq(*sourceRectangle) {
		sSize := sourceRectangle.Size()
		dSize := destinationRectangle.Size()

		scaleX = float64(dSize.X) / float64(sSize.X)
		scaleY = float64(dSize.Y) / float64(sSize.Y)
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest

	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(float64(destinationRectangle.Min.X), float64(destinationRectangle.Min.Y))

	destTexture := texture.SubImage(*sourceRectangle).(*ebiten.Image)

	if color != nil {
		// color channel modulation/tinting
		op.ColorScale.Scale(float32(color.R)/255, float32(color.G)/255, float32(color.B)/255, float32(color.A)/255)
	}

	screen.DrawImage(destTexture, op)
}

// -- minimap

func (g *Game) miniMap() *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, g.mapWidth, g.mapHeight))

	// wall/world positions
	worldMap := g.mapObj.Level(0)
	for x, row := range worldMap {
		for y := range row {
			c := g.getMapColor(x, y)
			if c.A == 255 {
				c.A = 142
			}
			m.Set(x, y, c)
		}
	}

	// sprite positions, sort by color to avoid random color getting chosen as last when using map keys
	sprites := make([]*Entity, 0, len(g.sprites))
	for s := range g.sprites {
		sprites = append(sprites, s.Entity)
	}
	sort.Slice(sprites, func(i, j int) bool {
		iComp := (sprites[i].mapColor.R + sprites[i].mapColor.G + sprites[i].mapColor.B)
		jComp := (sprites[j].mapColor.R + sprites[j].mapColor.G + sprites[j].mapColor.B)
		return iComp < jComp
	})

	for _, sprite := range sprites {
		if sprite.mapColor.A > 0 {

			m.Set(int(sprite.pos.X), int(sprite.pos.Y), sprite.mapColor)
		}
	}

	// projectile positions
	projectiles := make([]*Entity, 0, len(g.projectiles))
	for p := range g.projectiles {
		projectiles = append(projectiles, p.Entity)
	}
	sort.Slice(projectiles, func(i, j int) bool {
		iComp := (projectiles[i].mapColor.R + projectiles[i].mapColor.G + projectiles[i].mapColor.B)
		jComp := (projectiles[j].mapColor.R + projectiles[j].mapColor.G + projectiles[j].mapColor.B)
		return iComp < jComp
	})

	for _, projectile := range projectiles {
		if projectile.mapColor.A > 0 {

			m.Set(int(projectile.pos.X), int(projectile.pos.Y), projectile.mapColor)
		}
	}

	// player position
	m.Set(int(g.player.pos.X), int(g.player.pos.Y), g.player.mapColor)

	return m
}

func (g *Game) getMapColor(x, y int) color.RGBA {
	worldMap := g.mapObj.Level(0)
	switch worldMap[x][y] {
	case 0:
		return color.RGBA{43, 30, 24, 255}
	case 1:
		return color.RGBA{100, 89, 73, 255}
	case 2:
		return color.RGBA{51, 32, 0, 196}
	case 3:
		return color.RGBA{56, 36, 0, 196}
	case 6:
		// ebitengine splash logo color!
		return color.RGBA{219, 86, 32, 255}
	default:
		return color.RGBA{255, 194, 32, 255}
	}
}
