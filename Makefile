include .env
export

export PROJECT_ROOT=$(shell pwd)

ps:
	@docker ps

env-up:
	@docker compose up -d pinn-postgres redis kafka

env-down:
	@docker compose down pinn-postgres redis kafka

env-cleanup:
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/n]: " ans; \
	if [ !"$$ans" = "y" ]; then \
  	  echo "Очистка окружения отменена"; \
  	  exit 0; \
  	fi; \
	docker compose down pinn-postgres postgres-port-forwarder redis && \
	rm -rf ${PROJECT_ROOT}/out/pgdata ${PROJECT_ROOT}/out/redis_data ${PROJECT_ROOT}/out/kafka && \
	echo "Файлы окружения очищены"

postgres-port-forward:
	@docker compose up -d postgres-port-forwarder

postgres-port-close:
	docker compose down postgres-port-forwarder

goose-create:
	@if [ -z "$(name)" ]; then \
		echo "Отсутствует необходимый параметр name. Пример: make goose-create name=init"; \
		exit 1; \
	fi; \
	docker compose run --rm \
		-e GOOSE_COMMAND=create \
		-e GOOSE_COMMAND_ARG="$(name) sql" \
		pinn-postgres-goose

goose-up:
	@docker compose run --rm pinn-postgres-goose

pinnapp-local-run:
	@docker compose up --build pinn-backend

pinnapp-local-shutdown:
	@docker compose down pinn-backend

pinnapp-deploy:
	@docker compose up -d --build pinn-backend

pinnapp-undeploy:
	@docker compose down pinn-backend

swagger-gen:
	@docker compose run --rm swagger \
		init \
		-g cmd/pinn/main.go \
		-o docs \
		--parseInternal \
		--exclude internal/usecases/**/**/*_test.go \
		--quiet
swagger-fmt:
	@docker compose run --rm swagger fmt

mockery:
	@docker compose run --rm \
    		-v $(shell go env GOCACHE):/root/.cache/go-build \
    		-v $(shell go env GOMODCACHE):/go/pkg/mod \
    		-e GOCACHE=/root/.cache/go-build \
    		mockery

kafka-ui-up:
	@docker compose up -d kafka-ui

kafka-ui-down:
	@docker compose down kafka-ui