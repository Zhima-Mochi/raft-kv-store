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

	// Start heartbeat timer
	ticker := time.NewTicker(HeartbeatInterval + time.Duration(fr.rand.Float64()*float64(LeaderHeartbeatTimeout-HeartbeatInterval)))

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// If we haven't received any AppendEntries from leader
				if !fr.isEnd.Load() {
					if fr.node.leaderID == uuid.Nil {
						log := fr.node.GetLoggerEntry()
						log.Warn("No leader, converting to candidate")
						fr.node.StepDown(fr, RoleNameCandidate)
						return
					}
					peer, ok := fr.node.GetPeer(fr.node.leaderID)
					if !ok {
						log.WithField("leader_id", fr.node.leaderID.String()).Error("Leader not found")
						fr.node.StepDown(fr, RoleNameCandidate)
						return
					}
					if time.Since(peer.GetLastHeartbeat()) > LeaderHeartbeatTimeout {
						log := fr.node.GetLoggerEntry()
						log.WithFields(map[string]interface{}{
							"leader_id":      fr.node.leaderID.String(),
							"last_heartbeat": peer.GetLastHeartbeat(),
						}).Warn("Leader heartbeat timeout, converting to candidate")
						fr.node.StepDown(fr, RoleNameCandidate)
						return
					}
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

	// Update term if needed
	if req.Term > fr.node.currentTerm {
		fr.node.UpdateTerm(req.Term)
		return fr.node.role.HandleAppendEntries(ctx, req)
	}

	// Reply false if term < currentTerm
	if req.Term < fr.node.currentTerm {
		log.WithField("received_term", req.Term).Info("Rejecting AppendEntries due to lower term")
		return &pb.AppendEntriesResponse{
			CurrentTerm: fr.node.currentTerm,
			Success:     false,
		}, nil
	}

	// check if leader id is valid
	leaderID := uuid.MustParse(req.LeaderId.Value)
	if fr.node.leaderID != uuid.Nil && fr.node.leaderID != leaderID {
		log.WithField("received_leader_id", req.LeaderId.Value).Error("Invalid leader ID")
		return &pb.AppendEntriesResponse{
			CurrentTerm: fr.node.currentTerm,
			Success:     false,
		}, nil
	}

	fr.node.leaderID = leaderID
	fr.node.voteTo = leaderID

	return &pb.AppendEntriesResponse{
		CurrentTerm: fr.node.currentTerm,
		Success:     true,
	}, nil
}

func (fr *FollowerRole) HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	log := fr.node.GetLoggerEntry().WithFields(map[string]interface{}{
		"candidate_id": req.CandidateId.Value,
	})

	// Update term if needed
	if req.Term > fr.node.currentTerm {
		fr.node.UpdateTerm(req.Term)
		return fr.node.RequestVote(ctx, req)
	}

	// Reply false if term < currentTerm
	if req.Term < fr.node.currentTerm {
		log.WithField("received_term", req.Term).Info("Rejecting vote request due to lower term")
		return &pb.RequestVoteResponse{
			CurrentTerm: fr.node.currentTerm,
			VoteGranted: false,
		}, nil
	}

	// check if candidate id is valid
	candidateID := uuid.MustParse(req.CandidateId.Value)
	if candidateID == uuid.Nil {
		log.WithField("received_candidate_id", req.CandidateId.Value).Error("Invalid candidate ID")
		return &pb.RequestVoteResponse{
			CurrentTerm: fr.node.currentTerm,
			VoteGranted: false,
		}, nil
	}

	if fr.node.voteTo == uuid.Nil {
		log.WithField("voted_for", fr.node.voteTo.String()).Info("Granted vote to candidate")
		fr.node.voteTo = candidateID
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
