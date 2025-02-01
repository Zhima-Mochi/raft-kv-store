package node

import (
	"context"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
)

type Role interface {
	Enter(ctx context.Context) error
	OnExit() error
	Name() string
	HandleAppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error)
	HandleRequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error)
}

type RoleName uint8

const (
	RoleNameFollower RoleName = iota
	RoleNameCandidate
	RoleNameLeader
)

func (r RoleName) String() string {
	return []string{"Follower", "Candidate", "Leader"}[r]
}

var (
	HeartbeatInterval      = 200 * time.Millisecond
	LeaderHeartbeatTimeout = 3 * HeartbeatInterval
)
