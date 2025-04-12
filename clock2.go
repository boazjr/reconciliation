package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

type clock2 struct {
	time           time.Time
	clockEventLock *sync.Mutex
	clockEvents    []*clockEvent
}

func (c *clock2) Time() time.Time {
	return c.time
}

func (c *clock2) Run() {
	for {
		if len(c.clockEvents) == 0 {
			continue
		}
		sort.Slice(c.clockEvents, func(i, j int) bool {
			jat := c.clockEvents[j].at
			return c.clockEvents[i].at.Before(jat)
		})

		c.clockEventLock.Lock()
		ce := c.clockEvents[0]
		c.clockEvents = append([]*clockEvent{}, c.clockEvents[1:]...)
		c.clockEventLock.Unlock()
		lastTime := c.time
		c.time = ce.at
		if c.time.Before(lastTime) {
			log.Println("was before")
		}
		go func(ce *clockEvent) {
			ce.ch <- ce.at
		}(ce)
	}
}

func (c *clock2) String() string {
	return fmt.Sprint(c.clockEvents)
}
func (c *clock2) After(d time.Duration, owner string) <-chan time.Time {
	ch := make(chan time.Time)
	ce := &clockEvent{
		at:    c.time.Add(d),
		ch:    ch,
		owner: owner,
	}
	c.clockEventLock.Lock()
	c.clockEvents = append(c.clockEvents, ce)
	c.clockEventLock.Unlock()
	return ch
}

type clockEvent struct {
	at    time.Time
	ch    chan time.Time
	owner string
}

func (c clockEvent) String() string {
	return fmt.Sprint(c.owner, " ", c.at)
}
