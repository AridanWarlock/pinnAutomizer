---
id: TASK-15
title: Повторять очистку файлов удалённых задач
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-23 12:17'
updated_date: '2026-10-23 12:47'
labels: []
dependencies: []
priority: high
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
После commit удаления task пути файлов теряются, а RemoveAll только логирует ошибку. Сбой процесса или файловой системы оставляет каталоги без возможности восстановления.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Удаление задачи и сохранение путей очистки атомарны в PostgreSQL
- [ ] #2 Файловый I/O выполняется после commit и повторяется после частичного сбоя или рестарта
- [ ] #3 Ошибочное задание не блокирует следующие; отмена и повтор безопасны
- [ ] #4 Unit и PostgreSQL интеграции проверяют rollback и повтор, HTTP-семантика описана
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Up-only таблица file_cleanup; валидная domain сущность; delete usecase сохраняет job и удаляет task одной транзакцией. Отдельный cleanup usecase/worker с keyset cursor без SQL-транзакций вокруг файлов, подключён в lifecycle. Создание/загрузка файлов — следующий самостоятельный PR.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Реализованы атомарная очередь очистки при DELETE и lifecycle worker. Проверено: make fmt; make check; PostgreSQL integration с -race (rollback, частичное удаление, сбой ACK, reopen, конкуренция, повтор DELETE). Ограничения: общий файловый volume; создание задачи с файловым I/O внутри транзакции вынесено в следующий этап.

Готово к review: https://github.com/AridanWarlock/pinnAutomizer/pull/26 (base codex/task-progress). Проверки успешны, одноразовый PostgreSQL остановлен.
<!-- SECTION:NOTES:END -->
