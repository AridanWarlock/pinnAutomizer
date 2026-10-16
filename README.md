# PINN Automizer

Backend для загрузки данных, постановки PINN-задач в очередь и выполнения через
Docker. Монорепозиторий: gateway, auth, tasks, solver и общий Go-модуль `pkg`.

## Документация

- [Карта архитектуры и состояние реализации](docs/architecture.md)
- [Причины требований review](docs/review-principles.md)
- [Аудит PR #16 и следующие шаги](docs/next-steps.md)
- [Гарантии и интеграционные проверки tasks](services/tasks/README.md)
- [SQLite-журнал solver и миграции](services/solver/internal/adapter/sqlite/README.md)
- [Краткие проверки качества](scripts/README.md)

## Работа с кодом

Правила проекта — в `AGENTS.md`, задачи — в Backlog.md. Версии Go указаны
в go.mod/go.work; инфраструктура и генераторы закреплены в Compose.

```sh
make tools-install
make fmt
make check
```

Эти команды не требуют `.env`. Для интеграционных PostgreSQL-тестов нужен отдельный
тестовый кластер; без DSN они пропускаются. Инструкции находятся в README tasks.

## Окружение и миграции

Конфигурация локального окружения берётся из `.env.example`; заполненный `.env`
не хранится в Git. Compose требует Docker и настроенных host-каталогов. Для PINN
нужен отдельный runner image; train/retrain используют GPU. Полный запуск
окружения не входит в `make check`.

Миграции запускаются отдельно через Compose и цели `make auth-goose-up`,
`make tasks-goose-up`, `make solver-goose-up`. Создание — соответствующие
`*-goose-create name=...`. Новые SQL-файлы содержат только Up; после генерации
проверять шаблон, применённые файлы не менять. Рабочую БД автоматически не мигрировать.
Для solver перед миграцией требуется освободить ownership lock остановкой владельца
журнала; runtime пока не подключён к SQLite.
