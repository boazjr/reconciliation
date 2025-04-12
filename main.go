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

const maxServerCycle = 200

func main() {
	// a := []int{1, 2, 3, 4}
	// for i := 0; i < len(a); i++ {
	// 	if a[i] == 4 {
	// 		a = append(a[:i], a[i+1:]...)
	// 		i--
	// 		continue
	// 	}
	// 	fmt.Println(a, a[i])
	// }
	fmt.Println(6.496125-4.640089, 0.928018)
	log.SetFlags(log.Lshortfile)
	fmt.Println("start")
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
	actions     *CircularArray[clientInput]
	lastUpdate  time.Time
	serverCycle int
	rnd         *rand.Rand
	network     *network
	ping        int
	// userEvents
}

func (c *client) update(t time.Time) {
	if t.Before(c.lastUpdate.Add(time.Second / 60)) {
		return
	}
	c.lastUpdate = t
	c.handleMessages()
	c.reconcileState()
	c.handleUserEvents()
	c.sendUserEvents()
	c.updateObjects()

	if c.serverCycle == maxServerCycle {
		os.Exit(0)
	}

}

func (c *client) String() string {
	if c.cliSimulation != nil {
		return fmt.Sprintf("client: servercycle: %d, obj %s, ping: %d", c.serverCycle, c.cliSimulation, c.ping)
	}
	return fmt.Sprintf("client: servercycle: %d, obj: %v, ping: %d", c.serverCycle, nil, c.ping)
}

func (c *client) Message(s serverMsg) {
	c.serverMsgLock.Lock()
	c.serverMessages = append(c.serverMessages, s)
	c.serverMsgLock.Unlock()
}

func (c *client) reconcileState() {
	c.serverCycle++
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
			// c.serverCycle = o.cycle
		}
	}
	if c.serverState != nil {
		oo := c.serverState
		if oo.clientID == c.id {
			op := *oo
			o := &op
			actions := c.actions.All()
			// loop until
			ai := 0
			for i := o.cycle; i <= c.serverCycle; i++ {
				for {
					if ai >= len(actions) {
						break
					}
					a := actions[ai]
					if a.sc < o.cycle {
						ai++
						continue
					}
					if a.sc == o.cycle {
						o.act(a)
						continue
					}
					if a.sc > o.cycle {
						break
					}
				}
				o.update(i)
			}
			// TODO: make this a smooth transition
			// need to add some stretching effect instead of jump
			c.cliSimulation = o
		}
	}
}

func (c *client) sendUserEvents() {
	ip := c.actions.Last(5)
	if len(ip) == 0 {
		return
	}
	m := clientMsg{
		client:      c,
		inputs:      ip,
		serverCycle: c.serverCycle,
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
		sc:       c.serverCycle,
	}
	c.actions.Add(ci)
	c.cliSimulation.act(ci)
	log.Println(ci)
}
func (c *client) handleMessages() {
	c.serverMsgLock.Lock()
	for _, m := range c.serverMessages {
		if m.lastCycleReceived != nil {
			c.ping = c.serverCycle - *m.lastCycleReceived
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
		connect:     true,
		client:      c,
		serverCycle: c.serverCycle,
	})
	c.clock.subscribe(c)
	return nil
}

func (c *client) updateObjects() {
	c.cliSimulation.update(c.serverCycle)
}

type clientMsg struct {
	connect     bool
	client      *client
	inputs      []clientInput
	serverCycle int
}

func (c clientMsg) String() string {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("clientMsg(connect: %t, sc: %d, inputs: ", c.connect, c.serverCycle))

	for i, ip := range c.inputs {
		sb.WriteString(fmt.Sprint(i, ip))
	}
	sb.WriteString(")")
	return sb.String()
}

type clientInput struct {
	sc       int
	velocity *float64
}

func (c clientInput) String() string {
	if c.velocity != nil {
		return fmt.Sprintf("clientInput(sc: %d, velocity: %f)", c.sc, *c.velocity)
	}
	return fmt.Sprintf("clientInput(sc: %d, velocity: nil)", c.sc)
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
	serverCycle       int
	lastUpdate        time.Time
	lastClientMessage *int
}

func (s *server) update(t time.Time) {
	if t.Before(s.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("server cycle", s.serverCycle)
	s.lastUpdate = t
	s.serverCycle++
	s.handleMessages()
	s.handleUserEvents()
	s.updateObjects()
	s.sendState()
}

func (s *server) String() string {
	return fmt.Sprintf("server: servercycle: %d, obj %s", s.serverCycle, s.clientState)
}

func (s *server) Setup(n *network) error {
	s.serverCycle = 1000
	n.server = s
	s.clock.subscribe(s)
	return nil
}

func (s *server) handleUserEvents() {
	for _, a := range s.newClientInputs {
		s.clientState.act(a)
		s.scLastInput = a.sc
	}
	s.newClientInputs = nil
}

func ptr[T any](t T) *T {
	return &t
}

func (s *server) handleMessages() {
	s.cliMsgLock.Lock()
	for _, m := range s.cliMsg {
		s.lastClientMessage = ptr(m.serverCycle)
		switch {
		case m.connect:
			s.client = m.client
			s.clientState = &obj{
				velocity: 0,
				pos:      0,
				cycle:    s.serverCycle,
				clientID: m.client.id,
			}
			s.scLastInput = -1
			s.newClientInputs = nil
		case len(m.inputs) != 0:
			// not concerned with out of sequence actions from client
			// use the timeOfLastInput to get only new inputs
			// then store them until the next update.
			for i, ip := range m.inputs {
				if ip.sc < s.scLastInput {
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
		state:             &o,
		lastCycleReceived: s.lastClientMessage,
	})
	s.lastClientMessage = nil
}
func (s *server) updateObjects() {
	if s.clientState == nil {
		return
	}
	s.clientState.update(s.serverCycle)
}
func (s *server) Message(c clientMsg) {
	s.cliMsgLock.Lock()
	s.cliMsg = append(s.cliMsg, c)
	s.cliMsgLock.Unlock()
}

type serverMsg struct {
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
