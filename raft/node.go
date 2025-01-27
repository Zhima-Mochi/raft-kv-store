package raft

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID             uuid.UUID
	address        string
	port           int
	state          NodeState
	currentTerm    int
	votedFor       *uuid.UUID
	votesReceived  int
	peers          []*Peer
	mutex          sync.Mutex
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
}

func NewNode(address string, port int) *Node {
	id := uuid.New()

	return &Node{
		ID:      id,
		address: address,
		port:    port,
		state:   &FollowerState{},
	}
}

func (n *Node) GetState() NodeState {
	return n.state
}

func (n *Node) SetState(state NodeState) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.state = state
}
func (n *Node) SendHeartbeat(peer *Peer) {
	panic("not implemented")
}

func (n *Node) ResetElectionTimeout() {
	panic("not implemented")
}

func (n *Node) AddPeer(peer *Peer) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.peers = append(n.peers, peer)
}

func (n *Node) GetPeers() []*Peer {
	return n.peers
}
