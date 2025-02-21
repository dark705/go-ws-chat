package pubsub

import "context"

type Logger interface {
	DebugfContext(ctx context.Context, format string, args ...any)
	InfofContext(ctx context.Context, format string, args ...any)
}
