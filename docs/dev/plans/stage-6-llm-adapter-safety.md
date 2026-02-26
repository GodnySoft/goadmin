# Stage 6 — LLM Adapter, Guardrails, Safe Assist

## 1. Цель этапа

Подключить LLM-слой в архитектуру `goadmin` без нарушения принципов безопасности: только как ассистивный контур с обязательным подтверждением действий человеком.

## 2. Scope

### In Scope

- `LLMProvider` интерфейс + adapters (`local`, `cloud`).
- PolicyGuard: redaction входа, policy-проверка промптов, validation ответа.
- Fallback стратегия `local -> cloud` по timeout/ошибкам.
- Режимы работы: diagnose/explain/suggest-command.

### Out of Scope

- Автономный auto-exec команд без подтверждения.
- Передача в LLM сырых секретов или персональных данных.

## 3. Входные условия

- [ ] Stage 5 закрыт.
- [ ] Утверждена политика обработки данных для LLM.
- [ ] Определен список разрешенных LLM use-cases v1.

## 4. Выходные артефакты

- `internal/llm` с provider adapters и policy guard.
- Конфигурация `llm.provider_order`, `llm.timeout_ms`, `llm.redaction`.
- Набор тест-кейсов на prompt injection и data leakage.
- Операционный runbook по деградации/отключению LLM.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S6-01 | Спроектировать `LLMProvider` интерфейс | Независимость от вендора | Stage 5 | 1д |
| S6-02 | Реализовать local adapter | Локальная модель как primary | S6-01 | 1д |
| S6-03 | Реализовать cloud adapter | Fallback при ошибках local | S6-01 | 1д |
| S6-04 | Реализовать PolicyGuard | Контроль данных и ответов | S6-02,S6-03 | 1.5д |
| S6-05 | Интеграция в CLI/Web как assistant mode | Безопасные рекомендации | S6-04 | 1д |
| S6-06 | Security и red-team тесты | Проверка на prompt-инъекции | S6-05 | 1д |

## 6. API/Контракты/Схемы

- `LLMProvider`:
  - `Name() string`
  - `Complete(ctx context.Context, req PromptRequest) (PromptResponse, error)`
  - `Health(ctx context.Context) error`
- `PromptRequest` содержит только санитизированные данные.
- `PromptResponse` проходит output policy validation.
- Для suggest-command формат ответа строго JSON schema.

## 7. Security Gates

- [ ] Redaction обязательна до любого вызова LLM.
- [ ] Запрещен доступ LLM к raw secret values.
- [ ] Prompt injection filter активен и покрыт тестами.
- [ ] Любая предлагаемая команда имеет `requires_confirmation=true`.
- [ ] LLM можно отключить runtime-флагом без рестарта.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] Ассистент выдает объяснение инцидента по метрикам/логам.
- [ ] Ассистент предлагает команду в structured формате.
- [ ] Команда не выполняется автоматически без явного подтверждения.

### Нефункциональные

- [ ] Timeout LLM-запроса соблюдается и не блокирует core.
- [ ] При недоступности local корректно включается cloud fallback.
- [ ] Логи не содержат содержимого чувствительных prompt фрагментов.

### Командные проверки

```bash
go test ./internal/llm/...
go test ./... -run TestPromptInjectionDefense
go test ./... -run TestLLMRedaction
go test -race ./...
gosec ./...
```

## 9. Наблюдаемость и эксплуатация

- Метрики: latency/timeout/error rate по provider.
- Метрики policy: blocked_prompt_count, redacted_fields_count.
- Алерты: резкий рост blocked_prompt_count или provider error rate.

## 10. Rollback

- Runtime переключение `llm.enabled=false`.
- Удаление cloud fallback из цепочки `provider_order`.
- Возврат к режиму только deterministic diagnostics.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Prompt injection обход фильтра | Средняя | Высокое | Многоуровневый guard + red-team тесты |
| Утечка данных в cloud provider | Низкая | Высокое | Жесткий redaction + data policy + audit |
| Нестабильность local модели | Средняя | Среднее | Fallback + timeout + circuit breaker |

## 12. Exit Criteria

- [ ] Все LLM security gates закрыты.
- [ ] Пройдены сценарии prompt-injection/data-leakage.
- [ ] Ассистент работает только в режиме safe assist (no auto-exec).
