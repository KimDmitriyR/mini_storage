# mini_storage

Мини-S3-like файловое хранилище на Go с локальным хранением файлов на диске и метаданными в SQLite.

## Возможности

- REST API для загрузки, скачивания, удаления и просмотра файлов
- UUID-based имена файлов в локальном хранилище
- SQLite для хранения метаданных
- конфигурация через переменные окружения
- логирование запросов, recovery middleware и JSON-ошибки
- ограничение размера загружаемого файла
- Docker и docker-compose
- базовые unit и handler тесты

## Структура проекта

```text
cmd/server/main.go
internal/config
internal/http
internal/metadata
internal/middleware
internal/storage
storage/
migrations/
Dockerfile
docker-compose.yml
README.md
```

## Переменные окружения

| Переменная | Описание | Значение по умолчанию |
| --- | --- | --- |
| `PORT` | порт HTTP-сервера | `8080` |
| `STORAGE_DIR` | директория хранения файлов | `storage` |
| `MAX_UPLOAD_SIZE_MB` | лимит размера upload в мегабайтах | `10` |
| `DATABASE_PATH` | путь к SQLite-файлу с метаданными | `storage/metadata.db` |

Пример:

```bash
cp .env.example .env
```

## Локальный запуск

Требования:

- Go `1.25+`

Запуск:

```bash
go run ./cmd/server
```

Запуск в PowerShell с явными переменными окружения:

```powershell
$env:PORT="8080"
$env:STORAGE_DIR="storage"
$env:MAX_UPLOAD_SIZE_MB="10"
$env:DATABASE_PATH="storage/metadata.db"
go run ./cmd/server
```

Проверка health endpoint:

```bash
curl http://localhost:8080/health
```

## Запуск через Docker

Сборка и запуск:

```bash
docker compose up --build
```

После старта API будет доступен по адресу `http://localhost:8080`.

## API примеры

### Upload

```bash
curl -X POST http://localhost:8080/files \
  -F "file=@./README.md"
```

Пример ответа:

```json
{
  "id": "2c5daec4-6217-4fd7-af49-d9742e16d7fd",
  "original_name": "README.md",
  "stored_name": "6cb81e3c-e536-4415-91c9-755f2ce0c643.md",
  "content_type": "text/markdown",
  "size": 1234,
  "created_at": "2026-04-26T11:00:00Z"
}
```

### List

```bash
curl http://localhost:8080/files
```

### Download

```bash
curl -L http://localhost:8080/files/<file_id> --output downloaded-file
```

### Metadata

```bash
curl http://localhost:8080/files/<file_id>/meta
```

### Delete

```bash
curl -X DELETE http://localhost:8080/files/<file_id>
```

Для PowerShell удобнее использовать `curl.exe`, чтобы не попасть на алиас `Invoke-WebRequest`:

```powershell
curl.exe -X POST http://localhost:8080/files -F "file=@README.md"
curl.exe http://localhost:8080/files
```

## Разработка и проверки

```bash
gofmt -w .
go test ./...
go vet ./...
```
