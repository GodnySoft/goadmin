# Stage 4 — Web API + External React UI Integration

## 1. Цель этапа

Стабилизировать HTTP API как backend для внешнего React UI (разрабатывается отдельно), с фокусом на безопасную интеграцию и контрактную совместимость.

## Статус реализации (2026-02-26)

- Выполнен стартовый web transport (`internal/transports/web`) и базовые v1 endpoint'ы.
- Реализован middleware hardening (request_id, timeout, body-size limit).
- Добавлен OpenAPI контракт и contract tests.
- Архитектурный вектор обновлен: UI не встраивается в бинарник, React UI разворачивается отдельно.
- Выполнен базовый load-test API (`/v1/metrics/latest` @ 100 RPS): p95 = `203.57µs`, error rate = `0.0000` (10s, 1000 запросов, 2026-02-26).

## 2. Scope

### In Scope

- HTTP transport на стандартном `net/http` (Go 1.22 router).
- API v1: health, me, modules, execute command, latest metrics, audit query.
- Интеграция с внешним React UI по OpenAPI контракту.
- Корреляция `request_id` между API, core и журналами.

### Out of Scope

- Встраивание frontend-ассетов в бинарник (`go:embed`).
- Публичный интернет-доступ без reverse proxy и mTLS.
- Полноценный multi-tenant UI.

## 3. Входные условия

- [ ] Stage 3 закрыт.
- [ ] Определен список разрешенных API методов v1.
- [x] Утверждена политика CORS allowlist для внешнего React origin.

## 4. Выходные артефакты

- `internal/transports/web` с безопасными middleware и bearer auth.
- OpenAPI контракт v1 (`docs/dev/api/openapi-v1.yaml`).
- Документация интеграции frontend/backend (`docs/dev/api/frontend-integration.md`).
- Contract tests для совместимости API.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S4-01 | Реализовать HTTP server lifecycle | Устойчивый web transport | Stage 3 | 1д |
| S4-02 | Реализовать v1 endpoints | Рабочий API контракт | S4-01 | 1.5д |
| S4-03 | Реализовать auth middleware | Защита API | S4-02 | 1д |
| S4-04 | Зафиксировать API для external React | Совместимый frontend/backend контракт | S4-03 | 1д |
| S4-05 | Добавить/расширить OpenAPI и contract tests | Контроль совместимости | S4-04 | 1д |
| S4-06 | Нагрузочные тесты API | Проверка производительности | S4-05 | 0.5д |
| S4-07 | Внедрить bearer auth + legacy fallback | Безопасная auth-модель для web | S4-04 | 1д |
| S4-08 | Реализовать CORS allowlist policy | Безопасный доступ из React UI | S4-04 | 0.5д |

## 6. API/Контракты/Схемы

- `GET /v1/health` -> `{"status":"ok"}`.
- `GET /v1/me` -> профиль текущего субъекта (subject/roles/auth_method).
- `GET /v1/modules` -> список доступных модулей.
- `POST /v1/commands/execute` -> принимает `ExecuteRequest`.
- `GET /v1/metrics/latest?module=host`.
- `GET /v1/audit?from=&to=&subject=&limit=`.
- Auth: `Authorization: Bearer <token>` (legacy `X-Subject-ID` только для совместимости).
- Ошибки: унифицированный `error_code` + `message`, без утечки внутренних деталей.

## 7. Security Gates

- [x] AuthN/AuthZ обязателен для всех endpoint, кроме `/health`.
- [x] Включены request size limit и timeout.
- [x] CORS policy на explicit allowlist.
- [ ] Валидация JSON schema для execute endpoint.
- [ ] pprof endpoint (если включен) защищен и выключен по умолчанию.

## 8. Проверки (Definition of Done)

### Функциональные

- [x] Все endpoint возвращают ожидаемые коды/контракты.
- [ ] Внешний React UI интегрирован через API v1 без контрактных расхождений.
- [x] `request_id` сквозной во всех логах запроса.

### Нефункциональные

- [x] p95 latency `GET /metrics/latest` < 250ms.
- [x] Поддержка 100 RPS на тестовом стенде без ошибок > 1%.

### Командные проверки

```bash
go test ./internal/transports/web/...
go test ./... -run TestHTTPContract
go test ./internal/transports/web -run TestWebLoadMetricsLatest100RPS -count=1 -v
go test -race ./...
# Пример smoke
curl -sS http://127.0.0.1:8080/v1/health
```

## 9. Наблюдаемость и эксплуатация

- Метрики: RPS, p95 latency, 4xx/5xx ratio.
- Логи: method/path/status/duration/request_id/subject.
- Алерт: 5xx > 3% за 5 минут.

## 10. Rollback

- Отключить web transport флагом `web.enabled=false`.
- Сохранить работу CLI/daemon без HTTP слоя.
- Откат API-контрактов через версионирование `/v1`.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Неполная валидация API input | Средняя | Высокое | JSON schema + contract tests |
| Деградация latency из-за БД | Средняя | Среднее | Индексы, кеш последних метрик |
| Утечка диагностических endpoint | Низкая | Высокое | Disable-by-default + auth |

## 12. Exit Criteria

- [x] API v1 стабилен и покрыт contract tests.
- [x] Модель external React UI зафиксирована в документации и OpenAPI.
- [ ] Выполнены performance/security критерии этапа.
