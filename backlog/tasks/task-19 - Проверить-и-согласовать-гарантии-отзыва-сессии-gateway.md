---
id: TASK-19
title: Проверять сессию в Redis на каждом защищённом запросе
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-24 21:19'
updated_date: '2026-10-24 21:27'
labels: []
dependencies: []
priority: high
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Gateway кеширует RedisSession на 30 секунд независимо от TTL Redis и удаления в auth. Гарантия отзыва и поведение при недоступном Redis требуют согласования.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Текущее поведение после истечения и удаления сессии воспроизведено на одноразовом Redis
- [x] #2 Предложены явные границы отзыва и поведения при сбое Redis; выбор пользователя отражён перед изменением runtime
- [x] #3 Согласованная гарантия покрыта адресными проверками без изменения PINN API
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Аудит Redis expire/delete выполнен. 2. Согласовано: прямой Redis без L1, backend error 503. 3. Обновить runtime и разделение ошибок, покрыть HTTP unit и Redis-интеграциями двух gateway. 4. make fmt/check, PR на review.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
На одноразовом Redis 7 воспроизведены: L1 возвращает сессию после DEL и естественного TTL expiry; прогретый L1 не читает закрытый Redis client. JWT не содержит exp. Аудит и варианты решения — docs/session-auth.md. Ожидается выбор пользователя: наличие L1 и 503/401 при сбое Redis; runtime не изменён.

Пользователь согласовал: убрать L1 из авторизации; читать Redis на каждом защищённом запросе. При недоступном Redis запрещать доступ с 503, отсутствующая сессия — 401. Уже авторизованные запросы продолжаются.

Согласованный вариант реализован: прямой Redis в gateway, отсутствие сессии 401, backend error 503. make fmt/check с PINN_TEST_REDIS_ADDR и адресный -race -count=3 прошли. HTTP-интеграции: два gateway, DEL отдельным клиентом, TTL expiry, обрыв TCP через proxy. CI получил одноразовый Redis. Гонки refresh/logout и Redis failover остаются вне гарантии.
<!-- SECTION:NOTES:END -->
