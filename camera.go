package main

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// -- camera

const (
	maxConcurrent = 100
)

type Camera struct {
	pos                       *Vec2
	camZ                      float64
	posZ                      float64
	dir                       *Vec2
	headingAngle              float64
	plane                     *Vec2
	w                         int
	h                         int
	pitch                     int
	pitchAngle                float64
	fovAngle, fovDepth        float64
	mapObj                    *Map
	mapWidth                  int
	mapHeight                 int
	floor                     *ebiten.Image
	sky                       *ebiten.Image
	texSize                   int
	levels                    []*Level
	floorLvl                  *HorizontalLevel
	slices                    []*image.Rectangle
	zBuffer                   []float64
	sprites                   []*Sprite
	spriteLvls                []*Level
	spriteOrder               []int
	spriteDistance            []float64
	tex                       *TextureManager
	lightFalloff              float64
	globalIllumination        float64
	minLightRGB               color.NRGBA
	maxLightRGB               color.NRGBA
	renderDistance            float64
	convergenceDistance       float64
	convergencePoint          *Vec3
	convergenceSprite         *Sprite
	alwaysSetSpriteScreenRect bool
	semaphore                 chan struct{}
}

func NewCamera(width int, height int, texSize int, mapObj *Map, tex *TextureManager) *Camera {
	c := &Camera{}
	c.mapObj = mapObj
	firstLevel := mapObj.Level(0)
	c.mapWidth = len(firstLevel)
	c.mapHeight = len(firstLevel[0])
	c.pos = &Vec2{X: 1.0, Y: 1.0}
	c.camZ = 0.0
	c.SetHeadingAngle(0)
	c.SetPitchAngle(0)
	fovDegrees := 70.0
	fovDepth := 1.0
	c.SetFovAngle(fovDegrees, fovDepth)
	c.SetRenderDistance(-1)
	c.SetLightFalloff(-100)
	c.SetGlobalIllumination(300)
	c.SetLightRGB(color.NRGBA{R: 0, G: 0, B: 0}, color.NRGBA{R: 255, G: 255, B: 255})
	c.texSize = texSize
	c.tex = tex
	c.SetViewSize(width, height)
	c.sprites = []*Sprite{}
	c.updateSpriteLevels(16)
	c.semaphore = make(chan struct{}, maxConcurrent)
	c.convergenceDistance = -1
	c.convergencePoint = nil
	c.convergenceSprite = nil
	c.raycast()
	return c
}

// Draw the raycasted camera view to the screen.
func (c *Camera) Draw(screen *ebiten.Image) {
	//--draw basic sky and floor--//
	texRect := image.Rect(0, 0, c.texSize, c.texSize)
	lightingRGBA := &color.RGBA{R: c.maxLightRGB.R, G: c.maxLightRGB.G, B: c.maxLightRGB.B, A: 255}

	floorRect := image.Rect(0, int(float64(c.h)*0.5)+c.pitch,
		c.w, c.h)
	drawTexture(screen, c.floor, &floorRect, &texRect, lightingRGBA)

	skyRect := image.Rect(0, 0, c.w, int(float64(c.h)*0.5)+c.pitch)
	drawTexture(screen, c.sky, &skyRect, &texRect, lightingRGBA)

	//--draw walls--//
	for x := 0; x < c.w; x++ {
		for i := cap(c.levels) - 1; i >= 0; i-- {
			drawTexture(screen, c.levels[i].CurrTex[x], c.levels[i].Sv[x], c.levels[i].Cts[x], c.levels[i].St[x])
		}
	}

	// draw textured floor
	if c.floorLvl != nil && c.floorLvl.image != nil {
		c.floorLvl.image.WritePixels(c.floorLvl.horBuffer.Pix)

		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(c.floorLvl.image, op)
	}

	// draw sprites
	for x := 0; x < c.w; x++ {
		for i := 0; i < cap(c.spriteLvls); i++ {
			spriteLvl := c.spriteLvls[i]
			if spriteLvl == nil {
				continue
			}

			texture := spriteLvl.CurrTex[x]
			if texture != nil {
				drawTexture(screen, texture, spriteLvl.Sv[x], spriteLvl.Cts[x], spriteLvl.St[x])
			}
		}
	}
}

func (c *Camera) SetViewSize(width, height int) {
	c.w = width
	c.h = height
	c.levels = c.createLevels(c.mapObj.NumLevels())
	c.slices = makeSlices(c.texSize, c.texSize, 0, 0)
	c.floorLvl = c.createFloorLevel()
	c.zBuffer = make([]float64, width)
}
func (c *Camera) ViewSize() (int, int) {
	return c.w, c.h
}
func (c *Camera) SetFovAngle(fovDegrees, fovDepth float64) {
	c.fovAngle = radians(fovDegrees)
	c.fovDepth = fovDepth
	var headingAngle float64 = 0
	if c.dir != nil {
		headingAngle = c.getAngleFromVec(c.dir)
	}
	c.dir = c.getVecForAngle(headingAngle)
	c.plane = c.getVecForFov(c.dir)
}
func (c *Camera) FovRadians() float64 {
	return c.fovAngle
}
func (c *Camera) FovAngle() float64 {
	return degrees(c.fovAngle)
}
func (c *Camera) FovRadiansVertical() float64 {
	return 2 * math.Atan(math.Tan(c.fovAngle/2)*(float64(c.h)/float64(c.w)))
}
func (c *Camera) FovAngleVertical() float64 {
	return degrees(c.FovRadiansVertical())
}
func (c *Camera) FovDepth() float64 {
	return c.fovDepth
}
func (c *Camera) SetFloorTexture(floor *ebiten.Image) {
	c.floor = floor
}
func (c *Camera) SetSkyTexture(sky *ebiten.Image) {
	c.sky = sky
}
func (c *Camera) SetRenderDistance(distance float64) {
	if distance < 0 {
		c.renderDistance = math.MaxFloat64
	} else {
		c.renderDistance = distance
	}
}
func (c *Camera) SetLightFalloff(falloff float64) {
	c.lightFalloff = falloff
}
func (c *Camera) SetGlobalIllumination(illumination float64) {
	c.globalIllumination = illumination
}
func (c *Camera) SetLightRGB(min, max color.NRGBA) {
	c.minLightRGB = min
	c.maxLightRGB = max
}
func (c *Camera) SetAlwaysSetSpriteScreenRect(b bool) {
	c.alwaysSetSpriteScreenRect = b
}
func (c *Camera) Update(sprites []*Sprite) {
	c.floorLvl.initialize(c.w, c.h)
	c.convergenceDistance = -1
	c.convergencePoint = nil
	c.convergenceSprite = nil
	if len(sprites) != len(c.sprites) {
		c.updateSpriteLevels(len(sprites))
	} else {
		c.clearAllSpriteLevels()
	}
	c.sprites = sprites
	c.raycast()
}
func (c *Camera) raycast() {
	numLevels := c.mapObj.NumLevels()
	var wg sync.WaitGroup
	for i := 0; i < numLevels; i++ {
		wg.Add(1)
		go c.asyncCastLevel(i, &wg)
	}
	wg.Wait()
	numSprites := len(c.sprites)
	c.spriteOrder = make([]int, numSprites)
	c.spriteDistance = make([]float64, numSprites)
	for i := 0; i < numSprites; i++ {
		sprite := c.sprites[i]
		c.spriteOrder[i] = i
		c.spriteDistance[i] = math.Sqrt(math.Pow(c.pos.X-sprite.Pos().X, 2) + math.Pow(c.pos.Y-sprite.Pos().Y, 2))
	}
	combSort(c.spriteOrder, c.spriteDistance, numSprites)
	for i := 0; i < numSprites; i++ {
		wg.Add(1)
		go c.asyncCastSprite(i, &wg)
	}
	wg.Wait()
}
func (c *Camera) asyncCastLevel(levelNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	rMap := c.mapObj.Level(levelNum)
	for x := 0; x < c.w; x++ {
		c.castLevel(x, rMap, c.levels[levelNum], levelNum, wg)
	}
}
func (c *Camera) asyncCastSprite(spriteNum int, wg *sync.WaitGroup) {
	defer wg.Done()
	c.semaphore <- struct{}{}
	defer func() {
		<-c.semaphore
	}()
	c.castSprite(spriteNum)
}
func (c *Camera) castLevel(x int, grid [][]int, lvl *Level, levelNum int, wg *sync.WaitGroup) {
	var _cts, _sv []*image.Rectangle
	var _st []*color.RGBA
	_cts = lvl.Cts
	_sv = lvl.Sv
	_st = lvl.St
	cameraX := 2.0*float64(x)/float64(c.w) - 1.0
	rayDirX := c.dir.X + c.plane.X*cameraX
	rayDirY := c.dir.Y + c.plane.Y*cameraX
	rayPosX := c.pos.X
	rayPosY := c.pos.Y
	mapX := int(rayPosX)
	mapY := int(rayPosY)
	var sideDistX float64
	var sideDistY float64
	deltaDistX := math.Abs(1 / rayDirX)
	deltaDistY := math.Abs(1 / rayDirY)
	var perpWallDist float64
	var stepX int
	var stepY int
	hit := 0
	side := -1
	if rayDirX < 0 {
		stepX = -1
		sideDistX = (rayPosX - float64(mapX)) * deltaDistX
	} else {
		stepX = 1
		sideDistX = (float64(mapX) + 1.0 - rayPosX) * deltaDistX
	}
	if rayDirY < 0 {
		stepY = -1
		sideDistY = (rayPosY - float64(mapY)) * deltaDistY
	} else {
		stepY = 1
		sideDistY = (float64(mapY) + 1.0 - rayPosY) * deltaDistY
	}
	for hit == 0 {
		if sideDistX < sideDistY {
			sideDistX += deltaDistX
			mapX += stepX
			side = 0
		} else {
			sideDistY += deltaDistY
			mapY += stepY
			side = 1
		}
		if side == 0 {
			perpWallDist = sideDistX - deltaDistX
		} else {
			perpWallDist = sideDistY - deltaDistY
		}
		if mapX >= 0 && mapY >= 0 && mapX < c.mapWidth && mapY < c.mapHeight {
			if perpWallDist > c.renderDistance {
				hit = 2
			} else if perpWallDist <= c.renderDistance && grid[mapX][mapY] > 0 {
				hit = 1
			}
		} else {
			hit = 2
		}
	}
	lineHeight := int(float64(c.h) / perpWallDist)
	drawStart := (-lineHeight/2 + c.h/2) + c.pitch + int(c.camZ/perpWallDist) - lineHeight*levelNum
	drawEnd := drawStart + lineHeight
	var wallX float64
	if side == 0 {
		wallX = rayPosY + perpWallDist*rayDirY
	} else {
		wallX = rayPosX + perpWallDist*rayDirX
	}
	wallX -= math.Floor(wallX)
	var texture *ebiten.Image
	if hit == 1 && mapX >= 0 && mapY >= 0 && mapX < c.mapWidth && mapY < c.mapHeight {
		texture = c.tex.TextureAt(mapX, mapY, levelNum, side)
	}
	c.levels[levelNum].CurrTex[x] = texture
	if texture != nil {
		texX := int(wallX * float64(c.texSize))
		if side == 0 && rayDirX > 0 {
			texX = c.texSize - texX - 1
		}
		if side == 1 && rayDirY < 0 {
			texX = c.texSize - texX - 1
		}
		_cts[x] = c.slices[texX]
		_sv[x].Min.Y = drawStart
		_sv[x].Max.Y = drawEnd
		shadowDepth := math.Sqrt(perpWallDist) * c.lightFalloff
		_st[x] = &color.RGBA{255, 255, 255, 255}
		_st[x].R = byte(clampInt(int(float64(_st[x].R)+shadowDepth+c.globalIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
		_st[x].G = byte(clampInt(int(float64(_st[x].G)+shadowDepth+c.globalIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
		_st[x].B = byte(clampInt(int(float64(_st[x].B)+shadowDepth+c.globalIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
		if side == 0 {
			wallDiff := 12
			_st[x].R = byte(clampInt(int(_st[x].R)-wallDiff, 0, 255))
			_st[x].G = byte(clampInt(int(_st[x].G)-wallDiff, 0, 255))
			_st[x].B = byte(clampInt(int(_st[x].B)-wallDiff, 0, 255))
		}
	}
	convergenceCol, convergenceRow := c.w/2-1, c.h/2-1
	if x == convergenceCol && drawStart <= convergenceRow && convergenceRow <= drawEnd {
		convergencePerpDist := perpWallDist * c.fovDepth
		convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
		convergenceDistance := convergenceLine3d.dist()
		if c.convergenceDistance == -1 || convergenceDistance < c.convergenceDistance {
			c.convergenceDistance = convergenceDistance
			c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
		}
	}
	if levelNum == 0 {
		c.zBuffer[x] = perpWallDist
	}
	if levelNum == 0 {
		if drawEnd < 0 {
			drawEnd = c.h
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			var floorXWall, floorYWall float64
			if side == 0 && rayDirX > 0 {
				floorXWall = float64(mapX)
				floorYWall = float64(mapY) + wallX
			} else if side == 0 && rayDirX < 0 {
				floorXWall = float64(mapX) + 1.0
				floorYWall = float64(mapY) + wallX
			} else if side == 1 && rayDirY > 0 {
				floorXWall = float64(mapX) + wallX
				floorYWall = float64(mapY)
			} else {
				floorXWall = float64(mapX) + wallX
				floorYWall = float64(mapY) + 1.0
			}
			var distWall, distPlayer, currentDist float64
			distWall = perpWallDist
			distPlayer = 0.0
			for y := drawEnd; y < c.h; y++ {
				currentDist = (float64(c.h) + (2.0 * c.camZ)) / (2.0*float64(y-c.pitch) - float64(c.h))
				if currentDist > c.renderDistance {
					continue
				}
				weight := (currentDist - distPlayer) / (distWall - distPlayer)
				currentFloorX := weight*floorXWall + (1.0-weight)*rayPosX
				currentFloorY := weight*floorYWall + (1.0-weight)*rayPosY
				if currentFloorX < 0 || currentFloorY < 0 || int(currentFloorX) >= c.mapWidth || int(currentFloorY) >= c.mapHeight {
					continue
				}
				if x == convergenceCol && y == convergenceRow {
					convergencePerpDist := currentDist * c.fovDepth
					convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
					convergenceDistance := convergenceLine3d.dist()
					if c.convergenceDistance == 0 || convergenceDistance < c.convergenceDistance {
						c.convergenceDistance = convergenceDistance
						c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
					}
				}
				floorTex := c.tex.FloorTextureAt(int(currentFloorX), int(currentFloorY))
				if floorTex == nil {
					continue
				}
				floorTexX := int(currentFloorX*float64(c.texSize)) % c.texSize
				floorTexY := int(currentFloorY*float64(c.texSize)) % c.texSize
				pxOffset := floorTex.PixOffset(floorTexX, floorTexY)
				if pxOffset < 0 {
					continue
				}
				pixel := color.RGBA{floorTex.Pix[pxOffset],
					floorTex.Pix[pxOffset+1],
					floorTex.Pix[pxOffset+2],
					floorTex.Pix[pxOffset+3]}
				pixelSt := &color.RGBA{R: 255, G: 255, B: 255}
				shadowDepth := math.Sqrt(currentDist) * c.lightFalloff
				pixelSt.R = byte(clampInt(int(float64(pixelSt.R)+shadowDepth+c.globalIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
				pixelSt.G = byte(clampInt(int(float64(pixelSt.G)+shadowDepth+c.globalIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
				pixelSt.B = byte(clampInt(int(float64(pixelSt.B)+shadowDepth+c.globalIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
				pixel.R = uint8(float64(pixel.R) * float64(pixelSt.R) / 256)
				pixel.G = uint8(float64(pixel.G) * float64(pixelSt.G) / 256)
				pixel.B = uint8(float64(pixel.B) * float64(pixelSt.B) / 256)
				pxOffset = c.floorLvl.horBuffer.PixOffset(x, y)
				c.floorLvl.horBuffer.Pix[pxOffset] = pixel.R
				c.floorLvl.horBuffer.Pix[pxOffset+1] = pixel.G
				c.floorLvl.horBuffer.Pix[pxOffset+2] = pixel.B
				c.floorLvl.horBuffer.Pix[pxOffset+3] = pixel.A
			}
		}()
	}
}
func (c *Camera) castSprite(spriteOrdIndex int) {
	sprite := c.sprites[c.spriteOrder[spriteOrdIndex]]
	spriteDist := c.spriteDistance[spriteOrdIndex]
	if spriteDist > c.renderDistance && !c.alwaysSetSpriteScreenRect {
		sprite.SetScreenRect(nil)
		return
	}
	renderSprite := false
	spriteX := sprite.Pos().X - c.pos.X
	spriteY := sprite.Pos().Y - c.pos.Y
	spriteTex := sprite.Texture()
	spriteTexRect := sprite.TextureRect()
	spriteTexWidth, spriteTexHeight := spriteTex.Bounds().Dx(), spriteTex.Bounds().Dy()
	spriteTexRatioWH := float64(spriteTexWidth) / float64(spriteTexHeight)
	spriteIllumination := sprite.Illumination()
	invDet := 1.0 / (c.plane.X*c.dir.Y - c.dir.X*c.plane.Y)
	transformX := invDet * (c.dir.Y*spriteX - c.dir.X*spriteY)
	transformY := invDet * (-c.plane.Y*spriteX + c.plane.X*spriteY)
	spriteScreenX := int(float64(c.w) / 2 * (1 + transformX/transformY))
	spriteScale := sprite.Scale()
	spriteAnchor := sprite.VerticalAnchor()
	var uDiv float64 = 1 / (spriteScale * spriteTexRatioWH)
	var vDiv float64 = 1 / spriteScale
	var vOffset float64 = getAnchorVerticalOffset(spriteAnchor, spriteScale, c.h)
	var vMove float64 = -sprite.PosZ()*float64(c.h) + vOffset
	vMoveScreen := int(vMove/transformY) + c.pitch + int(c.camZ/transformY)
	spriteHeight := int(math.Abs(float64(c.h)/transformY) / vDiv)
	drawStartY := -spriteHeight/2 + c.h/2 + vMoveScreen
	if drawStartY < 0 {
		drawStartY = 0
	}
	drawEndY := spriteHeight/2 + c.h/2 + vMoveScreen
	if drawEndY >= c.h {
		drawEndY = c.h - 1
	}
	spriteWidth := int(math.Abs(float64(c.h)/transformY) / uDiv)
	drawStartX := -spriteWidth/2 + spriteScreenX
	drawEndX := spriteWidth/2 + spriteScreenX
	if spriteWidth == 0 || spriteHeight == 0 || transformY <= 0 || drawStartX < -spriteWidth || drawEndX >= c.w+spriteWidth {
		sprite.SetScreenRect(nil)
		return
	}
	if drawStartX < 0 {
		drawStartX = 0
	}
	if drawEndX >= c.w {
		drawEndX = c.w - 1
	}
	canConverge := sprite.IsFocusable()
	convergenceCol, convergenceRow := c.w/2-1, c.h/2-1
	d := (drawStartY-vMoveScreen)*256 - c.h*128 + spriteHeight*128
	texStartY := ((d * spriteTexHeight) / spriteHeight) / 256
	d = (drawEndY-1-vMoveScreen)*256 - c.h*128 + spriteHeight*128
	texEndY := ((d * spriteTexHeight) / spriteHeight) / 256
	var spriteSlices []*image.Rectangle
	if !c.alwaysSetSpriteScreenRect || spriteDist <= c.renderDistance {
		for stripe := drawStartX; stripe < drawEndX; stripe++ {
			if transformY > 0 && stripe > 0 && stripe < c.w && transformY < c.zBuffer[stripe] {
				var spriteLvl *Level
				if !renderSprite {
					renderSprite = true
					spriteLvl = c.makeSpriteLevel(spriteOrdIndex)
					spriteSlices = makeSlices(spriteTexWidth, spriteTexHeight, spriteTexRect.Min.X, spriteTexRect.Min.Y)
				} else {
					spriteLvl = c.spriteLvls[spriteOrdIndex]
				}
				texX := int(256*(stripe-(-spriteWidth/2+spriteScreenX))*spriteTexWidth/spriteWidth) / 256
				if texX < 0 || texX >= cap(spriteSlices) {
					continue
				}
				if canConverge && stripe == convergenceCol && drawStartY <= convergenceRow && convergenceRow <= drawEndY {
					convergencePerpDist := spriteDist
					convergenceLine3d := line3dFromBaseAngle(c.pos.X, c.pos.Y, c.posZ, c.headingAngle, c.pitchAngle, convergencePerpDist)
					convergenceDistance := convergenceLine3d.dist()
					if c.convergenceDistance == -1 || convergenceDistance < c.convergenceDistance {
						c.convergenceDistance = convergenceDistance
						c.convergencePoint = &Vec3{X: convergenceLine3d.X2, Y: convergenceLine3d.Y2, Z: convergenceLine3d.Z2}
						c.convergenceSprite = sprite
					}
				}
				spriteLvl.Cts[stripe] = spriteSlices[texX]
				spriteLvl.Cts[stripe].Min.Y = spriteTexRect.Min.Y + texStartY
				spriteLvl.Cts[stripe].Max.Y = spriteTexRect.Min.Y + texEndY + 1
				spriteLvl.CurrTex[stripe] = spriteTex
				spriteLvl.Sv[stripe].Min.Y = drawStartY
				spriteLvl.Sv[stripe].Max.Y = drawEndY
				shadowDepth := math.Sqrt(transformY) * c.lightFalloff
				spriteLvl.St[stripe] = &color.RGBA{255, 255, 255, 255}
				spriteLvl.St[stripe].R = byte(clampInt(int(float64(spriteLvl.St[stripe].R)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.R), int(c.maxLightRGB.R)))
				spriteLvl.St[stripe].G = byte(clampInt(int(float64(spriteLvl.St[stripe].G)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.G), int(c.maxLightRGB.G)))
				spriteLvl.St[stripe].B = byte(clampInt(int(float64(spriteLvl.St[stripe].B)+shadowDepth+c.globalIllumination+spriteIllumination), int(c.minLightRGB.B), int(c.maxLightRGB.B)))
			}
		}
	}
	if renderSprite || c.alwaysSetSpriteScreenRect {
		spriteCastRect := image.Rect(drawStartX, drawStartY, drawEndX, drawEndY)
		sprite.SetScreenRect(&spriteCastRect)
	} else {
		c.clearSpriteLevel(spriteOrdIndex)
		sprite.SetScreenRect(nil)
	}
}
func makeSlices(width, height, xOffset, yOffset int) []*image.Rectangle {
	newSlices := make([]*image.Rectangle, width)
	for x := 0; x < width; x++ {
		thisRect := image.Rect(xOffset+x, yOffset, xOffset+x+1, yOffset+height)
		newSlices[x] = &thisRect
	}
	return newSlices
}
func (c *Camera) createLevels(numLevels int) []*Level {
	levelArr := make([]*Level, numLevels)
	for i := 0; i < numLevels; i++ {
		levelArr[i] = new(Level)
		levelArr[i].Sv = sliceView(c.w, c.h)
		levelArr[i].Cts = make([]*image.Rectangle, c.w)
		levelArr[i].St = make([]*color.RGBA, c.w)
		levelArr[i].CurrTex = make([]*ebiten.Image, c.w)
	}
	return levelArr
}
func (c *Camera) createFloorLevel() *HorizontalLevel {
	horizontalLevel := new(HorizontalLevel)
	horizontalLevel.initialize(c.w, c.h)
	return horizontalLevel
}
func (c *Camera) updateSpriteLevels(spriteCapacity int) {
	if c.spriteLvls != nil {
		capacity := len(c.spriteLvls)
		if spriteCapacity <= capacity {
			c.clearAllSpriteLevels()
			return
		}
		for capacity <= spriteCapacity {
			capacity *= 2
		}
		spriteCapacity = capacity
	}
	c.spriteLvls = make([]*Level, spriteCapacity)
}
func (c *Camera) makeSpriteLevel(spriteOrdIndex int) *Level {
	spriteLvl := new(Level)
	spriteLvl.Sv = sliceView(c.w, c.h)
	spriteLvl.Cts = make([]*image.Rectangle, c.w)
	spriteLvl.St = make([]*color.RGBA, c.w)
	spriteLvl.CurrTex = make([]*ebiten.Image, c.w)
	c.spriteLvls[spriteOrdIndex] = spriteLvl
	return spriteLvl
}
func (c *Camera) clearAllSpriteLevels() {
	for i := 0; i < len(c.spriteLvls); i++ {
		c.clearSpriteLevel(i)
	}
}
func (c *Camera) clearSpriteLevel(spriteOrdIndex int) {
	c.spriteLvls[spriteOrdIndex] = nil
}
func combSort(order []int, dist []float64, amount int) {
	gap := amount
	swapped := false
	for gap > 1 || swapped {
		gap = (gap * 10) / 13
		if gap == 9 || gap == 10 {
			gap = 11
		}
		if gap < 1 {
			gap = 1
		}
		swapped = false
		for i := 0; i < amount-gap; i++ {
			j := i + gap
			if dist[i] < dist[j] {
				dist[i], dist[j] = dist[j], dist[i]
				order[i], order[j] = order[j], order[i]
				swapped = true
			}
		}
	}
}
func (c *Camera) SetPosition(pos *Vec2) {
	c.pos = pos
}
func (c *Camera) GetPosition() *Vec2 {
	return c.pos
}
func (c *Camera) SetPositionZ(gridPosZ float64) {
	c.posZ = gridPosZ
	c.camZ = (gridPosZ - 0.5) * float64(c.h)
}
func (c *Camera) GetPositionZ() float64 {
	return c.posZ
}
func (c *Camera) SetHeadingAngle(headingAngle float64) {
	c.headingAngle = headingAngle
	cameraDir := c.getVecForAngle(headingAngle)
	c.dir = cameraDir
	c.plane = c.getVecForFov(cameraDir)
}
func (c *Camera) SetPitchAngle(pitchAngle float64) {
	c.pitchAngle = pitchAngle
	cameraPitch := getOppositeTriangleLeg(pitchAngle, float64(c.h)*c.fovDepth)
	c.pitch = clampInt(int(cameraPitch), -c.h/2, int(float64(c.h)*c.fovDepth))
}
func (c *Camera) getAngleFromVec(dir *Vec2) float64 {
	return math.Atan2(dir.Y, dir.X)
}
func (c *Camera) getVecForAngleLength(angle, length float64) *Vec2 {
	return &Vec2{X: length * math.Cos(angle), Y: length * math.Sin(angle)}
}
func (c *Camera) getVecForAngle(angle float64) *Vec2 {
	return &Vec2{X: c.fovDepth * math.Cos(angle), Y: c.fovDepth * math.Sin(angle)}
}
func (c *Camera) getVecForFov(dir *Vec2) *Vec2 {
	angle := c.getAngleFromVec(dir)
	length := math.Sqrt(math.Pow(dir.X, 2) + math.Pow(dir.Y, 2))
	hypotenuse := length / math.Cos(c.fovAngle/2)
	return dir.copy().sub(c.getVecForAngleLength(angle+c.fovAngle/2, hypotenuse))
}
func (c *Camera) GetConvergenceDistance() float64 {
	return c.convergenceDistance
}
func (c *Camera) GetConvergencePoint() *Vec3 {
	return c.convergencePoint
}
func (c *Camera) GetConvergenceSprite() *Sprite {
	return c.convergenceSprite
}

// Update camera to match player position and orientation
func (g *Game) updatePlayerCamera(forceUpdate bool) {
	if !g.player.moved && !forceUpdate {
		// only update camera position if player moved or forceUpdate set
		return
	}

	// reset player moved flag to only update camera when necessary
	g.player.moved = false

	g.camera.SetPosition(g.player.Position.copy())
	g.camera.SetPositionZ(g.player.cameraZ)
	g.camera.SetHeadingAngle(g.player.Angle)
	g.camera.SetPitchAngle(g.player.Pitch)
}
