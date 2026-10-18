---
id: TASK-14
title: 'Применить четвёртый раунд review PR 16: schema и Kafka API'
status: Done
assignee:
  - '@codex'
created_date: '2026-10-18 12:07'
updated_date: '2026-10-18 12:11'
labels: []
dependencies: []
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Шесть замечаний: таблицы в schema целиком по группам, Values для messageRow, Kafka PartialErr в адаптере, доверие Limit, set вместо bool map, Created в Output.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Все шесть замечаний исправлены и получили ответ Codex без resolve
- [x] #2 Типы, моки, тесты и docs согласованы; проверки пройдены
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Переместить Kafka-контракт и согласовать Output. 2. Перегруппировать schema, добавить messageRow.Values с использованием в eventRow. 3. Убрать повторный Validate Limit, использовать set. 4. Генерация моков, fmt/check/race, docs, commit/push и ответы.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Шесть правок внесены: kafka.PartialErr в пакете адаптера, core оставлен для Created/Changed; schema сгруппирована целиком attempts, затем outbox; messageRow.Values используется составной eventRow.Values; повторный Validate Limit удалён; confirmed positions set — map[int]struct{}; Output.Created имеет тип core.Created без потери типа. Assertions сравнивают typed Created. Mockery 3.7.1 перегенерирован: контрактный mock перенесён core→kafka. Документы/AGENTS исправлены, включая прежнее чрезмерное обобщение всех ошибок в core. make check прошёл, адресные race прошли; финальный fmt/check выполняется. Docker/PG/Kafka интеграции в этом раунде не запускались, дисковые SQLite-тесты входят в стандартный check.

Финальные make fmt и make check прошли. Адресные race-тесты Kafka, SQLite, outbox и accept прошли. Внешние интеграции в этом раунде не повторялись.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Все шесть замечаний исправлены коммитом 799f176 и опубликованы в PR #16. В каждом обсуждении оставлен ответ [Codex]; обсуждения не закрывались. Обновлены AGENTS.md, архитектура и правила review. make check и адресные race-тесты прошли; внешние интеграции не повторялись.
<!-- SECTION:FINAL_SUMMARY:END -->
