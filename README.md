<h1>Миграции:</h1>

1. Установить goose для миграций\
   go install github.com/pressly/goose/v3/cmd/goose@latest
2. goose -version
3. Накатить миграции:\
   goose up
4. Создать миграцию\
   goose create <name_of_migration> sql
5. Проверить статус миграций\
   goose status

<h1>PostgreSQL:</h1>

1. Установи docker и запусти его (лучше прям gui приложением)
2. Залезь в postgres.conf и поставь порт на любой, отличный от 5432
3. Останови текущий процесс постгри (версию psql заменить на свою): pg_ctl -D /usr/local/var/postgresql@14 stop\
4. Создай .env файл
5. Подгрузи переменные окружения: source .env
6. docker compose up -d
7. Теперь можешь подключаться через pgAdmin4:\
   Host: localhost\
   Port: 5432\
   Maintenance database: eazy_subtitle\
   Username: eazy_admin
