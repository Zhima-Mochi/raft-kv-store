package node

import (
	"context"
	"fmt"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Peer interface {
	GetID() uuid.UUID
	AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error)
	RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error)
	GetLastHeartbeat() time.Time
	UpdateHeartbeat()
	Close() error
}

// Peer represents a remote Raft node (client for another Raft server)
type peer struct {
	id            uuid.UUID
	addr          string
	port          string
	lastHeartbeat time.Time
	conn          *grpc.ClientConn
	client        pb.RaftClient // gRPC client interface to communicate with another Raft node
}

// NewPeer creates a new Peer instance and establishes a gRPC connection
func NewPeer(id uuid.UUID, addr, port string) (Peer, error) {
	p := &peer{
		id:   id,
		addr: addr,
		port: port,
	}

	endpoint := fmt.Sprintf("%s:%s", addr, port)

	// Establish gRPC connection
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %v", id.String(), err)
	}

	// Create gRPC client
	p.conn = conn
	p.client = pb.NewRaftClient(conn)
	return p, nil
}

func (p *peer) GetID() uuid.UUID {
	return p.id
}

// Send AppendEntries RPC to this peer
func (p *peer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resp, err := p.client.AppendEntries(ctx, req)
	if err != nil {
		return nil, err
	}

	p.lastHeartbeat = time.Now()

	return resp, nil
}

func (p *peer) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resp, err := p.client.RequestVote(ctx, req)
	if err != nil {
		return nil, err
	}

	p.lastHeartbeat = time.Now()

	return resp, nil
}

// Close gRPC connection when shutting down
func (p *peer) Close() error {
	if p.conn != nil {
		err := p.conn.Close()
		if err != nil {
			return err
		}
		p.conn = nil
		p.client = nil
	}
	return nil
}

func (p *peer) GetLastHeartbeat() time.Time {
	return p.lastHeartbeat
}

func (p *peer) UpdateHeartbeat() {
	p.lastHeartbeat = time.Now()
}
