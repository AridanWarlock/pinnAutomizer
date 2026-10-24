---
id: TASK-16
title: Создавать задачи без файлового I/O внутри SQL-транзакции
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-23 14:22'
updated_date: '2026-10-23 14:32'
labels: []
dependencies: []
priority: high
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Запись файлов удерживает PostgreSQL-транзакцию; сбой оставляет каталоги без устойчивой компенсации.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Файлы пишутся вне SQL-транзакции, задача доступна только после завершения записи
- [x] #2 Сбой или остановка процесса оставляют устойчивое задание очистки
- [x] #3 Конкурентная очистка не удаляет файлы активной или успешно созданной задачи
- [x] #4 Unit и PostgreSQL-интеграции проверяют сбои, отмену и устаревшую выборку worker
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Файловая блокировка записи/очистки. 2. До записи сохранить file_cleanup; после записи атомарно создать task и убрать cleanup. 3. Перепроверять задание под lock. 4. Проверить сбои и конкуренцию.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Готово к review: make fmt, make check и PostgreSQL integration с -race прошли. Проверены rollback, частичный upload, отмена, stale cleanup, ожидание commit/rollback и Process.Kill создателя. Ограничения: общий flock-compatible volume; POST без idempotency key; power loss/NFS не проверены.

PR на review: https://github.com/AridanWarlock/pinnAutomizer/pull/27 → codex/task-progress.
<!-- SECTION:NOTES:END -->
