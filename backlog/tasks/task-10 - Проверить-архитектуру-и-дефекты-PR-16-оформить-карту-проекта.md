---
id: TASK-10
title: 'Проверить архитектуру и дефекты PR 16, оформить карту проекта'
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-16 03:52'
updated_date: '2026-10-16 03:57'
labels: []
dependencies: []
ordinal: 15000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Пока PR 16 ожидает review, требуется независимо проверить добавленные механизмы и собрать компактный контекст для следующих задач: текущее поведение, ограничения и причины требований ревьюера. Исправлять подтверждённые дефекты PR; крупные изменения runtime отложить.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs описывает сервисы, потоки данных, границы слоёв и фактическое подключение механизмов PR
- [ ] #2 docs связывает требования review с причинами, компромиссами и приоритетами дальнейшей работы
- [ ] #3 Подтверждённые дефекты PR исправлены регрессионными тестами без смены PINN API и массового рефакторинга
- [ ] #4 make check и релевантные проверки выполнены, ограничения явно записаны
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Проверить текущий diff и маршруты runtime, прочитать адаптеры и тесты добавленных механизмов. 2. Воспроизвести подозрительные случаи минимальными тестами; исправить подтверждённые баги. 3. Написать docs с картой, оценкой, требованиями review и порядком задач. 4. Выполнить make fmt/check и адресные race; коммиты разделить по смыслу и обновить PR 16.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Аудит подтвердил незавершённость runtime solver: SQLite/accept/outbox подготовлены, но cmd/main.go использует прежний at-most-once путь. Регрессионные тесты сначала воспроизвели два дефекта: A→B transaction context скрывал привязку A (deadline вместо rollback/nesting error); CheckMigrations пропускал отсутствующий файл применённой промежуточной версии. Исправлено: ключ context с владельцем SQLite, проверка всех applied версий против sources Goose. make fmt и полный make check с отдельным временным PostgreSQL прошли; адресные race SQLite/solver adapter прошли. Временный PostgreSQL остановлен; Docker/Kafka/Redis и PINN не запускались.
<!-- SECTION:NOTES:END -->
