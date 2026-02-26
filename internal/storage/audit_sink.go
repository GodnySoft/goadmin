package storage

import "context"

// AuditWriter позволяет использовать Store как AuditSink.
type AuditWriter interface {
	Write(ctx context.Context, ev AuditEvent) error
}
