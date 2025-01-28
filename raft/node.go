package raft

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	id             uuid.UUID
	address        string
	port           int
	isLeader       bool
	state          State
	election       Election
	currentTerm    int32
	votedFor       *Peer
	votesReceived  int32
	peers          []*Peer
	mutex          sync.RWMutex
	heartbeatTimer *time.Timer
}

func CreateNode(id uuid.UUID, address string, port int) *Node {
	node := &Node{
		id:      id,
		address: address,
		port:    port,
	}
	return node
}

func (n *Node) GetID() uuid.UUID {
	return n.id
}

func (n *Node) GetAddress() string {
	return n.address
}

func (n *Node) GetPort() int {
	return n.port
}

func (n *Node) SetState(state State) {
	n.state = state
	go n.state.HandleHeartbeat()
}

// GetCurrentTerm returns the current term
func (n *Node) GetCurrentTerm() int32 {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.currentTerm
}

// SetCurrentTerm updates the current term
func (n *Node) SetCurrentTerm(term int32) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.currentTerm = term
}

func (n *Node) AddPeer(peer *Peer) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.peers = append(n.peers, peer)
}

func (n *Node) GetPeers() []*Peer {
	return n.peers
}

func (n *Node) Run() error {
	if n.state == nil {
		return fmt.Errorf("state is nil")
	}
	go n.state.HandleHeartbeat()
	return nil
}

func (n *Node) HeartbeatTimer() *time.Timer {
	return n.heartbeatTimer
}
