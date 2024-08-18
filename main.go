package main

import (
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 1024
	screenHeight = 768
	mapWidth     = 24
	mapHeight    = 24
	texWidth     = 64
	texHeight    = 64
	numSprites   = 1
)

var (
	worldMap = [mapWidth][mapHeight]int{
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 2, 2, 2, 2, 2, 0, 0, 0, 0, 3, 0, 3, 0, 3, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0, 0, 3, 0, 0, 0, 3, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 2, 2, 0, 2, 2, 0, 0, 0, 0, 3, 0, 3, 0, 3, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 4, 4, 4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 4, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 0, 0, 0, 5, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 4, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 4, 4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 4, 4, 4, 4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}
	texture [8]*ebiten.Image
	sprite  *ebiten.Image
)

type Sprite struct {
	x, y float64
}

type Game struct {
	posX, posY, dirX, dirY, planeX, planeY float64
	sprites                                []Sprite
	zBuffer                                []float64
	spriteOrder                            []int
	spriteDistance                         []float64
}

func init() {
	for i := 0; i < 8; i++ {
		img, _, err := ebitenutil.NewImageFromFile("assets/wall" + string(rune(i+'0')) + ".png")
		if err != nil {
			log.Fatal(err)
		}
		texture[i] = img
	}

	spriteImg, _, err := ebitenutil.NewImageFromFile("assets/sprite.png")
	if err != nil {
		log.Fatal(err)
	}
	sprite = spriteImg
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		if worldMap[int(g.posX+g.dirX*0.1)][int(g.posY)] == 0 {
			g.posX += g.dirX * 0.1
		}
		if worldMap[int(g.posX)][int(g.posY+g.dirY*0.1)] == 0 {
			g.posY += g.dirY * 0.1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		if worldMap[int(g.posX-g.dirX*0.1)][int(g.posY)] == 0 {
			g.posX -= g.dirX * 0.1
		}
		if worldMap[int(g.posX)][int(g.posY-g.dirY*0.1)] == 0 {
			g.posY -= g.dirY * 0.1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		oldDirX := g.dirX
		g.dirX = g.dirX*math.Cos(-0.05) - g.dirY*math.Sin(-0.05)
		g.dirY = oldDirX*math.Sin(-0.05) + g.dirY*math.Cos(-0.05)
		oldPlaneX := g.planeX
		g.planeX = g.planeX*math.Cos(-0.05) - g.planeY*math.Sin(-0.05)
		g.planeY = oldPlaneX*math.Sin(-0.05) + g.planeY*math.Cos(-0.05)
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		oldDirX := g.dirX
		g.dirX = g.dirX*math.Cos(0.05) - g.dirY*math.Sin(0.05)
		g.dirY = oldDirX*math.Sin(0.05) + g.dirY*math.Cos(0.05)
		oldPlaneX := g.planeX
		g.planeX = g.planeX*math.Cos(0.05) - g.planeY*math.Sin(0.05)
		g.planeY = oldPlaneX*math.Sin(0.05) + g.planeY*math.Cos(0.05)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.zBuffer = make([]float64, screenWidth)

	// Draw ceiling
	vector.DrawFilledRect(screen, 0, 0, screenWidth, screenHeight/2, color.RGBA{135, 206, 235, 255}, false)

	// Draw floor
	vector.DrawFilledRect(screen, 0, screenHeight/2, screenWidth, screenHeight/2, color.RGBA{100, 100, 100, 255}, false)

	for x := 0; x < screenWidth; x++ {
		cameraX := 2*float64(x)/float64(screenWidth) - 1
		rayDirX := g.dirX + g.planeX*cameraX
		rayDirY := g.dirY + g.planeY*cameraX

		mapX, mapY := int(g.posX), int(g.posY)
		var sideDistX, sideDistY, perpWallDist float64
		deltaDistX, deltaDistY := math.Abs(1/rayDirX), math.Abs(1/rayDirY)
		var stepX, stepY, side int

		if rayDirX < 0 {
			stepX = -1
			sideDistX = (g.posX - float64(mapX)) * deltaDistX
		} else {
			stepX = 1
			sideDistX = (float64(mapX) + 1.0 - g.posX) * deltaDistX
		}
		if rayDirY < 0 {
			stepY = -1
			sideDistY = (g.posY - float64(mapY)) * deltaDistY
		} else {
			stepY = 1
			sideDistY = (float64(mapY) + 1.0 - g.posY) * deltaDistY
		}

		// Perform DDA
		for hit := false; !hit; {
			if sideDistX < sideDistY {
				sideDistX += deltaDistX
				mapX += stepX
				side = 0
			} else {
				sideDistY += deltaDistY
				mapY += stepY
				side = 1
			}
			if worldMap[mapX][mapY] > 0 {
				hit = true
			}
		}

		// Calculate distance to the wall
		if side == 0 {
			perpWallDist = (float64(mapX) - g.posX + (1-float64(stepX))/2) / rayDirX
		} else {
			perpWallDist = (float64(mapY) - g.posY + (1-float64(stepY))/2) / rayDirY
		}

		// Calculate wall height
		lineHeight := int(float64(screenHeight) / perpWallDist)
		drawStart := -lineHeight/2 + screenHeight/2
		if drawStart < 0 {
			drawStart = 0
		}
		drawEnd := lineHeight/2 + screenHeight/2
		if drawEnd >= screenHeight {
			drawEnd = screenHeight - 1
		}

		// Texture calculations
		texNum := worldMap[mapX][mapY] - 1
		wallX := 0.0
		if side == 0 {
			wallX = g.posY + perpWallDist*rayDirY
		} else {
			wallX = g.posX + perpWallDist*rayDirX
		}
		wallX -= math.Floor(wallX)

		texX := int(wallX * float64(texWidth))
		if side == 0 && rayDirX > 0 {
			texX = texWidth - texX - 1
		}
		if side == 1 && rayDirY < 0 {
			texX = texWidth - texX - 1
		}

		// Draw the wall slice
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1, float64(drawEnd-drawStart)/float64(texHeight))
		op.GeoM.Translate(float64(x), float64(drawStart))
		if side == 1 {
			op.ColorM.Scale(0.5, 0.5, 0.5, 1)
		}
		screen.DrawImage(texture[texNum].SubImage(image.Rect(texX, 0, texX+1, texHeight)).(*ebiten.Image), op)

		g.zBuffer[x] = perpWallDist
	}

	g.drawSprites(screen)
}

func (g *Game) drawSprites(screen *ebiten.Image) {
	for i := range g.sprites {
		g.spriteOrder[i] = i
		g.spriteDistance[i] = ((g.posX-g.sprites[i].x)*(g.posX-g.sprites[i].x) + (g.posY-g.sprites[i].y)*(g.posY-g.sprites[i].y))
	}
	combSort(g.spriteOrder, g.spriteDistance, numSprites)

	for i := 0; i < numSprites; i++ {
		spriteX := g.sprites[g.spriteOrder[i]].x - g.posX
		spriteY := g.sprites[g.spriteOrder[i]].y - g.posY

		invDet := 1.0 / (g.planeX*g.dirY - g.dirX*g.planeY)
		transformX := invDet * (g.dirY*spriteX - g.dirX*spriteY)
		transformY := invDet * (-g.planeY*spriteX + g.planeX*spriteY)

		spriteScreenX := int((float64(screenWidth) / 2) * (1 + transformX/transformY))
		spriteHeight := int(math.Abs(float64(screenHeight) / transformY))

		drawStartY := -spriteHeight/2 + screenHeight/2
		if drawStartY < 0 {
			drawStartY = 0
		}
		drawEndY := spriteHeight/2 + screenHeight/2
		if drawEndY >= screenHeight {
			drawEndY = screenHeight - 1
		}

		spriteWidth := int(math.Abs(float64(screenHeight) / transformY))
		drawStartX := -spriteWidth/2 + spriteScreenX
		if drawStartX < 0 {
			drawStartX = 0
		}
		drawEndX := spriteWidth/2 + spriteScreenX
		if drawEndX >= screenWidth {
			drawEndX = screenWidth - 1
		}

		for stripe := drawStartX; stripe < drawEndX; stripe++ {
			if transformY > 0 && stripe > 0 && stripe < screenWidth && transformY < g.zBuffer[stripe] {
				texX := int(256*(stripe-(-spriteWidth/2+spriteScreenX))*texWidth/spriteWidth) / 256
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(drawEndX-drawStartX)/float64(texWidth), float64(drawEndY-drawStartY)/float64(texHeight))
				op.GeoM.Translate(float64(stripe), float64(drawStartY))
				screen.DrawImage(sprite.SubImage(image.Rect(texX, 0, texX+1, texHeight)).(*ebiten.Image), op)
			}
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
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

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Raycasting Game")

	game := &Game{
		posX: 22, posY: 12,
		dirX: -1, dirY: 0,
		planeX: 0, planeY: 0.66,
		sprites: []Sprite{
			{x: 12, y: 12},
		},
		spriteOrder:    make([]int, numSprites),
		spriteDistance: make([]float64, numSprites),
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
