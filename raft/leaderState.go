package raft

var _ NodeState = (*LeaderState)(nil)

type LeaderState struct{}

func (l *LeaderState) HandleHeartbeat(node *Node) {
	for _, peer := range node.GetPeers() {
		go node.SendHeartbeat(peer)
	}
}

func (l *LeaderState) StartElection(node *Node) {
	panic("not implemented")
}

func (l *LeaderState) BecomeLeader(node *Node) {
	panic("not implemented")
}
