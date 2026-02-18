package main

import (
	"fmt"
	"sort"

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

type Message struct {
	Type	MsgType
	Sender	int
}

// Each node is a process.
type Node struct {
	ID			int						// Node ID.
	Inbox		chan Message			// Incoming read-only messages.
	Network		map[int]chan Message	// Write-only access to neighbors.

	neighbors	map[int]bool			// Set of active neighbors.
	pair		int						// The ID of the node I paired with (final result).
}

func InitNode(id int, neighbors []int, inbox chan Message, network map[int]chan Message) *Node {
	neighborSet := make(map[int]bool)

	for _, n:= range neighbors {
		neighborSet[n] = true
	}

	return &Node {
		ID:			id,
		Inbox:		inbox,
		Network:	network,
		neighbors:	neighborSet,
		pair:		-1,
	}
}

func (n *Node) send(to int, typ MsgType) {
	select {
	case n.Network[to] <- Message{Type: typ, Sender: n.ID}:
	default:
		// Channel is full (should never happen), just log it.
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
	n.send(target_id, PROPOSE)
	
	// Wait for a response to the proposal
	waiting := true
	while waiting {
		msg := <-n.Inbox
		switch msg.Type {
		case ACCEPT:
			// Target has accepted our proposal. Yay!
			if msg.Sender == target_id{
				n.finalize(target_id)
				// log
				return
			}
		case PROPOSE:
			// We both proposed at the same time. Yay!
			if msg.Sender == target_id {
				n.finalize(target_id)
				// log
				return
			}
		case MATCHED:
			// Remove the neighbor from our neighbor list, it has already matched.
			delete(n.neighbors, msg.Sender)

			if msg.Sender == target_id {
				// Exit and re evaluate.
				waiting = false
			}
		}
	}
}

