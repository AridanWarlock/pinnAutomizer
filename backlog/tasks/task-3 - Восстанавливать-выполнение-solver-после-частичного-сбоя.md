---
id: TASK-3
title: Восстанавливать выполнение solver после частичного сбоя
status: In Progress
assignee:
  - '@codex'
created_date: '2026-10-13 19:18'
updated_date: '2026-10-23 07:48'
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
Runtime: открыть проверенную SQLite schema, собрать accept/start/observe/cleanup/recovery/outbox, обслуживать обе Kafka очереди at-least-once под lifecycle. Kafka success ack только после durable accept; retry/DLQ — существующая политика shared consumer. Закрывать зависимости после остановки workers. Добавить env/Compose, сборку всего cmd-пакета, проверки wiring/replay/shutdown и Docker CPU. Старых сообщений нет, cleanup ctx отменяемый.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
План подготовлен для совместного разбора. Требуется подтвердить новое архитектурное решение: собственное устойчивое хранилище solver и консервативная обработка неоднозначного старта. Код TASK-3 пока не менялся; автоматический повтор вычисления и новый пользовательский API попыток не входят в текущую задачу.

После разбора с пользователем PostgreSQL для solver заменён в плане на локальную SQLite: текущий объём метаданных и один Docker-host не требуют отдельной серверной БД. Сохраняются атомарные ledger/outbox, стабильный run_id и запрет слепого повторного Start. Ограничения: постоянный локальный volume, WAL, один владелец журнала; потерю хоста/диска и общий журнал нескольких хостов это не покрывает. Основание: sqlite.org/whentouse.html, sqlite.org/wal.html и synchronous pragma. Код реализации пока не менялся.

Review checkpoint: TASK-3.1/3.2/3.3 завершены отдельными коммитами: SQLite ledger, acceptance consumer и outbox worker. Runtime пока использует прежний consumer/runner. make check, race для solver и Linux CGO_ENABLED=0 сборка solver и SQLite test binary проходят. Docker daemon недоступен, настоящий Kafka и восстановление контейнеров не проверены. Следующий этап: recovery usecase/worker, Docker create-inspect-start с защитой от повторного Start, runtime ownership/volume/shutdown и rollout legacy сообщений. До переключения рабочей доставки нужен review этой границы и Docker интеграционные проверки.

По согласованному review выполнено механическое переименование Kafka-обёрток обоих уровней: Consumer/Producer, ConsumerConfig/ProducerConfig, NewConsumer/NewProducer и WithProducer. Места подключения во всех четырёх сервисах обновлены. Env prefixes KAFKA_READER_/KAFKA_WRITER_ сохранены для совместимости развёртывания; библиотечные kafka-go.Reader/Writer остаются внутри segmentio-адаптера. Интерфейсы и сигнатуры методов зависимостей не изменены, существующие моки не требуют перегенерации.

По запросу пользователя env также переименован: KAFKA_CONSUMER_/KAFKA_PRODUCER_ в Go config и docker-compose.yaml. .env.example документирует новые service keys. В игнорируемом локальном .env старые Kafka prefixes отсутствуют; секреты не выводились. Старые env names больше не поддерживаются; при review учитывать обновление внешней конфигурации запуска.

По согласованию Docker-слой выделен в отдельный PR https://github.com/AridanWarlock/pinnAutomizer/pull/18 (~500 строк): shared API без типов SDK, unit/race и Docker CPU lifecycle прошли, make check прошёл. Подключение PinnRunner и замечания #17 подготовлены локально; публикация второго этапа — после review и merge #18.

Этап безопасного запуска: PR https://github.com/AridanWarlock/pinnAutomizer/pull/17. После принятия #18 PinnRunner переводится на shared Docker API; runtime, recovery и cleanup остаются отдельными этапами.

После merge #18 PR #17 переведён на shared Docker API: интерфейс Docker и generated mock, fixture Attempt, методы стадий и приватное имя контейнера; инверсия AutoRemove исправлена. make check, адресные race и Docker CPU Create/replay/exited прошли. Runtime нового сценария по-прежнему не подключён.

Наблюдение и завершение — PR https://github.com/AridanWarlock/pinnAutomizer/pull/19 (~850 строк). make check, адресные race, SQLite rollback/reopen/concurrency и Docker CPU прошли. Runtime не подключён; следующий этап после review — recovery, затем deadline/cleanup.

Recovery — PR https://github.com/AridanWarlock/pinnAutomizer/pull/20 (~1,1 тыс. строк). make check, адресные race, SQLite pagination/reopen и Docker CPU intent/exited прошли. Worker не подключён в runtime; следующий этап — deadline/Stop/cleanup, затем wiring/rollout.

PR #21: https://github.com/AridanWarlock/pinnAutomizer/pull/21 — deadline и подтверждённый Docker Stop. make check, адресный race и Docker CPU integration (Stop → отказ outbox → повтор) прошли. Следом: устойчивый cleanup; runtime ещё не подключён.

PR #22: https://github.com/AridanWarlock/pinnAutomizer/pull/22 — durable cleaned_at, cleanup usecase, RemoveStopped без Force, recovery завершённых неочищенных попыток. make check и адресные race/SQLite/Docker CPU прошли, включая удаление → отказ записи → reopen → повтор и upgrade старой схемы. Runtime wiring остаётся следующим этапом.

PR #23: https://github.com/AridanWarlock/pinnAutomizer/pull/23 — перед runtime сохранён tasks.on.run: event + intent/deadline атомарны. make check, адресные race/SQLite/Docker CPU прошли (rollback/concurrency/reopen/replay после доставки). Cleanup-тест согласован с отменяемым ctx из #22. Следом runtime wiring; старых сообщений/данных нет, совместимость не требуется.
<!-- SECTION:NOTES:END -->
