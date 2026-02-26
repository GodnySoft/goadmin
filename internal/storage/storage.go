package storage

import (
	"context"
	"time"
)

// MetricRecord сохраняет метрики модуля.
type MetricRecord struct {
	Module  string
	Payload []byte
	TS      time.Time
}

// AuditEvent фиксирует действия пользователей/транспорта.
type AuditEvent struct {
	Subject   string
	Action    string
	Source    string
	Status    string
	RequestID string
	Payload   []byte
	TS        time.Time
}

// AuditQuery задает фильтры выборки аудита.
type AuditQuery struct {
	From    time.Time
	To      time.Time
	Subject string
	Limit   int
}

// Store описывает операции хранилища.
type Store interface {
	SaveMetric(ctx context.Context, rec MetricRecord) error
	SaveAudit(ctx context.Context, ev AuditEvent) error
	LatestMetric(ctx context.Context, module string) (MetricRecord, error)
	QueryAudit(ctx context.Context, q AuditQuery) ([]AuditEvent, error)
	Close() error
}
