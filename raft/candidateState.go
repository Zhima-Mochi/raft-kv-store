package raft

var _ NodeState = (*CandidateState)(nil)

// Candidate state implementation
type CandidateState struct{}

func (c *CandidateState) HandleHeartbeat(node *Node) {
	panic("not implemented")
}

func (c *CandidateState) StartElection(node *Node) {
	panic("not implemented")
}

func (c *CandidateState) BecomeLeader(node *Node) {
	panic("not implemented")
}
