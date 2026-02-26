# Stage 2 — Daemon Mode, Scheduler, SQLite

## 1. Цель этапа

Добавить устойчивый режим демона `serve`, фоновый scheduler и хранилище SQLite для истории метрик и аудита.

## 2. Scope

### In Scope

- Команда `goadmin serve`.
- Graceful shutdown (SIGINT/SIGTERM).
- Scheduler для периодического опроса модулей.
- SQLite: схема, миграции, retention policy.
- Репозиторий доступа к данным (`storage` слой).

### Out of Scope

- Внешние транспорты (Telegram/Max).
- Полноценный web dashboard.

## 3. Входные условия

- [ ] Stage 1 закрыт.
- [ ] Утвержден выбор SQLite-драйвера (cgo/pure-go) ADR.
- [ ] Утверждена политика retention (например, 30/90 дней).

## 4. Выходные артефакты

- Команда `serve` и жизненный цикл процесса.
- Миграции БД и таблицы `metrics`, `audit_events`, `jobs`.
- Scheduler с джиттером и bounded workers.
- Runbook по восстановлению sqlite-файла.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S2-01 | Реализовать lifecycle manager | Управление старт/стоп сервисов | Stage 1 | 1д |
| S2-02 | Команда `serve` + signal handling | Корректный graceful shutdown | S2-01 | 1д |
| S2-03 | Реализовать scheduler | Периодический сбор метрик | S2-01 | 1д |
| S2-04 | Добавить SQLite storage + migrations | История телеметрии | S2-03 | 1.5д |
| S2-05 | Retention job | Контроль размера БД | S2-04 | 0.5д |
| S2-06 | Интеграционные тесты daemon + db | Проверка устойчивости | S2-02,S2-04 | 1д |

## 6. API/Контракты/Схемы

- `Storage` интерфейсы:
  - `SaveMetric(ctx, MetricRecord) error`
  - `SaveAuditEvent(ctx, AuditEvent) error`
  - `QueryMetrics(ctx, Query) ([]MetricRecord, error)`
- Схема БД:
  - `metrics(id, ts, module, key, value, tags_json)`
  - `audit_events(id, ts, subject, action, source, status, request_id, payload_json)`
  - индексы: `(ts)`, `(module, ts)`, `(subject, ts)`

## 7. Security Gates

- [ ] Путь к sqlite-файлу валидирован и не указывает на world-writable location.
- [ ] SQL запросы только параметризованные.
- [ ] Любой транспортный ввод в БД проходит schema validation.
- [ ] Логи не содержат raw payload чувствительных команд.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] `goadmin serve` запускается и пишет heartbeat в лог.
- [ ] По SIGTERM завершаются все worker-горутины < 10 сек.
- [ ] Метрики сохраняются в SQLite по расписанию.

### Нефункциональные

- [ ] Нет утечек горутин в soak-тесте 30 минут.
- [ ] Рост sqlite-файла контролируется retention job.
- [ ] p95 запись метрики < 50ms.

### Командные проверки

```bash
go test ./internal/core/...
go test ./internal/storage/...
go test -race ./...
go test -run TestServeGracefulShutdown ./...
go test -run TestSchedulerPersistsMetrics ./...
```

## 9. Наблюдаемость и эксплуатация

- Метрики: размер очереди scheduler, длительность job, ошибки записи БД.
- Логи: старт/стоп сервиса, причина остановки, число активных worker.

## 10. Rollback

- При деградации IO/latency: отключить scheduler флагом конфига.
- Откат миграций через `down` скрипт до предыдущей схемы.
- Вернуться на тег `stage1-stable` при критической нестабильности.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Блокировки SQLite под нагрузкой | Средняя | Высокое | WAL mode, batch writes, индексы |
| Утечки горутин scheduler | Средняя | Высокое | Контроль context + тест на goroutine diff |
| Рост БД без лимита | Высокая | Среднее | Retention + аварийный purge job |

## 12. Exit Criteria

- [ ] Все пункты DoD выполнены.
- [ ] Стабильный soak-тест 30 минут без деградации.
- [ ] Runbook backup/restore SQLite опубликован.
