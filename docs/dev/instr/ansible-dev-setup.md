# Развертывание dev-окружения (Ansible)

## Назначение
Документ описывает автоматическое развёртывание окружения разработки `goadmin` через Ansible.

## Что настраивается
- системные пакеты для сборки (`build-essential`, `gcc`, `make`, `sqlite3`, `libsqlite3-dev` и др.);
- локальная установка Go в `.local/go` внутри репозитория;
- локальные кэши Go (`.cache/go-build`, `.cache/gomod`);
- переменные окружения в `.bashrc` пользователя разработки;
- очистка прокси-переменных (опционально);
- предзагрузка офлайн-зависимостей (`utils/fetch-mods.sh`);
- запуск `make tidy` в офлайн-режиме.

## Файлы Ansible
- `deployments/ansible/ansible.cfg`
- `deployments/ansible/dev-setup.yml`
- `deployments/ansible/group_vars/all.yml`
- `deployments/ansible/inventories/local/hosts.ini`
- `deployments/ansible/roles/dev_env/defaults/main.yml`
- `deployments/ansible/roles/dev_env/tasks/main.yml`

## Подготовка
1. Убедиться, что репозиторий `goadmin` уже склонирован на целевой машине.
2. Проверить значения в `deployments/ansible/group_vars/all.yml`:
   - `goadmin_project_root`
   - `goadmin_dev_user`
   - `goadmin_go_version`
   - `goadmin_prefetch_modules`
   - `goadmin_run_make_tidy`
   - `goadmin_install_system_packages` (если нет sudo, установить `false`)
   - `goadmin_tidy_offline` (`false` для online tidy, `true` для strict offline)

## Запуск
Из корня репозитория:

```bash
cd deployments/ansible
ansible-playbook dev-setup.yml
```

С переопределением параметров:

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
- Если `ansible-playbook` отсутствует, установите Ansible на управляющей машине.
- При ограниченной сети включайте предзагрузку модулей (`goadmin_prefetch_modules: true`).
- При полностью офлайн-среде заранее подготовьте `.cache/gomod` и отключите `goadmin_run_make_tidy` на первом прогоне.
