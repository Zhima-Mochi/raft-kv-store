package node

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Peer represents another node in the cluster.
type Peer struct {
	id            uuid.UUID
	lastHeartbeat time.Time

	es EventService
}

// NewPeer creates a new Peer instance and establishes a gRPC connection.
func NewPeer(id uuid.UUID, es EventService) *Peer {
	return &Peer{
		id: id,
		es: es,
	}
}

// SendEvent sends an event to this peer via gRPC.
func (p *Peer) SendEvent(ctx context.Context, event *Event) error {
	return p.es.SendEvent(ctx, event)
}
