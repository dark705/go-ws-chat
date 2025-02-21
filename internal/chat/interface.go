package chat

import "context"

type Logger interface {
	Debugf(format string, args ...any)
	DebugfContext(ctx context.Context, format string, args ...any)

	Infof(format string, args ...any)
	InfofContext(ctx context.Context, format string, args ...any)

	Warnf(format string, args ...any)
	WarnfContext(ctx context.Context, format string, args ...any)

	Errorf(format string, args ...any)
	ErrorfContext(ctx context.Context, format string, args ...any)
}

type PubSub interface {
	Sub(ctx context.Context, id string) (chan string, error)
	Pub(ctx context.Context, id, message string) error
}
