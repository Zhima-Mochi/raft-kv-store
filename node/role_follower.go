package node

import (
	"context"
	"errors"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
	"github.com/google/uuid"
)

var _ Role = (*FollowerRole)(nil)

type FollowerRole struct {
	ctx    context.Context
	cancel context.CancelFunc
	node   *Node
	rand   *rand.Rand

	isEnd atomic.Bool
}

func NewFollowerRole(node *Node) *FollowerRole {
	return &FollowerRole{
		node: node,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (fr *FollowerRole) Name() string {
	return RoleNameFollower.String()
}

func (fr *FollowerRole) Enter(ctx context.Context) error {
	log := fr.node.GetLoggerEntry()
	log.Info("Entering Follower state")

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	fr.ctx = ctx
	fr.cancel = cancel

	// Start election timer
	ticker := time.NewTicker(LeaderHeartbeatTimeout)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// If we haven't received any AppendEntries from leader
				if !fr.isEnd.Load() {
					log.Warn("Leader heartbeat timeout, converting to candidate")
					fr.node.StepDown(fr, RoleNameCandidate)
					return
				}
			}
		}
	}()

	return nil
}

func (fr *FollowerRole) OnExit() error {
	log := fr.node.GetLoggerEntry()
	log.Info("Exiting Follower state")

	if fr.cancel != nil {
		fr.cancel()
	}

	if !fr.isEnd.CompareAndSwap(false, true) {
		return errors.New("follower already ended")
	}
	return nil
}

func (fr *FollowerRole) HandleAppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	log := fr.node.GetLoggerEntry().WithFields(map[string]interface{}{
		"leader_id": req.LeaderId.Value,
	})

	// Reply false if term < currentTerm
	if req.Term < fr.node.currentTerm {
		log.WithField("received_term", req.Term).Info("Rejecting AppendEntries due to lower term")
		return &pb.AppendEntriesResponse{
			CurrentTerm: fr.node.currentTerm,
			Success:     false,
		}, nil
	}

	// Update term if needed
	if req.Term > fr.node.currentTerm {
		fr.node.currentTerm = req.Term
		fr.node.voteTo = uuid.Nil
		log.WithField("new_term", req.Term).Info("Updated term from AppendEntries")
	}

	// Update leader ID
	fr.node.leaderID = uuid.MustParse(req.LeaderId.Value)

	return &pb.AppendEntriesResponse{
		CurrentTerm: fr.node.currentTerm,
		Success:     true,
	}, nil
}

func (fr *FollowerRole) HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	log := fr.node.GetLoggerEntry().WithFields(map[string]interface{}{
		"candidate_id": req.CandidateId.Value,
	})

	// Reply false if term < currentTerm
	if req.Term < fr.node.currentTerm {
		log.WithField("received_term", req.Term).Info("Rejecting vote request due to lower term")
		return &pb.RequestVoteResponse{
			CurrentTerm: fr.node.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// Update term if needed
	if req.Term > fr.node.currentTerm {
		fr.node.UpdateTerm(req.Term)
		log.WithField("new_term", req.Term).Info("Updated term from RequestVote")
	}

	// If votedFor is null or candidateId, and candidate's log is at least as up-to-date as receiver's log, grant vote
	if fr.node.voteTo == uuid.Nil || fr.node.voteTo.String() == req.CandidateId.Value {
		fr.node.voteTo = uuid.MustParse(req.CandidateId.Value)
		log.Info("Granted vote to candidate")
		return &pb.RequestVoteResponse{
			CurrentTerm: fr.node.currentTerm,
			VoteGranted: true,
		}, nil
	}

	log.WithField("voted_for", fr.node.voteTo.String()).Info("Rejecting vote request, already voted")
	return &pb.RequestVoteResponse{
		CurrentTerm: fr.node.currentTerm,
		VoteGranted: false,
	}, nil
}
