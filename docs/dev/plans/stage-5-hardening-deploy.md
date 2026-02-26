# Stage 5 — Hardening, Systemd, Ansible Deploy

## 1. Цель этапа

Довести `goadmin` до production-safe развертывания: системный hardening, воспроизводимый deployment, и управляемый rollback.

## 2. Scope

### In Scope

- `deployments/systemd/goadmin.service` с ограничениями безопасности.
- Ansible role/playbook для идемпотентного развёртывания.
- Создание сервисного пользователя и прав на конфиги/данные.
- Поставка бинарника + checksum/signature verification.

### Out of Scope

- Kubernetes deployment.
- Кросс-ОС поддержка.

## 3. Входные условия

- [ ] Stage 4 закрыт.
- [ ] Подготовлены release-артефакты и политика версионирования.
- [ ] Доступен тестовый Ubuntu 24.04 стенд.

## 4. Выходные артефакты

- `goadmin.service` с hardening directives.
- Ansible role `deployments/ansible/roles/goadmin`.
- Runbook deploy/rollback.
- Проверки CIS-baseline для целевого хоста.

## 5. Backlog этапа

| ID | Задача | Результат | Зависимость | Оценка |
|---|---|---|---|---|
| S5-01 | Описать systemd unit hardening | Ограниченный runtime профиль | Stage 4 | 1д |
| S5-02 | Реализовать Ansible role | Идемпотентный deploy | S5-01 | 1.5д |
| S5-03 | Добавить verify checksum/signature | Защита цепочки поставки | S5-02 | 0.5д |
| S5-04 | Настроить лог- и конфиг-права | Least privilege на ФС | S5-01 | 0.5д |
| S5-05 | Провести deploy/rollback drill | Проверка восстановления | S5-02 | 1д |

## 6. API/Контракты/Схемы

- Публичный API не расширяется.
- Добавляются операционные контракты:
  - `systemctl start|stop|status goadmin`
  - Ansible vars: `goadmin_version`, `goadmin_checksum`, `goadmin_config`.

## 7. Security Gates

- [ ] Запуск только от `goadmin` non-root user.
- [ ] В unit включены: `NoNewPrivileges=true`, `ProtectSystem=strict`, `PrivateTmp=true`, `ProtectHome=true`.
- [ ] Ограничены capabilities (`CapabilityBoundingSet=` минимум).
- [ ] Конфиг и sqlite-файл не доступны world-readable.
- [ ] Подпись/хэш бинарника проверяется перед установкой.

## 8. Проверки (Definition of Done)

### Функциональные

- [ ] Fresh install на чистой Ubuntu 24.04 проходит полностью.
- [ ] Повторный запуск playbook не вносит лишних изменений (idempotency).
- [ ] Сервис стартует и проходит healthcheck.

### Нефункциональные

- [ ] Время развёртывания < 5 минут на узел.
- [ ] Rollback до предыдущей версии < 3 минут.

### Командные проверки

```bash
ansible-playbook -i inventories/stage deploy.yml --check
ansible-playbook -i inventories/stage deploy.yml
systemctl status goadmin --no-pager
systemd-analyze security goadmin.service
go test ./... -race
gosec ./...
```

## 9. Наблюдаемость и эксплуатация

- Логи сервиса в journald, парсинг JSON полей.
- Алерты: restart loop, healthcheck fail, disk usage sqlite > порога.

## 10. Rollback

- Ansible задача переключения на `goadmin_previous_version`.
- Автоматический restore предыдущего бинарника и unit-reload.
- Верификация после отката: health endpoint + smoke CLI.

## 11. Риски и меры

| Риск | Вероятность | Влияние | Митигирующее действие |
|---|---|---|---|
| Слишком строгий unit ломает функционал | Средняя | Среднее | Пошаговый hardening + integration tests |
| Неидемпотентный playbook | Средняя | Высокое | Обязательный `--check` и повторный прогон |
| Компрометация цепочки поставки | Низкая | Высокое | Подпись и checksum verify |

## 12. Exit Criteria

- [ ] Успешный deploy и rollback drill на стенде.
- [ ] Security score systemd приемлем для production.
- [ ] Runbook эксплуатации и инцидент-процедуры утверждены.
