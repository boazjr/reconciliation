package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type network struct {
	clk        *clock
	cons       []*connection
	lastUpdate time.Time
	server     *server
}

func (n *network) ConnectToServer(cli *client) *connection {
	c := n.server.NewConnection(cli)
	n.cons = append(n.cons, c)
	return c
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

func (n *network) Setup() error {
	n.clk.subscribe(n)
	return nil
}

func (n network) String() string {
	sb := &strings.Builder{}
	for _, c := range n.cons {
		sb.WriteString(c.String())
	}
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
	sLock      *sync.Mutex
	serverMsgs []NetworkWrapper[serverMsg]
	cLock      *sync.Mutex
	clientMsgs []NetworkWrapper[clientMsg]
	rnd        *rand.Rand
}

func newConnection(cli *client, ser *server) *connection {
	return &connection{
		client: cli,
		server: ser,
		sLock:  &sync.Mutex{},
		cLock:  &sync.Mutex{},
		rnd:    rand.New(rand.NewSource(0)),
	}
}

func (c *connection) update() {
	log.Println("!!!", c)
	c.cLock.Lock()
	for i := 0; i < len(c.clientMsgs); i++ {
		cm := c.clientMsgs[i]
		cm.counter--
		if cm.counter < 0 {
			c.clientMsgs = append(c.clientMsgs[:i], c.clientMsgs[i+1:]...)
			i--
			cli := c.server.client(cm.msg.client)
			if cli != nil {
				cli.Message(cm.msg)
			}
			continue
		}
		c.clientMsgs[i] = cm
	}
	c.cLock.Unlock()
	c.sLock.Lock()
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
	c.sLock.Unlock()
}

func (c *connection) SendToClient(m serverMsg) {
	c.sLock.Lock()
	defer c.sLock.Unlock()
	c.serverMsgs = append(c.serverMsgs, NetworkWrapper[serverMsg]{ //TODO: deep copy
		msg:     m,
		counter: c.rnd.Intn(2) + 3,
	})
}

func (c connection) String() string {
	return fmt.Sprintf("connection: smsg %d, cmsg: %d", len(c.serverMsgs), len(c.clientMsgs))
}
func (c *connection) SendToServer(m clientMsg) {
	c.cLock.Lock()
	defer c.cLock.Unlock()
	c.clientMsgs = append(c.clientMsgs, NetworkWrapper[clientMsg]{ //TODO: deep copy
		msg:     m,
		counter: c.rnd.Intn(2) + 3,
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
