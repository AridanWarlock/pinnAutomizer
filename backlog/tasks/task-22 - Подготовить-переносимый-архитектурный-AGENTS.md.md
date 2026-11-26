---
id: TASK-22
title: Подготовить переносимый архитектурный AGENTS.md
status: In Progress
assignee:
  - codex
created_date: '2026-11-26 09:59'
updated_date: '2026-11-26 10:49'
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
- [x] #3 Подготовлен пакет исходных примеров PostgreSQL/poolx, usecase/DI и HTTP с явно обозначенными legacy-ограничениями
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
PR #34: https://github.com/AridanWarlock/pinnAutomizer/pull/34. Шаблон для одного сервиса/модуля; примеры в docs/templates/backend/examples. 22 исходных блока сверены с файлами проекта, composition root адаптирован явно. Markdown и make check прошли; архив для передачи агенту сформирован в out/exports. Исходники runtime не менялись.
<!-- SECTION:NOTES:END -->
