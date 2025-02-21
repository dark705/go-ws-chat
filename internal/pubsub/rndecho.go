package pubsub

import (
	"context"
)

type rndecho struct {
	logger Logger
	ch     chan string
}

func NewRndEcho(logger Logger) *rndecho {
	return &rndecho{logger: logger, ch: make(chan string)}
}

func (ps *rndecho) Sub(_ context.Context, _ string) (chan string, error) {
	return ps.ch, nil
}

func (ps *rndecho) Pub(ctx context.Context, id, message string) error {
	ps.logger.DebugfContext(ctx, "pubsub, rndecho, Pub, subscriber with id: %s send: %s", id, message)
	ps.ch <- message

	return nil
}
