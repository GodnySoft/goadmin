package storage

import "context"

// MetricRecord сохраняет метрики модуля.
type MetricRecord struct {
	Module  string
	Payload []byte
}

// AuditEvent фиксирует действия пользователей/транспорта.
type AuditEvent struct {
	Subject   string
	Action    string
	Source    string
	Status    string
	RequestID string
	Payload   []byte
}

// Store описывает операции хранилища.
type Store interface {
	SaveMetric(ctx context.Context, rec MetricRecord) error
	SaveAudit(ctx context.Context, ev AuditEvent) error
	Close() error
}
