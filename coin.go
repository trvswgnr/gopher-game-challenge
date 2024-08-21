// coin.go
package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	coinAttractionDistance = 5.0
	coinLifespan           = 6.0
	coinDropDistance       = 2.0
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
	if playerCoinCoint <= 0 {
		return
	}

	// calculate position in front of the player
	coinX := g.player.x + g.player.dirX*coinDropDistance
	coinY := g.player.y + g.player.dirY*coinDropDistance

	// check if calculated position is valid (not inside a wall)
	if !g.checkCollision(coinX, coinY) {
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
