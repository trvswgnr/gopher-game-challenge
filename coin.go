// coin.go
package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	coinSprite      = loadCoinSprite()
	coins           []Coin
	playerCoinCoint int = 3
)

type Coin struct {
	x, y float64
}

func loadCoinSprite() *ebiten.Image {
	return loadImageAsset("coin.png")
}

func (g *Game) dropCoin() {
	if playerCoinCoint > 0 {
		coins = append(coins, Coin{x: g.player.x, y: g.player.y})
		playerCoinCoint--
	}
}
