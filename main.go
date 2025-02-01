package main

import (
	"context"

	"github.com/Zhima-Mochi/raft-kv-store/node"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	// set log level
	logrus.SetLevel(logrus.DebugLevel)

	// Create nodes
	n1 := node.New(uuid.New(), "localhost", "8080")
	n2 := node.New(uuid.New(), "localhost", "8081")
	n3 := node.New(uuid.New(), "localhost", "8082")

	// Create peers
	p1, err := node.NewPeer(n1.ID, "localhost", "8080")
	if err != nil {
		logrus.Fatalf("Failed to create peer: %v", err)
		return
	}
	p2, err := node.NewPeer(n2.ID, "localhost", "8081")
	if err != nil {
		logrus.Fatalf("Failed to create peer: %v", err)
		return
	}
	p3, err := node.NewPeer(n3.ID, "localhost", "8082")
	if err != nil {
		logrus.Fatalf("Failed to create peer: %v", err)
		return
	}
	// Add peers to nodes
	n1.AddPeer(p1)
	n1.AddPeer(p2)
	n2.AddPeer(p1)
	n2.AddPeer(p3)
	n3.AddPeer(p2)
	n3.AddPeer(p3)

	// Run nodes
	go n1.Run(context.Background())
	go n2.Run(context.Background())
	go n3.Run(context.Background())

	select {}
}
