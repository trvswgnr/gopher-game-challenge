// main.go
package main

import (
	"embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*
var assets embed.FS

const (
	screenWidth  int = 1024
	screenHeight int = 768
)

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("office escape!")
	ebiten.SetCursorMode(ebiten.CursorModeCaptured)

	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
