package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

const maxClientCycle = 1110

func main() {
	log.SetFlags(log.Lshortfile)
	clk := &clock{
		updatersLock: &sync.Mutex{},
	}
	n := &network{
		clk: clk,
		rnd: rand.New(rand.NewSource(0)),
	}
	n.Setup()
	go clk.Run()
	s := &server{
		clock:      clk,
		cliMsgLock: &sync.Mutex{},
		network:    n,
	}

	s.Setup(n)

	c := &client{
		server:        s,
		clock:         clk,
		serverMsgLock: &sync.Mutex{},
		rnd:           rand.New(rand.NewSource(0)),
		actions:       NewCircularArray[clientInput](20),
		network:       n,
	}
	c.Setup(n)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

type client struct {
	id             int
	server         *server
	serverMsgLock  *sync.Mutex
	serverMessages []serverMsg
	clock          *clock
	serverState    *obj
	// cliSimulation
	cliSimulation *obj
	// actions - velocity and server cycle // circularArray
	actions        *CircularArray[clientInput]
	lastUpdate     time.Time
	cycle          int
	rnd            *rand.Rand
	network        *network
	ping           int
	nextCorrection int
	// userEvents
}

func (c *client) update(t time.Time) {
	if t.Before(c.lastUpdate.Add(time.Second / 60)) {
		return
	}
	c.lastUpdate = t
	c.cycle++
	c.handleMessages()
	c.reconcileState()
	c.handleUserEvents()
	c.sendToServer()
	c.updateObjects()

	if c.cycle > maxClientCycle {
		os.Exit(0)
	}

}

func (c *client) String() string {
	if c.cliSimulation != nil {
		return fmt.Sprintf("client: cycle: %d, obj %s, serverState: %s, ping: %d", c.cycle, c.cliSimulation, c.serverState, c.ping)
	}
	return fmt.Sprintf("client: cycle: %d, obj: %v, ping: %d", c.cycle, nil, c.ping)
}

func (c *client) Message(s serverMsg) {
	c.serverMsgLock.Lock()
	c.serverMessages = append(c.serverMessages, s)
	c.serverMsgLock.Unlock()
}

func (c *client) reconcileState() {
	if c.cliSimulation == nil {
		// will happen once when the client connects because c.cliObj is nil
		if c.serverState != nil {
			o := c.serverState
			if o.clientID == c.id {
				c.cliSimulation = &obj{
					velocity: 1.,
					pos:      o.pos,
					cycle:    o.cycle,
					clientID: o.clientID,
				}
			}
		}
	}
	if c.serverState != nil {
		if c.serverState.clientID == c.id {
			c.cliSimulation = simulateObj(c.serverState, c.actions.All(), c.cycle)
		}
	}
}

func simulateObj(serverState *obj, actions []clientInput, curCycle int) *obj {
	oo := serverState
	op := *oo
	o := &op
	// loop until
	ai := 0
	for i := o.cycle + 1; i < curCycle; i++ {
		for {
			if ai >= len(actions) {
				break
			}
			a := actions[ai]
			if a.cycle < i {
				ai++
				continue
			}
			if a.cycle == i {
				log.Println("client reconcile messages", a)
				o.act(a)
				ai++
				continue
			}
			if a.cycle > i {
				break
			}
		}
		o.update(i)
	}
	// TODO: make this a smooth transition
	// need to add some stretching effect instead of jump
	return o
}

func (c *client) sendToServer() {
	ip := c.actions.Last(5)
	m := clientMsg{
		client: c,
		inputs: ip,
		cycle:  c.cycle,
	}
	c.network.SendToServer(m)
}
func (c *client) handleUserEvents() {
	if c.cliSimulation == nil {
		return
	}
	if c.rnd.Float32()*1000 > 10 {
		return
	}
	newVel := c.rnd.Float64()
	ci := clientInput{
		velocity: &newVel,
		cycle:    c.cycle + 2, // 2 is the speed at which the client reacts to the input
	}
	c.actions.Add(ci)
	log.Println(ci)
}
func (c *client) handleMessages() {
	c.serverMsgLock.Lock()
	for _, m := range c.serverMessages {
		if m.lastCycleReceived != nil {
			p := c.cycle - *m.lastCycleReceived
			offset := 0
			if p < 10 {
				c.ping = p
				if c.nextCorrection < c.cycle {
					offset = p
					c.nextCorrection = c.cycle + offset + 4
					p := c.cycle
					c.cycle = m.cycle + offset
					log.Println("made correction", p, c.cycle)
				}
			}
		}
		if m.state != nil {
			c.serverState = m.state
		}
	}
	c.serverMsgLock.Unlock()
}
func (c *client) Setup(n *network) error {
	n.client = c
	c.network.SendToServer(clientMsg{
		connect: true,
		client:  c,
		cycle:   c.cycle,
	})
	c.clock.subscribe(c)
	return nil
}

func (c *client) updateObjects() {
	c.cliSimulation.update(c.cycle)
}

type clientMsg struct {
	connect bool
	client  *client
	inputs  []clientInput
	cycle   int
}

func (c clientMsg) String() string {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("clientMsg(connect: %t, sc: %d, inputs: ", c.connect, c.cycle))

	for i, ip := range c.inputs {
		sb.WriteString(fmt.Sprint(i, ip))
	}
	sb.WriteString(")")
	return sb.String()
}

type clientInput struct {
	cycle    int
	velocity *float64
}

func (c clientInput) String() string {
	if c.velocity != nil {
		return fmt.Sprintf("clientInput(sc: %d, velocity: %f)", c.cycle, *c.velocity)
	}
	return fmt.Sprintf("clientInput(sc: %d, velocity: nil)", c.cycle)
}

type server struct {
	cliMsgLock        *sync.Mutex
	cliMsg            []clientMsg
	client            *client
	clock             *clock
	clientState       *obj
	scLastInput       int
	newClientInputs   []clientInput
	network           *network
	cycle             int
	lastUpdate        time.Time
	lastClientMessage *int
}

func (s *server) update(t time.Time) {
	if t.Before(s.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("server cycle", s.serverCycle)
	s.lastUpdate = t
	s.cycle++
	s.handleMessages()
	s.handleUserEvents()
	s.updateObjects()
	s.sendState()
}

func (s *server) String() string {
	return fmt.Sprintf("server: cycle: %d, obj %s", s.cycle, s.clientState)
}

func (s *server) Setup(n *network) error {
	s.cycle = 1000
	n.server = s
	s.clock.subscribe(s)
	return nil
}

func (s *server) handleUserEvents() {
	n := []clientInput{}
	for _, a := range s.newClientInputs {
		if a.cycle < s.cycle {
			continue
		}
		if a.cycle == s.cycle {
			s.clientState.act(a)
			s.scLastInput = a.cycle
			log.Println("server reconciling message", a)
			continue
		}
		if a.cycle > s.cycle {
			n = append(n, a)
		}
	}
	s.newClientInputs = n
}

func ptr[T any](t T) *T {
	return &t
}

func (s *server) handleMessages() {
	s.cliMsgLock.Lock()
	for _, m := range s.cliMsg {
		s.lastClientMessage = ptr(m.cycle)
		switch {
		case m.connect:
			s.client = m.client
			s.clientState = &obj{
				velocity: 0.5,
				pos:      0,
				cycle:    s.cycle,
				clientID: m.client.id,
			}
			s.scLastInput = -1
			s.newClientInputs = nil
		case len(m.inputs) != 0:
			// not concerned with out of sequence actions from client
			// use the timeOfLastInput to get only new inputs
			// then store them until the next update.
			for i, ip := range m.inputs {
				if ip.cycle < s.scLastInput {
					continue
				}
				ta := make([]clientInput, len(m.inputs)-i)
				copy(ta, m.inputs[i:])
				s.newClientInputs = ta
			}
		}
	}
	s.cliMsg = nil
	s.cliMsgLock.Unlock()
}
func (s *server) sendState() {
	if s.client == nil {
		return
	}
	o := *s.clientState
	s.network.SendToClient(serverMsg{
		cycle:             s.cycle,
		state:             &o,
		lastCycleReceived: s.lastClientMessage,
	})
	s.lastClientMessage = nil
}
func (s *server) updateObjects() {
	if s.clientState == nil {
		return
	}
	s.clientState.update(s.cycle)
}
func (s *server) Message(c clientMsg) {
	s.cliMsgLock.Lock()
	s.cliMsg = append(s.cliMsg, c)
	s.cliMsgLock.Unlock()
}

type serverMsg struct {
	cycle             int
	state             *obj
	lastCycleReceived *int
}

func (c serverMsg) String() string {
	if c.state == nil {
		return "nil"
	}
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("serverMsg(state: %v)", c.state))
	return sb.String()
}
