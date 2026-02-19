package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"
)

// MsgType defines the intent of a network message.
type MsgType int

const (
	PROPOSE MsgType = iota
	ACCEPT
	MATCHED_MSG
)

// String representation for JSON serialization.
func (m MsgType) String() string {
	switch m {
	case PROPOSE:
		return "PROPOSE"
	case ACCEPT:
		return "ACCEPT"
	case MATCHED_MSG:
		return "MATCHED"
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON ensures custom enums serialize to strings.
func (m MsgType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// NodeState tracks the algorithm phase for visualization.
type NodeState string

const (
	SINGLE   NodeState = "SINGLE"
	PROPOSER NodeState = "PROPOSER"
	LISTENER NodeState = "LISTENER"
	MATCHED  NodeState = "MATCHED"
)

// Message encapsulates the data sent between nodes.
type Message struct {
	Type   MsgType `json:"type"`
	Sender int     `json:"sender"`
	Target int     `json:"target"`
}

// Event represents a single atomic occurrence in the simulation.
type Event struct {
	Timestamp int64     `json:"timestamp"`
	Type      string    `json:"type"` 
	Node      int       `json:"node"` 
	State     NodeState `json:"state,omitempty"`
	Msg       *Message  `json:"msg,omitempty"`
	Partner   int       `json:"partner"` 
	Nodes     []int     `json:"nodes,omitempty"`
	Edges     [][2]int  `json:"edges,omitempty"`
}

// EventRecorder acts as a synchronized sink for all simulation events.
type EventRecorder struct {
	EventChan chan Event
	DoneChan  chan bool
	file      *os.File
}

// NewEventRecorder initializes the JSON file and starts the listening goroutine.
func NewEventRecorder(filename string) *EventRecorder {
	f, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("Failed to create event log: %v", err))
	}
	f.WriteString("[\n")

	rec := &EventRecorder{
		EventChan: make(chan Event, 5000),
		DoneChan:  make(chan bool),
		file:      f,
	}

	go rec.listen()
	return rec
}

// listen sequentially writes events to disk to prevent interleaving.
func (r *EventRecorder) listen() {
	first := true
	for ev := range r.EventChan {
		data, _ := json.Marshal(ev)
		if !first {
			r.file.WriteString(",\n")
		}
		r.file.Write(data)
		first = false
	}
	r.file.WriteString("\n]\n")
	r.file.Close()
	r.DoneChan <- true
}

// Record queues an event with an automatic nanosecond timestamp.
func (r *EventRecorder) Record(ev Event) {
	ev.Timestamp = time.Now().UnixNano()
	r.EventChan <- ev
}

// Close signals the recorder to finish writing and flush to disk.
func (r *EventRecorder) Close() {
	close(r.EventChan)
	<-r.DoneChan
}

// Node represents an independent concurrent process in the graph.
type Node struct {
	ID        int                  
	State     NodeState            
	Inbox     chan Message         
	Network   map[int]chan Message 
	neighbors map[int]bool         
	pair      int                  
	recorder  *EventRecorder       
}

// InitNode provisions a new Node with its initial topology.
func InitNode(id int, neighbors []int, inbox chan Message, network map[int]chan Message, rec *EventRecorder) *Node {
	neighborSet := make(map[int]bool)
	for _, n := range neighbors {
		neighborSet[n] = true
	}

	return &Node{
		ID:        id,
		State:     SINGLE,
		Inbox:     inbox,
		Network:   network,
		neighbors: neighborSet,
		pair:      id,
		recorder:  rec,
	}
}

// simulateNetworkLatency introduces a brief block to mimic real network propagation,
// forcing concurrent events to cluster temporally for the visualizer.
func simulateNetworkLatency() {
	// 50ms is enough to create distinct temporal windows without dragging the simulation.
	time.Sleep(50 * time.Millisecond)
}

// changeState safely updates the node's state and logs the transition.
func (n *Node) changeState(newState NodeState) {
	if n.State != newState {
		n.State = newState
		n.recorder.Record(Event{
			Type:  "STATE_CHANGE",
			Node:  n.ID,
			State: newState,
		})
	}
}

// send issues a non-blocking message to a target node with simulated latency.
func (n *Node) send(to int, typ MsgType) {
	simulateNetworkLatency()
	msg := Message{Type: typ, Sender: n.ID, Target: to}
	select {
	case n.Network[to] <- msg:
		n.recorder.Record(Event{
			Type: "MSG_SENT",
			Node: n.ID,
			Msg:  &msg,
		})
	default:
		// Drop message if channel is full to prevent absolute deadlock.
	}
}

// finalize registers a successful pairing and notifies remaining neighbors.
func (n *Node) finalize(partnerID int) {
	n.pair = partnerID
	n.changeState(MATCHED)

	n.recorder.Record(Event{
		Type:    "MATCHED",
		Node:    n.ID,
		Partner: partnerID,
	})

	for nid := range n.neighbors {
		if nid != partnerID {
			n.send(nid, MATCHED_MSG)
		}
	}
}

// propose attempts to pair with the specified high-priority target.
func (n *Node) propose(targetID int) {
	n.changeState(PROPOSER)
	n.send(targetID, PROPOSE)

	waiting := true
	for waiting {
		msg := <-n.Inbox
		simulateNetworkLatency() // Mimic processing time
		n.recorder.Record(Event{Type: "MSG_RECV", Node: n.ID, Msg: &msg})

		switch msg.Type {
		case ACCEPT:
			if msg.Sender == targetID {
				n.finalize(targetID)
				return
			}
		case MATCHED_MSG:
			delete(n.neighbors, msg.Sender)
			if msg.Sender == targetID {
				waiting = false
			}
		case PROPOSE:
			// Ignore proposals while actively waiting for a response to our own.
		}
	}
}

// listen waits for incoming proposals and greedily accepts the first valid one.
func (n *Node) listen() {
	n.changeState(LISTENER)

	msg := <-n.Inbox
	simulateNetworkLatency() // Mimic processing time
	n.recorder.Record(Event{Type: "MSG_RECV", Node: n.ID, Msg: &msg})

	switch msg.Type {
	case PROPOSE:
		n.send(msg.Sender, ACCEPT)
		n.finalize(msg.Sender)
	case MATCHED_MSG:
		delete(n.neighbors, msg.Sender)
	}
}

// makePairs executes the core maximal matching algorithm.
func (n *Node) makePairs() {
	for n.pair == n.ID {
		if len(n.neighbors) == 0 {
			n.changeState(SINGLE)
			return
		}

		maxNeighborID := -1
		for id := range n.neighbors {
			if id > maxNeighborID {
				maxNeighborID = id
			}
		}

		if n.ID > maxNeighborID {
			n.propose(maxNeighborID)
		} else {
			n.listen()
		}
	}
}

// GenerateGraph creates a connected graph with additional random edges.
func GenerateGraph(numNodes int, extraEdges int) map[int][]int {
	adj := make(map[int][]int)
	for i := 0; i < numNodes; i++ {
		adj[i] = []int{}
	}

	shuffledIDs := make([]int, numNodes)
	for i := range numNodes {
		shuffledIDs[i] = i
	}
	rand.Shuffle(numNodes, func(i, j int) {
		shuffledIDs[i], shuffledIDs[j] = shuffledIDs[j], shuffledIDs[i]
	})

	addEdge := func(u, v int) {
		if u == v || slices.Contains(adj[u], v) {
			return
		}
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
	}

	for i := 0; i < numNodes-1; i++ {
		addEdge(shuffledIDs[i], shuffledIDs[i+1])
	}
	for i := 0; i < extraEdges; i++ {
		addEdge(rand.Intn(numNodes), rand.Intn(numNodes))
	}
	return adj
}

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <N nodes> <E extra edges>")
		os.Exit(2)
	}

	numNodes, _ := strconv.Atoi(os.Args[1])
	extraEdgesFactor, _ := strconv.Atoi(os.Args[2])
	extraEdges := numNodes * extraEdgesFactor

	recorder := NewEventRecorder("simulation_events.json")
	adj := GenerateGraph(numNodes, extraEdges)

	var edges [][2]int
	nodes := make([]int, numNodes)
	for u, neighbors := range adj {
		nodes[u] = u
		for _, v := range neighbors {
			if u < v {
				edges = append(edges, [2]int{u, v})
			}
		}
	}
	
	// Record initialization topology. 
	recorder.Record(Event{
		Type:  "INIT",
		Nodes: nodes,
		Edges: edges,
	})

	network := make(map[int]chan Message)
	for i := 0; i < numNodes; i++ {
		network[i] = make(chan Message, 1000)
	}

	var nodeInstances []*Node
	for i := 0; i < numNodes; i++ {
		nodeInstances = append(nodeInstances, InitNode(i, adj[i], network[i], network, recorder))
	}

	var wg sync.WaitGroup
	wg.Add(numNodes)

	for _, node := range nodeInstances {
		go func(n *Node) {
			defer wg.Done()
			n.makePairs()
		}(node)
	}

	wg.Wait()
	recorder.Close()

	fmt.Println("Simulation complete. Event log written to 'simulation_events.json'.")
}