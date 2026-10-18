---
id: TASK-8.1
title: Перейти на Go 1.27+ и стандартный uuid.UUID
status: To Do
assignee: []
created_date: '2026-10-14 09:59'
labels: []
dependencies: []
references:
  - 'https://go.dev/doc/go1.27'
  - 'https://pkg.go.dev/uuid'
parent_task_id: TASK-8
priority: medium
type: enhancement
ordinal: 9000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Go 1.27 добавляет стандартный uuid. Сейчас пять модулей и модели используют google/uuid; отличаются Nil/NewString и набор реализованных интерфейсов. Перед заменой нужно проверить JSON/Kafka/Redis, validator, pgx/scany, JWT и моки, сохранив формат существующих идентификаторов и контракт API.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Выбрана поддерживаемая стабильная версия Go не ниже 1.27; go.mod, builders Docker, CI и локальная инструкция согласованы
- [ ] #2 Исходный Go-код использует стандартный uuid вместо google/uuid; сохранены UUIDv4, нулевые значения и текстовый формат существующих идентификаторов
- [ ] #3 Пройдены тесты совместимости JSON/JWT/Kafka/Redis, validator и записи/чтения UUID через PostgreSQL pgx/scany, включая nullable значения
- [ ] #4 Удалены ненужные прямые зависимости google/uuid, обновлены моки закреплённым генератором и пройден make check с интеграционными тестами
<!-- AC:END -->
