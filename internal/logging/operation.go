package logging

import (
	"context"
	"strconv"
	"sync/atomic"
)

type operationContextKey struct{}

var operationSequence atomic.Uint64

// NewOperationID returns an ID that is unique in this process, such as
// "chat-12", so that every entry of one operation can be found.
func NewOperationID(prefix string) string {
	return prefix + "-" + strconv.FormatUint(operationSequence.Add(1), 10)
}

func WithOperation(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, operationContextKey{}, operationID)
}
