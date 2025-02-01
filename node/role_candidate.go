package node

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
)

var _ Role = (*CandidateRole)(nil)

type CandidateRole struct {
	ctx    context.Context
	cancel context.CancelFunc
	node   *Node
	rand   *rand.Rand

	isEnd atomic.Bool

	votesReceived int
	voteMutex     sync.Mutex
}

func NewCandidateRole(node *Node) *CandidateRole {
	return &CandidateRole{
		node:          node,
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
		votesReceived: 1, // Vote for self
	}
}

func (cr *CandidateRole) Name() string {
	return RoleNameCandidate.String()
}

func (cr *CandidateRole) Enter(ctx context.Context) error {
	log := cr.node.GetLoggerEntry()
	log.Info("Entering Candidate state")

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	cr.ctx = ctx
	cr.cancel = cancel

	// Increment current term and vote for self
	cr.node.termMutex.Lock()
	cr.node.currentTerm++
	cr.node.voteTo = cr.node.ID
	cr.node.termMutex.Unlock()

	// Start election
	cr.startElection(ctx)

	// Set election timeout
	timeout := ElectionTimeoutMin + time.Duration(cr.rand.Float64()*float64(ElectionTimeoutMax-ElectionTimeoutMin))
	timer := time.NewTimer(timeout)

	go func() {
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			// Election timeout, start new election if we haven't stepped down
			if !cr.isEnd.Load() {
				log.Info("Election timeout, restarting election")
				cr.node.StepDown(cr, RoleNameCandidate) // Restart election
			}
		}
	}()

	return nil
}

func (cr *CandidateRole) OnExit() error {
	log := cr.node.GetLoggerEntry()
	log.Info("Exiting Candidate state")

	if cr.cancel != nil {
		cr.cancel()
	}

	if !cr.isEnd.CompareAndSwap(false, true) {
		return errors.New("candidate already ended")
	}
	return nil
}

func (cr *CandidateRole) HandleAppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	// If we receive an AppendEntries RPC from new leader with higher term
	if req.Term > cr.node.currentTerm {
		cr.node.StepDown(cr, RoleNameFollower)
		return &pb.AppendEntriesResponse{
			CurrentTerm: req.Term,
			Success:     true,
		}, nil
	}

	return &pb.AppendEntriesResponse{
		CurrentTerm: cr.node.currentTerm,
		Success:     false,
	}, nil
}

func (cr *CandidateRole) HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	// If we receive a RequestVote RPC with higher term
	if req.Term > cr.node.currentTerm {
		cr.node.StepDown(cr, RoleNameFollower)
		return &pb.RequestVoteResponse{
			CurrentTerm: req.Term,
			VoteGranted: false,
		}, nil
	}

	return &pb.RequestVoteResponse{
		CurrentTerm: cr.node.currentTerm,
		VoteGranted: false,
	}, nil
}

func (cr *CandidateRole) startElection(ctx context.Context) {
	log := cr.node.GetLoggerEntry()
	log.Info("Starting election")

	// Send RequestVote RPCs to all peers
	for _, peer := range cr.node.GetPeers() {
		go func(peer Peer) {
			resp, err := peer.RequestVote(ctx, &pb.RequestVoteRequest{
				Term:        cr.node.currentTerm,
				CandidateId: &pb.UUID{Value: cr.node.ID.String()},
			})
			if err != nil {
				log.WithError(err).WithField("peer_id", peer.GetID().String()).Error("Failed to send RequestVote")
				return
			}

			if resp.VoteGranted {
				cr.voteMutex.Lock()
				cr.votesReceived++

				// Check if we have majority
				if cr.votesReceived > (len(cr.node.peers)+1)/2 {
					log.Info("Received majority votes, becoming leader")
					cr.node.StepDown(cr, RoleNameLeader)
				}
				cr.voteMutex.Unlock()
			} else if resp.CurrentTerm > cr.node.currentTerm {
				log.WithField("received_term", resp.CurrentTerm).Info("Received higher term, stepping down")
				cr.node.StepDown(cr, RoleNameFollower)
			}
		}(peer)
	}
}
