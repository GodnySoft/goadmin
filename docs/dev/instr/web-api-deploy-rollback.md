# Runbook: Deploy / Rollback Web API (External React)

Документ описывает эксплуатационные шаги для включения web API `goadmin` под внешний React UI и безопасного отката.

## 1. Цель

- Включить web transport с bearer auth и CORS allowlist.
- Проверить доступность API и базовые сценарии.
- Иметь быстрый rollback при инциденте.

## 2. Предусловия

- Бинарник `goadmin` установлен и запускается через systemd.
- Есть рабочий конфиг (рекомендуется от `configs/config.prod.example.yaml`).
- Токен UI выдан, в конфиге сохранен **SHA-256** (`token_sha256`), а не plaintext.
- Origin React UI известен и внесен в `web.cors.allowed_origins`.

## 3. Конфиг (production baseline)

Ключевые параметры:

- `web.enabled: true`
- `web.auth.mode: bearer`
- `web.auth.allow_legacy_subject_header: false`
- `web.auth.tokens[*].enabled: true`
- `web.cors.allowed_origins: [https://<react-ui-domain>]`

## 4. Deploy (пошагово)

1. Подготовить/обновить конфиг на сервере.
2. Проверить синтаксис YAML и значения auth/cors.
3. Перезапустить сервис:

```bash
sudo systemctl daemon-reload
sudo systemctl restart goadmin
sudo systemctl status goadmin --no-pager
```

4. Проверить API smoke:
- `GET /v1/health`
- `GET /v1/me` с bearer
- CORS preflight `OPTIONS /v1/commands/execute`

Использовать:
- `docs/dev/api/smoke-runbook-react.md`

## 5. Наблюдаемость после deploy

- Проверить логи:

```bash
sudo journalctl -u goadmin -n 200 --no-pager
```

- Убедиться в отсутствии всплеска `401/403/5xx`.
- Проверить, что `request_id` присутствует в ответах и логах.

## 6. Rollback (быстрый)

### Сценарий A: отключение web transport

Использовать при критическом инциденте web API:

1. `web.enabled: false`
2. Перезапуск сервиса.

Ожидаемый эффект:
- CLI/daemon функции сохраняются.
- Web API становится недоступным.

### Сценарий B: временный fallback на legacy header

Использовать при проблемах с bearer-секретами/ротацией:

1. `web.auth.allow_legacy_subject_header: true`
2. (Опционально) `web.auth.mode: bearer` оставить, чтобы bearer продолжал работать.
3. Перезапуск сервиса.

Важно:
- Это временная мера.
- После стабилизации вернуть `allow_legacy_subject_header: false`.

### Сценарий C: откат бинарника/конфига на предыдущую версию

1. Вернуть предыдущий бинарник/релизный пакет.
2. Вернуть предыдущий рабочий конфиг.
3. Перезапустить сервис.
4. Прогнать smoke-runbook.

## 7. Инциденты и действия

- `401 auth_required/invalid_token`:
  - проверить `Authorization` формат в UI;
  - проверить `token_sha256` в конфиге.
- `403 cors_denied`:
  - проверить `Origin` и `web.cors.allowed_origins`.
- `504 request_timeout`:
  - проверить нагрузку/БД;
  - временно увеличить `web.request_timeout_ms`.

## 8. Критерии успешного deploy

- Все smoke-сценарии проходят.
- Нет ошибок `5xx` в позитивных API сценариях.
- React UI успешно выполняет `/v1/me`, `/v1/modules`, `/v1/metrics/latest`, `/v1/commands/execute`.
