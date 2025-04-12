package main

import (
	"log"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func newCmdline() *cmdline {
	return &cmdline{
		updatersLock: &sync.Mutex{},
	}
}

type cmdline struct {
	time         time.Time
	updatersLock *sync.Mutex
	updaters     []updater
}

func (c *cmdline) subscribe(u updater) {
	c.updatersLock.Lock()
	c.updaters = append(c.updaters, u)
	c.updatersLock.Unlock()
}

func (c *cmdline) unsubscribe(u updater) {
	c.updatersLock.Lock()
	c.updaters = append(c.updaters, u)
	c.updatersLock.Unlock()
}

func (c *cmdline) Run() {
	lastPrint := time.Time{}
	for {
		c.time = c.time.Add(time.Millisecond)
		c.updatersLock.Lock()
		for _, u := range c.updaters {
			u.update(c.time)
		}
		c.updatersLock.Unlock()
		if c.time.Sub(lastPrint) > time.Second/60 {
			lastPrint = c.time
			for _, u := range c.updaters {
				log.Println(u)
			}
		}
	}
}

type updater interface {
	update(time.Time)
	Draw(screen *ebiten.Image)
}

type ui interface {
	subscribe(u updater)
}
