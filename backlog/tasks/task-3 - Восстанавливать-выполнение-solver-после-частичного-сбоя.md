---
id: TASK-3
title: Восстанавливать выполнение solver после частичного сбоя
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-18 12:30'
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

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Следующий этап — отдельный PR до 2000 изменённых строк: идемпотентный Docker Create по run_id и labels; usecase запуска сохраняет start_requested до единственного Start; неопределённый Start не повторяется. Unit-тесты частичных сбоев/конкуренции и короткий Docker CPU-тест. Recovery, остановка и подключение runtime — следующие PR.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
План подготовлен для совместного разбора. Требуется подтвердить новое архитектурное решение: собственное устойчивое хранилище solver и консервативная обработка неоднозначного старта. Код TASK-3 пока не менялся; автоматический повтор вычисления и новый пользовательский API попыток не входят в текущую задачу.

После разбора с пользователем PostgreSQL для solver заменён в плане на локальную SQLite: текущий объём метаданных и один Docker-host не требуют отдельной серверной БД. Сохраняются атомарные ledger/outbox, стабильный run_id и запрет слепого повторного Start. Ограничения: постоянный локальный volume, WAL, один владелец журнала; потерю хоста/диска и общий журнал нескольких хостов это не покрывает. Основание: sqlite.org/whentouse.html, sqlite.org/wal.html и synchronous pragma. Код реализации пока не менялся.

Review checkpoint: TASK-3.1/3.2/3.3 завершены отдельными коммитами: SQLite ledger, acceptance consumer и outbox worker. Runtime пока использует прежний consumer/runner. make check, race для solver и Linux CGO_ENABLED=0 сборка solver и SQLite test binary проходят. Docker daemon недоступен, настоящий Kafka и восстановление контейнеров не проверены. Следующий этап: recovery usecase/worker, Docker create-inspect-start с защитой от повторного Start, runtime ownership/volume/shutdown и rollout legacy сообщений. До переключения рабочей доставки нужен review этой границы и Docker интеграционные проверки.

По согласованному review выполнено механическое переименование Kafka-обёрток обоих уровней: Consumer/Producer, ConsumerConfig/ProducerConfig, NewConsumer/NewProducer и WithProducer. Места подключения во всех четырёх сервисах обновлены. Env prefixes KAFKA_READER_/KAFKA_WRITER_ сохранены для совместимости развёртывания; библиотечные kafka-go.Reader/Writer остаются внутри segmentio-адаптера. Интерфейсы и сигнатуры методов зависимостей не изменены, существующие моки не требуют перегенерации.

По запросу пользователя env также переименован: KAFKA_CONSUMER_/KAFKA_PRODUCER_ в Go config и docker-compose.yaml. .env.example документирует новые service keys. В игнорируемом локальном .env старые Kafka prefixes отсутствуют; секреты не выводились. Старые env names больше не поддерживаются; при review учитывать обновление внешней конфигурации запуска.
<!-- SECTION:NOTES:END -->
