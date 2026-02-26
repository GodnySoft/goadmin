# Master Checklist — Сквозные условия выполнения и проверки

## 0. Текущий статус работ (обновлено: 2026-02-26)

- [x] Подготовлен Makefile с целями для локальной разработки и проверок качества.
- [x] Подготовлена инструкция развёртывания окружения для разработки через Ansible.
- [x] Реализован core-пайплайн command -> authz -> ratelimit -> module execution.
- [x] Добавлены transport-адаптеры Telegram/MaxBot (скелеты) с общим пайплайном.
- [x] Реализовано хранение метрик и аудита в SQLite.
- [x] Добавлен аудит событий в messaging-пайплайн (status: ok/error/denied/rate_limited).
- [x] Добавлены базовые интеграционные и unit-тесты для transport-пайплайна.
- [x] Начата реализация Stage 4: добавлен базовый Web transport (`/v1/health`, execute, latest metrics, audit).
- [x] Усилен Web transport middleware-слой (auth/request_id/timeout/body limit).
- [x] Добавлен OpenAPI v1 и базовые HTTP contract tests.
- [x] Зафиксирована модель external React UI (без `go:embed`) и документ интеграции frontend/backend.
- [x] Для web API добавлены bearer auth, CORS allowlist и endpoint'ы `/v1/me`, `/v1/modules`.

## 1. Глобальные quality gates (каждый этап)

- [ ] `golangci-lint run ./...` без ошибок.
- [ ] `go test ./...` успешно.
- [ ] `go test -race ./...` успешно.
- [ ] `gosec ./...` без critical/high.
- [ ] SBOM сформирован и приложен к артефакту релиза.

## 2. Архитектурные инварианты

- [ ] Core не зависит от transport реализаций.
- [ ] Transport не выполняет бизнес-логику напрямую.
- [ ] Любой вызов системных команд идет только через безопасный wrapper.
- [ ] Все длительные операции управляются `context.Context`.

## 3. Security baseline

- [ ] Deny-by-default для внешних команд.
- [ ] Allowlist субъектов для удаленных transport.
- [ ] Redaction чувствительных данных в логах и внешних интеграциях.
- [ ] Секреты не коммитятся в репозиторий.

## 4. Reliability baseline

- [ ] Graceful shutdown реализован и протестирован.
- [ ] Нет утечек горутин в soak-тестах.
- [ ] Обработаны transient ошибки сети/диска с retry/backoff.

## 5. Release readiness

- [x] Release notes и changelog обновлены.
- [x] Runbook deploy/rollback актуализирован.
- [ ] Проведен smoke-test после деплоя.
- [ ] Подписан и верифицирован бинарный артефакт.

## 6. Контрольные команды перед релизом

```bash
go mod tidy
golangci-lint run ./...
go test ./...
go test -race ./...
gosec ./...
go build -trimpath -ldflags "-s -w" -o bin/goadmin ./cmd/goadmin
```
