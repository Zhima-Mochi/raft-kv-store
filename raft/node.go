package raft

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID             uuid.UUID
	State          NodeState
	CurrentTerm    int
	VotedFor       *uuid.UUID
	VotesReceived  int
	Peers          []uuid.UUID
	mutex          sync.Mutex
	electionTimer  *time.Timer
	heartbeatTimer *time.Timer
}

func NewNode(peers []uuid.UUID) *Node {
	id := uuid.New()

	return &Node{
		ID:    id,
		Peers: peers,
		State: &FollowerState{},
	}
}

func (n *Node) SetState(state NodeState) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.State = state
}
func (n *Node) SendHeartbeat(peer uuid.UUID) {
	panic("not implemented")
}

func (n *Node) ResetElectionTimeout() {
	panic("not implemented")
}

func (n *Node) AddPeer(peer uuid.UUID) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.Peers = append(n.Peers, peer)
}
