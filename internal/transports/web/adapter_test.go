package web

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	mu     sync.Mutex
	latest storage.MetricRecord
	audit  []storage.AuditEvent
}

func (s *fakeStore) SaveMetric(ctx context.Context, rec storage.MetricRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = rec
	return nil
}
func (s *fakeStore) SaveAudit(ctx context.Context, ev storage.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, ev)
	return nil
}
func (s *fakeStore) LatestMetric(ctx context.Context, module string) (storage.MetricRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest, nil
}
func (s *fakeStore) QueryAudit(ctx context.Context, q storage.AuditQuery) ([]storage.AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func TestProtectedEndpointRequiresAuth(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})
	req := httptest.NewRequest(http.MethodGet, "/v1/audit", nil)
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
	assertErrorHasRequestID(t, rr)
}

func TestExecuteEndpointAuthorizedBearer(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("Authorization", "Bearer test-token")
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

func TestInvalidTokenDenied(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("Authorization", "Bearer bad-token")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestLegacySubjectHeaderWhenEnabled(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{AllowLegacySubjectHeader: true})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestLegacySubjectHeaderWhenDisabled(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{AllowLegacySubjectHeader: false})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestInvalidRequestIDGetsReplaced(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("Authorization", "Bearer test-token")
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
	req.Header.Set("Authorization", "Bearer test-token")
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
	req.Header.Set("Authorization", "Bearer test-token")
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
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestMeEndpointContainsSubjectRolesAndAuthMethod(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Subject    string   `json:"subject"`
		Roles      []string `json:"roles"`
		AuthMethod string   `json:"auth_method"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Subject != "u1" {
		t.Fatalf("expected subject u1, got %q", resp.Subject)
	}
	if resp.AuthMethod != "bearer" {
		t.Fatalf("expected auth_method bearer, got %q", resp.AuthMethod)
	}
	if len(resp.Roles) == 0 || resp.Roles[0] != "admin" {
		t.Fatalf("expected roles to include admin, got %#v", resp.Roles)
	}
}

func TestAuthModeLegacyHeaderForcesHeaderFlow(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{
		AuthMode:                 "legacy_header",
		AllowLegacySubjectHeader: false,
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 when legacy mode ignores bearer, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req2.Header.Set("X-Subject-ID", "u1")
	rr2 := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status 200 with legacy header, got %d", rr2.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{CORSAllowedOrigins: []string{"https://ui.local"}})
	req := httptest.NewRequest(http.MethodOptions, "/v1/commands/execute", nil)
	req.Header.Set("Origin", "https://ui.local")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://ui.local" {
		t.Fatalf("unexpected allow-origin: %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSDeniedOrigin(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{CORSAllowedOrigins: []string{"https://ui.local"}})
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set("Origin", "https://evil.local")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
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
	authz := core.NewAllowlistAuthorizer(map[string][]string{"web": {"u1", "ui-admin"}})
	if len(cfg.Tokens) == 0 {
		cfg.Tokens = []TokenEntry{{
			ID:          "t1",
			TokenSHA256: tokenSHA256("test-token"),
			Subject:     "u1",
			Roles:       []string{"admin"},
			Enabled:     true,
		}}
	}
	return NewAdapter(registry, authz, store, cfg)
}

func tokenSHA256(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
