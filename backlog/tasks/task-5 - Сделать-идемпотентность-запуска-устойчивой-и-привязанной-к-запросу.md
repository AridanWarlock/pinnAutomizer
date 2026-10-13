---
id: TASK-5
title: Сделать идемпотентность запуска устойчивой и привязанной к запросу
status: Done
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-13 19:42'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 5000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Redis pending имеет несовместимый формат чтения; глобальный ключ не привязан к пользователю и задаче, а commit БД и completed Redis разделены окном сбоя.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Параллельный повтор pending не вызывает ошибку декодирования
- [x] #2 Одинаковый ключ с другими параметрами даёт конфликт
- [x] #3 Повтор после commit и сбоя Redis возвращает сохранённый результат без нового запуска
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Сохранять принятый запрос запуска в PostgreSQL по паре user_id/idempotency_key в одной транзакции со статусом и outbox. 2. Повтор с той же задачей возвращает успех, другая задача даёт конфликт; конкурентные одинаковые ключи сериализуются уникальным ограничением. 3. Использовать отдельный UUID события вместо глобального клиентского ключа. 4. Убрать Redis из пути запуска и startup tasks: PG уже обеспечивает координацию, Redis добавляет окно сбоя. Исправить формат pending общего Redis store и проверить его через узкий интерфейс. 5. Добавить миграцию и интеграционные проверки повторов, конкуренции, отката публикации и повторного запроса после удаления outbox-события; действующую БД не мигрировать.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Принятый запрос запуска хранится в task_run_requests: уникальная пара user_id/id_key, отдельный UUID события, отсутствие FK к удаляемой задаче. Redis убран из tasks startup/config/compose и пути запуска: сбой Redis не участвует в исходе этой операции. Общий Redis store сохраняет pending как envelope; 16 конкурентных JSON-повторов и round-trip completed проверены unit-тестами. Ключ ограничен 256 байтами в domain и SQL, чтобы размер пользовательского ключа был ограничен и не приводил к ошибкам индекса. Общий go test пяти модулей с PostgreSQL integration проходит; race для postgres/domain/delete/Redis store проходит. Интеграционные тесты применили новую миграцию в собственных временных БД PostgreSQL 14.19 и покрыли область пользователя, другой task_id, конкуренцию, rollback outbox, потерю подтверждения commit и повтор после удаления события/задачи. Рабочая БД не мигрировалась. Условия внедрения и ограничения описаны в services/tasks/README.md; старые запросы не backfill-ятся и retention записей пока бессрочный.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Устойчивая идемпотентность запуска обеспечена PostgreSQL-транзакцией вместе со статусом и outbox. Повтор возвращает успех, другой task_id — конфликт; Redis больше не нужен сервису tasks. Исправлен общий JSON-формат pending. Подтверждено общими тестами, PostgreSQL integration и race detector; перед запуском обновлённого сервиса нужна новая миграция.
<!-- SECTION:FINAL_SUMMARY:END -->
