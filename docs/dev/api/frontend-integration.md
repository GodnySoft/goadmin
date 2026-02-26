# Frontend Integration (React External)

## Контекст

Frontend (`React`) разрабатывается и разворачивается отдельно от `goadmin`.
`goadmin` предоставляет только API (`/v1`) и не встраивает UI-ассеты в бинарник.

## Auth

- Основной режим: `Authorization: Bearer <token>`.
- Legacy fallback (временный): `X-Subject-ID` при `web.auth.allow_legacy_subject_header=true`.
- Для production рекомендуется отключить legacy-header и использовать только bearer.

## CORS

- Политика: explicit allowlist (`web.cors.allowed_origins`).
- Разрешенные методы: `web.cors.allowed_methods` (по умолчанию `GET,POST,OPTIONS`).
- Разрешенные заголовки: `web.cors.allowed_headers` (по умолчанию `Authorization,Content-Type,X-Request-ID`).

## Корреляция запросов

- Клиент может передать `X-Request-ID`.
- Если `X-Request-ID` отсутствует или невалиден — сервер генерирует новый.
- В каждом ответе сервер выставляет `X-Request-ID` и поле `request_id` в JSON.

## Endpoint минимум для UI

- `GET /v1/health` — healthcheck.
- `GET /v1/me` — текущий субъект, роли, метод auth.
- `GET /v1/modules` — список доступных модулей.
- `GET /v1/metrics/latest?module=host` — последние метрики.
- `GET /v1/audit?...` — аудит.
- `POST /v1/commands/execute` — исполнение команды.

## Ошибки

Базовый формат:

```json
{
  "request_id": "9f7e8d1f...",
  "error_code": "access_denied",
  "message": "access denied"
}
```

Рекомендуемая обработка на frontend:

- `401` — невалидный/отсутствующий токен.
- `403` — токен валиден, но действие запрещено.
- `413` — payload слишком большой.
- `504` — timeout операции.
