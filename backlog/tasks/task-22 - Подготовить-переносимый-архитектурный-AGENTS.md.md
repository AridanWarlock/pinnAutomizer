---
id: TASK-22
title: Подготовить переносимый архитектурный AGENTS.md
status: In Progress
assignee:
  - codex
created_date: '2026-11-26 09:59'
updated_date: '2026-11-26 10:00'
labels: []
dependencies: []
ordinal: 28000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Самодостаточный шаблон правил Go-бэкенда для другой предметной области.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Описаны пакеты, допустимые зависимости, domain/core и контракты usecase/адаптеров
- [x] #2 Описаны тесты, моки, отказоустойчивость и проверки без привязки к бизнес-логике текущего проекта
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Готов docs/templates/backend/AGENTS.md; ссылка добавлена в docs/README.md. Проверены Markdown и make check. Интеграционные системы отдельно не запускались: изменение только документации.
<!-- SECTION:NOTES:END -->
