# Утилита анализа GC и памяти

Проект представляет собой HTTP-сервер на Go, который показывает состояние памяти и сборщика мусора через endpoint `/metrics` в формате Prometheus.

В проекте используются:

- `runtime.ReadMemStats` — чтение текущей статистики памяти и GC;
- `debug.SetGCPercent` — настройка процента запуска GC;
- `net/http/pprof` — встроенные endpoint-ы профилирования;
- HTTP API для проверки работы сервера;
- Dockerfile, docker-compose и Makefile для удобного запуска.

## Структура проекта

```text
.
├── cmd/gc-mem-exporter/main.go       # точка входа
├── internal/config/config.go         # загрузка настроек из env
├── internal/httpapi/server.go        # HTTP handlers
├── internal/metrics/collector.go     # сбор runtime-метрик
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```

## Endpoint-ы

| Метод | URL | Назначение |
|---|---|---|
| GET | `/` | список доступных endpoint-ов |
| GET | `/health` | проверка состояния сервера |
| GET | `/metrics` | метрики памяти и GC в формате Prometheus |
| GET | `/gc-percent` | текущее значение GOGC |
| POST | `/gc-percent?value=50` | изменить GOGC во время работы |
| POST | `/gc` | принудительно запустить сборку мусора |
| POST | `/alloc?mb=10&keep=true` | выделить память для демонстрации изменения метрик |
| DELETE | `/alloc` | очистить удерживаемую память |
| GET | `/debug/pprof/` | pprof-профилирование |

## Переменные окружения

| Переменная | Значение по умолчанию | Описание |
|---|---:|---|
| `ADDR` | `:8080` | адрес HTTP-сервера |
| `GC_PERCENT` | `100` | значение GOGC, которое применяется через `debug.SetGCPercent` |

## Запуск локально

```bash
make run
```

Или напрямую:

```bash
ADDR=:8080 GC_PERCENT=100 go run ./cmd/gc-mem-exporter
```

## Запуск в Docker

```bash
make docker-up
```

Посмотреть логи:

```bash
make docker-logs
```

Остановить контейнер:

```bash
make docker-down
```

Также можно запустить без docker-compose:

```bash
make docker-run
```

## Проверка работы

Проверка health endpoint-а:

```bash
curl http://localhost:8080/health
```

Ожидаемый ответ:

```json
{"status":"ok"}
```

Получить метрики:

```bash
curl http://localhost:8080/metrics
```

Пример части ответа:

```text
# HELP go_mem_total_alloc_bytes Cumulative bytes allocated for heap objects.
# TYPE go_mem_total_alloc_bytes counter
go_mem_total_alloc_bytes 583184
# HELP go_gc_cycles_total Cumulative number of completed GC cycles.
# TYPE go_gc_cycles_total counter
go_gc_cycles_total 0
# HELP go_heap_alloc_bytes Bytes of allocated heap objects.
# TYPE go_heap_alloc_bytes gauge
go_heap_alloc_bytes 583184
```

Создать нагрузку на память:

```bash
curl -X POST "http://localhost:8080/alloc?mb=50&keep=true"
```

После этого можно снова посмотреть метрики:

```bash
curl http://localhost:8080/metrics | grep -E "go_heap_alloc_bytes|go_mem_total_alloc_bytes|go_heap_objects"
```

Запустить GC вручную:

```bash
curl -X POST http://localhost:8080/gc
```

Проверить, что количество сборок мусора изменилось:

```bash
curl http://localhost:8080/metrics | grep go_gc_cycles_total
```

Изменить процент запуска GC:

```bash
curl -X POST "http://localhost:8080/gc-percent?value=50"
```

Проверить новое значение:

```bash
curl http://localhost:8080/gc-percent
curl http://localhost:8080/metrics | grep go_gc_percent
```

Очистить удерживаемую память:

```bash
curl -X DELETE http://localhost:8080/alloc
```

## pprof

Открыть список профилей:

```bash
curl http://localhost:8080/debug/pprof/
```

Посмотреть goroutine-профиль в текстовом виде:

```bash
curl "http://localhost:8080/debug/pprof/goroutine?debug=1"
```

Снять heap-профиль:

```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

Снять CPU-профиль за 30 секунд:

```bash
go tool pprof "http://localhost:8080/debug/pprof/profile?seconds=30"
```

Открыть pprof в браузере:

```bash
go tool pprof -http=:9090 http://localhost:8080/debug/pprof/heap
```

## Тесты и проверки

Форматирование:

```bash
make fmt
```

Статическая проверка:

```bash
make vet
```

Тесты:

```bash
make test
```

Сборка:

```bash
make build
```

Полная локальная проверка:

```bash
make all
```

Если установлен `golint`, можно дополнительно выполнить:

```bash
make lint
```

## Основные метрики

| Метрика | Тип | Описание |
|---|---|---|
| `go_mem_alloc_bytes` | gauge | текущий объем выделенной heap-памяти |
| `go_mem_total_alloc_bytes` | counter | суммарное количество выделенных байт за время работы |
| `go_mem_sys_bytes` | gauge | память, полученная runtime от ОС |
| `go_heap_alloc_bytes` | gauge | занятая heap-память |
| `go_heap_objects` | gauge | количество объектов в heap |
| `go_gc_cycles_total` | counter | количество завершенных GC-циклов |
| `go_gc_pause_total_ns` | counter | суммарная длительность GC-пауз |
| `go_gc_last_timestamp_seconds` | gauge | Unix-время последней сборки мусора |
| `go_gc_last_pause_ns` | gauge | длительность последней GC-паузы |
| `go_gc_cpu_fraction` | gauge | доля CPU, использованная GC |
| `go_gc_percent` | gauge | текущее значение GOGC |
| `go_goroutines` | gauge | количество goroutine |

## Что демонстрирует проект

Проект показывает, как можно сделать небольшой observability-сервис для Go-приложения:

1. собрать статистику памяти через `runtime.ReadMemStats`;
2. отдать ее в формате Prometheus;
3. подключить pprof для анализа heap, goroutine, CPU и trace;
4. менять GOGC во время работы через `debug.SetGCPercent`;
5. запустить приложение локально или в Docker.
