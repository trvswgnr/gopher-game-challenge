// coin.go
package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	coinAttractionDistance = 5.0
	coinLifespan           = 6.0
)

var (
	coinSprite      = loadCoinSprite()
	coins           []Coin
	playerCoinCoint int = 3
)

type Coin struct {
	x, y     float64
	lifespan float64
}

func loadCoinSprite() *ebiten.Image {
	return loadImageAsset("coin.png")
}

func (g *Game) dropCoin() {
	if playerCoinCoint > 0 {
		// Calculate position in front of the player
		coinX := g.player.x + g.player.dirX
		coinY := g.player.y + g.player.dirY

		coins = append(coins, Coin{x: coinX, y: coinY, lifespan: coinLifespan})
		playerCoinCoint--
	}
}

func (g *Game) updateCoins(deltaTime float64) {
	// store active coins
	activeCoinCount := 0
	for i := range coins {
		coins[i].lifespan -= deltaTime
		if coins[i].lifespan > 0 {
			// keep active coins
			coins[activeCoinCount] = coins[i]
			activeCoinCount++
		}
	}
	// remove expired coins
	coins = coins[:activeCoinCount]
}
