package raft

var _ NodeState = (*FollowerState)(nil)

type FollowerState struct{}

func (f *FollowerState) HandleHeartbeat(node *Node) {
	panic("not implemented")
}

func (f *FollowerState) StartElection(node *Node) {
	panic("not implemented")
}

func (f *FollowerState) BecomeLeader(node *Node) {
	panic("not implemented")
}
