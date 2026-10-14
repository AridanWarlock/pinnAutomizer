---
id: TASK-3.2
title: Принимать команды solver в устойчивый журнал
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 10:46'
updated_date: '2026-10-14 10:58'
labels: []
dependencies: []
parent_task_id: TASK-3
priority: high
type: enhancement
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Kafka consumer должен подтверждать принятие только после сохранения попытки и не запускать вычисление повторно. Новый usecase принятия команды сохраняет стабильный ID события, не меняя API уравнения.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Usecase и consumer валидируют UUID run_id, task_id и mode до записи
- [x] #2 Повтор команды возвращает успех без вычисления; конфликт не маскируется успехом
- [x] #3 Сбой записи возвращается вызывающему consumer и покрыт тестами
- [x] #4 Моки сгенерированы закреплённой версией, make check проходит
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Добавить отдельный usecase принятия команды с проверкой идентификаторов и параметров. Проверить реальные повторы и ошибки записи на SQLite; не подключать новый consumer до проверки восстановления Docker.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлены отдельные usecase/consumer принятия команды с run_id из X-Idempotency-Key. Валидация UUID/mode предшествует записи; повтор возвращает сохранённую попытку, конфликт и ошибка записи доходят до consumer. Реальный reopen/replay проверен на SQLite; мок Repository сгенерирован mockery v3.7.1 и включён в Git. make check и race solver проходят. Новый consumer пока не подключён к runtime; rollout старых сообщений вынесен в продолжение TASK-3.
<!-- SECTION:FINAL_SUMMARY:END -->
