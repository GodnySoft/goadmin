package web

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"goadmin/internal/core"
	"goadmin/internal/storage"
)

type contextKey string

const (
	ctxRequestID  contextKey = "request_id"
	ctxSubjectID  contextKey = "subject_id"
	ctxRoles      contextKey = "roles"
	ctxAuthMethod contextKey = "auth_method"
	ctxExecuteReq contextKey = "execute_req"
)

// TokenEntry описывает web bearer-токен.
type TokenEntry struct {
	ID          string
	TokenSHA256 string
	Subject     string
	Roles       []string
	Enabled     bool
}

// Config определяет параметры HTTP-транспорта.
type Config struct {
	ListenAddr               string
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	ShutdownTimeout          time.Duration
	RequestTimeout           time.Duration
	MaxRequestBody           int64
	AllowLegacySubjectHeader bool
	Tokens                   []TokenEntry
	CORSAllowedOrigins       []string
	CORSAllowedMethods       []string
	CORSAllowedHeaders       []string
}

// Adapter реализует web transport поверх net/http.
type Adapter struct {
	registry   *core.Registry
	authorizer core.Authorizer
	store      storage.Store
	cfg        Config

tokensByHash map[string]TokenEntry
	corsOrigins map[string]struct{}

	mu     sync.Mutex
	server *http.Server
}

type executeRequest struct {
	Module  string   `json:"module"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// NewAdapter создает web transport.
func NewAdapter(registry *core.Registry, authorizer core.Authorizer, store storage.Store, cfg Config) *Adapter {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 5 * time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 3 * time.Second
	}
	if cfg.MaxRequestBody <= 0 {
		cfg.MaxRequestBody = 1 << 20
	}
	if len(cfg.CORSAllowedMethods) == 0 {
		cfg.CORSAllowedMethods = []string{"GET", "POST", "OPTIONS"}
	}
	if len(cfg.CORSAllowedHeaders) == 0 {
		cfg.CORSAllowedHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	}

	tokensByHash := make(map[string]TokenEntry, len(cfg.Tokens))
	for _, token := range cfg.Tokens {
		h := strings.ToLower(strings.TrimSpace(token.TokenSHA256))
		if len(h) != 64 {
			continue
		}
		tokensByHash[h] = token
	}

	corsOrigins := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, origin := range cfg.CORSAllowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		corsOrigins[trimmed] = struct{}{}
	}

	return &Adapter{
		registry:    registry,
		authorizer:  authorizer,
		store:       store,
		cfg:         cfg,
		tokensByHash: tokensByHash,
		corsOrigins: corsOrigins,
	}
}

func (a *Adapter) Name() string { return "web" }

// Start запускает HTTP server и останавливает его при отмене контекста.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.server != nil {
		a.mu.Unlock()
		return errors.New("web transport already started")
	}
	srv := &http.Server{
		Addr:         a.cfg.ListenAddr,
		Handler:      a.routes(),
		ReadTimeout:  a.cfg.ReadTimeout,
		WriteTimeout: a.cfg.WriteTimeout,
	}
	a.server = srv
	a.mu.Unlock()

	go func() {
		<-ctx.Done()
		stopCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		_ = a.Stop(stopCtx)
	}()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = a.writeAudit(context.Background(), "", "web:serve", "error", map[string]string{"error": err.Error()}, "")
		}
	}()
	return nil
}

// Stop завершает HTTP server.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	srv := a.server
	a.server = nil
	a.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

type middleware func(http.Handler) http.Handler

func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func (a *Adapter) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /v1/health", http.HandlerFunc(a.handleHealth))

	protected := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}), a.timeoutMiddleware(), a.authSubjectMiddleware())

	mux.Handle("GET /v1/", protected)
	mux.Handle("POST /v1/", protected)

	mux.Handle("GET /v1/me", chain(http.HandlerFunc(a.handleMe),
		a.timeoutMiddleware(),
		a.authSubjectMiddleware(),
		a.authorizeActionMiddleware("web:me", core.Action{Module: "web", Command: "me"}),
	))

	mux.Handle("GET /v1/modules", chain(http.HandlerFunc(a.handleModules),
		a.timeoutMiddleware(),
		a.authSubjectMiddleware(),
		a.authorizeActionMiddleware("web:modules", core.Action{Module: "web", Command: "modules"}),
	))

	mux.Handle("POST /v1/commands/execute", chain(http.HandlerFunc(a.handleExecute),
		a.timeoutMiddleware(),
		a.authSubjectMiddleware(),
		a.maxBodyMiddleware(),
		a.authorizeExecuteMiddleware(),
	))

	mux.Handle("GET /v1/metrics/latest", chain(http.HandlerFunc(a.handleLatestMetric),
		a.timeoutMiddleware(),
		a.authSubjectMiddleware(),
		a.authorizeMetricMiddleware(),
	))

	mux.Handle("GET /v1/audit", chain(http.HandlerFunc(a.handleAudit),
		a.timeoutMiddleware(),
		a.authSubjectMiddleware(),
		a.authorizeActionMiddleware("web:audit_query", core.Action{Module: "audit", Command: "read"}),
	))

	return chain(mux, a.requestIDMiddleware(), a.corsMiddleware())
}

func (a *Adapter) requestIDMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := sanitizeRequestID(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = newRequestID()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), ctxRequestID, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *Adapter) corsMiddleware() middleware {
	allowMethods := strings.Join(a.cfg.CORSAllowedMethods, ", ")
	allowHeaders := strings.Join(a.cfg.CORSAllowedHeaders, ", ")

	isMethodAllowed := func(method string) bool {
		for _, m := range a.cfg.CORSAllowedMethods {
			if strings.EqualFold(strings.TrimSpace(m), method) {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := a.corsOrigins[origin]; !ok {
				writeError(w, r, http.StatusForbidden, "cors_denied")
				return
			}

			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", allowMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)

			if r.Method == http.MethodOptions {
				preflightMethod := strings.TrimSpace(r.Header.Get("Access-Control-Request-Method"))
				if preflightMethod != "" && !isMethodAllowed(preflightMethod) {
					writeError(w, r, http.StatusForbidden, "cors_method_denied")
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (a *Adapter) timeoutMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), a.cfg.RequestTimeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *Adapter) authSubjectMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subjectID, roles, authMethod, code := a.resolveSubject(r)
			if code != "" {
				writeError(w, r, http.StatusUnauthorized, code)
				return
			}
			ctx := context.WithValue(r.Context(), ctxSubjectID, subjectID)
			ctx = context.WithValue(ctx, ctxRoles, roles)
			ctx = context.WithValue(ctx, ctxAuthMethod, authMethod)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *Adapter) resolveSubject(r *http.Request) (string, []string, string, string) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		token := strings.TrimSpace(authHeader[7:])
		if token == "" {
			return "", nil, "", "invalid_token"
		}
		sum := sha256.Sum256([]byte(token))
		hash := hex.EncodeToString(sum[:])
		entry, ok := a.tokensByHash[hash]
		if !ok || !entry.Enabled || entry.Subject == "" {
			return "", nil, "", "invalid_token"
		}
		roles := append([]string(nil), entry.Roles...)
		return entry.Subject, roles, "bearer", ""
	}

	if a.cfg.AllowLegacySubjectHeader {
		subjectID := strings.TrimSpace(r.Header.Get("X-Subject-ID"))
		if subjectID != "" {
			return subjectID, nil, "legacy_header", ""
		}
	}

	return "", nil, "", "auth_required"
}

func (a *Adapter) maxBodyMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, a.cfg.MaxRequestBody)
			next.ServeHTTP(w, r)
		})
	}
}

func (a *Adapter) authorizeActionMiddleware(auditAction string, authAction core.Action) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subjectID := subjectIDFromContext(r.Context())
			if subjectID == "" {
				writeError(w, r, http.StatusUnauthorized, "auth_required")
				return
			}
			if err := a.authorizer.Authorize(core.Subject{Source: "web", ID: subjectID}, authAction); err != nil {
				writeError(w, r, http.StatusForbidden, "access_denied")
				_ = a.writeAudit(r.Context(), subjectID, auditAction, "denied", map[string]string{"auth_method": authMethodFromContext(r.Context())}, requestIDFromContext(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *Adapter) authorizeExecuteMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subjectID := subjectIDFromContext(r.Context())
			if subjectID == "" {
				writeError(w, r, http.StatusUnauthorized, "auth_required")
				return
			}

			req, code, statusCode := decodeExecuteRequest(r)
			if code != "" {
				writeError(w, r, statusCode, code)
				_ = a.writeAudit(r.Context(), subjectID, "web:execute", "error", map[string]string{"error_code": code, "auth_method": authMethodFromContext(r.Context())}, requestIDFromContext(r.Context()))
				return
			}

			action := core.Action{Module: req.Module, Command: req.Command}
			if err := a.authorizer.Authorize(core.Subject{Source: "web", ID: subjectID}, action); err != nil {
				writeError(w, r, http.StatusForbidden, "access_denied")
				_ = a.writeAudit(r.Context(), subjectID, "web:execute", "denied", map[string]string{"module": req.Module, "command": req.Command, "auth_method": authMethodFromContext(r.Context())}, requestIDFromContext(r.Context()))
				return
			}

			ctx := context.WithValue(r.Context(), ctxExecuteReq, req)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *Adapter) authorizeMetricMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subjectID := subjectIDFromContext(r.Context())
			module := r.URL.Query().Get("module")
			if module == "" {
				next.ServeHTTP(w, r)
				return
			}
			action := core.Action{Module: module, Command: "read_metrics"}
			if err := a.authorizer.Authorize(core.Subject{Source: "web", ID: subjectID}, action); err != nil {
				writeError(w, r, http.StatusForbidden, "access_denied")
				_ = a.writeAudit(r.Context(), subjectID, "web:metrics_latest", "denied", map[string]string{"module": module, "auth_method": authMethodFromContext(r.Context())}, requestIDFromContext(r.Context()))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func decodeExecuteRequest(r *http.Request) (executeRequest, string, int) {
	var req executeRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		if isBodyTooLargeErr(err) {
			return executeRequest{}, "payload_too_large", http.StatusRequestEntityTooLarge
		}
		return executeRequest{}, "invalid_json", http.StatusBadRequest
	}
	if dec.More() {
		return executeRequest{}, "invalid_json", http.StatusBadRequest
	}
	if req.Module == "" || req.Command == "" {
		return executeRequest{}, "bad_command", http.StatusBadRequest
	}
	return req, "", 0
}

func isBodyTooLargeErr(err error) bool {
	return strings.Contains(err.Error(), "request body too large")
}

func sanitizeRequestID(v string) string {
	id := strings.TrimSpace(v)
	if id == "" || len(id) > 64 {
		return ""
	}
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			continue
		}
		switch ch {
		case '-', '_', '.', ':':
			continue
		default:
			return ""
		}
	}
	return id
}

func (a *Adapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Adapter) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"request_id": requestIDFromContext(r.Context()),
		"subject":    subjectIDFromContext(r.Context()),
		"roles":      rolesFromContext(r.Context()),
		"auth_method": authMethodFromContext(r.Context()),
	})
}

func (a *Adapter) handleModules(w http.ResponseWriter, r *http.Request) {
	providers := a.registry.Providers()
	sort.Strings(providers)
	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"request_id": requestIDFromContext(r.Context()),
		"items":      providers,
	})
}

func (a *Adapter) handleExecute(w http.ResponseWriter, r *http.Request) {
	subjectID := subjectIDFromContext(r.Context())
	requestID := requestIDFromContext(r.Context())
	authMethod := authMethodFromContext(r.Context())

	raw := r.Context().Value(ctxExecuteReq)
	req, ok := raw.(executeRequest)
	if !ok {
		writeError(w, r, http.StatusBadRequest, "bad_command")
		_ = a.writeAudit(r.Context(), subjectID, "web:execute", "error", map[string]string{"error_code": "bad_command", "auth_method": authMethod}, requestID)
		return
	}

	resp, err := a.registry.Execute(r.Context(), req.Module, req.Command, req.Args)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			writeError(w, r, http.StatusGatewayTimeout, "request_timeout")
			_ = a.writeAudit(r.Context(), subjectID, "web:execute", "error", map[string]string{"module": req.Module, "command": req.Command, "error_code": "request_timeout", "auth_method": authMethod}, requestID)
			return
		}
		writeJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"request_id": requestID,
			"status":     resp.Status,
			"error_code": resp.ErrorCode,
		})
		_ = a.writeAudit(r.Context(), subjectID, "web:execute", "error", map[string]string{"module": req.Module, "command": req.Command, "auth_method": authMethod}, requestID)
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"request_id": requestID,
		"status":     resp.Status,
		"data":       resp.Data,
		"error_code": resp.ErrorCode,
	})
	_ = a.writeAudit(r.Context(), subjectID, "web:execute", "ok", map[string]string{"module": req.Module, "command": req.Command, "auth_method": authMethod}, requestID)
}

func (a *Adapter) handleLatestMetric(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFromContext(r.Context())
	subjectID := subjectIDFromContext(r.Context())
	authMethod := authMethodFromContext(r.Context())

	module := r.URL.Query().Get("module")
	if module == "" {
		writeError(w, r, http.StatusBadRequest, "module_required")
		return
	}

	rec, err := a.store.LatestMetric(r.Context(), module)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			writeError(w, r, http.StatusGatewayTimeout, "request_timeout")
			_ = a.writeAudit(r.Context(), subjectID, "web:metrics_latest", "error", map[string]string{"module": module, "error_code": "request_timeout", "auth_method": authMethod}, requestID)
			return
		}
		writeError(w, r, http.StatusNotFound, "metric_not_found")
		_ = a.writeAudit(r.Context(), subjectID, "web:metrics_latest", "error", map[string]string{"module": module, "auth_method": authMethod}, requestID)
		return
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"request_id": requestID,
		"module":     rec.Module,
		"ts":         rec.TS.UTC().Format(time.RFC3339),
		"payload":    json.RawMessage(rec.Payload),
	})
	_ = a.writeAudit(r.Context(), subjectID, "web:metrics_latest", "ok", map[string]string{"module": module, "auth_method": authMethod}, requestID)
}

func (a *Adapter) handleAudit(w http.ResponseWriter, r *http.Request) {
	requestID := requestIDFromContext(r.Context())
	subjectID := subjectIDFromContext(r.Context())
	authMethod := authMethodFromContext(r.Context())

	q := storage.AuditQuery{
		Subject: r.URL.Query().Get("subject"),
		Limit:   parseLimit(r.URL.Query().Get("limit")),
	}
	if from := r.URL.Query().Get("from"); from != "" {
		ts, err := time.Parse(time.RFC3339, from)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "bad_from")
			return
		}
		q.From = ts
	}
	if to := r.URL.Query().Get("to"); to != "" {
		ts, err := time.Parse(time.RFC3339, to)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "bad_to")
			return
		}
		q.To = ts
	}

	events, err := a.store.QueryAudit(r.Context(), q)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
			writeError(w, r, http.StatusGatewayTimeout, "request_timeout")
			_ = a.writeAudit(r.Context(), subjectID, "web:audit_query", "error", map[string]string{"error_code": "request_timeout", "auth_method": authMethod}, requestID)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "query_failed")
		_ = a.writeAudit(r.Context(), subjectID, "web:audit_query", "error", map[string]string{"auth_method": authMethod}, requestID)
		return
	}

	type eventDTO struct {
		Subject   string          `json:"subject"`
		Action    string          `json:"action"`
		Source    string          `json:"source"`
		Status    string          `json:"status"`
		RequestID string          `json:"request_id"`
		Payload   json.RawMessage `json:"payload,omitempty"`
		TS        string          `json:"ts"`
	}
	payload := make([]eventDTO, 0, len(events))
	for _, ev := range events {
		payload = append(payload, eventDTO{
			Subject:   ev.Subject,
			Action:    ev.Action,
			Source:    ev.Source,
			Status:    ev.Status,
			RequestID: ev.RequestID,
			Payload:   json.RawMessage(ev.Payload),
			TS:        ev.TS.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, r, http.StatusOK, map[string]interface{}{
		"request_id": requestID,
		"items":      payload,
	})
	_ = a.writeAudit(r.Context(), subjectID, "web:audit_query", "ok", map[string]string{"items": strconv.Itoa(len(payload)), "auth_method": authMethod}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(ctxRequestID).(string)
	if !ok || v == "" {
		return newRequestID()
	}
	return v
}

func subjectIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxSubjectID).(string)
	return v
}

func rolesFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(ctxRoles).([]string)
	return v
}

func authMethodFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxAuthMethod).(string)
	return v
}

func (a *Adapter) writeAudit(ctx context.Context, subject, action, status string, payload interface{}, requestID string) error {
	var rawPayload []byte
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		rawPayload = data
	}
	return a.store.SaveAudit(ctx, storage.AuditEvent{
		Subject:   subject,
		Action:    action,
		Source:    "web",
		Status:    status,
		RequestID: requestID,
		Payload:   rawPayload,
	})
}

func parseLimit(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 50
	}
	return n
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code string) {
	writeJSON(w, r, statusCode, map[string]string{
		"request_id": requestIDFromContext(r.Context()),
		"error_code": code,
		"message":    errorMessage(code),
	})
}

func errorMessage(code string) string {
	switch code {
	case "auth_required":
		return "authentication is required"
	case "invalid_token":
		return "token is invalid"
	case "access_denied":
		return "access denied"
	case "payload_too_large":
		return "request payload is too large"
	case "request_timeout":
		return "request timeout"
	case "cors_denied", "cors_method_denied":
		return "cors policy denied request"
	default:
		return code
	}
}

func writeJSON(w http.ResponseWriter, r *http.Request, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestIDFromContext(r.Context()))
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}
