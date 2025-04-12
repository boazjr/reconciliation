package main

import (
	"fmt"
	"sync"
	"time"
)

type clock struct {
	time         time.Time
	updatersLock *sync.Mutex
	updaters     []updater
}

func (c *clock) subscribe(u updater) {
	c.updatersLock.Lock()
	c.updaters = append(c.updaters, u)
	c.updatersLock.Unlock()
}

func (c *clock) unsubscribe(u updater) {
	c.updatersLock.Lock()
	c.updaters = append(c.updaters, u)
	c.updatersLock.Unlock()
}

func (c *clock) Run() {
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
				fmt.Println(u)
			}
		}
	}
}

type updater interface {
	update(time.Time)
}
