---
id: TASK-4
title: Защитить состояния задач от повторов и конкурирующего удаления
status: Done
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-13 19:35'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 4000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Повтор события onRun попадает в retry, afterRun может перезаписать результат, а параллельные run/delete не защищены одной транзакцией вокруг чтения.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Повтор уже применённого события завершается успешно без изменения результата
- [x] #2 Завершение допускает обратный порядок доставки начала и конца
- [x] #3 Конкурирующее удаление и запуск не удаляют данные принятой к выполнению задачи
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Выразить допустимые переходы и идемпотентные повторы методами domain.Task. 2. Добавить отдельные repository-методы чтения FOR UPDATE для изменения задач; run/delete/onRun/afterRun используют одну блокировку строки внутри транзакции. 3. Завершённый результат не перезаписывать; завершение разрешить из in_queue и running. 4. Файлы удалять только после успешного commit удаления. 5. Проверить повторы unit-тестами и конкуренцию интеграционными тестами на отдельном временном PostgreSQL.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Unit-тесты domain.Task и удаления после commit проходят. Интеграционные тесты на отдельном PostgreSQL 14.19 проверили оба порядка run/delete, реальное ожидание блокировки через pg_stat_activity, повтор onRun, обратный порядок start/finish и конкурирующие противоречивые завершения. go test -race для domain, delete и postgres проходит. Действующие БД не затрагивались. Очистка файлов теперь после commit, но её повторное выполнение после остановки процесса требует отдельного механизма cleanup.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Переходы определены в domain.Task; изменяющие usecase-ы используют FOR UPDATE внутри транзакции. Первый финальный результат сохраняется при повторах, run/delete сериализуются. Файлы удаляются после commit. Подтверждено unit, PostgreSQL integration и race-проверками.
<!-- SECTION:FINAL_SUMMARY:END -->
