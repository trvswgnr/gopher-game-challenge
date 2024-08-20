// minimap.go
package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (g *Game) generateStaticMinimap() {
	g.minimap = ebiten.NewImage(g.level.width()*minimapScale, g.level.height()*minimapScale)
	for y := 0; y < g.level.height(); y++ {
		for x := 0; x < g.level.width(); x++ {
			switch g.level.getEntityAt(x, y) {
			case LevelEntity_Wall:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{50, 50, 50, 255}, false)
			case LevelEntity_Construct:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{140, 140, 140, 255}, false)
			default:
				vector.DrawFilledRect(g.minimap, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{140, 140, 140, 255}, false)
			}
		}
	}
}

func (g *Game) drawDynamicMinimap(screen *ebiten.Image) {
	minimapImage := ebiten.NewImage(g.level.width()*minimapScale, g.level.height()*minimapScale)

	for y := 0; y < g.level.height(); y++ {
		for x := 0; x < g.level.width(); x++ {
			visibility := g.discoveredAreas[y][x]
			if visibility > 0 {
				var tileColor color.RGBA
				switch g.level.getEntityAt(x, y) {
				case LevelEntity_Wall:
					tileColor = color.RGBA{50, 50, 50, 255}
				case LevelEntity_Construct:
					tileColor = color.RGBA{140, 140, 140, 255}
				default:
					tileColor = color.RGBA{200, 200, 200, 255}
				}

				// apply fog effect
				tileColor.R = uint8(float64(tileColor.R) * visibility)
				tileColor.G = uint8(float64(tileColor.G) * visibility)
				tileColor.B = uint8(float64(tileColor.B) * visibility)

				vector.DrawFilledRect(minimapImage, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), tileColor, false)
			} else {
				vector.DrawFilledRect(minimapImage, float32(x*minimapScale), float32(y*minimapScale), float32(minimapScale), float32(minimapScale), color.RGBA{20, 20, 20, 255}, false)
			}
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(screenWidth-g.level.width()*minimapScale-10), 10)
	screen.DrawImage(minimapImage, op)

	g.drawMinimapPlayer(screen)
	g.drawMinimapEnemies(screen)
}

func (g *Game) drawMinimapPlayer(screen *ebiten.Image) {
	// calculate player position on minimap
	playerX := float32(screenWidth - g.level.width()*minimapScale - 10 + int(g.player.x*float64(minimapScale)))
	playerY := float32(10 + int(g.player.y*float64(minimapScale)))

	// calculate triangle points
	triangleSize := float32(minimapScale)
	angle := math.Atan2(g.player.dirY, g.player.dirX)

	x1 := playerX + triangleSize*float32(math.Cos(angle))
	y1 := playerY + triangleSize*float32(math.Sin(angle))

	x2 := playerX + triangleSize*float32(math.Cos(angle+2.5))
	y2 := playerY + triangleSize*float32(math.Sin(angle+2.5))

	x3 := playerX + triangleSize*float32(math.Cos(angle-2.5))
	y3 := playerY + triangleSize*float32(math.Sin(angle-2.5))

	// define triangle vertices
	vertices := []ebiten.Vertex{
		{DstX: x1, DstY: y1, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: x2, DstY: y2, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: x3, DstY: y3, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}

	// define triangle indices
	indices := []uint16{0, 1, 2}

	// choose color based on crouching state
	var playerColor color.RGBA
	if g.player.isCrouching {
		playerColor = color.RGBA{0, 255, 0, 255} // green when crouching
	} else {
		playerColor = color.RGBA{0, 255, 255, 255} // teal when standing
	}

	// create a 1x1 image with the player color
	playerColorImage := ebiten.NewImage(1, 1)
	playerColorImage.Fill(playerColor)

	// draw the triangle
	screen.DrawTriangles(vertices, indices, playerColorImage, nil)
}

func (g *Game) drawMinimapEnemies(screen *ebiten.Image) {
	for _, enemy := range g.enemies {
		enemyX, enemyY := int(enemy.x), int(enemy.y)

		if g.discoveredAreas[enemyY][enemyX] > 0 {
			screenX := float32(screenWidth - g.level.width()*minimapScale - 10 + int(enemy.x*float64(minimapScale)))
			screenY := float32(10 + int(enemy.y*float64(minimapScale)))

			// draw enemy (red)
			vector.DrawFilledCircle(screen, screenX, screenY, float32(minimapScale)/2, color.RGBA{255, 0, 0, 255}, false)

			// draw field of vision
			centerAngle := math.Atan2(enemy.dirY, enemy.dirX)
			leftAngle := centerAngle - enemy.fovAngle/2
			rightAngle := centerAngle + enemy.fovAngle/2

			// create vertices for the fov arc
			const segments = 20
			vertices := make([]ebiten.Vertex, segments+2)
			indices := make([]uint16, (segments+1)*3)

			// center vertex
			vertices[0] = ebiten.Vertex{
				DstX:   screenX,
				DstY:   screenY,
				SrcX:   0,
				SrcY:   0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 0,
				ColorA: 0.25,
			}

			// arc vertices
			for i := 0; i <= segments; i++ {
				angle := leftAngle + (rightAngle-leftAngle)*float64(i)/float64(segments)
				x := screenX + float32(math.Cos(angle)*enemy.fovDistance*float64(minimapScale))
				y := screenY + float32(math.Sin(angle)*enemy.fovDistance*float64(minimapScale))
				vertices[i+1] = ebiten.Vertex{
					DstX:   x,
					DstY:   y,
					SrcX:   0,
					SrcY:   0,
					ColorR: 1,
					ColorG: 1,
					ColorB: 0,
					ColorA: 0.25,
				}

				if i < segments {
					indices[i*3] = 0
					indices[i*3+1] = uint16(i + 1)
					indices[i*3+2] = uint16(i + 2)
				}
			}

			// draw the filled fov arc
			screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
				CompositeMode: ebiten.CompositeModeLighter,
			})

			// draw the fov outline
			for i := 0; i <= segments; i++ {
				if i < segments {
					vector.StrokeLine(screen, vertices[i+1].DstX, vertices[i+1].DstY, vertices[i+2].DstX, vertices[i+2].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
				}
			}
			vector.StrokeLine(screen, screenX, screenY, vertices[1].DstX, vertices[1].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
			vector.StrokeLine(screen, screenX, screenY, vertices[segments+1].DstX, vertices[segments+1].DstY, 1, color.RGBA{255, 255, 0, 128}, false)
		}
	}
}

var emptySubImage = ebiten.NewImage(3, 3).SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
