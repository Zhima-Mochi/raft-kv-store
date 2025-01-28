package node

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/anypb"
)

type Role int32

const (
	FollowerRole  Role = 0
	CandidateRole Role = 1
	LeaderRole    Role = 2
)

const (
	HeartbeatInterval = 10 * time.Second
	ElectTimeout      = 10 * time.Second
)

type Node struct {
	id      uuid.UUID
	address string
	port    int

	role Role

	peers    map[uuid.UUID]*Peer
	leaderID uuid.UUID

	eventChan   chan *Event
	voteTo      uuid.UUID
	earnedVotes uint32

	currentTerm uint64

	peerMutex sync.RWMutex
}

func New(id uuid.UUID, address string, port int) *Node {
	node := &Node{
		id:        id,
		address:   address,
		port:      port,
		role:      FollowerRole,
		peers:     make(map[uuid.UUID]*Peer),
		eventChan: make(chan *Event, 1024),
	}
	return node
}

func (n *Node) GetID() uuid.UUID {
	return n.id
}

func (n *Node) AddPeer(peer *Peer) {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	n.peers[peer.id] = peer
}

func (n *Node) RemovePeer(peer *Peer) {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	delete(n.peers, peer.id)
}

func (n *Node) GetPeers() map[uuid.UUID]*Peer {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()
	return n.peers
}

func (n *Node) toCandidate() error {
	if n.role == CandidateRole {
		return nil
	}
	n.role = CandidateRole
	n.voteTo = uuid.Nil
	n.currentTerm++
	n.earnedVotes = 0
	log.Printf("[Node %s] Becoming candidate for term %d", n.id.String(), n.currentTerm)
	return nil
}

func (n *Node) toFollower(leaderID uuid.UUID) error {
	if n.role != CandidateRole {
		return errors.New("not a candidate")
	}
	n.role = FollowerRole
	n.leaderID = leaderID
	log.Printf("[Node %s] Becoming follower, leader is %s", n.id.String(), leaderID.String())
	return nil
}

func (n *Node) toLeader() error {
	if n.role != CandidateRole {
		return errors.New("not a candidate")
	}
	n.role = LeaderRole
	n.leaderID = n.id
	log.Printf("[Node %s] Becoming leader for term %d", n.id.String(), n.currentTerm)
	return nil
}

func (n *Node) keepAlive(ctx context.Context) {
	log.Printf("[Node %s] Starting keepAlive loop", n.id.String())
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n.role == LeaderRole {
				log.Printf("[Node %s] Sending heartbeat to peers as leader", n.id.String())
				n.peerMutex.RLock()
				for _, peer := range n.peers {
					go func(peer *Peer) {
						ctx, cancel := context.WithTimeout(ctx, HeartbeatInterval)
						defer cancel()
						peer.SendEvent(ctx, &Event{
							Type: EventType_EVENT_TYPE_HEARTBEAT,
							From: &UUID{Value: n.id.String()},
							Data: nil,
						})
					}(peer)
				}
				n.peerMutex.RUnlock()
			}
		}
	}
}

func (n *Node) Run(ctx context.Context) {
	go n.keepAlive(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-n.eventChan:
			log.Printf("[Node %s] Received event type %s", n.id.String(), event.GetType())
			if err := n.handleEvent(ctx, event); err != nil {
				log.Printf("Failed to handle event: %v", err)
			}
		}
	}
}

func (n *Node) handleEvent(ctx context.Context, event *Event) error {
	fromUUID, err := uuid.Parse(event.From.Value)
	if err != nil {
		return err
	}

	if event.Term > n.currentTerm {
		log.Printf("[Node %s] Updating term from %d to %d", n.id.String(), n.currentTerm, event.Term)
		n.currentTerm = event.Term
		n.toFollower(fromUUID)
	} else if event.Term < n.currentTerm {
		return nil
	}

	switch event.GetType() {
	case EventType_EVENT_TYPE_HEARTBEAT:
		peer, ok := n.peers[fromUUID]
		if !ok {
			return errors.New("received heartbeat from unknown peer")
		}

		if n.leaderID == fromUUID {
			peer.lastHeartbeat = time.Now()
			log.Printf("[Node %s] Received valid heartbeat from leader %s", n.id.String(), fromUUID.String())

			if n.role != FollowerRole {
				n.toFollower(fromUUID)
			}
		}
	case EventType_EVENT_TYPE_VOTE_REQUEST:
		peer, ok := n.peers[fromUUID]
		if !ok {
			return errors.New("received vote request from unknown peer")
		}

		if n.role == FollowerRole && n.voteTo == uuid.Nil {
			log.Printf("[Node %s] Sending vote to %s", n.id.String(), fromUUID.String())
			vote := &Vote{VoteTo: n.id}
			b, err := vote.Serialize()
			if err != nil {
				return err
			}
			peer.SendEvent(ctx, &Event{
				Type: EventType_EVENT_TYPE_VOTE_RESPONSE,
				From: &UUID{Value: n.id.String()},
				Data: &anypb.Any{Value: b},
			})
		}
	case EventType_EVENT_TYPE_VOTE_RESPONSE:
		if n.role != CandidateRole {
			return nil
		}

		var vote *Vote
		if vote, err = vote.Deserialize(event.Data.Value); err != nil {
			return err
		}
		if vote.VoteTo == n.id {
			n.earnedVotes++
			log.Printf("[Node %s] Received vote from %s (votes: %d/%d)",
				n.id.String(), fromUUID.String(), n.earnedVotes, len(n.peers)/2+1)
			if n.earnedVotes > uint32(len(n.peers)/2) {
				n.toLeader()
			}
		}
	}
	return nil
}
