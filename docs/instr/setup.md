# Развертывание окружения разработки goadmin (локально, без внешнего доступа)

## Требования
- Linux (Ubuntu 24.04). Нет прав sudo и ограничен интернет.
- Под рукой скачанный архив Go 1.22.0 (если нет сети). Makefile умеет скачивать при доступе.

## Шаги
1) Установка Go локально (внутрь репозитория)
- Если интернет есть:
  ```bash
  make go-install
  ```
  Go распакуется в `.local/go`. Версия управляется переменной `GO_TARBALL` в Makefile.
- Если интернета нет: положите заранее `go1.22.0.linux-amd64.tar.gz` в `/tmp/go.tgz` и выполните:
  ```bash
  tar -C .local -xzf /tmp/go.tgz
  ```

2) Настройка локальных кешей
- Makefile уже задаёт локальные пути:
  - `GOCACHE=.cache/go-build`
  - `GOMODCACHE=.cache/gomod`
- Переменные передаются во все команды через `ENV_VARS`.

3) Установка тулов
```bash
make tools
```
Установятся `golangci-lint` и `gosec` в `$(GOBIN)` (по умолчанию `$HOME/go/bin`).

4) Проверки качества и сборка
```bash
make check    # tidy + fmt + lint + test + race + gosec + build
make build    # только сборка
make run      # вывод версии
make serve    # запуск демона
```

5) Офлайн-модули (если нет интернета)
- Скачайте необходимые модули и сложите в `.cache/gomod` согласно структуре Go proxy (пример для gopsutil):
  - `.cache/gomod/github.com/shirou/gopsutil/v3/@v/v3.24.1.zip`
  - `.cache/gomod/github.com/shirou/gopsutil/v3/@v/v3.24.1.mod`
  - `.cache/gomod/github.com/shirou/gopsutil/v3/@v/v3.24.1.info`
- Затем запустите:
  ```bash
  GOPROXY=file://$(pwd)/.cache/gomod,off GOSUMDB=off make tidy
  ```

6) Очистка
```bash
make clean
rm -rf .cache
```

## Примечания безопасности
- Все команды используют локальные кеши, чтобы не писать вне репозитория.
- При отсутствии сети используйте заранее заготовленный набор модулей, иначе Go попытается выйти во внешний прокси.

## Ansible-вариант
- Для автоматизированного развёртывания используйте playbook:
  - `deployments/ansible/dev-setup.yml`
- Пошаговая инструкция:
  - `docs/instr/ansible-dev.md`
  - `docs/dev/instr/ansible-dev-setup.md`
