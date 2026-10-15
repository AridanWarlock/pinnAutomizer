---
id: TASK-6
title: Обеспечить управляемую остановку HTTP и фоновых workers
status: Done
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-13 19:23'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
HTTP Shutdown наследует отменённый контекст. Solver записывает общий err из двух goroutine; ошибка consumer оставляет процесс без работающей очереди.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Активный HTTP-запрос получает время на завершение при отмене родительского контекста
- [x] #2 Зависимости закрываются после остановки workers
- [x] #3 Ошибка критического consumer приводит к управляемой остановке приложения
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. HTTP Shutdown использует отдельный ограниченный контекст. 2. Подключить HTTP и Kafka workers к errgroup в auth/tasks; solver ждёт errgroup и не пишет общий err. 3. Outbox Close ждёт завершения собственной goroutine. 4. Проверить HTTP shutdown с активным запросом и ожидание outbox на тестах, затем общий go test и race.

Общий небольшой pkg/lifecycle закрепляет одинаковое поведение errgroup для auth/tasks/solver и позволяет проверить отмену, неожиданный выход и ожидание cleanup без реальной Kafka.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Общий go test пяти модулей проходит. go test -race ./pkg/cache ./pkg/lifecycle ./pkg/httpsrv ./services/tasks/internal/outbox ./services/tasks/internal/usecases/v1/tasks/create проходит. HTTP-тест проверяет завершение активного настоящего запроса при отменённом родительском контексте. Lifecycle-тест проверяет отмену второго worker при ошибке первого и ожидание cleanup. Outbox-тест проверяет ожидание транзакции в Close. Локальный HTTP listener требовал выхода из sandbox для проверки. Реальные Kafka/Docker сервисы не запускались.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
HTTP shutdown использует отдельный ограниченный контекст. Auth/tasks/solver останавливают workers при критической ошибке и ждут их перед закрытием зависимостей; общий err solver устранён. Outbox Close ждёт завершения работы. Подтверждено общими тестами и race detector выбранных пакетов.
<!-- SECTION:FINAL_SUMMARY:END -->
