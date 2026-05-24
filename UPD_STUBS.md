# upd-stubs: Генератор OpenAPI-заглушек
`upd-stubs` — это консольная утилита (CLI) для автоматизации создания и сопровождения реализаций OpenAPI mock серверов на
языке Go. Инструмент сканирует спецификации OpenAPI, формирует типобезопасные заглушки, настраивает внедрение
зависимостей (Dependency Injection) и выполняет «умное» обновление существующих файлов, сохраняя ранее написанную
бизнес-логику.
## Ключевые возможности
### 1. Автоматическая генерация заглушек
* **Анализ OpenAPI спецификаций:** Рекурсивно сканирует директорию `specs/` для поиска файлов `openapi.yaml`, `openapi.yml` или `openapi.json`.
* **Распознавание операций:** Автоматически выявляет все HTTP операции и группирует их по тегам.
* **Строгий режим (Strict Server):** Генерирует код, совместимый с `StrictServerInterface`, используя типизированные запросы и ответы вместо `http.ResponseWriter`.
* **Стабильный порядок методов:** Парсит сгенерированный Go-код (`server.gen.go`), чтобы гарантировать порядок методов, идентичный интерфейсу (не зависит от порядка в map/yaml).
* **Формирование структуры:** Генерирует файлы реализации в директории `internal/stubs`, зеркально повторяя структуру спецификаций.

### 2. Интеллектуальное (безопасное) обновление
В отличие от стандартных генераторов, которые часто просто перезаписывают файлы, `upd-stubs` анализирует существующий код, чтобы сохранить ваш ручной труд:
* **Добавление методов:** Автоматически обнаруживает новые операции, появившиеся в спецификации, и добавяет их заготовки в код, если они отсутствуют.
* **Сохранение методов:** Существующие реализации методов остаются нетронутыми.
* **Primary-tag стратегия:** Если операция имеет несколько тегов, заглушка генерируется только для первого тега (без дублирующих методов в разных файлах).
* **Управление импортами:** Автоматически добавляет необходимые зависимости.
* **Режим prune (`--prune`):** Не удаляет код, но добавляет аннотацию со списком "осиротевших" методов, которые больше не описаны в спецификации.

### 3. Качественная кодогенерация
Создаваемый код соответствует минимальным стандартам качества и содержит необходимую обвязку:
* **Интеллектуальный выбор ответа:** По умолчанию генерирует заглушку с наименьшим успешным кодом ответа (например, выберет 200 вместо 404, обрабатывает `default` как `Default`).
* **Структура обработчиков:** Генерирует структуры для каждого тега (группы операций) с поддержкой логирования.
* **Конструкторы:** Создает функции `New<Tag>Handlers`, которые принимают флаг конфигурации `enableLogging`.
* **Композитные обработчики:** Создает файл `provider.go` с типом `CompositeHandlers`, объединяющим все обработчики тегов и реализующим интерфейс `StrictServerInterface`.

### 4. Логирование и работа с контекстом
* **Контекст запроса:** Все сгенерированные методы умеют извлекать `reqID` (Request ID), используя ключ из пакета `openapi-mock/pkg/ctxkeys`.
* **Управляемое логирование:** Структура обработчика содержит поле `EnableLogging`.
    * Если опция включена, в лог автоматически пишется имя метода и идентификатор запроса.

### 5. Интеграция с Google Wire
* **Автоматическое связывание:** Генерирует файл `internal/app/openapi_wire.go` для настройки dependency injection с помощью [Google Wire](https://github.com/google/wire).
* **Единая точка входа:** Собирает все сгенерированные заглушки в общую структуру `HTTPApp`.
* **Набор провайдеров:** Формирует `wire.NewSet`, включающий конструкторы всех обнаруженных обработчиков.

## Параметры CLI

```bash
go run ./cmd/upd-stubs \
  --specs-dir specs \
  --generated-dir internal/generated \
  --stubs-dir internal/stubs \
  --wire-out internal/app/openapi_wire.go \
  --prune \
  --verbose
```

Поддерживаемые флаги:

| Флаг | Описание |
|:--|:--|
| `--specs-dir` | Директория со спецификациями OpenAPI |
| `--generated-dir` | Директория с `oapi-codegen` файлами |
| `--stubs-dir` | Куда писать заглушки |
| `--wire-out` | Путь для генерируемого wire-файла |
| `--dry-run` | Сгенерировать изменения без записи файлов |
| `--prune` | Добавлять блок с orphaned-методами при рассинхронизации |
| `--verbose` | Печатать расширенные диагностические сообщения |

---

## 📂 Структура проекта
Инструмент ориентирован на следующую структуру директорий:
```text
./specs/                     # Входные данные: OpenAPI спецификации
└── <api-name>/              # Каждая API в отдельной директории
    └── openapi.yaml         # OpenAPI 3.0 спецификация
./internal/
├── app
│   ├── openapi_wire.go      # Обновляется: Wire файл для HTTP сервера
│   └── wire_gen.go          # Обновляется: Реальный файл для компиляции
├── generated                # Выходные данные: Сгенерированный oapi-codegen код
│   └── <api-name>/
│       ├── server.gen.go    # Chi-сервер и StrictServerInterface
│       ├── spec.gen.go      # Спецификация
│       └── types.gen.go     # Типы данных
└── stubs                    # Выходные данные: Сгенерированные заглушки
    └── <api-name>/
        ├── <tag>.go         # Обработчики для каждого тега
        └── provider.go      # Композитный провайдер StrictHandler
```

---

## 🛠 Примеры генерируемого кода

### 1. Файл реализации заглушки для тега "pets"
Для спецификации `petstore` инструмент создаст файл `internal/stubs/petstore/pets.go` в стиле Strict Server:

```go
package petstore

import (
	"context"
	"log"

	gen "openapi-mock/internal/generated/petstore"
	"openapi-mock/pkg/ctxkeys"
)

type PetsHandlers struct {
	EnableLogging bool
}

func NewPetsHandlers(enableLogging bool) *PetsHandlers {
	return &PetsHandlers{EnableLogging: enableLogging}
}

func (h *PetsHandlers) ListPets(ctx context.Context, request gen.ListPetsRequestObject) (gen.ListPetsResponseObject, error) {
	if h.EnableLogging {
		reqID, _ := ctx.Value(ctxkeys.RequestID{}).(string)
		log.Printf("[req_id=%s] [PetsHandlers] ListPets", reqID)
	}

	_ = request

	return gen.ListPets200JSONResponse{}, nil
}
```

### 2. Файл provider.go
```go
// Code generated by upd-stubs. DO NOT EDIT.
package petstore

import (
	gen "openapi-mock/internal/generated/petstore"
)

type CompositeHandlers struct {
	default_ *DefaultHandlers
	pets     *PetsHandlers
}

func NewCompositeHandlers(default_ *DefaultHandlers, pets *PetsHandlers) gen.StrictServerInterface {
	return &CompositeHandlers{default_: default_, pets: pets}
}
```

### 3. Файл конфигурации Wire
Генерирует `internal/app/openapi_wire.go`:
```go
//go:build wireinject
// +build wireinject
package app
import (
  "github.com/go-chi/chi/v5"
  "github.com/google/wire"
  petstoregen "openapi-mock/internal/generated/petstore"
  petstorestub "openapi-mock/internal/stubs/petstore"
)
type HTTPApp struct {
  Router          *chi.Mux
  PetstoreDefault *petstorestub.DefaultHandlers
  PetstorePets    *petstorestub.PetsHandlers
}
var HTTPProviderSet = wire.NewSet(
  petstorestub.NewDefaultHandlers,
  petstorestub.NewPetsHandlers,
  providePetstoreHandlers,
  provideHTTPRouter,
  wire.Struct(new(HTTPApp), "*"),
)
func InitializeHTTPApp(enableLogging bool) (*HTTPApp, error) {
  wire.Build(HTTPProviderSet)
  return nil, nil
}
```
