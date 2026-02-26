# Master Checklist — Сквозные условия выполнения и проверки

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

- [ ] Release notes и changelog обновлены.
- [ ] Runbook deploy/rollback актуализирован.
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
