# Stage 4 — Web API, Embedded UI, Fleet Operations

## 1. Цель этапа

Добавить HTTP API и встроенный UI (через `go:embed`) для локального контроля агента и базовых fleet-операций, не нарушая изоляцию core.

## Статус реализации (2026-02-26)

- Выполнен стартовый каркас `internal/transports/web` с lifecycle `Start/Stop`.
- Реализованы базовые endpoint'ы v1: `health`, `commands/execute`, `metrics/latest`, `audit`.
- Для API реализован middleware hardening:
  - auth через `X-Subject-ID` в middleware;
  - сквозной `request_id` через `X-Request-ID` (или генерация сервером);
  - общий request-timeout middleware и body-size limit middleware.

## 2. Scope

### In Scope

- HTTP transport на стандартном `net/http` (Go 1.22 router).
- API v1: health, execute command, latest metrics, audit query.
- Встроенный UI: базовый dashboard (status, metrics, recent audit).
- Корреляция `request_id` между API, core и журналами.

### Out of Scope

- Публичный интернет-доступ без reverse proxy и mTLS.
- Полноценный multi-tenant UI.

## 3. Входные условия

- [ ] Stage 3 закрыт.
- [ ] Определен список разрешенных API методов v1.
- [ ] Утверждена политика CORS/CSRF.

## 4. Выходные артефакты

- `internal/transports/web` с роутами v1.
- Контракт OpenAPI (минимум для v1 endpoints).
- Embedded UI assets в бинарнике.
- Документация API и примеры запросов.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S4-01 | Реализовать HTTP server lifecycle | Устойчивый web transport | Stage 3 | 1д |
| S4-02 | Реализовать v1 endpoints | Рабочий API контракт | S4-01 | 1.5д |
| S4-03 | Реализовать auth middleware | Защита API | S4-02 | 1д |
| S4-04 | Встроить UI через `go:embed` | Self-contained бинарник | S4-02 | 1д |
| S4-05 | Добавить OpenAPI и contract tests | Контроль совместимости | S4-02 | 1д |
| S4-06 | Нагрузочные тесты API | Проверка производительности | S4-02 | 0.5д |

## 6. API/Контракты/Схемы

- `GET /v1/health` -> `{"status":"ok"}`.
- `POST /v1/commands/execute` -> принимает `CommandEnvelope`.
- `GET /v1/metrics/latest?module=host`.
- `GET /v1/audit?from=&to=&subject=`.
- Ошибки: унифицированный `error_code`, без утечки внутренних деталей.

## 7. Security Gates

- [ ] AuthN/AuthZ обязателен для всех endpoint, кроме `/health`.
- [ ] Включены request size limit и timeout.
- [ ] CORS и CSRF политика зафиксированы.
- [ ] Валидация JSON schema для execute endpoint.
- [ ] pprof endpoint (если включен) защищен и выключен по умолчанию.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] Все endpoint возвращают ожидаемые коды/контракты.
- [ ] UI отображает статус агента и последние метрики.
- [ ] `request_id` сквозной во всех логах запроса.

### Нефункциональные

- [ ] p95 latency `GET /metrics/latest` < 250ms.
- [ ] Поддержка 100 RPS на тестовом стенде без ошибок > 1%.

### Командные проверки

```bash
go test ./internal/transports/web/...
go test ./... -run TestHTTPContract
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

- [ ] API v1 стабилен и покрыт contract tests.
- [ ] UI встроен в бинарник и работает без внешнего web server.
- [ ] Выполнены performance/security критерии этапа.
