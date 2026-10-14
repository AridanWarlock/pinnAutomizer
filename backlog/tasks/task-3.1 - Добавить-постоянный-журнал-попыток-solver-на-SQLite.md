---
id: TASK-3.1
title: Добавить постоянный журнал попыток solver на SQLite
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 10:46'
updated_date: '2026-10-14 13:43'
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

По review migrate schema через goose Provider и embed SQL; в Go библиотеке закрепить v3.26.0, так как v3.27.0 требует обновления Docker API/otel. CLI goose в Compose остаётся v3.27.0; SQL-формат совместим. Проверить повторный Up, incremental Up, SQL rollback, отказ older binary при newer schema, Down только на временной БД, и сохранение старого unversioned журнала при отказе. Из-за goose module graph lz4 меняется с v4.1.15 на v4.1.22; Docker API, Go и sqlite driver не меняются.

Новое review: отказаться от embed и автоматического Up. Общий lifecycle SQLite/ownership/проверка готовности схемы вынести в pkg/sqlite; solver оставляет адаптер domain queries. SQL только Up, отдельный goose-solver контейнер из существующего image с flock той же sidecar lock и make solver-goose-create/up. Runtime New не создаёт DB/schema; только открывает заранее мигрированный журнал и отклоняет pending/newer schema. Тесты явно мигрируют временные файлы до открытия и проверяют отказ при running owner; действующие DB не затрагивать.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Review пользователя: сделать владение ресурсами в New явным через defer очистки на неуспешной инициализации; разделить открытие пути, ownership, соединение и schema init на приватные методы. Сохранить время жизни DB/lock до Repository.Close и SQL поведение. Проверить освобождение lock после ошибок открытия/версии/schema transaction и общий make check.

Исправление review завершено: New вынесен в connection.go и читает последовательность preparePath/acquireOwner/createJournalFile/openDatabase/initializeSchema. Deferred cleanup освобождает DB/lock до передачи результата вызывающему, включая разворачивание panic; временный file.Close передаёт ошибку через errors.Join. createSchema имеет defer Rollback. Дисковые regression tests проверяют ошибки создания файла, версии и второго CREATE с rollback первой таблицы и повторным успешным открытием. make check и race для SQLite проходят; PG/Docker/Kafka интеграции в этом запуске не выполнялись.

Review пользователя: заменить ручные schemaVersion/PRAGMA user_version на goose, уже используемый в проекте. Поддержка SQLite/modernc подтверждена официальной документацией. План: goose v3.27.0 как в compose, SQL Up/Down с embed.FS, instance Provider без глобальной конфигурации, запуск Up под ownership lock; тесты повторного применения, pending migration с сохранением данных, rollback и неизвестной версии. Runtime solver ещё не подключён, действующие БД не мигрировать.

Goose внедрён: SQL Up/Down в services/solver/migrations, embed.FS, instance Provider без глобального registry. schema.go/schemaVersion/PRAGMA user_version удалены; новые версии определяются SQL-файлами, история в goose_db_version. Provider Up выполняется под ownership lock, более новая схема отклоняется; старый unversioned journal не baselined автоматически, тест подтверждает сохранность данных при отказе. Проверены reopen без повторного Up, pending Up ровно один раз с сохранением попытки, rollback SQL+version при сбое, Down только на временной БД. make check, race SQLite и CGO_ENABLED=0 Linux test binary проходят. Рабочие БД не затрагивались, PG/Docker/Kafka end-to-end не запускались.

Последнее review реализовано: connection/WAL/ownership/check migrations перенесены в pkg/sqlite; solver адаптер хранит domain SQL. Embed удалён; SQL только Up; runtime требует существующий мигрированный файл и не запускает Up. В Compose goose-solver на общем goose image с flock, make solver-goose-create/up; SQL копируется в runtime image отдельно, общий журнал volume. В .mockery.yml общие настройки и recursive all для пяти Go-модулей, исключение SQLite удалено; генерация v3.7.1 успешна. Docker daemon недоступен: запуск/сборка контейнера пока не проверены.

Review завершён: shared pkg/sqlite хранит connection/lock/check schema, solver адаптер использует его без embed и автоматического Up. Внешний goose-solver из общей goose image, make create/up, общий persisted journal path и flock. Up-only wrapper запрещает Down и удаляет Down только из нового goose create-файла. Проверены make check, race shared SQLite/solver, CGO_ENABLED=0 Linux test binary, shell syntax и Compose config с .env.example; Docker daemon недоступен, реальный контейнер migration/solver и PG integration не запускались. .mockery.yml recursive all по пяти модулям успешно отработал host mockery v3.7.1; Docker generator создаёт временный workspace для clean clone. Правила review по нерешённым комментариям добавлены в AGENTS.md.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлены SQLite ledger (modernc.org/sqlite v1.60.1, без CGO), ownership flock, WAL/FULL/busy timeout, идемпотентная регистрация, CAS записи контейнера/start intent и атомарный результат+outbox. Дисковые тесты проверяют конкуренцию, rollback, busy, отмену, reopen и SIGKILL владельца. make check и go test -race solver проходят. Адаптер пока не подключён к runtime; Docker/Kafka end-to-end и потеря host не проверены.
<!-- SECTION:FINAL_SUMMARY:END -->
