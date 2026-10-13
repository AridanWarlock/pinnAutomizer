---
id: TASK-2
title: Ограничить загрузку файлов до разбора multipart
status: Done
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-13 19:20'
labels: []
dependencies: []
priority: high
type: bug
ordinal: 2000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
FormValue запускает разбор multipart до установки лимита body. При ошибке открытия следующего обязательного файла уже открытые файлы не закрываются; train повторно читает data.mat.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Превышение лимита отклоняется до вызова usecase и проверено HTTP-тестом
- [x] #2 Корректная multipart-загрузка вызывает usecase с уникальным набором обязательных файлов
- [x] #3 Открытые файлы закрываются при неполном наборе входных файлов
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Ограничить body до первого разбора multipart и уменьшить порог памяти до 8 MiB. 2. Удалить повтор data.mat из train и закрывать открытые файлы при ошибке сбора. 3. Проверить лимит настоящим HTTP multipart-запросом и корректную передачу файлов usecase-у; проверить закрытие файлов при частичной ошибке.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
go test ./services/tasks/internal/usecases/v1/tasks/create ./services/tasks/internal/domain проходит. Oversized HTTP-тест потоково передаёт более 200 MiB, подтверждает отсутствие вызова usecase и очистку временных файлов. Успешные тесты покрывают train/retrain/predict; отдельный тест подтверждает закрытие ранее открытого файла при отсутствии следующего.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Лимит body применяется до разбора multipart; память для файлов ограничена порогом 8 MiB. Устранены дублирование data.mat и утечка частично открытого набора. Проверено HTTP-тестами и тестом закрытия файлов.
<!-- SECTION:FINAL_SUMMARY:END -->
