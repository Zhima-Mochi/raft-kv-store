package raft

type State interface {
	HandleHeartbeat()
}
