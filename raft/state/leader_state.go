package state

import (
	"github.com/Zhima-Mochi/raft-kv-store/raft"
)

type leaderState struct {
	node *raft.Node
}

func NewLeaderState(node *raft.Node) *leaderState {
	return &leaderState{node: node}
}

func (l *leaderState) HandleHeartbeat() {
	// Send heartbeats to all peers
	peers := l.node.GetPeers()
	for _, peer := range peers {
		peer.SendHeartbeat()
	}
}
