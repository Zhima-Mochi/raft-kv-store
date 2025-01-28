package raft

type Election interface {
	StartElection()
}
