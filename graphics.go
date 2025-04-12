package main

import (
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 600
	screenHeight = 600
)

func newGraphics() *graphics {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Server reconceliation")
	return &graphics{updatersLock: &sync.Mutex{}}
}

type graphics struct {
	updatersLock *sync.Mutex
	updaters     []updater
}

func (g *graphics) subscribe(u updater) {
	g.updatersLock.Lock()
	g.updaters = append(g.updaters, u)
	g.updatersLock.Unlock()
}

func (g *graphics) unsubscribe(u updater) {
	g.updatersLock.Lock()
	g.updaters = append(g.updaters, u)
	g.updatersLock.Unlock()
}

func (g *graphics) Update() error {
	t := time.Now()
	for _, u := range g.updaters {
		u.update(t)
	}
	return nil
}

func (g *graphics) Draw(screen *ebiten.Image) {

	for _, u := range g.updaters {
		u.Draw(screen)
	}
}

func (g *graphics) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

var colors = []color.Color{
	color.White,
	color.RGBA{255, 0, 0, 255},
	color.RGBA{0, 255, 0, 255},
	color.RGBA{0, 0, 255, 255},
}
