---
id: TASK-18
title: Исправить владение pool и транзакциями PostgreSQL
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-24 16:31'
updated_date: '2026-10-24 16:38'
labels: []
dependencies: []
priority: high
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
poolx не закрывает pool после ошибки Ping и передаёт транзакцию всем клиентам через общий context key.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Ошибка первоначального Ping освобождает созданный pool
- [x] #2 Операции используют только транзакцию своего клиента; вложенная транзакция того же клиента отклоняется
- [x] #3 Unit и PostgreSQL интеграции проверяют независимость commit/rollback двух клиентов и отмену
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Регрессионные тесты владения ресурсами и context. 2. Закрытие при неуспешной инициализации и ключ с владельцем. 3. Проверки на одноразовом PostgreSQL и make check.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Проверено: регрессия общего ключа воспроизведена на PostgreSQL (rollback A терял запись B); с исправлением адресный -race и полный make check с PINN_TEST_POSTGRES_DSN прошли. Unit проверяет Close при Ping failure, вложенность, panic и объединение ошибок rollback без logger. Обнаруженный panic при отмене устранён; WithoutCancel не добавлялся.
<!-- SECTION:NOTES:END -->
