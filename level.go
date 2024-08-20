// level.go
package main

import (
	"image"
	"image/color"
	"io/fs"
	"log"
)

type LevelEntity int

const (
	LevelEntity_Empty LevelEntity = iota
	LevelEntity_Wall
	LevelEntity_Enemy
	LevelEntity_Exit
	LevelEntity_Player
	LevelEntity_Construct
)

type LevelEntityColor = color.RGBA

var (
	LevelEntityColor_Empty     = color.RGBA{255, 255, 255, 255}
	LevelEntityColor_Wall      = color.RGBA{0, 0, 0, 255}
	LevelEntityColor_Enemy     = color.RGBA{255, 0, 0, 255}
	LevelEntityColor_Exit      = color.RGBA{0, 255, 0, 255}
	LevelEntityColor_Player    = color.RGBA{0, 0, 255, 255}
	LevelEntityColor_Construct = color.RGBA{255, 255, 0, 255}
)

type Level [][]LevelEntity

func NewLevel(file fs.File) Level {
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	matrix := make(Level, height)
	for i := range matrix {
		matrix[i] = make([]LevelEntity, width)
	}

	// fill matrix based on pixel colors
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := img.At(x, y)

			switch {
			case c == LevelEntityColor_Empty:
				matrix[y][x] = LevelEntity_Empty
			case c == LevelEntityColor_Wall:
				matrix[y][x] = LevelEntity_Wall
			case c == LevelEntityColor_Enemy:
				matrix[y][x] = LevelEntity_Enemy
			case c == LevelEntityColor_Exit:
				matrix[y][x] = LevelEntity_Exit
			case c == LevelEntityColor_Player:
				matrix[y][x] = LevelEntity_Player
			case c == LevelEntityColor_Construct:
				matrix[y][x] = LevelEntity_Construct
			}
		}
	}

	return matrix
}

func (level Level) getPlayer() (float64, float64) {
	playerX := 0
	playerY := 0
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Player {
				playerX = x
				playerY = y
				// remove player block from level so it doesn't render or collide
				level[y][x] = LevelEntity_Empty
				break
			}
		}
	}

	return float64(playerX), float64(playerY)
}

func (level Level) getEnemies() []Enemy {
	enemies := []Enemy{}
	for y := 0; y < len(level); y++ {
		for x := 0; x < len(level[y]); x++ {
			if level[y][x] == LevelEntity_Enemy {
				enemies = append(enemies, Enemy{x: float64(x), y: float64(y)})
				// remove enemy block from level so it doesn't render or collide
				level[y][x] = LevelEntity_Empty
			}
		}
	}
	return enemies
}

func (l Level) width() int                       { return len(l[0]) }
func (l Level) height() int                      { return len(l) }
func (l Level) getEntityAt(x, y int) LevelEntity { return l[y][x] }
