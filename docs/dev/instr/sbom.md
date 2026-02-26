# SBOM (Software Bill of Materials)

`goadmin` использует встроенный генератор SBOM на базе `go list -m`.

## Формат

- CycloneDX JSON (spec `1.5`).
- Файл по умолчанию: `bin/sbom-gomod.cdx.json`.

## Генерация

```bash
make sbom
```

Команда использует:

```bash
go run ./utils/sbom -o bin/sbom-gomod.cdx.json
```

## Проверка

- Убедиться, что файл существует и содержит список `components`.
- Для релиза прикладывать файл SBOM вместе с бинарником.

## Примечания

- В SBOM попадают зависимости из `go.mod` (`go list -m all`).
- В офлайн/ограниченной сети генератор автоматически использует `go.mod` + `go.sum` без запросов во внешний proxy.
- Для main-модуля фиксируется `metadata.component`.
