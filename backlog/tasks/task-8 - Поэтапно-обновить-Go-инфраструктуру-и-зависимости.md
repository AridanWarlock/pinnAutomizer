---
id: TASK-8
title: 'Поэтапно обновить Go, инфраструктуру и зависимости'
status: In Progress
assignee: []
created_date: '2026-10-14 09:57'
updated_date: '2026-10-25 10:24'
labels: []
dependencies: []
references:
  - 'https://go.dev/doc/go1.27'
priority: medium
type: enhancement
ordinal: 8000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Проект использует Go 1.26.1, Kafka 4.1.2, Redis 7, PostgreSQL 18.1 и отдельные версии SDK/генераторов. Нужен управляемый переход на поддерживаемые стабильные версии с проверкой совместимости и сохранности данных. В Go 1.27 подтверждён стандартный пакет uuid; переход с google/uuid требует отдельной проверки сериализации и pgx.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Составлена матрица текущих и целевых версий Go, Kafka, Redis, PostgreSQL, Docker SDK/образов, Go-зависимостей, goose, mockery и CI-инструментов с официальными источниками
- [ ] #2 Обновления разделены на небольшие изменения; закреплены конкретные версии и проверены условия совместимости, backup и ограничения отката для компонентов с данными
- [ ] #3 go.mod, Dockerfiles, compose, CI и инструкции согласованы; моки перегенерированы закреплённым генератором
- [ ] #4 Пройдены make check, необходимые race и интеграционные проверки; описаны изменения поведения и процедура rollout без автоматического изменения рабочей БД
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Go 1.27.3 выполняется отдельно в TASK-8.2; стандартный UUID остаётся TASK-8.1. Остальные обновления инфраструктуры не входят в текущий PR; после review вернуться к задачам auth.
<!-- SECTION:NOTES:END -->
