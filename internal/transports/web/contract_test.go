package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPContractHealth(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID header")
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", body["status"])
	}
}

func TestHTTPContractExecuteSuccess(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	req.Header.Set("X-Subject-ID", "u1")
	req.Header.Set("X-Request-ID", "contract-1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["request_id"] != "contract-1" {
		t.Fatalf("request_id = %v, want contract-1", resp["request_id"])
	}
	if _, ok := resp["status"]; !ok {
		t.Fatal("missing status")
	}
}

func TestHTTPContractExecuteUnauthorized(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	body := bytes.NewBufferString(`{"module":"host","command":"status","args":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/commands/execute", body)
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["request_id"]; !ok {
		t.Fatal("missing request_id")
	}
	if resp["error_code"] != "subject_required" {
		t.Fatalf("error_code = %v, want subject_required", resp["error_code"])
	}
}

func TestHTTPContractMetricsLatest(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/metrics/latest?module=host", nil)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["request_id"]; !ok {
		t.Fatal("missing request_id")
	}
	if _, ok := resp["payload"]; !ok {
		t.Fatal("missing payload")
	}
}

func TestHTTPContractAudit(t *testing.T) {
	adapter := newTestAdapter(t, false, Config{})

	req := httptest.NewRequest(http.MethodGet, "/v1/audit", nil)
	req.Header.Set("X-Subject-ID", "u1")
	rr := httptest.NewRecorder()
	adapter.routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["request_id"]; !ok {
		t.Fatal("missing request_id")
	}
	if _, ok := resp["items"]; !ok {
		t.Fatal("missing items")
	}
}
