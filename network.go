package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type network struct {
	clk        *clock
	client     *client
	server     *server
	serverMsgs []NetworkWrapper[serverMsg]
	clientMsgs []NetworkWrapper[clientMsg]
	rnd        *rand.Rand
	lastUpdate time.Time
}

func (n *network) SendToClient(m serverMsg) {

	n.serverMsgs = append(n.serverMsgs, NetworkWrapper[serverMsg]{
		msg:     m,
		counter: n.rnd.Intn(2) + 3,
	})
}

func (n *network) SendToServer(m clientMsg) {
	n.clientMsgs = append(n.clientMsgs, NetworkWrapper[clientMsg]{
		msg:     m,
		counter: n.rnd.Intn(2) + 3,
	})
}

func (n *network) update(t time.Time) {
	if t.Before(n.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("server cycle", s.serverCycle)
	n.lastUpdate = t
	for i := 0; i < len(n.clientMsgs); i++ {
		cm := n.clientMsgs[i]
		cm.counter--
		if cm.counter < 0 {
			n.clientMsgs = append(n.clientMsgs[:i], n.clientMsgs[i+1:]...)
			i--
			n.server.Message(cm.msg)
			continue
		}
		n.clientMsgs[i] = cm
	}

	for i := 0; i < len(n.serverMsgs); i++ {
		cm := n.serverMsgs[i]
		cm.counter--
		if cm.counter < 0 {
			n.serverMsgs = append(n.serverMsgs[:i], n.serverMsgs[i+1:]...)
			i--
			n.client.Message(cm.msg)
			continue
		}
		n.serverMsgs[i] = cm
	}

}

func (n network) String() string {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("clientMSG: %d ", len(n.clientMsgs)))
	// for i := 0; i < len(n.clientMsgs); i++ {
	// 	cm := n.clientMsgs[i]
	// 	sb.WriteString(fmt.Sprint(cm.msg))
	// }
	sb.WriteString(fmt.Sprintf("serverMSG: %d ", len(n.serverMsgs)))
	// for i := 0; i < len(n.serverMsgs); i++ {
	// 	cm := n.serverMsgs[i]
	// 	sb.WriteString(fmt.Sprint(cm.msg))
	// }

	return fmt.Sprintf("network(%s)", sb.String())
}

func (n *network) Setup() error {
	n.clk.subscribe(n)
	return nil
}

type NetworkWrapper[T fmt.Stringer] struct {
	msg     T
	counter int
}

func (n *NetworkWrapper[T]) Decrement() bool {
	n.counter--
	return n.counter < 1
}
