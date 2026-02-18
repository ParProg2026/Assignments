package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
	"slices"
)

type MsgType int

const (
	// PROPOSE: "I want to pair with you"
	PROPOSE MsgType = iota
	// ACCEPT: "I received your proposal, and accepted it"
	ACCEPT
	// MATCHED: "I received your proposal, but I am either paired or single."
	MATCHED
)

func (m MsgType) String() string {
	switch m {
	case PROPOSE:
		return "PROPOSE"
	case ACCEPT:
		return "ACCEPT"
	case MATCHED:
		return "MATCHED"
	default:
		return "UNKNOWN"
	}
}

type Message struct {
	Type   MsgType
	Sender int
}

// Each node is a process.
type Node struct {
	ID      int                  // Node ID.
	Inbox   chan Message         // Incoming read-only messages.
	Network map[int]chan Message // Write-only access to neighbors.

	neighbors map[int]bool // Set of active neighbors.
	pair      int          // The ID of the node I paired with (final result).

	logger *log.Logger
}

func keys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func InitNode(id int, neighbors []int, inbox chan Message, network map[int]chan Message) *Node {
	neighborSet := make(map[int]bool)

	for _, n := range neighbors {
		neighborSet[n] = true
	}

	log_prefix := fmt.Sprintf("[Node %d] ", id)
	logger := log.New(os.Stdout, log_prefix, log.LUTC)

	return &Node{
		ID:        id,
		Inbox:     inbox,
		Network:   network,
		neighbors: neighborSet,
		logger:    logger,
		pair:      -1,
	}
}

func (n *Node) send(to int, typ MsgType) {
	// Non-blocking send to avoid potential deadlocks if buffers fill up
	select {
	case n.Network[to] <- Message{Type: typ, Sender: n.ID}:
	default:
		n.logger.Printf("WARNING: Network channel to Node %d is full!", to)
	}
}

func (n *Node) finalize(partner_id int) {
	// Save partner id in pair.
	n.pair = partner_id

	// Notify all the other neighbors of the new pair.
	for nid := range n.neighbors {
		if nid != partner_id {
			n.send(nid, MATCHED)
		}
	}
}

func (n *Node) propose(target_id int) {
	n.logger.Printf("Local Highest ID detected. Proposing to node %d...", target_id)
	n.send(target_id, PROPOSE)

	// Wait for a response to the proposal
	waiting := true
	for waiting {
		msg := <-n.Inbox
		switch msg.Type {
		case ACCEPT:
			if msg.Sender == target_id {
				// Target has accepted our proposal. Yay!
				n.logger.Printf("Node %d accepted my proposal!", target_id)
				n.finalize(target_id)
				return
			}
		case PROPOSE:
			// We both proposed at the same time. Yay!
			if msg.Sender == target_id {
				n.logger.Printf("Node %d cross-proposed with me!", target_id)
				n.finalize(target_id)
				return
			}
		case MATCHED:
			n.logger.Printf("Node %d is already MATCHED, removing from neighbors", msg.Sender)
			// Remove the neighbor from our neighbor list, it has already matched.
			delete(n.neighbors, msg.Sender)

			if msg.Sender == target_id {
				// Exit waiting loop and re-evaluate who is the new local max
				waiting = false
			}
		}
	}
}

func (n *Node) listen() {
	msg := <-n.Inbox

	switch msg.Type {
	case PROPOSE:
		n.logger.Printf("Received PROPOSE from [Node %d], accepting", msg.Sender)
		// We are not the node with the highest ID -> we accept any proposal that comes, greedy!
		n.send(msg.Sender, ACCEPT)
		n.finalize(msg.Sender)
	case MATCHED:
		n.logger.Printf("Node %d notified that he has MATCHED, removing from neighbors", msg.Sender)
		// We are being notified that the sender has been already matched, so we delete him.
		delete(n.neighbors, msg.Sender)
	}
}

func (n *Node) makePairs() {
	n.logger.Printf("Started. Neighbors: %v", keys(n.neighbors))
	
	for n.pair == -1 {
		if len(n.neighbors) == 0 {
			n.logger.Printf("No active neighbors. SINGLE Node")
			n.finalize(n.ID) // We are a single node, pair with ourselves :C
			return
		}

		// We need to find out if we have the highest ID to take priority as proposers.
		maxNeighborID := -1

		for id := range n.neighbors {
			if id > maxNeighborID {
				maxNeighborID = id
			}
		}

		haveMaxId := n.ID > maxNeighborID

		if haveMaxId {
			n.propose(maxNeighborID)
		} else {
			n.listen()
		}
	}
}
func main() {
	// Setup random source (math/rand v1 style, widely used)
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	// Configuration
	numNodes := 22
	extraEdgesFactor := 7

	// 1. Generate Random Connected Graph
	adj := make(map[int][]int)
	
	// Go 1.22+: 'range' over integer
	for i := range numNodes {
		adj[i] = []int{}
	}

	// Ensure connectivity: Backbone 0-1-2...
	// Go 1.22+: 'range' over integer expression
	for i := range numNodes - 1 {
		addEdge(adj, i, i+1)
	}

	// Add random edges
	// Go 1.22+: 'range' without index for simple repetition
	for range numNodes * extraEdgesFactor {
		u := rng.Intn(numNodes)
		v := rng.Intn(numNodes)
		if u != v {
			addEdge(adj, u, v)
		}
	}

	fmt.Println("--- Generated Random Graph (Adjacency List) ---")
	for i := range numNodes {
		fmt.Printf("Node %d: %v\n", i, adj[i])
	}
	fmt.Println("----------------------------------------------")

	// 2. Setup Network Channels
	network := make(map[int]chan Message)
	for i := range numNodes {
		network[i] = make(chan Message, 100)
	}

	// 3. Initialize Nodes
	var nodes []*Node
	for i := range numNodes {
		// InitNode assumed to be defined as in previous context
		node := InitNode(i, adj[i], network[i], network)
		nodes = append(nodes, node)
	}

	// 4. Run Algorithm concurrently
	var wg sync.WaitGroup
	wg.Add(numNodes)

	for _, node := range nodes {
		// Go 1.22+: Loop variables are recreated each iteration.
		// We no longer need to pass 'node' as an argument to the closure
		// to avoid the "sharing" bug. It captures the correct instance safely.
		go func() {
			defer wg.Done()
			node.makePairs()
		}()
	}

	// 5. Wait for termination
	wg.Wait()

	// 6. Print and Verify Results
	fmt.Println("\n--- Final Results ---")
	results := make(map[int]int)

	for _, n := range nodes {
		results[n.ID] = n.pair
		status := fmt.Sprintf("Paired with %d", n.pair)
		if n.pair == n.ID {
			status = "SINGLE"
		}
		fmt.Printf("Node %d: %s\n", n.ID, status)
	}

	// Verification
	valid := true
	for id, pair := range results {
		if pair == id { // If I am single
			for _, neighborID := range adj[id] {
				neighborPair := results[neighborID]
				if neighborPair == neighborID {
					fmt.Printf("VIOLATION: Node %d and Neighbor %d are both SINGLE!\n", id, neighborID)
					valid = false
				}
			}
		}
	}

	if valid {
		fmt.Println("\nSUCCESS: Algorithm terminated correctly (Maximal Matching).")
	} else {
		fmt.Println("\nFAILURE: Algorithm failed verification.")
	}
}

// Helper to add undirected edge avoiding duplicates
func addEdge(adj map[int][]int, u, v int) {
	if slices.Contains(adj[u], v) {
		return
	}
	adj[u] = append(adj[u], v)
	adj[v] = append(adj[v], u)
}
