package node

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Peer struct {
	id            uuid.UUID
	address       string
	port          int
	conn          *grpc.ClientConn
	lastHeartbeat time.Time
}

func NewPeer(node *Node) *Peer {
	p := &Peer{
		id:      node.id,
		address: node.address,
		port:    node.port,
	}
	conn, err := grpc.NewClient(p.GetAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to peer %s: %v", p.GetAddress(), err)
	}
	p.conn = conn
	log.Printf("Connected to peer %s", p.GetAddress())
	return p
}

func (p *Peer) GetAddress() string {
	return fmt.Sprintf("%s:%d", p.address, p.port)
}

func (p *Peer) SendEvent(ctx context.Context, event *Event) error {
	// TODO: Implement gRPC event sending
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
