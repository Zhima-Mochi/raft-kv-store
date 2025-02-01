package node

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
	"github.com/google/uuid"
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
	term := cr.node.GetTerm()
	cr.node.UpdateTerm(term + 1)
	cr.node.voteTo = cr.node.ID

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
		cr.node.UpdateTerm(req.Term)
		cr.node.StepDown(cr, RoleNameFollower)
		return cr.node.role.HandleAppendEntries(ctx, req)
	}

	return &pb.AppendEntriesResponse{
		CurrentTerm: cr.node.currentTerm,
		Success:     false,
	}, nil
}

func (cr *CandidateRole) HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	log := cr.node.GetLoggerEntry().WithFields(map[string]interface{}{
		"candidate_id": req.CandidateId.Value,
	})

	// Update term if needed
	if req.Term > cr.node.currentTerm {
		cr.node.StepDown(cr, RoleNameCandidate)
		return cr.node.RequestVote(ctx, req)
	}

	// Reply false if term < currentTerm
	if req.Term < cr.node.currentTerm {
		log.WithField("received_term", req.Term).Info("Rejecting vote request due to lower term")
		return &pb.RequestVoteResponse{
			CurrentTerm: cr.node.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// As a candidate, we've already voted for ourselves in the current term
	// So we should reject all other vote requests in our term
	if req.Term == cr.node.currentTerm {
		log.Info("Rejecting vote request, already voted for self in current term")
		return &pb.RequestVoteResponse{
			CurrentTerm: cr.node.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// If we receive a RequestVote RPC with higher term

	log.WithField("received_term", req.Term).Info("Received higher term, updating term")
	cr.node.UpdateTerm(req.Term)
	leaderID := uuid.MustParse(req.CandidateId.Value)
	if leaderID == uuid.Nil {
		return nil, errors.New("invalid leader ID")
	}
	cr.node.leaderID = leaderID
	cr.node.voteTo = leaderID

	// Step down to follower since we discovered a higher term
	cr.node.StepDown(cr, RoleNameFollower)

	// Continue with vote handling as a follower would
	return &pb.RequestVoteResponse{
		CurrentTerm: cr.node.currentTerm,
		VoteGranted: true,
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
			// check if the role is ended
			if cr.isEnd.Load() {
				return
			}
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
