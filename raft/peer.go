package raft

import (
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type Peer struct {
	id      uuid.UUID
	address string
	port    int
	conn    *grpc.ClientConn
	stream  *grpc.ClientStream
}

func NewPeer(id uuid.UUID, address string, port int) *Peer {
	return &Peer{
		id:      id,
		address: address,
		port:    port,
	}
}

func (p *Peer) GetAddress() string {
	return fmt.Sprintf("%s:%d", p.address, p.port)
}

func (p *Peer) SendHeartbeat() {

}
