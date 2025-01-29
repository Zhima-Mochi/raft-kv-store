package node

import "context"

type EventService interface {
	SendEvent(ctx context.Context, event *Event) error
}
