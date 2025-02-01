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

	// Create and run node
	n := node.New(uuid.New(), "localhost", "8080")
	n.Run(context.Background())
}
