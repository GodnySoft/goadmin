package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // sqlite driver

	"goadmin/internal/storage"
)

// Store реализует storage.Store поверх SQLite.
type Store struct {
	db *sql.DB
}

// Open инициализирует соединение и выполняет миграции.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_journal=WAL&_busy_timeout=5000", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			module TEXT NOT NULL,
			payload BLOB NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_ts ON metrics(ts);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_module_ts ON metrics(module, ts);`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			subject TEXT,
			action TEXT,
			source TEXT,
			status TEXT,
			request_id TEXT,
			payload BLOB
		);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_events(ts);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_subject_ts ON audit_events(subject, ts);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// SaveMetric сохраняет метрику.
func (s *Store) SaveMetric(ctx context.Context, rec storage.MetricRecord) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO metrics(module, payload, ts) VALUES(?,?,?)`, rec.Module, rec.Payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}
	return nil
}

// SaveAudit сохраняет аудиторное событие.
func (s *Store) SaveAudit(ctx context.Context, ev storage.AuditEvent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_events(subject, action, source, status, request_id, payload, ts) VALUES(?,?,?,?,?,?,?)`,
		ev.Subject, ev.Action, ev.Source, ev.Status, ev.RequestID, ev.Payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

// Write реализует общий AuditSink интерфейс.
func (s *Store) Write(ctx context.Context, ev storage.AuditEvent) error {
	return s.SaveAudit(ctx, ev)
}

// Close закрывает соединение.
func (s *Store) Close() error {
	return s.db.Close()
}

// MarshalPayload упрощает сериализацию данных метрик.
func MarshalPayload(data interface{}) ([]byte, error) {
	buf, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return buf, nil
}
