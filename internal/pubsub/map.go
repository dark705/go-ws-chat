package pubsub

import (
	"context"
	"fmt"
	"sync"
)

type inmemory struct {
	logger Logger
	mu     sync.Mutex
	chs    map[string]chan string
}

func NewInmemory(logger Logger) *inmemory {
	return &inmemory{logger: logger, chs: make(map[string]chan string)}
}

func (ps *inmemory) Sub(ctx context.Context, id string) (chan string, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ch := make(chan string)
	ps.chs[id] = ch
	ps.logger.InfofContext(ctx,
		"pubsub, inmemory, Sub, subscriber with id: %s subscribed, total subscribers is: %d",
		id, len(ps.chs))

	go func() {
		<-ctx.Done()
		ps.mu.Lock()
		defer ps.mu.Unlock()
		delete(ps.chs, id)
		ps.logger.InfofContext(ctx,
			"pubsub, inmemory, Sub, subscriber with id: %s unsubscribed, total subscribers is: %d",
			id, len(ps.chs))
	}()

	return ch, nil
}

func (ps *inmemory) Pub(ctx context.Context, id, message string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch, found := ps.chs[id]
	if !found {
		return fmt.Errorf("subscriber with id: %s not found", id)
	}
	ps.logger.DebugfContext(ctx, "pubsub, inmemory, Pub, subscriber with id: %s send: %s", id, message)
	ch <- message

	return nil
}
