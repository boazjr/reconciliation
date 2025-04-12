package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type network struct {
	clk        *clock
	cons       []*connection
	lastUpdate time.Time
}

func (n *network) update(t time.Time) {
	if t.Before(n.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("server cycle", s.serverCycle)
	n.lastUpdate = t
	for _, c := range n.cons {
		c.update()
	}
}

func (n *network) NewConnection() *connection {
	c := &connection{
		rnd: rand.New(rand.NewSource(0)),
	}
	n.cons = append(n.cons, c)
	return c
}

func (n *network) Setup() error {
	n.clk.subscribe(n)
	return nil
}

func (n network) String() string {
	return ""
	sb := &strings.Builder{}
	// sb.WriteString(fmt.Sprintf("clientMSG: %d ", len(n.clientMsgs)))
	// for i := 0; i < len(n.clientMsgs); i++ {
	// 	cm := n.clientMsgs[i]
	// 	sb.WriteString(fmt.Sprint(cm.msg))
	// }
	// sb.WriteString(fmt.Sprintf("serverMSG: %d ", len(n.serverMsgs)))
	// for i := 0; i < len(n.serverMsgs); i++ {
	// 	cm := n.serverMsgs[i]
	// 	sb.WriteString(fmt.Sprint(cm.msg))
	// }

	return fmt.Sprintf("network(%s)", sb.String())
}

type connection struct {
	client     *client
	server     *server
	serverMsgs []NetworkWrapper[serverMsg]
	clientMsgs []NetworkWrapper[clientMsg]
	rnd        *rand.Rand
}

func (c *connection) update() {
	for i := 0; i < len(c.clientMsgs); i++ {
		cm := c.clientMsgs[i]
		cm.counter--
		if cm.counter < 0 {
			c.clientMsgs = append(c.clientMsgs[:i], c.clientMsgs[i+1:]...)
			i--
			c.server.Message(cm.msg)
			continue
		}
		c.clientMsgs[i] = cm
	}

	for i := 0; i < len(c.serverMsgs); i++ {
		cm := c.serverMsgs[i]
		cm.counter--
		if cm.counter < 0 {
			c.serverMsgs = append(c.serverMsgs[:i], c.serverMsgs[i+1:]...)
			i--
			c.client.Message(cm.msg)
			continue
		}
		c.serverMsgs[i] = cm
	}

}

func (n *connection) SendToClient(m serverMsg) {

	n.serverMsgs = append(n.serverMsgs, NetworkWrapper[serverMsg]{
		msg:     m,
		counter: n.rnd.Intn(2) + 3,
	})
}

func (n *connection) SendToServer(m clientMsg) {
	n.clientMsgs = append(n.clientMsgs, NetworkWrapper[clientMsg]{
		msg:     m,
		counter: n.rnd.Intn(2) + 3,
	})
}

type NetworkWrapper[T fmt.Stringer] struct {
	msg     T
	counter int
}

func (n *NetworkWrapper[T]) Decrement() bool {
	n.counter--
	return n.counter < 1
}
