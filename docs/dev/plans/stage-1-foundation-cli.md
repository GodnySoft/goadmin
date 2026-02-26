# Stage 1 — Foundation, Core, CLI, CI

## 1. Цель этапа

Собрать минимально жизнеспособный фундамент `goadmin`: каркас модульного ядра, CLI-транспорт, базовый модуль `host`, и CI-конвейер качества/безопасности.

## 2. Scope

### In Scope

- Структура репозитория по Go standard layout.
- `internal/core`: регистрация модулей, диспетчер команд, базовая маршрутизация.
- `internal/transports/cli`: команды `goadmin host status`, `goadmin version`.
- `internal/modules/host`: CPU/RAM/uptime/os-release (read-only).
- `pkg/logger`: structured logging на `slog`.
- CI: lint, test, race, gosec, сборка бинарника.

### Out of Scope

- Демон, scheduler, БД.
- HTTP, Telegram, Max.
- Изменяющие системные операции.

## 3. Входные условия (Entry Criteria)

- [ ] Утверждены манифест и архитектурный документ.
- [ ] Назначен владелец этапа и reviewer по безопасности.
- [ ] Выбран формат конфигурации (`yaml`) и стратегия загрузки.

## 4. Выходные артефакты (Deliverables)

- Код каркаса `cmd`, `internal`, `pkg`.
- Рабочий бинарник `goadmin` с CLI-командами.
- Конфиг-шаблон `configs/config.example.yaml`.
- CI pipeline (`.github/workflows` или `.gitlab-ci.yml`).
- Документация запуска CLI и локальной разработки.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S1-01 | Инициализировать каркас директорий | Базовая структура проекта | Нет | 0.5д |
| S1-02 | Реализовать `CommandProvider` и registry | Модульный core-контракт | S1-01 | 1д |
| S1-03 | Реализовать CLI transport (cobra) | Команды `host status`, `version` | S1-02 | 1д |
| S1-04 | Реализовать `host` module | Метрики узла read-only | S1-02 | 1д |
| S1-05 | Внедрить structured logger | Единый формат логов | S1-01 | 0.5д |
| S1-06 | Настроить CI quality gates | Автоматизированные проверки | S1-01 | 1д |
| S1-07 | Покрыть core unit-тестами | Минимум 70% для core | S1-02 | 1д |

## 6. API/Контракты/Схемы

- `CommandProvider`:
  - `Name() string`
  - `Init(ctx context.Context, cfg *Config) error`
  - `Execute(ctx context.Context, cmd string, args []string) (Response, error)`
- `Response` должен содержать `status`, `data`, `error_code`.
- Все ошибки оборачиваются через `%w` и категоризируются.

## 7. Security Gates

- [ ] CLI-аргументы валидируются до вызова core.
- [ ] Нет прямых shell-вызовов в module/transport.
- [ ] Логи не содержат секретов (token/password redact).
- [ ] `gosec` без high/critical.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] `goadmin version` выводит версию/commit/build date.
- [ ] `goadmin host status` возвращает валидный JSON.
- [ ] Ошибки пользователя возвращают контролируемый код/сообщение.

### Нефункциональные

- [ ] Покрытие `internal/core` >= 70%.
- [ ] Нет data race.
- [ ] Время ответа `host status` p95 < 200ms на тестовом узле.

### Командные проверки

```bash
go mod tidy
golangci-lint run ./...
go test ./...
go test -race ./...
gosec ./...
go build -trimpath -ldflags "-s -w" -o bin/goadmin ./cmd/goadmin
./bin/goadmin host status
```

## 9. Наблюдаемость и эксплуатация

- Лог-формат: JSON, поля `ts`, `level`, `msg`, `request_id`, `module`.
- Для CLI обязательно поле `command` в audit-событии.

## 10. Rollback

- Откат по git-тегу этапа `stage1-rc`.
- Если CI нестабилен > 2 дня, freeze новых фич, только исправления.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Размытый контракт модулей | Средняя | Высокое | Утвердить интерфейсы до реализации |
| Переусложнение CLI на старте | Средняя | Среднее | Оставить только обязательные команды |
| Низкое тест-покрытие | Средняя | Высокое | Gate на покрытие core |

## 12. Exit Criteria

- [ ] Все пункты DoD выполнены.
- [ ] CI green на `main` 3 последовательных прогона.
- [ ] Обновлена документация локального запуска.
