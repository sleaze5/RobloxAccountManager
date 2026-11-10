package logging

import "context"

type operationContextKey struct{}

func WithOperation(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, operationContextKey{}, operationID)
}
