package node

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
)

type LeaderState struct {
	node *Node
}

func NewLeaderState(node *Node) *LeaderState {
	return &LeaderState{node: node}
}

func (ls *LeaderState) getRole() Role {
	return LeaderRole
}

func (ls *LeaderState) run(ctx context.Context) {
	log.Printf("[Leader %s] Leader state running", ls.node.id.String())

	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ls.sendHeartbeats(ctx)
		}
	}
}

func (ls *LeaderState) onExit() {
	log.Printf("[Leader %s] Exiting Leader state", ls.node.id.String())
}

func (ls *LeaderState) handleEvent(ctx context.Context, event *Event) error {
	fromUUID, err := uuid.Parse(event.From.Value)
	if err != nil {
		return err
	}

	// If there's a higher term, step down
	if ls.node.updateTermAndStepDown(ctx, event.Term, fromUUID) {
		return nil
	}

	switch event.Type {
	case EventType_EVENT_TYPE_HEARTBEAT:
		// Leader ignores heartbeats from others with the same or lower term
		return nil
	case EventType_EVENT_TYPE_VOTE_REQUEST:
		// If another candidate is requesting votes with the same term, we typically ignore.
		// If the candidate's term is higher, updateTermAndStepDown above already handled it.
		return nil
	case EventType_EVENT_TYPE_VOTE_RESPONSE:
		// Leader usually ignores vote responses after becoming leader
		return nil
	}
	return nil
}

func (ls *LeaderState) sendHeartbeats(ctx context.Context) {
	ls.node.termMutex.RLock()
	currentTerm := ls.node.currentTerm
	ls.node.termMutex.RUnlock()

	log.Printf("[Leader %s] Sending heartbeats (term %d)", ls.node.id.String(), currentTerm)

	ls.node.peerMutex.RLock()
	defer ls.node.peerMutex.RUnlock()

	for _, peer := range ls.node.peers {
		go func(p *Peer) {
			hbCtx, cancel := context.WithTimeout(ctx, HeartbeatInterval)
			defer cancel()

			ls.node.sendEventToPeer(hbCtx, p, &Event{
				Type: EventType_EVENT_TYPE_HEARTBEAT,
				From: &UUID{Value: ls.node.id.String()},
				Term: currentTerm,
			})
		}(peer)
	}
}
