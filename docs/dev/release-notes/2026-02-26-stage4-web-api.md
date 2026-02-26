# Release Notes — 2026-02-26 (Stage 4 Web API)

## Scope

Релиз фокусируется на web API для внешнего React интерфейса.

## Highlights

- Введен API-first подход для frontend:
  - UI разрабатывается отдельно от backend.
- Реализована web security модель:
  - bearer auth,
  - CORS allowlist,
  - timeout/body-limit middleware.
- Добавлены endpoint'ы:
  - `GET /v1/me`
  - `GET /v1/modules`
- Стабилизирован контракт API:
  - OpenAPI v1,
  - HTTP contract tests.

## Backward compatibility

- Legacy `X-Subject-ID` поддерживается только в режиме совместимости:
  - `web.auth.allow_legacy_subject_header: true`
- Для production рекомендуется:
  - `web.auth.allow_legacy_subject_header: false`
  - использовать только bearer token.

## Ops

- Пример production-конфига:
  - `configs/config.prod.example.yaml`
- Smoke runbook для React команды:
  - `docs/dev/api/smoke-runbook-react.md`
- Deploy/Rollback runbook для web API:
  - `docs/dev/instr/web-api-deploy-rollback.md`
- SBOM инструкция:
  - `docs/dev/instr/sbom.md`

## Quality status

- `make test` — passed
- `make race` — passed
- `make lint` — passed
- `make sec` — passed
