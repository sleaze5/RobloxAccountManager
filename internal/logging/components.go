package logging

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type ComponentAttribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ComponentRecord struct {
	Module     string               `json:"module"`
	Message    string               `json:"message"`
	RecordedAt int64                `json:"recordedAt"`
	Attributes []ComponentAttribute `json:"attributes"`
}

type componentRegistry struct {
	mu      sync.Mutex
	order   []string
	records map[string]ComponentRecord
}

func newComponentRegistry() *componentRegistry {
	return &componentRegistry{records: map[string]ComponentRecord{}}
}

func (registry *componentRegistry) add(record ComponentRecord) {
	key := record.Module + "\x00" + record.Message
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, ok := registry.records[key]; !ok {
		registry.order = append(registry.order, key)
	}
	registry.records[key] = record
}

func (registry *componentRegistry) snapshot() []ComponentRecord {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	result := make([]ComponentRecord, 0, len(registry.order))
	for _, key := range registry.order {
		result = append(result, registry.records[key])
	}
	return result
}

type componentHandler struct {
	next       slog.Handler
	registry   *componentRegistry
	attributes []slog.Attr
}

func isDiagnostic(ctx context.Context) bool {
	diagnostic, _ := ctx.Value(diagnosticContextKey{}).(bool)
	return diagnostic
}

func (handler *componentHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return (handler.registry != nil && isDiagnostic(ctx)) || handler.next.Enabled(ctx, level)
}

func (handler *componentHandler) Handle(ctx context.Context, record slog.Record) error {
	if handler.registry != nil && isDiagnostic(ctx) {
		handler.capture(record)
	}
	if !handler.next.Enabled(ctx, record.Level) {
		return nil
	}
	return handler.next.Handle(ctx, record)
}

func (handler *componentHandler) capture(record slog.Record) {
	attributes := append([]slog.Attr(nil), handler.attributes...)
	record.Attrs(func(attribute slog.Attr) bool {
		attributes = append(attributes, attribute)
		return true
	})
	component := ComponentRecord{Message: record.Message, RecordedAt: record.Time.UnixMilli(), Attributes: []ComponentAttribute{}}
	if record.Time.IsZero() {
		component.RecordedAt = time.Now().UnixMilli()
	}
	for _, attribute := range attributes {
		switch attribute.Key {
		case "module":
			component.Module = attribute.Value.String()
		case "launch_id", "operation", "operation_id", "record_type":
		default:
			component.Attributes = append(component.Attributes, ComponentAttribute{Key: attribute.Key, Value: attribute.Value.Resolve().String()})
		}
	}
	handler.registry.add(component)
}

func (handler *componentHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	return &componentHandler{next: handler.next.WithAttrs(attributes), registry: handler.registry,
		attributes: append(append([]slog.Attr(nil), handler.attributes...), attributes...)}
}

func (handler *componentHandler) WithGroup(name string) slog.Handler {
	return &componentHandler{next: handler.next.WithGroup(name), registry: handler.registry, attributes: handler.attributes}
}
