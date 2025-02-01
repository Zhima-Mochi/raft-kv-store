package node

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
)

var _ Role = (*LeaderRole)(nil)

type LeaderRole struct {
	ctx    context.Context
	cancel context.CancelFunc
	node   *Node

	isEnd atomic.Bool
}

func NewLeaderRole(node *Node) *LeaderRole {
	return &LeaderRole{node: node}
}

func (ls *LeaderRole) Name() string {
	return RoleNameLeader.String()
}

func (ls *LeaderRole) Enter(ctx context.Context) error {
	log := ls.node.GetLoggerEntry()
	log.Info("Entering Leader state")

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	ls.ctx = ctx
	ls.cancel = cancel

	// Start heartbeat ticker
	ticker := time.NewTicker(HeartbeatInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ls.sendHeartbeats(ctx)
			}
		}
	}()

	return nil
}

func (ls *LeaderRole) OnExit() error {
	log := ls.node.GetLoggerEntry()
	log.Info("Exiting Leader state")

	if ls.cancel != nil {
		ls.cancel()
	}

	if !ls.isEnd.CompareAndSwap(false, true) {
		return errors.New("leader already ended")
	}
	return nil
}

func (ls *LeaderRole) HandleAppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	// Update term if needed
	if req.Term > ls.node.currentTerm {
		ls.node.StepDown(ls, RoleNameFollower)
		return ls.node.role.HandleAppendEntries(ctx, req)
	}

	return &pb.AppendEntriesResponse{
		CurrentTerm: ls.node.currentTerm,
		Success:     false,
	}, nil
}

func (ls *LeaderRole) HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	// Update term if needed
	if req.Term > ls.node.currentTerm {
		ls.node.StepDown(ls, RoleNameFollower)
		return ls.node.RequestVote(ctx, req)
	}

	return &pb.RequestVoteResponse{
		CurrentTerm: ls.node.currentTerm,
		VoteGranted: false,
	}, nil
}

func (ls *LeaderRole) sendHeartbeats(ctx context.Context) {
	log := ls.node.GetLoggerEntry()

	for _, peer := range ls.node.GetPeers() {
		go func(peer Peer) {
			ctx, cancel := context.WithTimeout(ctx, LeaderHeartbeatTimeout)
			defer cancel()

			resp, err := peer.AppendEntries(ctx, &pb.AppendEntriesRequest{
				Term:     ls.node.currentTerm,
				LeaderId: &pb.UUID{Value: ls.node.ID.String()},
			})
			if err != nil {
				log.WithError(err).WithField("peer_id", peer.GetID().String()).Error("Failed to send heartbeat")
				return
			}

			if resp.CurrentTerm > ls.node.currentTerm {
				log.WithField("received_term", resp.CurrentTerm).Info("Received higher term in heartbeat response, stepping down")
				ls.node.UpdateTerm(resp.CurrentTerm)
				ls.node.StepDown(ls, RoleNameFollower)
			}
		}(peer)
	}
}
