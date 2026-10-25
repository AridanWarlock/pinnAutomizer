---
id: TASK-21
title: Закрыть гонку refresh/logout
status: In Progress
assignee:
  - codex
created_date: '2026-10-25 11:27'
updated_date: '2026-10-25 11:32'
labels: []
dependencies: []
priority: high
ordinal: 27000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Logout с уже авторизованным старым JTI должен отзывать и сессию конкурентного refresh.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 После успешного logout ни старый access, ни access конкурентного refresh не имеют Redis-сессии
- [x] #2 Условная ротация после удаления refresh-записи не выдаёт токены; повтор logout допустим
- [x] #3 Unit и PostgreSQL/Redis integration проверяют оба порядка гонки, отмену и частичный сбой
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
FOR UPDATE сериализует logout с refresh CAS; Redis DEL актуального и request JTI выполняются перед удалением строки и commit. RED на прежнем коде: новая Redis-сессия оставалась после успешного logout. Проверено make fmt/check с PostgreSQL/Redis, адресный race logout + auth PostgreSQL; оба порядка гонки, DEL до/после эффекта, отмена, повтор, отсутствующая refresh-запись. Ограничения: короткий Redis I/O под SQL-блокировкой; нет durable intent до подтверждения; новый login и неизвестный commit отдельно. docs/session-auth.md.
<!-- SECTION:NOTES:END -->
