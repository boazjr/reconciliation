package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const maxClientCycle = 1110

func main() {
	log.SetFlags(log.Lshortfile)

	// g := newCmdline()
	g := newGraphics()

	n := &network{
		clk: g,
	}
	n.Setup()
	s := &server{
		viz:        g,
		worldState: &worldState{},
	}

	s.Setup(n)

	c := &client{
		id:            1,
		viz:           g,
		serverMsgLock: &sync.Mutex{},
		rnd:           rand.New(rand.NewSource(0)),
		actions:       NewCircularArray[clientInput](20),
	}
	c.Setup(n)

	c2 := &client{
		id:            2,
		viz:           g,
		serverMsgLock: &sync.Mutex{},
		rnd:           rand.New(rand.NewSource(3)),
		actions:       NewCircularArray[clientInput](20),
	}
	c2.Setup(n)

	// clk.Run()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

type client struct {
	id             int
	serverMsgLock  *sync.Mutex
	serverMessages []serverMsg
	viz            ui
	cliServerState *obj
	// cliSimulation
	cliSimulation *obj
	worldState    *worldState
	// actions - velocity and server cycle // circularArray
	actions        *CircularArray[clientInput]
	lastUpdate     time.Time
	cycle          int
	rnd            *rand.Rand
	con            *connection
	ping           int
	nextCorrection int
	network        *network
	// userEvents
}

func (c *client) Draw(screen *ebiten.Image) {
	height := (c.id + 1) * 10 * 3
	c.worldState.Draw(screen, height)
	if c.cliSimulation == nil {
		return
	}
	draw(screen, int(c.cliSimulation.pos), height, colors[c.id])
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

}

func (c *client) String() string {
	cs := "nil"
	if c.cliSimulation != nil {
		cs = fmt.Sprint(c.cliSimulation)
	}
	p := "nil"
	if c.worldState != nil {
		p = fmt.Sprint(c.worldState.players)
	}
	return fmt.Sprintf("client %d: cycle: %d, simulation: %v, worldState: %v, ping: %d", c.id, c.cycle, cs, p, c.ping)
}

func (c *client) Message(s serverMsg) {
	c.serverMsgLock.Lock()
	c.serverMessages = append(c.serverMessages, s)
	c.serverMsgLock.Unlock()
}

func (c *client) reconcileState() {
	if c.cliSimulation == nil {
		// will happen once when the client connects because c.cliObj is nil
		if c.cliServerState != nil {
			o := c.cliServerState
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
	if c.cliServerState != nil {
		if c.cliServerState.clientID == c.id {
			c.cliSimulation = simulateObj(c.cliServerState, c.actions.All(), c.cycle)
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
				// log.Println("client reconcile messages", a)
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
	c.con.SendToServer(m)
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
	// log.Println(ci)
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
					_ = p
					// log.Println("made correction", p, c.cycle)
				}
			}
		}
		if m.state != nil {
			for _, st8 := range m.state {
				if st8.clientID == c.id {
					s2 := st8
					c.cliServerState = s2
				}
			}
			c.worldState = &worldState{
				players: m.state,
			}
		}
	}
	c.serverMsgLock.Unlock()
}
func (c *client) Setup(n *network) error {
	c.network = n
	c.con = c.network.ConnectToServer(c)
	c.con.SendToServer(clientMsg{
		connect: true,
		client:  c,
		cycle:   c.cycle,
	})
	c.viz.subscribe(c)
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
	viz        ui
	cycle      int
	lastUpdate time.Time
	worldState *worldState
	sc         []*serverClient
}

func (s *server) Draw(screen *ebiten.Image) {
	s.worldState.Draw(screen, 10)
}
func (s *server) client(c *client) *serverClient {
	for _, cl := range s.sc {
		if cl.con.client.id == c.id {
			return cl
		}
	}
	return nil
}
func (s *server) NewConnection(cli *client) *connection {
	con := newConnection(cli, s)
	s.sc = append(s.sc, &serverClient{
		cliMsgLock: &sync.Mutex{},
		con:        con,
		server:     s,
	})
	return con
}

func (s *server) update(t time.Time) {
	if t.Before(s.lastUpdate.Add(time.Second / 60)) {
		return
	}

	s.lastUpdate = t
	s.cycle++

	// deal with user inputs
	for _, c := range s.sc {
		c.handleMessages(s.cycle)
		c.handleUserEvents(s.cycle)
	}

	// update world state
	s.worldState.update(s.cycle)

	// send back the state
	for _, c := range s.sc {
		c.sendState(s.cycle, s.worldState)
	}
}

type worldState struct {
	players []*obj
}

func (w *worldState) Draw(screen *ebiten.Image, height int) {
	if w == nil {
		return
	}
	for _, p := range w.players {
		draw(screen, int(p.pos), height, colors[p.clientID])
	}
}

func draw(screen *ebiten.Image, x, y int, c color.Color) {
	screen.Set(x+1, y+1, c)
	screen.Set(x+1, y, c)
	screen.Set(x+1, y-1, c)
	screen.Set(x, y+1, c)
	screen.Set(x, y, c)
	screen.Set(x, y-1, c)
	screen.Set(x-1, y+1, c)
	screen.Set(x-1, y, c)
	screen.Set(x-1, y-1, c)
}

func (w *worldState) update(cycle int) {
	// update world state
	for _, c := range w.players {
		c.update(cycle)
	}
}

func (w *worldState) getCli(id int) *obj {
	for i := range len(w.players) {
		if w.players[i].clientID == id {
			return w.players[i]
		}
	}
	return nil
}

func (w *worldState) saveCli(o *obj) {
	w.players = append(w.players, o)
}

func (s *worldState) String() string {
	sb := &strings.Builder{}

	sb.WriteString("worldState:(")
	for _, o := range s.players {
		sb.WriteString(o.String())
	}

	sb.WriteString(")")
	return sb.String()
}

func (s *server) String() string {
	return fmt.Sprintf("server: cycle: %d, %s", s.cycle, s.worldState)
}

func (s *server) Setup(n *network) error {
	s.cycle = 1000
	n.server = s
	// con.server = s //TODO:
	s.viz.subscribe(s)
	return nil
}

type serverClient struct {
	server            *server
	cliMsgLock        *sync.Mutex
	cliMsg            []clientMsg
	cycleLastInput    int
	newClientInputs   []clientInput
	con               *connection
	lastClientMessage *int
}

func (s *serverClient) handleUserEvents(cycle int) {
	n := []clientInput{}
	for _, a := range s.newClientInputs {
		if a.cycle < cycle {
			continue
		}
		if a.cycle == cycle {
			s.server.worldState.getCli(s.con.client.id).act(a)
			s.cycleLastInput = a.cycle
			// log.Println("server reconciling message", a)
			continue
		}
		if a.cycle > cycle {
			n = append(n, a)
		}
	}
	s.newClientInputs = n
}

func ptr[T any](t T) *T {
	return &t
}

func (s *serverClient) handleMessages(cycle int) {
	s.cliMsgLock.Lock()
	for _, m := range s.cliMsg {
		s.lastClientMessage = ptr(m.cycle)
		switch {
		case m.connect:
			// s.con.client = m.client //TODO: does this make sense.
			s.server.worldState.saveCli(&obj{
				velocity: 0.5,
				pos:      0,
				cycle:    cycle,
				clientID: m.client.id,
			})
			s.cycleLastInput = -1
			s.newClientInputs = nil
		case len(m.inputs) != 0:
			// not concerned with out of sequence actions from client
			// use the timeOfLastInput to get only new inputs
			// then store them until the next update.
			for i, ip := range m.inputs {
				if ip.cycle < s.cycleLastInput {
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
func (s *serverClient) sendState(cycle int, ws *worldState) {
	s.con.SendToClient(serverMsg{
		cycle:             cycle,
		state:             ws.players,
		lastCycleReceived: s.lastClientMessage,
	})
	s.lastClientMessage = nil
}

func (s *serverClient) Message(c clientMsg) {
	s.cliMsgLock.Lock()
	s.cliMsg = append(s.cliMsg, c)
	s.cliMsgLock.Unlock()
}

func (c serverClient) String() string {
	return fmt.Sprintf("serverClient(last Message: %d)", c.lastClientMessage)
}

type serverMsg struct {
	cycle             int
	state             []*obj
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
