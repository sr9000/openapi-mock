# Генератор OpenAPI Мок-серверов

Этот репозиторий предлагает оптимизированный, полуавтоматический процесс создания OpenAPI мок-серверов. Инструмент берет
на
себя рутинные задачи генерации кода из спецификаций OpenAPI и генерирует Go-код (заглушки), сохраняя при этом вашу
кастомную бизнес-логику при обновлениях.
Детали генерации заглушек даны в описании работы утилиты [upd-stubs](./UPD_STUBS.md).

## 📂 Структура проекта

```text
.
├── Makefile             # Главный оркестратор рабочего процесса
├── bin/                 # Скомпилированные бинарные файлы
├── cmd/                 # Точки входа (сервер и утилиты)
├── internal/
│   ├── app/             # Связывание приложения (DI)
│   ├── generated/       # Сгенерированные Go-файлы из OpenAPI (НЕ РЕДАКТИРОВАТЬ)
│   │   ├── echo/        # Сгенерированный код для Echo API
│   │   └── petstore/    # Сгенерированный код для Petstore API
│   └── stubs/           # Реализации обработчиков (<--- РЕДАКТИРУЙТЕ ЗАГЛУШКИ ЗДЕСЬ)
│       ├── echo/        # Заглушки для Echo API
│       └── petstore/    # Заглушки для Petstore API
├── pkg/
│   ├── ctxkeys/         # Определения ключей контекста
│   ├── metrics/         # Сервер метрик Prometheus
│   ├── mgmt/            # Сервер управления для e2e-тестирования
│   ├── middleware/      # HTTP middleware (логирование, запись)
│   └── recorder/        # Запись HTTP-вызовов
├── api/                 # <--- Поместите ваши OpenAPI спецификации сюда
│   ├── echo/            # Пример: Echo API (без версии)
│   │   └── openapi.yaml
│   └── petstore/        # Пример: Petstore API
│       ├── openapi.yaml # Базовая версия
│       └── v3/          # Версионированная версия
│           └── openapi.yaml
└── scripts/             # Вспомогательные скрипты для генерации
```

## 📋 Предварительные требования

* **Go**: версия 1.25 или выше
* **Make**
* **oapi-codegen**: Генератор Go кода из OpenAPI (
  `go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`)

## 🚀 Быстрый старт

Рабочий процесс максимально упрощен:

1. **Создайте OpenAPI спецификацию** в директории `api/`.

   Например, для нового API `myservice`:
   ```bash
   mkdir -p api/myservice
   # Создайте api/myservice/openapi.yaml с вашей спецификацией

   # Дополнительно можно держать версию рядом
   mkdir -p api/myservice/v3
   # Создайте api/myservice/v3/openapi.yaml
   ```

2. **Запустите команду сборки**:
   ```bash
   make all
   ```
   Эта команда выполнит:
    - `make openapi` — генерация Go-кода из всех спецификаций в `api/`
    - `make stub` — создание/обновление заглушек в `internal/stubs/`
    - `make wire` — обновление DI-конфигурации для всех API
    - `make build` — компиляция бинарного файла

3. **Проверьте сгенерированные заглушки** в папке `internal/stubs/<api-name>/`.
    * Инструмент автоматически создаст структуры Go и сигнатуры методов, совместимые с `StrictServerInterface`.
    * Генерируется код с типизированными запросами и ответами (вместо сырого `http.ResponseWriter`).
    * Заглушки группируются по тегам OpenAPI (например, `pets.go`, `users.go`).
    * Вы можете реализовать логику мока внутри этих файлов. Будущие запуски **не перезапишут** ваш код; они лишь добавят
      новые методы, если изменится спецификация (в стабильном порядке, соответствующем интерфейсу).

4. **Запустите сервер**:
   ```bash
   ./bin/openapi-mock run
   ```

   Все API будут доступны на одном порту. Например:
    - Petstore API: `http://localhost:8080/pets`
    - Echo API: `http://localhost:8080/echo`

## ✨ Ключевые возможности

* **Мульти-API поддержка**: Добавляйте любое количество OpenAPI спецификаций в `api/`. Все API автоматически
  объединяются в один HTTP-сервер с единой точкой входа.
* **Поддержка версий API**: Одновременно поддерживаются оба формата:
    * `api/<api_name>/openapi.yaml`
    * `api/<api_name>/<version>/openapi.yaml` (например, `api/petstore/v3/openapi.yaml`)
      Для каждого найденного spec-path генерируются отдельные `internal/generated/...` и `internal/stubs/...` модули.
* **Умная генерация заглушек**: Утилита `upd-stubs` генерирует бойлерплейт для реализации сервера. Она сканирует
  существующие файлы, чтобы гарантировать, что **существующая логика внутри методов сохранится** при повторной генерации
  моков.
* **Автоматическое внедрение зависимостей**: Используется [Google Wire](https://github.com/google/wire) для
  автоматического
  связывания всех обработчиков с основным HTTP-сервером. При добавлении нового API не требуется изменять `main.go`.
* **Группировка по тегам**: Операции группируются по тегам OpenAPI, создавая отдельные файлы обработчиков для каждого
  тега.
* **Middleware из коробки**: Встроенная поддержка логирования, метрик Prometheus и записи вызовов для e2e-тестирования.

## ⚙️ Архитектура сервера и CLI

Сервер `openapi-mock` построен на базе фреймворка Cobra и Chi роутера.

### Внутреннее устройство сервера

* **Внедрение зависимостей (Wire)**:
  Приложение использует `google/wire` в пакете `internal/app`. При запуске `make wire` (или `make all`) инструмент
  обнаруживает новые заглушки в `internal/stubs` и регистрирует их.
* **Middleware и логирование**:
  Сервер включает middleware для записи запросов, которое генерирует уникальный
  **Request ID**, внедряет его в контекст и логирует статус запроса.
* **Ключи контекста**:
  Вы можете получить сгенерированный Request ID внутри ваших заглушек, используя хелпер из `pkg/ctxkeys`.
* **Плавная остановка (Graceful Shutdown)**:
  Сервер прослушивает системные сигналы (`SIGINT`, `SIGTERM`) для корректного завершения соединений.

### Использование CLI

Управление сервером осуществляется через команду `run`. Конфигурацию можно передать через позиционные аргументы, флаги
или переменные окружения.

#### Синтаксис:

```bash
./bin/openapi-mock run [host] [port] [flags]
```

#### Позиционные аргументы:

* `[host]`: Интерфейс для привязки (например, `127.0.0.1`).
* `[port]`: Порт для прослушивания (например, `8080`).

#### Флаги:

| Флаг     | Сокр. | Описание                                 |
|:---------|:------|:-----------------------------------------|
| `--port` | `-p`  | Порт (переопределяет переменную `PORT`). |
| `--help` | `-h`  | Показать справку.                        |

#### Переменные окружения:

| Переменная                   | По умолч.        | Описание                                            |
|:-----------------------------|:-----------------|:----------------------------------------------------|
| `HOST`                       | 0.0.0.0          | Хост интерфейса для привязки                        |
| `PORT`                       | 8080             | Порт для прослушивания                              |
| `MGMT_PORT`                  | 9000             | Порт сервера управления                             |
| `METRICS_PORT`               | 9100             | Порт сервера метрик Prometheus                      |
| `MGMT_ENABLED`               | true             | Включить сервер управления                          |
| `METRICS_ENABLED`            | true             | Включить сервер метрик                              |
| `HTTP_LOGGING`               | true             | Включить логирование запросов/ответов               |
| `REQUEST_ID_HEADERS`         | X-Request-ID,... | Заголовки для входящего request id                  |
| `REQUEST_ID_RESPONSE_HEADER` | X-Request-ID     | Каноничный response header c request id             |
| `LOG_FORMAT`                 | json             | Формат логов (`json`/`console`)                     |
| `LOG_OUTPUT`                 | stdout           | Куда писать логи (`stdout`/`file`)                  |
| `LOG_FILE`                   | -                | Путь до лог-файла, если `LOG_OUTPUT=file`           |
| `LOG_LEVEL`                  | info             | Уровень логирования (`debug..error`)                |
| `TRACE_ENABLED`              | false            | Включить OpenTelemetry tracing                      |
| `TRACE_EXPORTER`             | none             | Экспортер (`none`/`file`/`otlp-http`)               |
| `TRACE_ENDPOINT`             | -                | OTLP HTTP endpoint (например `otel-collector:4318`) |
| `TRACE_FILE`                 | ./traces.json    | Файл трейсов при `TRACE_EXPORTER=file`              |
| `TRACE_SAMPLING_RATIO`       | 1.0              | Доля семплирования трейсов                          |

#### Примеры:

```bash
# Запуск с настройками по умолчанию (0.0.0.0:8080)
./bin/openapi-mock run
# Запуск на порту 3000
./bin/openapi-mock run -p 3000
# Запуск только на localhost
./bin/openapi-mock run 127.0.0.1 8080
```

## 🔭 Обозреваемость (Observability)

Проект включает встроенную поддержку современных практик Observability для удобства отладки и прозрачности:

### Метрики (Prometheus)

На выделенном порту (`9100` по умолчанию) работает сервер метрик. В метриках автоматически подставляются конкретные
OpenAPI-операции (поле `operation`), чтобы можно было легко агрегировать статистику по конкретным ручкам даже при
использовании путей с параметрами:

```bash
curl -sS http://localhost:9100/metrics | grep http_requests_total
# Пример вывода:
# http_requests_total{endpoint="/echo",method="POST",operation="Echo",status="200"} 1
# http_requests_total{endpoint="/pets/{petId}",method="GET",operation="GetPetById",status="200"} 1
```

В случае ошибок парсинга (например, неверный JSON) метрики будут содержать `kind="request_parse"`, но при этом сохранят
правильную привязку к исходному `endpoint` и `operation`.

### Трассировка (OpenTelemetry / Tempo)

При включении флага `TRACE_ENABLED=true` (по умолчанию включено в `docker-compose.observability.yaml`) сервер
автоматически извлекает `traceparent` из заголовков HTTP-запросов и отправляет спаны в OpenTelemetry Collector.

```bash
# Пример запроса с пробросом конкретного trace_id = 4bf92f3577b34da6a3ce929d0e0e4736
curl -X POST http://localhost:8080/echo \
  -H 'Content-Type: application/json' \
  -H 'traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01' \
  -d '{"message":"traced"}'
```

### Структурированное логирование (Loki / ZeroLog)

Все логи HTTP-запросов (access logs) и внутреннее логирование автоматически связываются с `request_id`, `trace_id` и
`operation`.
Формат логов по умолчанию — `json` (управляется переменной `LOG_FORMAT`), что делает их готовыми для парсинга. В
заглушках (stubs) рекомендуется использовать контекстный логгер из `pkg/observability` для сохранения привязки к
`trace_id`:

```go
logger := observability.Logger(ctx, zerolog.Nop())
logger.Info().Msg("Запрос достиг бизнес-логики") // Будет содержать trace_id и operation
```

### Полный локальный стек (Grafana)

Compose-файлы лежат в корне репозитория, а все конфиги observability — в `deploy/`:

- `docker-compose.dev.yaml` — dev/watch-окружение.
- `docker-compose.observability.yaml` — полный observability-стек.
- `.env.example` — пример локальных переменных окружения.
- `deploy/prometheus.yaml`, `deploy/otel.yaml`, `deploy/promtail.yaml`, `deploy/tempo.yaml` — плоские конфиги сервисов.
- `deploy/grafana/` — образ Grafana и provisioning (datasources + dashboards).

Вы можете поднять полный изолированный стек (Prometheus, Loki, Tempo, OTel Collector, Grafana), выполнив:

```bash
make compose-up
```

- **Grafana** будет доступна по адресу `http://localhost:3000`.
- Заранее настроенные дашборды и источники данных покажут RPS, latency, распределение по OpenAPI-операциям,
  ошибки/паники по `kind`, а трассы и логи будут доступны через Grafana Explore (Tempo + Loki).

## 🔧 Сервер управления (Management API)

Для поддержки e2e-тестирования OpenAPI-мок включает встроенный HTTP-сервер управления на порту 9000 (по умолчанию).
Этот сервер записывает все HTTP-вызовы и предоставляет API для их просмотра и очистки.

### Эндпоинты:

| Метод       | Путь                           | Описание                                           |
|:------------|:-------------------------------|:---------------------------------------------------|
| `GET`       | `/logs`                        | Получить все записанные HTTP-вызовы в формате JSON |
| `GET`       | `/logs/{request_id}`           | Получить записи только для конкретного request id  |
| `DELETE`    | `/logs`                        | Очистить все записи                                |
| `GET`       | `/context-values`              | Получить все context values по request id          |
| `PUT/PATCH` | `/context-values`              | Полная замена/обновление всех context values       |
| `DELETE`    | `/context-values`              | Очистить все context values                        |
| `GET`       | `/context-values/{request_id}` | Получить values для request id                     |
| `PUT/PATCH` | `/context-values/{request_id}` | Замена/обновление values для request id            |
| `DELETE`    | `/context-values/{request_id}` | Удалить values для request id                      |
| `GET`       | `/doc`                         | Интерактивная страница Swagger UI                  |
| `POST`      | `/reset`                       | Soft reset mock HTTP-сервера без остановки процесса |
| `GET`       | `/docs`                        | Список OpenAPI-документов моков                    |
| `GET`       | `/docs/{api_name}`             | Swagger UI для API или индекс версий               |
| `GET`       | `/docs/{api_name}/openapi.json` | OpenAPI JSON для API (если версия однозначна)    |
| `GET`       | `/docs/{api_name}/{api_ver}`   | Swagger UI для конкретной версии API               |
| `GET`       | `/docs/{api_name}/{api_ver}/openapi.json` | OpenAPI JSON для версии API            |
| `GET`       | `/openapi.json`                | Спецификация OpenAPI в формате JSON                |
| `GET`       | `/metrics`                     | Метрики Prometheus (RPS, тайминги, ошибки)         |

### Формат записи вызова:

```json
{
  "request_id": "ABCD1234",
  "method": "GET",
  "path": "/pets",
  "timestamp": "2026-02-10T12:00:00Z",
  "request": {
    "query": "limit=10"
  },
  "response": {
    "body": "[...]"
  },
  "error": "",
  "panic": "",
  "duration_ms": 5
}
```

### Примеры использования:

```bash
# Получить все записанные вызовы
curl http://localhost:9000/logs
# Получить вызовы только для request id
curl http://localhost:9000/logs/test-id
# Очистить записи
curl -X DELETE http://localhost:9000/logs

# Задать значения в контекст для request id
curl -X PUT http://localhost:9000/context-values/case-a \
  -H 'Content-Type: application/json' \
  -d '{"low":10,"high":20}'

# Прочитать значения для request id
curl http://localhost:9000/context-values/case-a

# Обновить только одно поле
curl -X PATCH http://localhost:9000/context-values/case-a \
  -H 'Content-Type: application/json' \
  -d '{"high":30}'

# Удалить конкретный ключ
curl -X DELETE http://localhost:9000/context-values/case-a \
  -H 'Content-Type: application/json' \
  -d '{"keys":["low"]}'

# Список доступных OpenAPI-доков моков
curl http://localhost:9000/docs

# Swagger UI для API (или индекс версий)
curl http://localhost:9000/docs/petstore

# OpenAPI JSON для API
curl http://localhost:9000/docs/petstore/openapi.json

# OpenAPI JSON для конкретной версии
curl http://localhost:9000/docs/echo/v1/openapi.json

# Soft reset mock runtime
curl -X POST http://localhost:9000/reset

# После reset context-values очищаются
curl http://localhost:9000/context-values/case-a
# Открыть Swagger UI в браузере
open http://localhost:9000/doc
```

### Миграционные заметки

- Эндпоинты `POST /clear` и `DELETE /clear` удалены.
- Для очистки логов используйте `DELETE /logs`.
- Для выборки логов по request id используйте `GET /logs/{request_id}`.

## 🧪 Генератор нагрузки для примера Petstore

В репозитории есть вспомогательная утилита `cmd/openapi-petstore-client`, которая генерирует фоновую HTTP-нагрузку на
`Petstore`-эндпоинты. Она полезна для локальной проверки метрик, логов, panic/error-path и observability-дашбордов.

Быстрый запуск:

```bash
# В одном терминале — сервер
./bin/openapi-mock run

# В другом — генератор нагрузки
make petstore-load
```

Утилита поддерживает базовый URL и частоту пересчета нагрузки через флаги:

```bash
go run ./cmd/openapi-petstore-client --base-url http://localhost:8080 --tick 100ms
```

## 🐳 Docker

### Разработка

```bash
# Запуск в режиме разработки с hot-reload
make docker-dev

# Эквивалентная команда Docker Compose
docker compose -f docker-compose.dev.yaml up --build
```

### Продакшен

```bash
# Сборка образа
make docker-build
# Запуск контейнера
make docker-run
```

### Полный стек с мониторингом

```bash
# Запуск с Prometheus + Grafana + Tempo + Loki + OTel Collector
make compose-up

# Явный compose-файл полного observability-стека
docker compose -f docker-compose.observability.yaml --progress plain build
docker compose -f docker-compose.observability.yaml up -d

# Автоматическая smoke-проверка всего observability стека
make compose-smoke
# Просмотр логов
make compose-logs
# Остановка
make compose-down
```

> Примечание по Docker Compose: для снижения риска известных сбоев CLI при `up --build` репозиторий использует
> двухшаговый запуск (`build` с plain progress, затем `up -d`).
> Это внешняя проблема плагина Compose, а не ошибка Go-кода этого проекта. Локально проверялась версия:
> `Docker Compose v2.29.7-desktop.1`.


> Рекомендуемое место для локальных overrides — корневой `.env` (например `cp .env.example .env`).
> `make docker-dev` и `make compose-*` автоматически передадут его в Docker Compose. Для обратной совместимости также
> поддерживается `deploy/.env`.
