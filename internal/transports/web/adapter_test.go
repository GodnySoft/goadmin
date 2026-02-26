package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goadmin/internal/core"
	"goadmin/internal/storage"
)

type fakeProvider struct {
	block bool
}

func (p *fakeProvider) Name() string { return "host" }
func (p *fakeProvider) Init(ctx context.Context) error {
	return nil
}
func (p *fakeProvider) Execute(ctx context.Context, cmd string, args []string) (core.Response, error) {
	if p.block {
		<-ctx.Done()
		return core.Response{Status: "error", ErrorCode: "timeout"}, ctx.Err()
	}
	if cmd != "status" {
		return core.Response{Status: "error", ErrorCode: "unknown_command"}, nil
	}
	return core.Response{Status: "ok", Data: map[string]string{"node": "n1"}}, nil
}

type fakeStore struct {
	latest storage.MetricRecord
	audit  []storage.AuditEvent
}

func (s *fakeStore) SaveMetric(ctx context.Context, rec storage.MetricRecord) error {
	s.latest = rec
	return nil
}
func (s *fakeStore) SaveAudit(ctx context.Context, ev storage.AuditEvent) error {
	s.audit = append(s.audit, ev)
	return nil
}
func (s *fakeStore) LatestMetric(ctx context.Context, module string) (storage.MetricRecord, error) {
	return s.latest, nil
}
func (s *fakeStore) QueryAudit(ctx context.Context, q storage.AuditQuery) ([]storage.AuditEvent, error) {
	return s.audit, nil
}
func (s *fakeStore) Close() error { return nil }

func TestHealthEndpoint(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestProtectedEndpointRequiresSubject(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})
	req := httptest.NewRequest(http.MethodGet, "/v1/audit", nil)
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	assertErrorHasRequestID(t, rr)
}

func TestExecuteEndpointAuthorized(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	req.Header.Set("X-Request-ID", "abc-123")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("expected request id header abc-123, got %q", got)
	}

	var resp struct {
		RequestID string `json:"request_id"`
		Status    string `json:"status"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %q", resp.Status)
	}
	if resp.RequestID != "abc-123" {
		t.Fatalf("expected request id in body abc-123, got %q", resp.RequestID)
	}
}

func TestInvalidRequestIDGetsReplaced(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	req.Header.Set("X-Request-ID", "bad id with spaces")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("X-Request-ID"); got == "" || got == "bad id with spaces" {
		t.Fatalf("expected sanitized generated request id, got %q", got)
	}
}

func TestExecuteEndpointBodyTooLarge(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{MaxRequestBody: 16})

	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", strings.NewReader(`{"module":"host","command":"status","args":["a","b","c"]}`))
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rr.Code)
	}
}

func TestExecuteEndpointTimeout(t *testing.T) {
	adapter := newTestAdapter(t, true, Config{RequestTimeout: 20 * time.Millisecond})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status 504, got %d", rr.Code)
	}
}

func TestLatestMetricEndpoint(t *testing.T) {
	store := &fakeStore{
		latest: storage.MetricRecord{
			Module:  "host",
			Payload: []byte(`{"cpu":10}`),
			TS:      time.Now().UTC(),
		},
	}
	adapter := newAdapterWithStore(t, store, false, Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/metrics/latest?module=host", nil)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func assertErrorHasRequestID(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	var resp struct {
		RequestID string `json:"request_id"`
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RequestID == "" {
		t.Fatal("expected request_id in error response")
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header in error response")
	}
}

func newTestAdapter(t *testing.T, block bool, cfg Config) *Adapter {
	t.Helper()
	return newAdapterWithStore(t, &fakeStore{
		latest: storage.MetricRecord{Module: "host", Payload: []byte(`{"cpu":1}`), TS: time.Now().UTC()},
	}, block, cfg)
}

func newAdapterWithStore(t *testing.T, store storage.Store, block bool, cfg Config) *Adapter {
	t.Helper()
	registry := core.NewRegistry()
	if err := registry.Register(context.Background(), &fakeProvider{block: block}); err != nil {
		t.Fatalf("register fake provider: %v", err)
	}
	authz := core.NewAllowlistAuthorizer(map[string][]string{"web": {"u1"}})
	return NewAdapter(registry, authz, store, cfg)
}
