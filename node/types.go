package node

import (
	"time"

	"github.com/google/uuid"
)

type Role int32

const (
	InitialRole   Role = 0
	FollowerRole  Role = 1
	CandidateRole Role = 2
	LeaderRole    Role = 3
)

// Constants for timeouts/intervals
const (
	HeartbeatInterval      = 1 * time.Second
	LeaderHeartbeatTimeout = 3 * time.Second

	// For randomized election timeout
	ElectionTimeoutMin = 150 * time.Millisecond
	ElectionTimeoutMax = 300 * time.Millisecond
)

type Vote struct {
	VoteTo uuid.UUID
}

func (v *Vote) Serialize() ([]byte, error) {
	return v.VoteTo.MarshalBinary()
}

func (v *Vote) Deserialize(b []byte) error {
	return v.VoteTo.UnmarshalBinary(b)
}
