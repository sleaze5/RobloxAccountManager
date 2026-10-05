package logging

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

const redacted = "[REDACTED]"

var forbiddenFragments = []string{
	"cookie",
	"roblosecurity",
	"csrf",
	"bound_auth",
	"bound-auth",
	"bat",
	"private_key",
	"ticket",
	"password",
	"hint",
	"secret",
	"database_key",
	"encryption_key",
	"key_encryption",
	"dpapi",
	"ciphertext",
	"authorization",
	"request_body",
	"response_body",
	"set-cookie",
}

type redactingHandler struct {
	next slog.Handler
}

type levelFilter struct {
	mu      sync.RWMutex
	enabled map[slog.Level]bool
}

func newLevelFilter(levels []Level) *levelFilter {
	filter := &levelFilter{}
	filter.set(levels)
	return filter
}

func (filter *levelFilter) set(levels []Level) {
	enabled := make(map[slog.Level]bool, len(levels))
	for _, level := range levels {
		enabled[level.slogLevel()] = true
	}
	filter.mu.Lock()
	filter.enabled = enabled
	filter.mu.Unlock()
}

func (filter *levelFilter) includes(level slog.Level) bool {
	filter.mu.RLock()
	defer filter.mu.RUnlock()
	return filter.enabled[level]
}

func (filter *levelFilter) any() bool {
	filter.mu.RLock()
	defer filter.mu.RUnlock()
	return len(filter.enabled) > 0
}

func (filter *levelFilter) allows(ctx context.Context, level slog.Level) bool {
	if metadata, _ := ctx.Value(diagnosticContextKey{}).(bool); metadata {
		filter.mu.RLock()
		defer filter.mu.RUnlock()
		return len(filter.enabled) > 0
	}
	return filter.includes(level)
}

type filteringHandler struct {
	next   slog.Handler
	filter *levelFilter
}

func (handler *filteringHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.filter.allows(ctx, level) && handler.next.Enabled(ctx, level)
}

func (handler *filteringHandler) Handle(ctx context.Context, record slog.Record) error {
	if !handler.filter.allows(ctx, record.Level) {
		return nil
	}
	return handler.next.Handle(ctx, record)
}

func (handler *filteringHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	return &filteringHandler{next: handler.next.WithAttrs(attributes), filter: handler.filter}
}

func (handler *filteringHandler) WithGroup(name string) slog.Handler {
	return &filteringHandler{next: handler.next.WithGroup(name), filter: handler.filter}
}

func (handler *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.next.Enabled(ctx, level)
}

func (handler *redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	clean := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	if id, _ := ctx.Value(operationContextKey{}).(string); id != "" {
		clean.AddAttrs(slog.String("operation_id", id))
	}
	record.Attrs(func(attribute slog.Attr) bool {
		clean.AddAttrs(cleanAttribute(attribute))
		return true
	})
	return handler.next.Handle(ctx, clean)
}

func (handler *redactingHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attributes))
	for index, attribute := range attributes {
		clean[index] = cleanAttribute(attribute)
	}
	return &redactingHandler{next: handler.next.WithAttrs(clean)}
}

func (handler *redactingHandler) WithGroup(name string) slog.Handler {
	return &redactingHandler{next: handler.next.WithGroup(name)}
}

func cleanAttribute(attribute slog.Attr) slog.Attr {
	key := strings.ToLower(attribute.Key)
	// Wails debug records include serialized binding input and return values.
	// These fields can contain passwords, cookies, and account data regardless
	// of the bound method's name.
	if key == "args" || key == "result" {
		return slog.String(attribute.Key, redacted)
	}
	for _, fragment := range forbiddenFragments {
		if strings.Contains(key, fragment) {
			return slog.String(attribute.Key, redacted)
		}
	}
	if attribute.Value.Kind() == slog.KindGroup {
		group := attribute.Value.Group()
		for index := range group {
			group[index] = cleanAttribute(group[index])
		}
		return slog.Group(attribute.Key, attrsToAny(group)...)
	}
	return attribute
}

func attrsToAny(attributes []slog.Attr) []any {
	result := make([]any, len(attributes))
	for index := range attributes {
		result[index] = attributes[index]
	}
	return result
}
