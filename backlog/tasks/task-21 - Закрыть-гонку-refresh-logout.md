---
id: TASK-21
title: Закрыть гонку refresh/logout
status: In Progress
assignee:
  - codex
created_date: '2026-10-25 11:27'
updated_date: '2026-10-25 12:16'
labels: []
dependencies: []
priority: high
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Отзыв refresh фиксируется вместе с устойчивыми заданиями Redis DEL; после commit сбой Redis или отмена запроса не восстанавливают авторизацию через refresh.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Logout атомарно удаляет refresh и сохраняет JTI запроса/актуальной сессии в очереди PostgreSQL до Redis I/O
- [x] #2 После commit отзыва отказ Redis, отмена и неизвестный ответ commit не позволяют refresh выдать новую пару при committed исходе
- [x] #3 Подключённый worker повторяет DEL/подтверждение; повтор logout учитывает все pending JTI user/fingerprint
- [x] #4 Тесты покрывают оба порядка гонки, транзакционные/Redis/ACK сбои, конкурентную очистку, scope и keyset
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Короткая SQL-транзакция: lock, очередь JTI, удаление refresh, commit. Вне транзакции — Redis DEL/ACK и устойчивый повтор через pkg/worker.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
PR #33: https://github.com/AridanWarlock/pinnAutomizer/pull/33. make fmt/check с одноразовыми PostgreSQL/Redis и адресный race прошли. Семантика и ограничения — docs/session-auth.md; новый login и немедленный отзыв access при отказе Redis остаются отдельными решениями.
<!-- SECTION:NOTES:END -->
