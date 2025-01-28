package state

import "github.com/Zhima-Mochi/raft-kv-store/raft"

var _ raft.State = &followerState{}

type followerState struct {
	node *raft.Node
}

func NewFollowerState(node *raft.Node) *followerState {
	return &followerState{node: node}
}

func (f *followerState) HandleHeartbeat() {
	raft.ResetHeartbeatTimeout(f.node)
	go func() {
		for range f.node.HeartbeatTimer().C {
			f.node.SetState(NewCandidateState(f.node))
		}
	}()
}
