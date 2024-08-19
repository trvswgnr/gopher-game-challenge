package engine

import (
	"image"
	"image/color"
	"math"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	screenWidth  = 1024
	screenHeight = 768
)

type Image struct {
	pixels []byte
	width  int
	height int
}

func NewImage(width, height int) *Image {
	return &Image{
		pixels: make([]byte, width*height*4),
		width:  width,
		height: height,
	}
}

func (img *Image) Set(x, y int, c color.Color) {
	if x < 0 || x >= img.width || y < 0 || y >= img.height {
		return
	}
	r, g, b, a := c.RGBA()
	index := (y*img.width + x) * 4
	img.pixels[index] = byte(r >> 8)
	img.pixels[index+1] = byte(g >> 8)
	img.pixels[index+2] = byte(b >> 8)
	img.pixels[index+3] = byte(a >> 8)
}

func (img *Image) SubImage(r image.Rectangle) *Image {
	return &Image{
		pixels: img.pixels,
		width:  r.Dx(),
		height: r.Dy(),
	}
}

func (img *Image) DrawImage(src *Image, op *DrawImageOptions) {
	if op == nil {
		op = &DrawImageOptions{}
	}

	for sy := 0; sy < src.height; sy++ {
		for sx := 0; sx < src.width; sx++ {
			// Apply scaling and translation
			dx := int(float32(sx)*op.GeoM.ScaleVec.X + op.GeoM.Trans.X)
			dy := int(float32(sy)*op.GeoM.ScaleVec.Y + op.GeoM.Trans.Y)

			// Skip if outside the destination image
			if dx < 0 || dx >= img.width || dy < 0 || dy >= img.height {
				continue
			}

			// Get source color
			srcIndex := (sy*src.width + sx) * 4
			srcR := src.pixels[srcIndex]
			srcG := src.pixels[srcIndex+1]
			srcB := src.pixels[srcIndex+2]
			srcA := src.pixels[srcIndex+3]

			// Skip fully transparent pixels
			if srcA == 0 {
				continue
			}

			// Simple alpha blending
			dstIndex := (dy*img.width + dx) * 4
			dstR := img.pixels[dstIndex]
			dstG := img.pixels[dstIndex+1]
			dstB := img.pixels[dstIndex+2]
			dstA := img.pixels[dstIndex+3]

			outA := srcA + dstA*(255-srcA)/255
			if outA > 0 {
				outR := (srcR*srcA + dstR*dstA*(255-srcA)/255) / outA
				outG := (srcG*srcA + dstG*dstA*(255-srcA)/255) / outA
				outB := (srcB*srcA + dstB*dstA*(255-srcA)/255) / outA

				img.pixels[dstIndex] = outR
				img.pixels[dstIndex+1] = outG
				img.pixels[dstIndex+2] = outB
				img.pixels[dstIndex+3] = outA
			}
		}
	}
}

func (img *Image) Bounds() image.Rectangle {
	return image.Rect(0, 0, img.width, img.height)
}

type Vector struct {
	X, Y float32
}

type DrawImageOptions struct {
	GeoM GeoM
}

type GeoM struct {
	ScaleVec Vector
	Trans    Vector
}

func (g *GeoM) Scale(x, y float64) {
	g.ScaleVec.X *= float32(x)
	g.ScaleVec.Y *= float32(y)
}

func (g *GeoM) Translate(x, y float64) {
	g.Trans.X += float32(x)
	g.Trans.Y += float32(y)
}

type Game interface {
	Update() error
	Draw(screen *Image)
	Layout(outsideWidth, outsideHeight int) (int, int)
}

type Engine struct {
	window   *sdl.Window
	renderer *sdl.Renderer
	texture  *sdl.Texture
	game     Game
}

func NewEngine(game Game) *Engine {
	if err := sdl.Init(uint32(sdl.INIT_EVERYTHING)); err != nil {
		panic(err)
	}

	window, err := sdl.CreateWindow("Custom Game Engine", int32(sdl.WINDOWPOS_UNDEFINED), int32(sdl.WINDOWPOS_UNDEFINED),
		screenWidth, screenHeight, uint32(sdl.WINDOW_SHOWN))
	if err != nil {
		panic(err)
	}

	renderer, err := sdl.CreateRenderer(window, -1, uint32(sdl.RENDERER_ACCELERATED))
	if err != nil {
		panic(err)
	}

	texture, err := renderer.CreateTexture(uint32(sdl.PIXELFORMAT_ABGR8888), int(sdl.TEXTUREACCESS_STREAMING), screenWidth, screenHeight)
	if err != nil {
		panic(err)
	}

	return &Engine{
		window:   window,
		renderer: renderer,
		texture:  texture,
		game:     game,
	}
}

func NewImageFromFile(path string) (*Image, error) {
	img, err := sdl.LoadBMP(path)
	if err != nil {
		return nil, err
	}
	return &Image{pixels: img.Pixels(), width: int(img.W), height: int(img.H)}, nil
}

func (e *Engine) Run() {
	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent:
				running = false
			}
		}

		if err := e.game.Update(); err != nil {
			panic(err)
		}

		screen := NewImage(screenWidth, screenHeight)
		e.game.Draw(screen)

		e.texture.Update(nil, unsafe.Pointer(&screen.pixels[0]), screenWidth*4)
		e.renderer.Clear()
		e.renderer.Copy(e.texture, nil, nil)
		e.renderer.Present()

		sdl.Delay(16) // Cap at roughly 60 FPS
	}
}

func (e *Engine) Destroy() {
	e.texture.Destroy()
	e.renderer.Destroy()
	e.window.Destroy()
	sdl.Quit()
}

const (
	KeyW      sdl.Scancode = sdl.SCANCODE_W
	KeyA      sdl.Scancode = sdl.SCANCODE_A
	KeyS      sdl.Scancode = sdl.SCANCODE_S
	KeyD      sdl.Scancode = sdl.SCANCODE_D
	KeyShift  sdl.Scancode = sdl.SCANCODE_LSHIFT
	KeyEscape sdl.Scancode = sdl.SCANCODE_ESCAPE
	KeySpace  sdl.Scancode = sdl.SCANCODE_SPACE
)

// Utility functions to replace Ebitengine's functions
func IsKeyPressed(key sdl.Scancode) bool {
	keyState := sdl.GetKeyboardState()
	return keyState[key] == 1
}

func CursorPosition() (int, int) {
	x, y, _ := sdl.GetMouseState()
	return int(x), int(y)
}

func SetCursorMode(mode int) {
	if mode == 1 { // CursorModeCaptured
		sdl.SetRelativeMouseMode(true)
	} else {
		sdl.SetRelativeMouseMode(false)
	}
}

func ActualFPS() float64 {
	return 60.0 // For simplicity, we're assuming 60 FPS
}

// DrawFilledRect replaces vector.DrawFilledRect
func DrawFilledRect(screen *Image, x, y, width, height float32, clr color.Color) {
	r, g, b, a := clr.RGBA()
	for dy := int(y); dy < int(y+height); dy++ {
		for dx := int(x); dx < int(x+width); dx++ {
			screen.Set(dx, dy, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}
}

// DrawFilledCircle replaces vector.DrawFilledCircle
func DrawFilledCircle(screen *Image, x, y, radius float32, clr color.Color) {
	r, g, b, a := clr.RGBA()
	rSquared := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= rSquared {
				screen.Set(int(x+dx), int(y+dy), color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
			}
		}
	}
}

// StrokeLine replaces vector.StrokeLine
func StrokeLine(screen *Image, x1, y1, x2, y2, thickness float32, clr color.Color) {
	// This is a simple implementation. For better results, you might want to use Bresenham's line algorithm
	dx := x2 - x1
	dy := y2 - y1
	distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if distance == 0 {
		return
	}
	dx /= distance
	dy /= distance

	for i := float32(0); i < distance; i++ {
		DrawFilledCircle(screen, x1+dx*i, y1+dy*i, thickness/2, clr)
	}
}

// DebugPrintAt replaces ebitenutil.DebugPrintAt
func DebugPrintAt(screen *Image, str string, x, y int) {
	// This is a placeholder. Implementing text rendering is complex and beyond the scope of this example.
	// You might want to use a library like github.com/golang/freetype for text rendering.
}
