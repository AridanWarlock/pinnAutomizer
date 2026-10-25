# Go 1.27.3

Версия подтверждена в [официальных релизах](https://go.dev/doc/devel/release#go1.27.3).
Изменение обновляет toolchain, а не API приложения и не использование UUID.

| Место | Версия / источник |
| --- | --- |
| Пять go.mod | go 1.27.3 |
| CI | setup-go читает pkg/go.mod |
| Четыре Docker builders, Swagger и CPU-тесты | golang:1.27.3-alpine |
| Mockery | v3.8.1, образ закреплён digest; GOTOOLCHAIN=go1.27.3 |
| golangci-lint | Прежняя v2.14.0, совместимость проверена make check |
| Swag | Прежняя v1.16.6, собрана новым Go |

Mockery v3.7.0 не читает экспорт типов Go 1.27 даже после пересборки.
[v3.8.1 содержит исправление](https://github.com/vektra/mockery/releases/tag/v3.8.1).
Компактный шаблон и schema перенесены с этой версии, общая рекурсивная конфигурация
сохранена. Generated-файлы редактирует только генератор.

`go mod tidy` выполнен отдельно для каждого модуля с GOWORK=off. Это исправляет
несогласованность графов, скрывавшуюся workspace: Docker собирает каждый сервис
отдельно. `go get -u` не запускался; прямые версии SDK/UUID не обновлялись,
транзитивные требования согласованы с уже выбранными версиями shared pkg.

## Локальная работа

Старый вручную заданный GOROOT может направить новый go в каталог старых compiler tools.
В таком окружении можно выполнить без изменения глобальных настроек:

```sh
env -u GOROOT GOTOOLCHAIN=go1.27.3 go version
env -u GOROOT GOTOOLCHAIN=go1.27.3 make check
```

Корневой go.work и go.work.sum остаются локальными игнорируемыми файлами. Если
workspace уже существует, выровнять его через `go work edit -go=1.27.3` новым Go.
Если отсутствует — `go work init ./pkg ./services/auth ./services/gateway ./services/solver ./services/tasks`.
Quality runner в чистом checkout сам создаёт временный workspace из версий go.mod.

Перед opt-in CPU Docker-тестами загрузить `docker pull golang:1.27.3-alpine`:
наличие образа в BuildKit cache после сборки ещё не делает его доступным Docker engine.

Сборка не применяет миграции и не выполняет deploy. Обновление стандартного uuid,
Kafka/Redis/PostgreSQL и прочих зависимостей остаётся отдельными задачами TASK-8.
