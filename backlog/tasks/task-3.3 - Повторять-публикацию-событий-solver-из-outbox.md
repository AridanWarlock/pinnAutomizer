---
id: TASK-3.3
title: Повторять публикацию событий solver из outbox
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 10:46'
updated_date: '2026-10-14 11:29'
labels: []
dependencies: []
parent_task_id: TASK-3
priority: high
type: enhancement
ordinal: 12000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Итог должен сохраняться вместе с событием; отказ Kafka или потерянное подтверждение публикации не должны требовать повторного вычисления. Outbox сохраняет порядок событий попытки и стабильный ID события.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Событие удаляется только после успешной публикации, сбой Kafka оставляет его в журнале
- [x] #2 После перезапуска публикуется сохранённое событие со стабильным ключом и заголовками
- [x] #3 Частичный сбой публикации не теряет последующие события, shutdown ожидает worker
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Добавить синхронный outbox worker с последовательной публикацией, стабильным payload и удалением только после подтверждения. Дисковые тесты воспроизводят неоднозначное подтверждение, перезапуск и отмену; подключение к runtime вынести в следующий этап.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
По review пользователя DI Repository/Writer переименованы в SQLite/Kafka; поля и параметры sqlite/kafka. Поведение не изменено.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлен outbox usecase: последовательная синхронная публикация и удаление только после подтверждения; Run принадлежит supervisor и возвращается после отмены writer. Дисковые тесты проверяют потерянное подтверждение с повтором после reopen, стабильные key/value/headers, частичный сбой и shutdown. make check и race solver проходят. Настоящий Kafka не запускался; подключение worker к runtime выполняется следующим этапом.
<!-- SECTION:FINAL_SUMMARY:END -->
