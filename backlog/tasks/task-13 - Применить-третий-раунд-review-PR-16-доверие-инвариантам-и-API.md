---
id: TASK-13
title: 'Применить третий раунд review PR 16: доверие инвариантам и API'
status: Done
assignee:
  - '@codex'
created_date: '2026-10-17 20:26'
updated_date: '2026-10-17 20:32'
labels: []
dependencies: []
ordinal: 18000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
13 новых замечаний: убрать повторную валидацию готовых сущностей, общие результаты/partial error, разделить schema/converters и уточнить fail-fast для нарушения persistence-контракта.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Все 13 замечаний обработаны, ответы Codex опубликованы без resolve
- [x] #2 Готовые сущности не перевалидируются в WithIdKey/RegisterAttempt; общие типы в core, row-конструкторы/конверторы разделены
- [x] #3 Нарушения внутренних row/partial-index контрактов явно проверены тестами, штатные I/O ошибки остаются ошибками
- [x] #4 make check, адресные race и релевантные интеграции пройдены, docs обновлены
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Упростить WithIdKey и обновить вызовы. 2. Перенести Created/Changed/PartialErr в core; упростить partial positions и проверить fail-fast. 3. Разделить schema/converters, добавить row-конструкторы и shared SQLite Returning, конвертацию DB-инвариантов сделать fail-fast. 4. Моки/тесты/fmt/check/race, docs, commits/push и ответы в PR.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
13 замечаний обработаны в коде: WithIdKey без error/повторной валидации; slices.Clone в delivery; empty struct inline; RegisterAttempt доверяет сущности и использует CanRegister; Created/Changed/PartialErr в core; Returning в shared SQLite; row constructors и конверторы в converters.go, constants разделены по таблицам. toDomain теперь panic при нарушении persistence-инварианта; malformed PartialErr positions тоже panic до удаления событий. SQL I/O и неизвестный Kafka исход остаются обычными errors. Тесты проверили row-roundtrip, invalid state/deadline/UUID/headers/identity и реальную SQL запись, допустимую CHECK, но недопустимую domain; после panic транзакция откатывается и повтор работает. Полный make check с Docker PostgreSQL/Kafka/Goose прошёл; race выполняется. Mockery 3.7.1 перегенерирован по общей конфигурации; docs/AGENTS уточнены. Контейнеры одноразовые, без рабочего окружения и PINN.

Финальный make check с Docker PostgreSQL/Kafka/Goose и адресные race для core/kafka/sqlite/solver прошли. Тестовые контейнеры остановлены. Нормальные SQL I/O errors не превращены в panic; fail-fast только при нарушении row-conversion/partial-index контрактов. Валидные фикстуры WithIdKey используют NewIdempotencyKey; метод без error, headers копируются. Осталось опубликовать ответы на все 13 обсуждений без resolve.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Все 13 новых замечаний PR16 обработаны, ответы [Codex] опубликованы без resolve. f07db92: WithIdKey без error/повторной валидации, shared core.Created/Changed/PartialErr, CanRegister, row-конструкторы и converters.go, shared SQLite Returning, раздельные schema constants, slices.Clone, compact empty struct. row-to-domain и нарушение partial-index контракта паникуют; обычные I/O ошибки сохранены. Добавлены roundtrip/invalid-row/real-SQL-panic-rollback тесты, обновлены unit-тесты и моки Mockery 3.7.1. 30259a9: AGENTS/docs уточняют доверие готовым сущностям, fail-fast и пределы DB CHECK. Полный make check с Docker PostgreSQL/Kafka/Goose и адресные race прошли; тестовые контейнеры остановлены. Runtime solver/PINN не менялись, merge не выполнялся.
<!-- SECTION:FINAL_SUMMARY:END -->
