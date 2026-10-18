---
id: TASK-9
title: 'Обработать замечания review PR #16'
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-15 13:38'
updated_date: '2026-10-15 14:07'
labels: []
dependencies: []
ordinal: 14000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Ревьюер оставил 29 нерешённых обсуждений к PR #16: shared SQLite API/scanning/query builder, lifecycle options, тесты, границы слоёв, конвенции и объяснения решений. Исправления должны остаться в текущем PR, ответы помечаются [Codex], обсуждения закрывает ревьюер.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Общий SQLite API поддерживает Sqlizer, Getx/Selectx/Execx, структурное сканирование и транзакции с дисковыми тестами; solver использует этот API
- [x] #2 Замечания lifecycle, HTTP shutdown, cache TTL, Redis tests и quality runner обработаны и проверены
- [x] #3 Замечания solver/domain/outbox, usecase интерфейсов, конвенций и миграций обработаны без смены PINN API
- [x] #4 make check и адресные race проходят; обновления отправлены в тот же PR, все 29 обсуждений получили ответ [Codex] без auto-resolve
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Сопоставить 29 комментариев с текущим кодом и разделить на связанные группы. 2. Исправить инфраструктуру/тесты/lifecycle и простые конвенции, проверить и закоммитить. 3. Реализовать SQLite API по poolx через существующий scany/sqlscan и squirrel; адаптировать solver, протестировать атомарность/сканирование/cancel. 4. Уточнить attempt/domain/outbox/config и migration workflow. 5. Перегенерировать моки, make check, race, push; ответить по каждому обсуждению с commit/test и обоснованием, не resolve.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Первая группа готова: lifecycle Worker options с валидацией до старта; shutdown на Background timeout + тест принудительного закрытия; TTL<=0 не хранится, фикстуры expired заданы явно, ticker.Close/close race исправлены. Redis unit tests используют generated MockRedis и TryLock до общего барьера. Runner больше не знает env Postgres. Именованы multipart memory threshold, вставлен TxManager, инвертирован happy path run; удалены Down и лишние IDE/Python ignore правила; объявления развёрнуты. Полный make check и реальные PG integration на временном TCP-кластере прошли.

Исправления опубликованы в PR #16: 189eee8, 1ffaf40, c2b5c1c. Shared SQLite использует Sqlizer, scany/sqlscan и контекстные транзакции; solver переведён на Squirrel; domain validation и naming уточнены; polling outbox вынесен в infrastructure worker. Полный make check с реальным временным PostgreSQL прошёл; адресные race-проверки cache/lifecycle/HTTP/Redis Store/SQLite/solver прошли с разрешением локального listener. Linux CGO_ENABLED=0 solver build и SQLite test compile прошли. Все 29 обсуждений получили ответы [Codex]; обсуждения не закрыты, PR не влит. Docker daemon и реальный Redis недоступны: образ миграций и Redis TTL интеграционно не проверены; PINN не запускался. Временный PostgreSQL остановлен. Ожидается review пользователя.
<!-- SECTION:NOTES:END -->
