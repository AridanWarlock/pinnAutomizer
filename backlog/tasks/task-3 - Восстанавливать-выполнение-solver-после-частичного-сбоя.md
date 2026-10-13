---
id: TASK-3
title: Восстанавливать выполнение solver после частичного сбоя
status: To Do
assignee: []
created_date: '2026-10-13 19:18'
labels: []
dependencies: []
priority: high
type: enhancement
ordinal: 3000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
At-most-once чтение и отсутствие устойчивой попытки запуска могут оставить задачу в очереди или running после остановки процесса либо ошибки публикации результата.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Повтор сообщения не запускает второе вычисление одной попытки
- [ ] #2 Завершение попытки восстанавливается после сбоя публикации результата
- [ ] #3 Интеграционные проверки покрывают остановку между запуском и завершением
<!-- AC:END -->
