package logging

import (
	"context"
	"strconv"
	"sync/atomic"
)

type operationContextKey struct{}

var operationSequence atomic.Uint64

// StartOperation gives ctx a new operation ID that starts with prefix. The
// handler adds the ID to every entry that any module logs with ctx.
func StartOperation(ctx context.Context, prefix string) context.Context {
	operationID := prefix + "-" + strconv.FormatUint(operationSequence.Add(1), 10)
	return context.WithValue(ctx, operationContextKey{}, operationID)
}
