# Stage 3 — Messaging Transports (Telegram/Max) + AuthZ

## 1. Цель этапа

Подключить удаленные transport-каналы (Telegram, MaxBot) без компромиссов по безопасности: строгая авторизация, аудит, и детерминированная маршрутизация команд в core.

## 2. Scope

### In Scope

- `internal/transports/telegram` (long polling).
- `internal/transports/maxbot` (по API контракта мессенджера).
- Единый middleware: authn/authz, rate-limit, command validation.
- Маппинг chat-команд в `CommandEnvelope`.

### Out of Scope

- Автоматическое выполнение опасных recovery-команд.
- Сложный RBAC с внешним IAM.

## 3. Входные условия

- [ ] Stage 2 закрыт.
- [ ] Подготовлен allowlist пользователей/чатов.
- [ ] Секреты токенов передаются только через защищенный механизм.

## 4. Выходные артефакты

- Реализация telegram/max transport adapters.
- Модуль авторизации по whitelist ID/role.
- Таблица аудита удаленных команд и ответов.
- Документация incident-процедуры по компрометации bot token.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S3-01 | Реализовать transport adapter abstraction | Единый старт/стоп контракт | Stage 2 | 0.5д |
| S3-02 | Реализовать Telegram adapter | Прием и обработка команд | S3-01 | 1.5д |
| S3-03 | Реализовать Max adapter | Прием и обработка команд | S3-01 | 1.5д |
| S3-04 | AuthN/AuthZ middleware | deny-by-default | S3-02,S3-03 | 1д |
| S3-05 | Rate-limit и anti-flood | Защита от abuse | S3-04 | 0.5д |
| S3-06 | E2E тесты transport->core | Проверка маршрутизации | S3-02..S3-05 | 1д |

## 6. API/Контракты/Схемы

- `Subject` контракт: `id`, `source`, `roles[]`, `is_allowed`.
- `Authorize(subject, action, module)` возвращает deny reason.
- Стандарт ответа в чат: `request_id`, `status`, короткая ошибка без внутренних деталей.

## 7. Security Gates

- [ ] Все входные сообщения проходят strict parser (без shell-like интерпретации).
- [ ] Запросы не из allowlist блокируются и логируются.
- [ ] Включен per-subject rate limit.
- [ ] В аудит пишутся и успешные, и отклоненные попытки.
- [ ] Секреты токенов не логируются.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] Авторизованный пользователь выполняет `host status` через чат.
- [ ] Неавторизованный пользователь получает отказ.
- [ ] Ошибки transport не ломают core loop.

### Нефункциональные

- [ ] p95 обработка чат-команды < 700ms (без тяжелых модулей).
- [ ] Восстановление polling после transient network error.

### Командные проверки

```bash
go test ./internal/transports/telegram/...
go test ./internal/transports/maxbot/...
go test ./internal/core/... -run TestAuthorization
go test -race ./...
gosec ./...
```

## 9. Наблюдаемость и эксплуатация

- Метрики: accepted/denied commands, rate-limit hits, polling reconnect count.
- Логи: `source`, `subject_id`, `action`, `decision`, `request_id`.

## 10. Rollback

- Feature-flag отключения каждого transport отдельно.
- При инциденте токена: rotate token, disable adapter, revoke sessions.
- Откат на режим только CLI/HTTP.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Командные инъекции через текст | Средняя | Высокое | Strict parser + command allowlist |
| Flood/DDoS на бота | Средняя | Среднее | Rate limit + backoff + circuit breaker |
| Утечка токена | Низкая | Высокое | Rotate, secret store, аудит доступа |

## 12. Exit Criteria

- [ ] Все transport security gates закрыты.
- [ ] Пройден E2E сценарий авторизованного и неавторизованного доступа.
- [ ] Подготовлен playbook response на инцидент bot-token.
