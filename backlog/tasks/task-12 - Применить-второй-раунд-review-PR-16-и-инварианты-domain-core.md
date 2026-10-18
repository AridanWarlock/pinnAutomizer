---
id: TASK-12
title: Применить второй раунд review PR 16 и инварианты domain/core
status: Done
assignee:
  - '@codex'
created_date: '2026-10-17 16:54'
updated_date: '2026-10-17 17:16'
labels: []
dependencies: []
ordinal: 17000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Новые замечания требуют явных доменных инвариантов, читаемого SQLite API и batch outbox. Docker теперь доступен для изолированных интеграций. Исправления остаются в PR 16, без смены PINN runtime.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Новые нерешённые замечания PR обработаны с ответами Codex и проверками
- [x] #2 Domain/core конструкторы и методы защищают валидность; понимание и ограничения отражены в docs/AGENTS
- [x] #3 SQLite репозиторий разделён, типы/конверторы/моки согласованы с API и тестами
- [x] #4 make check/race и подходящие Docker интеграции выполнены без рабочего окружения и PINN
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Исправить gitignore и документацию инвариантов; уточнить id_key/event_id. 2. Ввести проверяемые limit/timestamp/message API, обновить domain attempt и persistence mapping. 3. Разделить SQLite методы, добавить узкий интерфейс и понятные результаты повторяемых операций. 4. Batch outbox с корректной обработкой частичного ACK, валидация input/JSON. 5. Моки, unit/race, изолированные Docker PostgreSQL/Kafka/goose, commit/push и ответы review.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Обработаны API-критерии: limit.Limit с закрытым положительным значением; Attempt.Validate/RestoreAttempt и методы переходов; Created/Changed вместо безымянных bool; SQLite интерфейс у адаптера и перегенерированные Mockery 3.7.1 моки; методы разделены на attempts/outbox repository; INSERT RETURNING на новом запросе и проверка совпадения replay в транзакции. UUID передаются как driver.Valuer, SQL-колонки вынесены в константы. shared SQLTime реализует Scan/Value для прежних INTEGER nanoseconds, eventRow/messageRow отвечают за преобразования. Batch outbox удаляет только подтверждённые позиции, ошибки Kafka переводит адаптер без driver types в usecase. Возвращены IDE ignores, удалён docker ignore. Docker Kafka + Goose lock/up интеграции прошли; полный check с Docker PostgreSQL выполняется. Экспортируемые поля legacy-моделей пока не дают compile-time защиты от обхода конструкторов; это будет явно отражено в docs.

Финальные проверки: make check с PINN_TEST_POSTGRES_DSN Docker PostgreSQL, PINN_TEST_KAFKA_ADDR и PINN_TEST_DOCKER=1 прошёл; race для core/sqlite/kafka/solver/tasks-outbox прошёл. Отдельный Linux test binary внутри Docker подтвердил ownership/pragmas, concurrent register, finish rollback/replay/reopen, start intent и SIGKILL/reopen. Stock Goose 3.27.0 уже содержит flock: inline build удалён, wrapper монтируется readonly; Up/lock/repeat Up проверены реальным контейнером. Kafka happy batch в две темы проверен реальным broker; частичные/неизвестные ACK покрыты unit. Core KafkaMessage конструкторы теперь возвращают error и валидируют сообщение, копируют буферы/headers; все call sites обновлены. JSON decoder pkg/httpin привязан к HTTP, в Kafka оставлен json.Unmarshal + Input.Validate. Domain/core правила и их пределы записаны в AGENTS/docs; id_key/event_id namespace объяснён. Mockery 3.7.1 перегенерирован локально; Docker image 3.7 содержит 3.7.0/Go1.26.1, прямой запуск без compose-workspace не использован для генерации. Runtime solver и PINN API не переключались.

Контракты и unit/race исправления оформлены отдельным коммитом; Docker Kafka/Goose интеграции и упрощение Compose — следующим. Осталось опубликовать docs и ответы во всех 29 обсуждениях, не resolve. IDE ignore возвращён, docker ignore удалён. Генерация mockery локальным v3.7.1 успешна; проверено отсутствие несвязанных изменений generated files.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Все 29 новых нерешённых замечаний PR 16 получили ответы [Codex], без resolve. 5d3d554: domain/core конструкторы/валидация, методы Attempt, typed outcomes/limit, shared timestamp, SQL интерфейс/файлы/row converters и batch outbox. 6d7fcbe: Docker Kafka/Goose интеграции, готовый Goose image без лишней сборки. 8f9d9b5: правила доменных инвариантов, разграничение DTO/snapshot/entity, id_key/event_id и реальные пределы гарантий в docs/AGENTS; исправлен gitignore. Полный make check с Docker PostgreSQL/Kafka/Goose и адресные race прошли. Отдельный Linux Docker-прогон проверил SQLite ownership, конкуренцию, rollback/reopen, intent и SIGKILL. PINN и runtime solver не переключались, рабочие volumes не использовались; PR ждёт review.
<!-- SECTION:FINAL_SUMMARY:END -->
