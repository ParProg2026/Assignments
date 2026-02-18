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


