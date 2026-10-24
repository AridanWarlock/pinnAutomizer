---
id: TASK-17
title: Сократить комментарии во всех генерируемых моках
status: Done
assignee:
  - '@codex'
created_date: '2026-10-24 16:21'
updated_date: '2026-10-24 16:39'
labels: []
dependencies: []
priority: medium
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Шаблон matryer добавляет повторяющиеся примеры и комментарии, затрудняя review generated-кода. Перегенерация всех модулей выполняется единообразно; пользователь исключил generated-моки из лимита PR.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Локальный шаблон соответствует фактическому Mockery v3.7.0 штатной make-команды и сохраняет поведение и generated marker
- [x] #2 Все моки пяти модулей перегенерированы без изменения значимых Go-токенов
- [x] #3 Повторная штатная генерация воспроизводима, make check проходит
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Удалить только шумовые комментарии из копии закреплённого шаблона с сохранением лицензии. 2. Подключить шаблон общими настройками для пяти модулей и перегенерировать все моки. 3. Сравнить Go-токены всех файлов, повторить штатную генерацию и выполнить проверки.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Все 38 моков переведены на компактный локальный шаблон; значимые Go-токены совпадают с исходными. Два штатных make mockery дали побайтово одинаковый результат; make fmt и make check прошли. Фактический Docker Mockery v3.7.0 закреплён digest, версия не обновлялась. Generated mocks исключены из лимита PR по правилу пользователя.

PR на review: https://github.com/AridanWarlock/pinnAutomizer/pull/28 → codex/task-progress.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
PR #28 принят пользователем. Все 38 моков генерируются компактным шаблоном; Go-токены сохранены, повторная генерация побайтово идентична, make check прошёл.
<!-- SECTION:FINAL_SUMMARY:END -->
