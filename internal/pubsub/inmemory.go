package pubsub

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var errNotFound = errors.New("not found")

type inmemory struct {
	logger Logger
	mu     sync.Mutex
	chs    map[string]chan string
}

func NewInmemory(logger Logger) *inmemory {
	return &inmemory{
		logger: logger,
		chs:    make(map[string]chan string),
	}
}

func (ps *inmemory) Sub(ctx context.Context, id string) (chan string, error) { //nolint:varnamelen
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ch := make(chan string) //nolint:varnamelen
	ps.chs[id] = ch
	ps.logger.InfofContext(ctx,
		"pubsub, inmemory, Sub, subscribed ID: %s, total: %d",
		id, len(ps.chs))

	go func() {
		<-ctx.Done()
		ps.mu.Lock()
		defer ps.mu.Unlock()
		close(ch)
		delete(ps.chs, id)
		ps.logger.InfofContext(ctx,
			"pubsub, inmemory, Sub, unsubscribed ID: %s , total: %d",
			id, len(ps.chs))
	}()

	return ch, nil
}

func (ps *inmemory) Pub(ctx context.Context, id, message string) error { //nolint:varnamelen
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ch, found := ps.chs[id]
	if !found {
		return fmt.Errorf("subscriber ID: %s, %w", id, errNotFound)
	}
	ps.logger.DebugfContext(ctx, "pubsub, inmemory, Pub, subscriber ID: %s send: %s", id, message)
	ch <- message

	return nil
}
