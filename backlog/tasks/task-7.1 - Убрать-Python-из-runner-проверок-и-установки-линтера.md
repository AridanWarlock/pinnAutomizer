---
id: TASK-7.1
title: Убрать Python из runner проверок и установки линтера
status: Done
assignee:
  - '@codex'
created_date: '2026-10-14 12:33'
updated_date: '2026-10-14 12:40'
labels: []
dependencies: []
parent_task_id: TASK-7
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Проверки Go backend требуют отдельный Python runtime только ради служебного runner. Пользователь согласовал перенос на Go с сохранением тихого вывода, кодов ошибок и полных логов.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Makefile и CI запускают проверки без Python; бинарь runner собирается штатным Go
- [x] #2 Успех не разбирает лог, ошибки показывают только упавшие тесты или диагностику сборки с ограничением вывода
- [x] #3 Установщик сохраняет закреплённую версию линтера и проверку SHA256
- [x] #4 Go-тесты проверяют реальные успешные, упавшие и несобираемые тесты, workspace чистого clone и installer; make check проходит
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Разместить стандартный Go command в pkg/cmd/quality без новых зависимостей. Make собирает игнорируемый .tools/quality с GOWORK=off. Перенести запуск, workspace, диагностику и checksum installer; перенести Python-тесты на Go и удалить старые скрипты. Проверить make check, адресный race и сборку runner из clone без go.work.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Python runner и unittest удалены. Добавлен Go command pkg/cmd/quality только на стандартной библиотеке; make автоматически собирает игнорируемый .tools/quality с GOWORK=off и stripped binary. Сохранены тихий успех без разбора лога, выборка упавших тестов/ошибок сборки, 20 строк диагностики, exit codes, полный лог, workspace чистого clone и SHA256 installer. Go-тесты проверяют реальные pass/fail/build-fail, bootstrap без workspace, ограничение вывода, отмену, fmt diff при нулевом exit code, HTTP failure и защиту установленного линтера при неверной checksum. make tools-install реально загрузил v2.14.0; make check и адресный race проходят. GitHub workflow использует прежние make-команды и больше не требует Python; сам CI и PG интеграционные тесты в этом запуске не выполнялись.
<!-- SECTION:FINAL_SUMMARY:END -->
