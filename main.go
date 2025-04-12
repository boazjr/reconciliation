package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile)
	fmt.Println("start")
	clk := &clock{
		updatersLock: &sync.Mutex{},
	}
	go clk.Run()
	s := &server{
		clock:      clk,
		cliMsgLock: &sync.Mutex{},
	}

	s.Setup()

	c := &client{
		server:        s,
		clock:         clk,
		serverMsgLock: &sync.Mutex{},
		rnd:           rand.New(rand.NewSource(0)),
		actions:       NewCircularArray[clientInputs](20),
	}
	c.Setup()

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
	serverState    []obj
	// cliSimulation
	cliSimulation *obj
	// actions - velocity and server cycle // circularArray
	actions     *CircularArray[clientInputs]
	lastUpdate  time.Time
	serverCycle int
	rnd         *rand.Rand
	// userEvents
}

func (c *client) String() string {
	if c.cliSimulation != nil {
		return fmt.Sprintf("client: servercycle: %d, obj %s", c.serverCycle, c.cliSimulation)
	}
	return "client: no objs"
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
		for _, o := range c.serverState {
			if o.clientID == c.id {
				c.cliSimulation = &obj{
					velocity: 1.,
					pos:      o.pos,
					id:       o.id,
					cycle:    o.cycle,
					clientID: o.clientID,
				}
			}
			c.serverCycle = o.cycle
		}
	}
	for _, oo := range c.serverState {
		if oo.clientID == c.id {
			op := oo
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
			c.cliSimulation = o
		}
	}
}

func (c *client) sendUserEvents() {
	c.server.Message(clientMsg{
		client:      c,
		inputs:      c.actions.Last(5),
		serverCycle: &c.serverCycle,
	})
}
func (c *client) handleUserEvents() {
	if c.cliSimulation == nil {
		return
	}
	if c.rnd.Float32()*1000 > 10 {
		return
	}
	newVel := c.rnd.Float64()
	ci := clientInputs{
		velocity: &newVel,
		sc:       c.serverCycle,
	}
	c.actions.Add(ci)
	c.cliSimulation.act(ci)
}
func (c *client) handleMessages() {
	c.serverMsgLock.Lock()
	for _, m := range c.serverMessages {
		if m.state != nil {
			c.serverState = m.state

		}
	}
	c.serverMsgLock.Unlock()
}
func (c *client) Setup() error {
	c.server.Message(clientMsg{
		connect: true,
		client:  c,
	})
	c.clock.subscribe(c)
	return nil
}
func (c *client) update(t time.Time) {
	if t.Before(c.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("client server cycle", c.serverCycle)
	// dt := t.Sub(c.lastUpdate)
	c.lastUpdate = t
	c.handleMessages()
	c.reconcileState()
	c.handleUserEvents()
	c.sendUserEvents()
	c.updateObjects()

	if c.serverCycle == 200 {
		os.Exit(0)
	}

}
func (c *client) updateObjects() {
	c.cliSimulation.update(c.serverCycle)
}

type clientMsg struct {
	connect     bool
	client      *client
	inputs      []clientInputs
	serverCycle *int
}

type clientInputs struct {
	sc       int
	velocity *float64
}

type server struct {
	cliMsgLock  *sync.Mutex
	cliMsg      []clientMsg
	clients     []*client
	clock       *clock
	objs        []obj
	serverCycle int
	lastUpdate  time.Time
}

func (s *server) String() string {
	if len(s.objs) > 0 {
		return fmt.Sprintf("server: servercycle: %d, obj %s", s.serverCycle, s.objs[0])
	}
	return "server: no objs"
}

func (s *server) Setup() error {
	s.clock.subscribe(s)
	return nil
}

func (s *server) update(t time.Time) {
	if t.Before(s.lastUpdate.Add(time.Second / 60)) {
		return
	}
	// log.Println("server cycle", s.serverCycle)
	s.lastUpdate = t
	s.serverCycle++
	s.handleMessages()
	s.updateObjects()
	s.sendState()
}

func (s *server) handleMessages() {
	s.cliMsgLock.Lock()
	for _, m := range s.cliMsg {
		switch {
		case m.connect:
			s.clients = append(s.clients, m.client)
			s.objs = append(s.objs, obj{
				velocity: 0,
				pos:      0,
				id:       0,
				cycle:    s.serverCycle,
				clientID: m.client.id,
			})
		case m.inputs != nil:
			// clientid := m.client.id

			// ca := server list of client actions
			// cs := server list of states
			// ca .append(newaction)
			// sort
			// rollback state to a specific time
			// linked list insert at position looping over the rest
			//

			// velocity := *m.inputs.velocity
			// serverCycle := *m.serverCycle
			// _, _, _ = clientid, velocity, serverCycle
			// for i := range s.objs {
			// 	if s.objs[i].id == clientid {
			// 		s.objs[i].velocity = velocity
			// 		// log.Printf("client server cycle: %d, server cycle: %d\n", serverCycle, s.serverCycle)
			// 		break
			// 	}
			// }
		}
	}
	s.cliMsg = nil
	s.cliMsgLock.Unlock()
}
func (s *server) sendState() {
	for i := range s.clients {
		if s.clients[i] == nil {
			continue
		}
		s.clients[i].Message(serverMsg{
			state: s.objs,
		})
	}
}
func (s *server) updateObjects() {
	for i := range s.objs {
		s.objs[i].update(s.serverCycle)
	}
}
func (s *server) Message(c clientMsg) {
	s.cliMsgLock.Lock()
	s.cliMsg = append(s.cliMsg, c)
	s.cliMsgLock.Unlock()
}

type serverMsg struct {
	state []obj
}

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
		if c.time.Sub(lastPrint) > time.Second/10 {
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

type obj struct {
	velocity float64
	pos      float64
	id       int
	cycle    int
	clientID int
}

func (o obj) String() string {
	return fmt.Sprintf("id: %d pos: %f velocity: %f", o.id, o.pos, o.velocity)
}
func (o *obj) update(sc int) {
	o.pos += o.velocity
	o.cycle = sc
}

func (o *obj) act(a clientInputs) {
	if a.velocity == nil {
		return
	}
	o.velocity = *a.velocity
}
