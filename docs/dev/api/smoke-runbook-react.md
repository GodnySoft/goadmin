# Smoke Runbook: React UI ↔ goadmin API

Краткий чеклист для команды React перед началом интеграции и после каждого релизного изменения backend.

## 1. Предусловия

- `goadmin` запущен с web transport (`web.enabled: true`).
- В конфиге выставлены корректные:
  - `web.auth.mode: bearer`
  - `web.auth.allow_legacy_subject_header: false` (для production smoke)
  - `web.cors.allowed_origins` включает origin UI.
- Есть рабочий `GOADMIN_TOKEN` (исходный токен) и `token_sha256` в конфиге.

## 2. Базовые проверки доступности

```bash
curl -sS http://127.0.0.1:8080/v1/health
```

Ожидание:
- HTTP `200`
- JSON: `{"status":"ok"}`
- Заголовок `X-Request-ID` присутствует.

## 3. Проверка auth (Bearer)

```bash
curl -i -sS http://127.0.0.1:8080/v1/me
```

Ожидание:
- HTTP `401`
- `error_code=auth_required`

```bash
curl -i -sS \
  -H "Authorization: Bearer ${GOADMIN_TOKEN}" \
  http://127.0.0.1:8080/v1/me
```

Ожидание:
- HTTP `200`
- Поля `subject`, `roles`, `auth_method` (обычно `bearer`).

## 4. Проверка CORS (для UI origin)

```bash
curl -i -sS -X OPTIONS \
  -H "Origin: https://goadmin-ui.example.com" \
  -H "Access-Control-Request-Method: POST" \
  http://127.0.0.1:8080/v1/commands/execute
```

Ожидание:
- HTTP `204`
- `Access-Control-Allow-Origin` совпадает с origin.
- `Access-Control-Allow-Methods` содержит `POST`.

Негативный сценарий:

```bash
curl -i -sS \
  -H "Origin: https://evil.example.com" \
  http://127.0.0.1:8080/v1/health
```

Ожидание:
- HTTP `403`
- `error_code=cors_denied`.

## 5. Проверка ключевых endpoint'ов для UI

```bash
curl -sS \
  -H "Authorization: Bearer ${GOADMIN_TOKEN}" \
  http://127.0.0.1:8080/v1/modules
```

Ожидание:
- HTTP `200`
- `items` (список модулей).

```bash
curl -sS \
  -H "Authorization: Bearer ${GOADMIN_TOKEN}" \
  "http://127.0.0.1:8080/v1/metrics/latest?module=host"
```

Ожидание:
- HTTP `200`
- `payload` с метриками.

```bash
curl -sS -X POST \
  -H "Authorization: Bearer ${GOADMIN_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"module":"host","command":"status","args":[]}' \
  http://127.0.0.1:8080/v1/commands/execute
```

Ожидание:
- HTTP `200`
- `status=ok`.

## 6. Корреляция request_id

```bash
curl -i -sS \
  -H "Authorization: Bearer ${GOADMIN_TOKEN}" \
  -H "X-Request-ID: ui-smoke-001" \
  http://127.0.0.1:8080/v1/me
```

Ожидание:
- `X-Request-ID: ui-smoke-001` в ответе.
- `request_id` в JSON совпадает.

## 7. Критерии прохождения smoke

- Все команды выше возвращают ожидаемые коды и поля.
- Нет `5xx` при позитивных сценариях.
- Для всех ответов есть `X-Request-ID`.
