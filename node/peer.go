package node

import (
	"context"
	"fmt"
	"time"

	"github.com/Zhima-Mochi/raft-kv-store/logger"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Peer interface {
	GetID() uuid.UUID
	AppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error)
	RequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error)
	GetLastHeartbeat() time.Time
	Close() error
}

// Peer represents a remote Raft node (client for another Raft server)
type peer struct {
	id            uuid.UUID
	addr          string
	port          string
	lastHeartbeat time.Time
	conn          *grpc.ClientConn
	client        RaftClient // gRPC client interface to communicate with another Raft node
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
	p.client = NewRaftClient(conn)
	return p, nil
}

func (p *peer) GetID() uuid.UUID {
	return p.id
}

// Send AppendEntries RPC to this peer
func (p *peer) AppendEntries(ctx context.Context, req *AppendEntriesRequest) (*AppendEntriesResponse, error) {
	log := p.GetLoggerEntry().WithFields(map[string]interface{}{
		"term": req.Term,
	})

	resp, err := p.client.AppendEntries(ctx, req)
	if err != nil {
		log.WithError(err).Error("Failed to send AppendEntries")
		return nil, err
	}

	log.WithFields(map[string]interface{}{
		"success":      resp.Success,
		"current_term": resp.CurrentTerm,
	}).Debug("Received AppendEntries response")

	p.lastHeartbeat = time.Now()
	return resp, nil
}

func (p *peer) RequestVote(ctx context.Context, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	log := p.GetLoggerEntry().WithFields(map[string]interface{}{
		"term":         req.Term,
		"candidate_id": req.CandidateId.Value,
	})

	resp, err := p.client.RequestVote(ctx, req)
	if err != nil {
		log.WithError(err).Error("Failed to send RequestVote")
		return nil, err
	}

	log.WithFields(map[string]interface{}{
		"vote_granted": resp.VoteGranted,
		"current_term": resp.CurrentTerm,
	}).Debug("Received RequestVote response")

	return resp, nil
}

// Close gRPC connection when shutting down
func (p *peer) Close() error {
	log := p.GetLoggerEntry()

	if p.conn != nil {
		err := p.conn.Close()
		if err != nil {
			log.WithError(err).Error("Failed to close connection")
			return err
		}
		log.Info("Disconnected from peer")
		p.conn = nil
		p.client = nil
	}
	return nil
}

func (p *peer) GetLastHeartbeat() time.Time {
	return p.lastHeartbeat
}

func (p *peer) GetLoggerEntry() *logrus.Entry {
	return logger.GetLogger().WithFields(map[string]interface{}{
		"peer_id": p.id.String(),
	})
}
