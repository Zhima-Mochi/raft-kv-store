package node

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"
)

type FollowerState struct {
	node     *Node
	leaderID uuid.UUID
}

func NewFollowerState(node *Node, leaderID uuid.UUID) *FollowerState {
	return &FollowerState{
		node:     node,
		leaderID: leaderID,
	}
}

func (fs *FollowerState) getRole() Role {
	return FollowerRole
}

func (fs *FollowerState) run(ctx context.Context) {
	log.Printf("[Follower %s] Follower state running", fs.node.id.String())
	ticker := time.NewTicker(LeaderHeartbeatTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fs.checkLeaderAlive(ctx)
		}
	}
}

func (fs *FollowerState) onExit() {
	log.Printf("[Follower %s] Exiting Follower state", fs.node.id.String())
}

func (fs *FollowerState) handleEvent(ctx context.Context, event *Event) error {
	fromUUID, err := uuid.Parse(event.From.Value)
	if err != nil {
		return err
	}

	// If event has a higher term, step down.
	if fs.node.updateTermAndStepDown(ctx, event.Term, fromUUID) {
		return nil
	}

	switch event.Type {
	case EventType_EVENT_TYPE_HEARTBEAT:
		peer, ok := fs.node.peers[fromUUID]
		if !ok {
			return errors.New("follower received heartbeat from unknown peer")
		}
		if fs.leaderID != fromUUID {
			fs.leaderID = fromUUID
			fs.node.termMutex.Lock()
			fs.node.leaderID = fromUUID
			fs.node.termMutex.Unlock()
		}
		peer.lastHeartbeat = time.Now()

		log.Printf("[Follower %s] Heartbeat from leader %s", fs.node.id.String(), fromUUID.String())

	case EventType_EVENT_TYPE_VOTE_REQUEST:
		// If we haven't voted yet, vote for the requester
		fs.node.termMutex.Lock()
		if fs.node.voteTo == uuid.Nil {
			fs.node.voteTo = fromUUID
			fs.node.termMutex.Unlock()

			// Send vote response
			vote := &Vote{VoteTo: fs.node.id}
			b, err := vote.Serialize()
			if err != nil {
				return err
			}
			peer, ok := fs.node.peers[fromUUID]
			if ok {
				return fs.node.sendEventToPeer(ctx, peer, &Event{
					Type: EventType_EVENT_TYPE_VOTE_RESPONSE,
					From: &UUID{Value: fs.node.id.String()},
					Data: &anypb.Any{Value: b},
					Term: fs.node.currentTerm, // currentTerm safe to read under termMutex or in a snapshot
				})
			}
		} else {
			fs.node.termMutex.Unlock()
		}

	case EventType_EVENT_TYPE_VOTE_RESPONSE:
		// Follower ignores vote responses
		return nil
	}
	return nil
}

func (fs *FollowerState) checkLeaderAlive(ctx context.Context) {
	fs.node.peerMutex.RLock()
	leader, ok := fs.node.peers[fs.leaderID]
	fs.node.peerMutex.RUnlock()

	if !ok {
		// Leader not found -> become candidate
		log.Printf("[Follower %s] Leader %s missing -> become candidate", fs.node.id.String(), fs.leaderID.String())
		fs.node.setState(ctx, NewCandidateState(fs.node))
		return
	}

	if time.Since(leader.lastHeartbeat) > LeaderHeartbeatTimeout {
		// Leader timed out -> become candidate
		log.Printf("[Follower %s] Leader %s timed out -> become candidate", fs.node.id.String(), fs.leaderID.String())
		fs.node.setState(ctx, NewCandidateState(fs.node))
	}
}
