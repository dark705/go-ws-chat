package chat

import "context"

type Logger interface {
	DebugfContext(ctx context.Context, format string, args ...any)
	InfofContext(ctx context.Context, format string, args ...any)
	WarnfContext(ctx context.Context, format string, args ...any)
	ErrorfContext(ctx context.Context, format string, args ...any)
}
