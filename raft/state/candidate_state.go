package state

import (
	"github.com/Zhima-Mochi/raft-kv-store/raft"
)

var _ raft.State = &candidateState{}

type candidateState struct {
	node *raft.Node
}

func NewCandidateState(node *raft.Node) *candidateState {
	return &candidateState{node: node}
}

func (c *candidateState) HandleHeartbeat() {
	// do nothing
}
