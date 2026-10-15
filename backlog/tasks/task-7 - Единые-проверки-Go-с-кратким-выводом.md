---
id: TASK-7
title: Единые проверки Go с кратким выводом
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 08:59'
updated_date: '2026-10-14 09:14'
labels: []
dependencies: []
type: chore
ordinal: 7000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Импорты не имеют общей конвенции, а полный вывод успешных тестов расходует контекст. Нужны воспроизводимые проверки форматирования и линтера и вывод только ошибок тестов.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Форматирование и группировка импортов проверяются автоматически
- [x] #2 Закреплённый современный линтер запускается для пяти модулей
- [x] #3 Успешные тесты выводят итог, упавшие — только диагностику ошибок, полный лог сохраняется
- [x] #4 Правила и команды записаны в AGENTS.md и проверены перед коммитом
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Зафиксировать golangci-lint v2 с gofmt/gci и узким набором проверок корректности. 2. Добавить команды quality для fmt, lint и test с сохранением логов и краткими сообщениями. 3. Нормализовать форматирование, исправить подтверждённые замечания линтера. 4. Проверить success/failure вывод тестового runner-а, обновить AGENTS.md и закоммитить.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Установлен golangci-lint 2.14.0 с проверкой SHA256; config verify проходит. gofmt/gci автоматически нормализовали импорты проекта. Включены govet/ineffassign/staticcheck SA*/unused, исключены стилевые переписывания и отдельные deprecation-миграции. Исправлены три assert.Equal(nil,nil), которые не проверяли значение, и reflect.Ptr заменён на reflect.Pointer. make check проходит с отдельной временной PostgreSQL 14.19: fmt-check, lint, runner-tests, test. Настоящие проходящие/упавшие Go-тесты и ошибка сборки проверили отсутствие успешного шума и сохранение ненулевого exit code; stdout ограничен 20 строками, полные логи в out/quality. Проверена генерация workspace для клона без go.work, включая symlink-пути macOS; прямой gofmt -l всех Go-файлов пуст. CI с PostgreSQL 18.1 и artifact логов подготовлен, на GitHub ещё не выполнялся. Конвенции закреплены в AGENTS.md; skill не создавался, поскольку это постоянные правила репозитория.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Добавлены воспроизводимые команды fmt/fmt-check/lint/test/test-race/check, закреплённый линтер, краткий test runner с самопроверками и CI. При успехе лог не разбирается; при ошибке видны только нужные сообщения и путь полного лога. Проверено make check с PostgreSQL и gofmt; правила записаны в AGENTS.md.
<!-- SECTION:FINAL_SUMMARY:END -->
