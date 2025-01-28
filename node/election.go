package node

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Vote struct {
	VoteTo uuid.UUID
}

func (v *Vote) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func (v *Vote) Deserialize(data []byte) (*Vote, error) {
	var vote Vote
	if err := json.Unmarshal(data, &vote); err != nil {
		return nil, err
	}
	return &vote, nil
}
