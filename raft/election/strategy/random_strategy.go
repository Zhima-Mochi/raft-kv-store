package election

import (
	"math/rand"
	"time"
)

type RandomElection struct{}

func (r *RandomElection) CalculateTimeout() time.Duration {
	return time.Duration(rand.Intn(150)+150) * time.Millisecond
}
