---
id: TASK-11
title: Дополнить PR 16 быстрыми поведенческими тестами
status: Done
assignee:
  - '@codex'
created_date: '2026-10-16 04:42'
updated_date: '2026-10-16 04:46'
labels: []
dependencies: []
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Пользователь просит больше читаемых и быстрых тестов до review PR. Приоритет — непокрытые ветки идемпотентности tasks/run, ошибки публикации solver outbox и преобразование команды accept; существующие дисковые и SQL-интеграции сохранить.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Добавлены независимые unit-тесты повтора, конфликта и частичного сбоя tasks/run с понятными названиями
- [x] #2 Solver outbox и accept проверены на границах ошибок и преобразования сообщения без внешнего окружения
- [x] #3 Тесты состояний проверяют перестановки и повторы событий без sleep и реального I/O
- [x] #4 make fmt/check и адресные race проходят; длительность новых unit-тестов и пределы проверок записаны
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Прочитать существующее покрытие и выбрать неповторяющиеся сценарии. 2. Добавить короткие unit-тесты с существующими generated mocks, небольшими именованными фикстурами и независимыми subtests. 3. Проверить новые пакеты, измерить test elapsed по краткому JSON-итогу, выполнить make fmt/check и race. 4. Коммит и push в текущий PR, без merge.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Добавлен 41 независимый сценарий в 4 файлах (21 test function, 28 subtests): tasks/run — повтор, конфликт, режимы, ошибки request/read/status/outbox/commit; solver outbox — пустая очередь, отказ чтения/публикации/удаления, повтор payload/ID, порядок confirm→delete и отмена; accept — envelope/context mapping, ошибки и возврат сохранённой попытки; domain — переставленные/повторные события с неизменностью первого результата. Новые тесты используют существующие Mockery-моки, без sleep и внешнего I/O; независимые случаи t.Parallel. Изолированный uncached запуск новых тестов: max test elapsed 0.01s, время процессов пакетов 0.664–1.257s. make fmt и полный make check с временным PostgreSQL прошли; адресные race четырёх пакетов прошли. Временный PostgreSQL остановлен; интерфейсы и runtime-код не менялись.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлен 41 читаемый unit-сценарий для tasks/run, solver accept/outbox и порядка событий Task; тесты независимы, без sleep/I/O, существующие SQL/crash-проверки сохранены. Использованы готовые generated mocks, runtime и интерфейсы не менялись. make fmt/check и race четырёх пакетов проходят; PostgreSQL-интеграции прошли отдельным полным check с временным кластером. Финальный check без DSN пропускает SQL-интеграции; Docker/Kafka/Redis этим тестированием не проверяются. Максимальный elapsed нового теста в контрольном запуске 0.01s (без порога времени в assertions). Изменения подготовлены для того же PR 16, merge остаётся за review пользователя.
<!-- SECTION:FINAL_SUMMARY:END -->
