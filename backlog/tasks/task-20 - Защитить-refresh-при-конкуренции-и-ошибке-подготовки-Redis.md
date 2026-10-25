---
id: TASK-20
title: Защитить refresh при конкуренции и ошибке подготовки Redis
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-25 10:42'
updated_date: '2026-10-25 10:55'
labels: []
dependencies: []
priority: high
ordinal: 26000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Refresh сначала потребляет PostgreSQL-токен, затем пишет Redis. Отказ Redis оставляет клиента без нового refresh token; конкурирующий проигравший получает серверную ошибку.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Неудачная подготовка Redis не потребляет старый refresh token и допускает повтор
- [x] #2 При конкурентной ротации одного токена только один запрос успешен; проигравший не удаляет сессию победителя
- [x] #3 Unit и PostgreSQL/Redis integration проверяют частичный сбой, конкуренцию и классификацию ошибок
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Воспроизвести Redis failure до/после эффекта и конкуренцию. 2. Подготовить недоступную клиенту candidate session до SQL CAS, сохранить условную ротацию и классифицировать проигрыш. 3. Проверить реальные системы и документировать пределы повтора без изменения Input/Output.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Проверено: make fmt/check с одноразовыми PostgreSQL/Redis; адресный race refresh + PostgreSQL adapter. SET failure до/после эффекта, повтор, конкурентный CAS, отмена после SET, ошибка очистки и потерянный SQL-ответ после commit. Input/Output сохранены. Неизвестный commit/потеря успешного HTTP-ответа требует повторного входа; идемпотентное восстановление и гонки logout отложены. Границы описаны в docs/refresh-rotation.md.
<!-- SECTION:NOTES:END -->
