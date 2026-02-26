# Changelog

## 2026-02-26

### Added
- Web API v1 contract (`docs/dev/api/openapi-v1.yaml`).
- Contract tests for web API (`TestHTTPContract*`).
- New API endpoints: `GET /v1/me`, `GET /v1/modules`.
- React integration docs:
  - `docs/dev/api/frontend-integration.md`
  - `docs/dev/api/smoke-runbook-react.md`
- Production config example for web integration: `configs/config.prod.example.yaml`.
- Load benchmark test for web API: `internal/transports/web/load_test.go`.

### Changed
- Stage 4 re-baselined to external React UI integration (no embedded UI in current scope).
- Web auth model switched to bearer-first:
  - `Authorization: Bearer <token>` as primary method.
  - `X-Subject-ID` only as temporary compatibility mode.
- Added CORS allowlist policy for web transport.
- Added unified error `message` field along with `error_code`.

### Security
- Enforced auth for protected web endpoints.
- Added request timeout and body size limit middleware.
- Added request correlation via `X-Request-ID`/`request_id`.

### Performance
- Baseline load test (`/v1/metrics/latest` @ 100 RPS):
  - p95: `203.57µs`
  - error rate: `0.0000`
