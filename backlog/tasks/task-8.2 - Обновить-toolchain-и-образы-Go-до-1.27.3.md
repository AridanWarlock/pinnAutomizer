---
id: TASK-8.2
title: Обновить toolchain и образы Go до 1.27.3
status: Done
assignee:
  - '@codex'
created_date: '2026-10-25 10:09'
updated_date: '2026-10-25 10:54'
labels: []
dependencies: []
parent_task_id: TASK-8
priority: high
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Пользователь запросил отдельный переход на Go 1.27.3 перед продолжением задач. Версии go.mod, workspace и Docker сейчас расходятся; UUID и прочие зависимости не входят в этот PR.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Все пять go.mod, go.work, builders, тестовые Go-образы и CI согласованы на 1.27.3
- [x] #2 make fmt/check, генерация моков и необходимые race/integration проверки выполняются Go 1.27.3
- [x] #3 Сервисные бинарники собираются новым Go; ограничения проверки и локальная инструкция документированы
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Проверить официальный релиз, toolchain и Docker image. 2. Согласовать закреплённые версии без миграции UUID. 3. Проверить штатные инструменты, тесты и Docker-сборки. 4. Открыть отдельный PR.

Выявлена обязательная совместимость: Mockery v3.8.1 исправляет чтение export data Go 1.27; компактный шаблон переносится с этой версии. go mod tidy согласует отдельные module graphs без go get -u.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Проверено Go 1.27.3: make fmt/check с PostgreSQL/Redis; race-тесты пяти модулей и CPU Docker-пакеты; четыре сервисных Docker build; Swagger CLI build metadata go1.27.3; повторная генерация 38 моков побайтово идентична. Пять standalone module graphs tidy. Обязательное сопутствующее обновление: Mockery v3.8.1 для export data Go 1.27. Корневой go.work остаётся ignored и обновлён локально.

PR на review: https://github.com/AridanWarlock/pinnAutomizer/pull/31 → codex/task-progress.

PR #31 принят пользователем. Остальные обновления ПО остаются в TASK-8.
<!-- SECTION:NOTES:END -->
