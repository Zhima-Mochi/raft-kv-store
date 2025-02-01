package node

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Zhima-Mochi/raft-kv-store/logger"
	"github.com/Zhima-Mochi/raft-kv-store/pb"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

var _ pb.RaftServer = (*Node)(nil)

type Node struct {
	ID     uuid.UUID
	ctx    context.Context
	cancel context.CancelFunc

	address string
	port    string

	// Protects term, voteTo, leaderID, earnedVotes
	termMutex sync.RWMutex

	currentTerm uint64
	voteTo      uuid.UUID
	leaderID    uuid.UUID

	// earnedVotes is used with atomic operations
	earnedVotes uint32

	// UnimplementedRaftServer is used to implement the RaftServer interface
	pb.UnimplementedRaftServer

	// The node's current role (Follower/Candidate/Leader)
	role      Role
	roleMutex sync.RWMutex

	// Holds references to other cluster members
	peers     map[uuid.UUID]Peer
	peerMutex sync.RWMutex

	entries []*pb.LogEntry
}

// New creates a new Node with some default values.
func New(id uuid.UUID, address, port string) *Node {

	node := &Node{
		ID:      id,
		address: address,
		port:    port,
		voteTo:  uuid.Nil,
		peers:   make(map[uuid.UUID]Peer),
	}
	return node
}

func (n *Node) GetTerm() uint64 {
	n.termMutex.Lock()
	defer n.termMutex.Unlock()
	return n.currentTerm
}

// Run starts this Node as a Follower by default, then listens for incoming events.
func (n *Node) Run(ctx context.Context) {
	n.ctx = ctx

	n.StepDown(nil, RoleNameFollower)
	n.run()
}

func (n *Node) run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", n.address, n.port))
	if err != nil {
		log.Fatalf("[Node %s] failed to listen: %v", n.ID.String(), err)
	}
	server := grpc.NewServer()
	pb.RegisterRaftServer(server, n)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("[Node %s] failed to serve: %v", n.ID.String(), err)
	}
}

// updateTerm atomically checks and steps down to a new term if eventTerm is larger.
func (n *Node) UpdateTerm(term uint64) bool {
	n.termMutex.Lock()
	defer n.termMutex.Unlock()

	if term > n.currentTerm {
		n.currentTerm = term
		n.voteTo = uuid.Nil
		n.leaderID = uuid.Nil
		return true
	}
	return false
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
func (n *Node) AddPeer(peer Peer) error {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	n.peers[peer.GetID()] = peer

	return nil
}

// RemovePeer safely remove a peer from the node's peer map.
func (n *Node) RemovePeer(id uuid.UUID) {
	n.peerMutex.Lock()
	defer n.peerMutex.Unlock()
	delete(n.peers, id)
}

// GetPeers safely gets a snapshot of the node's peer map.
func (n *Node) GetPeers() map[uuid.UUID]Peer {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()
	// Return a shallow copy if we want to avoid concurrency issues
	snap := make(map[uuid.UUID]Peer)
	for k, v := range n.peers {
		snap[k] = v
	}
	return snap
}

func (n *Node) GetPeer(id uuid.UUID) (Peer, bool) {
	n.peerMutex.RLock()
	defer n.peerMutex.RUnlock()
	peer, ok := n.peers[id]
	return peer, ok
}

func (n *Node) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	return n.role.HandleAppendEntries(ctx, req)
}

func (n *Node) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	return n.role.HandleRequestVote(ctx, req)
}

func (n *Node) StepDown(oldRole Role, newRole RoleName) {
	n.roleMutex.Lock()
	defer n.roleMutex.Unlock()

	if oldRole != nil {
		if err := oldRole.OnExit(); err != nil {
			log.Printf("[Node %s] previous role failed to exit: %s", n.ID.String(), err)
			return
		}
	}

	switch newRole {
	case RoleNameFollower:
		n.role = NewFollowerRole(n)
	case RoleNameCandidate:
		n.role = NewCandidateRole(n)
	case RoleNameLeader:
		n.role = NewLeaderRole(n)
	}
	n.role.Enter(n.ctx)
}

func (n *Node) GetLoggerEntry() *logrus.Entry {
	return logger.GetLogger().WithFields(map[string]interface{}{
		"node_id": n.ID.String(),
		"role":    n.role.Name(),
		"term":    n.currentTerm,
	})
}
