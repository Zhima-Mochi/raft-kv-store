package main

import (
	"context"

	"github.com/Zhima-Mochi/raft-kv-store/node"
	"github.com/google/uuid"
)

func main() {
	nodes := make([]*node.Node, 3)
	for i := 0; i < 3; i++ {
		nodes[i] = node.New(uuid.New(), "localhost", 8080+i)
	}

	// set peers
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i != j {
				nodes[i].AddPeer(node.NewPeer(nodes[j]))
			}
		}
	}

	for _, n := range nodes {
		go n.Run(context.Background())
	}

	select {}
}
