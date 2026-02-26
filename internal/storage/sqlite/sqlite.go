package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	ts := rec.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO metrics(module, payload, ts) VALUES(?,?,?)`, rec.Module, rec.Payload, ts)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}
	return nil
}

// SaveAudit сохраняет аудиторное событие.
func (s *Store) SaveAudit(ctx context.Context, ev storage.AuditEvent) error {
	ts := ev.TS
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_events(subject, action, source, status, request_id, payload, ts) VALUES(?,?,?,?,?,?,?)`,
		ev.Subject, ev.Action, ev.Source, ev.Status, ev.RequestID, ev.Payload, ts)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}

// LatestMetric возвращает последнюю метрику по модулю.
func (s *Store) LatestMetric(ctx context.Context, module string) (storage.MetricRecord, error) {
	row := s.db.QueryRowContext(ctx, `SELECT module, payload, ts FROM metrics WHERE module = ? ORDER BY ts DESC LIMIT 1`, module)
	var rec storage.MetricRecord
	var ts string
	if err := row.Scan(&rec.Module, &rec.Payload, &ts); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.MetricRecord{}, fmt.Errorf("latest metric not found: %w", err)
		}
		return storage.MetricRecord{}, fmt.Errorf("query latest metric: %w", err)
	}
	parsedTS, err := parseSQLiteTS(ts)
	if err != nil {
		return storage.MetricRecord{}, fmt.Errorf("parse metric timestamp: %w", err)
	}
	rec.TS = parsedTS
	return rec, nil
}

// QueryAudit возвращает аудит по фильтрам.
func (s *Store) QueryAudit(ctx context.Context, q storage.AuditQuery) ([]storage.AuditEvent, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	from := q.From
	if from.IsZero() {
		from = time.Unix(0, 0).UTC()
	}
	to := q.To
	if to.IsZero() {
		to = time.Now().UTC()
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT subject, action, source, status, request_id, payload, ts
FROM audit_events
WHERE ts >= ? AND ts <= ? AND (? = '' OR subject = ?)
ORDER BY ts DESC
LIMIT ?`, from, to, q.Subject, q.Subject, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit: %w", err)
	}
	defer rows.Close()

	events := make([]storage.AuditEvent, 0, limit)
	for rows.Next() {
		var ev storage.AuditEvent
		var ts string
		if err := rows.Scan(&ev.Subject, &ev.Action, &ev.Source, &ev.Status, &ev.RequestID, &ev.Payload, &ts); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		parsedTS, err := parseSQLiteTS(ts)
		if err != nil {
			return nil, fmt.Errorf("parse audit timestamp: %w", err)
		}
		ev.TS = parsedTS
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit: %w", err)
	}
	return events, nil
}

func parseSQLiteTS(v string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, v); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported sqlite time format: %q", v)
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
