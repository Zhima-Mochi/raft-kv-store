package node

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type CandidateState struct {
	node *Node
}

func NewCandidateState(node *Node) *CandidateState {
	return &CandidateState{node: node}
}

func (cs *CandidateState) getRole() Role {
	return CandidateRole
}

func (cs *CandidateState) run(ctx context.Context) {
	cs.startElection(ctx)

	// Generate a random election timeout
	randTimeout := time.Duration(rand.Int63n(int64(ElectionTimeoutMax-ElectionTimeoutMin))) + ElectionTimeoutMin
	electionTicker := time.NewTicker(randTimeout)
	defer electionTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-electionTicker.C:
			// If still candidate, start a new election
			log.Printf("[Candidate %s] Election timeout -> start new election", cs.node.id.String())
			cs.node.setState(ctx, NewCandidateState(cs.node))
			return
		}
	}
}

func (cs *CandidateState) onExit() {
	log.Printf("[Candidate %s] Exiting Candidate state", cs.node.id.String())
}

func (cs *CandidateState) handleEvent(ctx context.Context, event *Event) error {
	fromUUID, err := uuid.Parse(event.From.Value)
	if err != nil {
		return err
	}

	// If there's a higher term, step down
	if cs.node.updateTermAndStepDown(ctx, event.Term, fromUUID) {
		return nil
	}

	switch event.Type {
	case EventType_EVENT_TYPE_HEARTBEAT:
		// Another leader is out there -> become follower
		cs.node.setState(ctx, NewFollowerState(cs.node, fromUUID))
		return nil

	case EventType_EVENT_TYPE_VOTE_REQUEST:
		// Typically candidate might ignore or reject other requests in same term
		return nil

	case EventType_EVENT_TYPE_VOTE_RESPONSE:
		vote := &Vote{}
		if err := vote.Deserialize(event.Data.Value); err != nil {
			return err
		}
		if vote.VoteTo == cs.node.id {
			cs.node.incrementEarnedVotes()
			earned := cs.node.getEarnedVotes()

			// Check if majority
			cs.node.peerMutex.RLock()
			totalPeers := len(cs.node.peers) + 1 // include self
			cs.node.peerMutex.RUnlock()

			if int(earned) > totalPeers/2 {
				cs.node.setState(ctx, NewLeaderState(cs.node))
			}
		}
	}
	return nil
}

func (cs *CandidateState) startElection(ctx context.Context) {
	// Bump term, reset votes
	cs.node.termMutex.Lock()
	cs.node.currentTerm++
	newTerm := cs.node.currentTerm
	cs.node.voteTo = cs.node.id
	cs.node.resetEarnedVotes()
	cs.node.incrementEarnedVotes() // vote for self
	cs.node.termMutex.Unlock()

	log.Printf("[Candidate %s] Starting election in term %d", cs.node.id.String(), newTerm)

	// Send vote requests to peers
	cs.node.peerMutex.RLock()
	defer cs.node.peerMutex.RUnlock()

	for _, p := range cs.node.peers {
		go func(peer *Peer) {
			cs.node.sendEventToPeer(ctx, peer, &Event{
				Type: EventType_EVENT_TYPE_VOTE_REQUEST,
				From: &UUID{Value: cs.node.id.String()},
				Term: newTerm,
			})
		}(p)
	}
}
