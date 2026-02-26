# Утилита диагностики хоста (`diag.sh`)

Bash-утилита для сбора диагностики хоста и подготовки метрик Prometheus (textfile collector). Ориентирована на Ubuntu 24.04 LTS, но работает на большинстве современных Linux-систем.

## Что делает

- Собирает системную информацию в каталог с меткой времени.
- Создает ссылку `latest` на последний запуск.
- Формирует метрики Prometheus в `diag.prom`.
- Опционально записывает метрики в каталог textfile collector Node Exporter.
- Собирает единый отчет `report.txt` для администраторов.

## Структура вывода

По умолчанию все сохраняется в `./out`:

```
./out/
  run-YYYYmmdd_HHMMSS/
    uname.txt
    os-release.txt
    uptime.txt
    date.txt
    cpu.txt
    memory.txt
    lsblk.txt
    df.txt
    mount.txt
    ip_addr.txt
    ip_link.txt
    ip_route.txt
    resolv.conf
    sockets.txt
    ps_top.txt
    ps_mem.txt
    systemd_failed_units.txt
    systemd_timers.txt
    systemd_services.txt
    journal_last_1h.txt
    journal_last_boot_errors.txt
    apt_updates.txt
    ufw_status.txt
    apparmor_status.txt
    diag.prom
    report.txt
  latest -> run-YYYYmmdd_HHMMSS
```

Все команды выполняются в режиме best-effort: если команда недоступна или вернула ошибку, скрипт продолжает работу, а вывод ошибки сохраняется в соответствующий файл.

## Метрики Prometheus

Файл `diag.prom` содержит:

- `host_diag_info{hostname,os_id,os_version,kernel}` = `1`
- `host_diag_failed_systemd_units`
- `host_diag_last_run_timestamp_seconds`
- `host_diag_duration_seconds`

Если задан `--textfile-dir` (или `PROM_TEXTFILE_DIR`), метрики дополнительно записываются в `<textfile-dir>/diag.prom` атомарной операцией.

## Использование

```
./diag.sh [--output DIR] [--textfile-dir DIR] [--quiet]
```

### Параметры

- `--output DIR`
  Каталог для логов. По умолчанию: `./out`.

- `--textfile-dir DIR`
  Каталог textfile collector Node Exporter. Необязателен.

- `--quiet`
  Минимум вывода в консоль.

- `-h`, `--help`
  Показать справку.

### Переменные окружения

- `OUTPUT_DIR`
  Аналог `--output`.

- `PROM_TEXTFILE_DIR`
  Аналог `--textfile-dir`.

## Примеры

Базовый запуск:

```
utils/diag/diag.sh
```

Вывод в отдельный каталог:

```
utils/diag/diag.sh --output /var/tmp/diag
```

Запись метрик в textfile collector Node Exporter:

```
utils/diag/diag.sh --textfile-dir /var/lib/node_exporter/textfile_collector
```

Запуск с переменными окружения:

```
PROM_TEXTFILE_DIR=/var/lib/node_exporter/textfile_collector \
OUTPUT_DIR=/var/tmp/diag \
utils/diag/diag.sh
```

## Требования

- Bash 4+ (в Ubuntu 24.04 LTS используется Bash 5.x).
- Базовые утилиты: `uname`, `date`, `ip`, `lsblk`, `df`, `mount`, `ps`.
- Опционально: `lscpu`, `free`, `ss`, `systemctl`, `journalctl`, `apt`, `ufw`, `aa-status`.

Скрипт устойчив к отсутствию отдельных утилит.

## Безопасность

- Скрипт только читает состояние системы, ничего не изменяет.
- Вывод может содержать чувствительные данные (сеть, процессы, журналы). Храните и передавайте с осторожностью.

## Расширение

Для добавления новых данных используйте `write_cmd` или `write_file` в `utils/diag/diag.sh`.

## Траблшутинг

- Если `diag.prom` не появляется в каталоге textfile collector:
  - Проверьте, что каталог существует и доступен для записи.
  - Убедитесь, что Node Exporter запущен с textfile collector и указывает на этот каталог.

- Если нет некоторых файлов, проверьте наличие соответствующих команд на системе.
