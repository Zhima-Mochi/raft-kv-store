package node

import (
	"time"

	"github.com/google/uuid"
)

// Constants for timeouts/intervals
const (
	ElectionTimeoutMin = 10 * time.Second
	ElectionTimeoutMax = 30 * time.Second
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
