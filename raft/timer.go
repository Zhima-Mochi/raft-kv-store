package raft

import (
	"math/rand"
	"time"
)

const (
	minElectionTimeout = 150 * time.Millisecond
	maxElectionTimeout = 300 * time.Millisecond
	heartbeatInterval  = 50 * time.Millisecond
)

// ResetElectionTimeout resets the election timer with a random duration
func ResetElectionTimeout(node *Node) {
	timeout := minElectionTimeout + time.Duration(rand.Int63n(int64(maxElectionTimeout-minElectionTimeout)))
	if !node.heartbeatTimer.Stop() {
		select {
		case <-node.heartbeatTimer.C:
		default:
		}
	}
	node.heartbeatTimer.Reset(timeout)
}

// ResetHeartbeatTimeout resets the heartbeat timer
func ResetHeartbeatTimeout(node *Node) {
	if !node.heartbeatTimer.Stop() {
		select {
		case <-node.heartbeatTimer.C:
		default:
		}
	}
	node.heartbeatTimer.Reset(heartbeatInterval)
}
