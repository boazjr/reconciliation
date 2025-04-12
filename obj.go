package main

import "fmt"

type obj struct {
	velocity float64
	pos      float64
	cycle    int
	clientID int
}

func (o obj) String() string {
	return fmt.Sprintf("obj(id: %d pos: %f velocity: %f)", o.clientID, o.pos, o.velocity)
}
func (o *obj) update(sc int) {
	if o == nil {
		return
	}
	o.pos += o.velocity
	o.cycle = sc
}

func (o *obj) act(a clientInput) {
	if a.velocity == nil {
		return
	}
	o.velocity = *a.velocity
}
