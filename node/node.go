package node

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Node struct {
	id uuid.UUID
	es EventService

	// Protects term, voteTo, leaderID, earnedVotes
	termMutex sync.RWMutex

	currentTerm uint64
	voteTo      uuid.UUID
	leaderID    uuid.UUID

	// earnedVotes is used with atomic operations
	earnedVotes uint32

	// The node's current state (Follower/Candidate/Leader)
	state      RoleState
	stateMutex sync.RWMutex

	// Context management for the current RoleState
	roleCtx    context.Context
	roleCancel context.CancelFunc

	// Holds references to other cluster members
	peers     map[uuid.UUID]*Peer
	peerMutex sync.RWMutex

	// Channel for receiving events
	eventChan chan *Event
}

// New creates a new Node with some default values.
func New(id uuid.UUID, es EventService) *Node {
	node := &Node{
		id:        id,
		es:        es,
		voteTo:    uuid.Nil,
		peers:     make(map[uuid.UUID]*Peer),
		eventChan: make(chan *Event, 1024),
	}
	return node
}

// Run starts this Node as a Follower by default, then listens for incoming events.
func (n *Node) Run(ctx context.Context) {
	// Start in Follower state (leaderID = uuid.Nil initially)
	n.setState(ctx, NewFollowerState(n, uuid.Nil))

	for {
		select {
		case <-ctx.Done():
			// Cancel current state’s context
			if n.roleCancel != nil {
				n.roleCancel()
			}
			return
		case event := <-n.eventChan:
			// Delegate event handling to current state
			st := n.getCurrentState()
			if st != nil {
				if err := st.handleEvent(n.roleCtx, event); err != nil {
					log.Printf("[Node %s] Error handling event: %v", n.id.String(), err)
				}
			}
		}
	}
}

// updateTermAndStepDown atomically checks and steps down to a new term if eventTerm is larger.
func (n *Node) updateTermAndStepDown(ctx context.Context, eventTerm uint64, leaderID uuid.UUID) bool {
	n.termMutex.Lock()
	defer n.termMutex.Unlock()

	if eventTerm > n.currentTerm {
		n.currentTerm = eventTerm
		n.voteTo = uuid.Nil
		n.leaderID = leaderID
		// Step down to Follower
		n.setState(ctx, NewFollowerState(n, leaderID))
		return true
	}
	return false
}

// getCurrentState safely gets the current RoleState (Follower/Candidate/Leader)
func (n *Node) getCurrentState() RoleState {
	n.stateMutex.RLock()
	defer n.stateMutex.RUnlock()
	return n.state
}

// setState transitions the node to a new RoleState.
func (n *Node) setState(ctx context.Context, newState RoleState) {
	// Cancel old state context if exists
	if n.roleCancel != nil {
		n.roleCancel()
	}

	// Create a new context for the new state
	ctx, cancel := context.WithCancel(ctx)

	n.stateMutex.Lock()
	oldState := n.state
	n.state = newState
	n.roleCtx = ctx
	n.roleCancel = cancel
	n.stateMutex.Unlock()

	if oldState != nil {
		oldState.onExit()
	}

	// Log the role transition
	n.termMutex.RLock()
	currentTerm := n.currentTerm
	n.termMutex.RUnlock()

	log.Printf("[Node %s] Transitioning from %s to %s for term %d",
		n.id.String(),
		roleString(oldState),
		roleString(newState),
		currentTerm,
	)

	// Start the new state's run loop
	go newState.run(ctx)
}

// resetEarnedVotes atomically resets the earned votes to 0.
func (n *Node) resetEarnedVotes() {
	atomic.StoreUint32(&n.earnedVotes, 0)
}

// incrementEarnedVotes atomically increments the earned votes by 1.
func (n *Node) incrementEarnedVotes() {
	atomic.AddUint32(&n.earnedVotes, 1)
}

// getEarnedVotes safely gets the current number of earned votes.
func (n *Node) getEarnedVotes() uint32 {
	return atomic.LoadUint32(&n.earnedVotes)
}

// AddPeer safely adds a new peer to the node's peer map.
func (n *Node) AddPeer(peer *Peer) {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	n.peers[peer.id] = peer
}

// RemovePeer safely remove a peer from the node's peer map.
func (n *Node) RemovePeer(peer *Peer) {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	delete(n.peers, peer.id)
}

// GetPeers safely gets a snapshot of the node's peer map.
func (n *Node) GetPeers() map[uuid.UUID]*Peer {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()
	// Return a shallow copy if we want to avoid concurrency issues
	snap := make(map[uuid.UUID]*Peer)
	for k, v := range n.peers {
		snap[k] = v
	}
	return snap
}

// sendEventToPeer uses a short timeout to send an event to a peer.
func (n *Node) sendEventToPeer(ctx context.Context, peer *Peer, event *Event) error {
	// Create a separate context with a short timeout for sending
	c, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	return peer.SendEvent(c, event)
}
