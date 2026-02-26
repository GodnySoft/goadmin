# Развертывание dev-окружения через Ansible

## Что делает automation
- Устанавливает системные dev-пакеты (build-essential, sqlite3 dev headers, make, curl и т.д.).
- Ставит Go локально в репозиторий: `.local/go`.
- Настраивает локальные кеши: `.cache/go-build`, `.cache/gomod`.
- Прописывает Go-переменные в `.bashrc` пользователя.
- Удаляет proxy-переменные из `.bashrc` (опционально).
- Выгружает офлайн-зависимости через `utils/fetch-mods.sh`.
- Запускает `make tidy` в офлайн-режиме (`GOPROXY=file://... ,off`).

## Предпосылки
- Ubuntu (рекомендуется 24.04).
- Установлен `ansible` на управляющей машине.
- Репозиторий уже склонирован на целевой машине.

## Конфигурация
Основные переменные в `deployments/ansible/group_vars/all.yml`:
- `goadmin_project_root`
- `goadmin_dev_user`
- `goadmin_go_version`
- `goadmin_prefetch_modules`
- `goadmin_run_make_tidy`

## Запуск
Из корня репозитория:

```bash
cd deployments/ansible
ansible-playbook dev-setup.yml
```

Запуск с переопределением путей/пользователя:

```bash
cd deployments/ansible
ansible-playbook dev-setup.yml \
  -e goadmin_project_root=/home/<user>/work/goadmin \
  -e goadmin_dev_user=<user>
```

## Проверка результата

```bash
cd /path/to/goadmin
.local/go/bin/go version
make build
make test
```

## Примечания
- Если сеть ограничена, используйте `goadmin_prefetch_modules: true` и убедитесь, что есть доступ к `proxy.golang.org`.
- При полностью офлайн-среде заранее подготовьте `.cache/gomod` и выключите `goadmin_run_make_tidy` на первом прогоне.
