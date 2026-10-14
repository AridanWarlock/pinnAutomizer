---
id: TASK-3.1
title: Добавить постоянный журнал попыток solver на SQLite
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 10:46'
updated_date: '2026-10-14 12:48'
labels: []
dependencies: []
parent_task_id: TASK-3
priority: high
type: enhancement
ordinal: 10000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Для восстановления после остановки процесса solver нужен локальный устойчивый журнал с атомарными изменениями состояния и outbox. Файл должен сохраняться на volume, а второй процесс не должен одновременно владеть журналом одного Docker-host.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 SQLite открывается с WAL, synchronous FULL и ограниченным busy timeout; второй владелец отклоняется
- [x] #2 Повтор регистрации run_id не меняет попытку, другой payload или второй run_id той же задачи дают конфликт
- [x] #3 Состояния и outbox сохраняются атомарно; повтор финализации не перезаписывает результат
- [x] #4 Дисковые интеграционные тесты проверяют повторное открытие, конкуренцию, rollback и освобождение владения после смерти процесса
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Добавить domain.Attempt/Event и narrow SQLite repository с фиксированной версией modernc.org/sqlite без CGO. 2. Реализовать schema version, WAL/FULL/busy timeout и process flock на каноническом локальном пути. 3. Регистрация, условные переходы и финализация с outbox выполняются короткими атомарными транзакциями. 4. Дисковые тесты проверяют конкуренцию, payload-конфликт, rollback, reopen, SQLITE_BUSY и SIGKILL владельца. 5. make check и адресный race; коммит с тестами.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Review пользователя: сделать владение ресурсами в New явным через defer очистки на неуспешной инициализации; разделить открытие пути, ownership, соединение и schema init на приватные методы. Сохранить время жизни DB/lock до Repository.Close и SQL поведение. Проверить освобождение lock после ошибок открытия/версии/schema transaction и общий make check.

Исправление review завершено: New вынесен в connection.go и читает последовательность preparePath/acquireOwner/createJournalFile/openDatabase/initializeSchema. Deferred cleanup освобождает DB/lock до передачи результата вызывающему, включая разворачивание panic; временный file.Close передаёт ошибку через errors.Join. createSchema имеет defer Rollback. Дисковые regression tests проверяют ошибки создания файла, версии и второго CREATE с rollback первой таблицы и повторным успешным открытием. make check и race для SQLite проходят; PG/Docker/Kafka интеграции в этом запуске не выполнялись.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлены SQLite ledger (modernc.org/sqlite v1.60.1, без CGO), ownership flock, WAL/FULL/busy timeout, идемпотентная регистрация, CAS записи контейнера/start intent и атомарный результат+outbox. Дисковые тесты проверяют конкуренцию, rollback, busy, отмену, reopen и SIGKILL владельца. make check и go test -race solver проходят. Адаптер пока не подключён к runtime; Docker/Kafka end-to-end и потеря host не проверены.
<!-- SECTION:FINAL_SUMMARY:END -->
