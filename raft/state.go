package raft

// NodeState follows the state pattern
type NodeState interface {
	HandleHeartbeat(node *Node)
	StartElection(node *Node)
	BecomeLeader(node *Node)
}
