package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type TextureHandler interface {
	// TextureAt reutrns image used for rendered wall at the given x, y map coordinates and level number
	TextureAt(x, y, levelNum, side int) *ebiten.Image

	// FloorTextureAt returns image used for textured floor at the given x, y map coordinates
	FloorTextureAt(x, y int) *image.RGBA
}

type TextureHandlerInstance struct {
	mapObj         *MapInstance
	textures       []*ebiten.Image
	floorTex       *image.RGBA
	renderFloorTex bool
}

func NewTextureHandler(mapObj *MapInstance, textureCapacity int) *TextureHandlerInstance {
	t := &TextureHandlerInstance{
		mapObj:         mapObj,
		textures:       make([]*ebiten.Image, textureCapacity),
		renderFloorTex: true,
	}
	return t
}

func (t *TextureHandlerInstance) TextureAt(x, y, levelNum, side int) *ebiten.Image {
	texNum := -1

	mapLayer := t.mapObj.Level(levelNum)
	if mapLayer == nil {
		return nil
	}

	mapWidth := len(mapLayer)
	if mapWidth == 0 {
		return nil
	}
	mapHeight := len(mapLayer[0])
	if mapHeight == 0 {
		return nil
	}

	if x >= 0 && x < mapWidth && y >= 0 && y < mapHeight {
		texNum = mapLayer[x][y] - 1 // 1 subtracted from it so that texture 0 can be used
	}

	if side == 0 {
		//--some supid hacks to make the houses render correctly--//
		// this corrects textures on two sides of house since the textures are not symmetrical
		if texNum == 3 {
			texNum = 4
		} else if texNum == 4 {
			texNum = 3
		}

		if texNum == 1 {
			texNum = 4
		} else if texNum == 2 {
			texNum = 3
		}

		// make the ebitengine splash only show on one side
		if texNum == 5 {
			texNum = 0
		}
	}

	if texNum < 0 {
		return nil
	}
	return t.textures[texNum]
}

func (t *TextureHandlerInstance) FloorTextureAt(x, y int) *image.RGBA {
	// x/y could be used to render different floor texture at given coords,
	// but for this demo we will just be rendering the same texture everywhere.
	if t.renderFloorTex {
		return t.floorTex
	}
	return nil
}
