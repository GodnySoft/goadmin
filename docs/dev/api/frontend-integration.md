# Frontend Integration (React External)

## Контекст

Frontend (`React`) разрабатывается и разворачивается отдельно от `goadmin`.
`goadmin` предоставляет только API (`/v1`) и не встраивает UI-ассеты в бинарник.

## Auth

- Основной режим: `Authorization: Bearer <token>`.
- Legacy fallback (временный): `X-Subject-ID` при `web.auth.allow_legacy_subject_header=true`.
- Для production рекомендуется отключить legacy-header и использовать только bearer.
- Хэш токена хранится в конфиге (`token_sha256`), а не сам токен.

Пример генерации SHA-256 токена:

```bash
TOKEN='replace-me'
printf '%s' "$TOKEN" | sha256sum | awk '{print $1}'
```

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

## Примеры запросов

```bash
curl -sS \
  -H "Authorization: Bearer $GOADMIN_TOKEN" \
  -H "X-Request-ID: ui-req-001" \
  http://127.0.0.1:8080/v1/me
```

```bash
curl -sS \
  -H "Authorization: Bearer $GOADMIN_TOKEN" \
  "http://127.0.0.1:8080/v1/metrics/latest?module=host"
```

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $GOADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"module":"host","command":"status","args":[]}' \
  http://127.0.0.1:8080/v1/commands/execute
```

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

## Smoke-проверка

Пошаговый smoke-checklist для React команды:

- `docs/dev/api/smoke-runbook-react.md`
