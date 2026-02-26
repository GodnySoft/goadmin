# План внедрения `goadmin` (Execution Pack)

Этот каталог содержит декомпозированный план разработки `goadmin` по этапам.

## Документы

1. `_stage-template.md` — единый шаблон этапа.
2. `stage-1-foundation-cli.md` — фундамент, core, CLI, CI.
3. `stage-2-daemon-storage.md` — режим демона, scheduler, SQLite.
4. `stage-3-transports-messaging.md` — мессенджер-транспорты и авторизация.
5. `stage-4-web-fleet.md` — HTTP API, встроенный UI, fleet-операции.
6. `stage-5-hardening-deploy.md` — hardening, systemd, Ansible, поставка.
7. `stage-6-llm-adapter-safety.md` — LLM-адаптер и контуры безопасности.
8. `master-checklist.md` — сквозной контроль качества и критерии релиза.

## Правила использования

- Каждый этап выполняется только после закрытия обязательных критериев предыдущего этапа.
- Любое архитектурное изменение фиксируется отдельным ADR в `docs/dev/adr`.
- Запуск этапа блокируется, если не пройдены security gates из документа этапа.
- Все проверки из раздела `Проверки` должны быть автоматизированы в CI, если это возможно.

## Базовые нефункциональные требования (для всех этапов)

- Целевая платформа: только Ubuntu 24.04 LTS.
- Язык: Go 1.22+.
- Архитектура: модульный монолит (Ports & Adapters).
- Безопасность: Zero-Trust, deny-by-default.
- Надежность: graceful shutdown, отсутствие утечек горутин.
- Качество: `golangci-lint`, `go test`, `-race`, `gosec`, SBOM.

## Связанные инструкции

- Развертывание dev-окружения через Ansible: `docs/dev/instr/ansible-dev-setup.md`
- Операционная инструкция по setup: `docs/instr/ansible-dev.md`
