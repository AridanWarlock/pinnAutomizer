include .env
export

export PROJECT_ROOT=$(shell pwd)

ps:
	@docker ps

gateway-env-up:
	@docker compose up -d redis

gateway-env-down:
	@docker compose down redis

auth-env-up:
	@docker compose up -d pinn-postgres-auth redis kafka

auth-env-down:
	@docker compose down pinn-postgres-auth redis kafka

tasks-env-up:
	@docker compose up -d pinn-postgres-tasks redis kafka

tasks-env-down:
	@docker compose down pinn-postgres-tasks redis kafka

env-cleanup:
	@read -p "Очистить все volume файлы окружения? Опасность утери данных. [y/n]: " ans; \
	if [ !"$$ans" = "y" ]; then \
  	  echo "Очистка окружения отменена"; \
  	  exit 0; \
  	fi; \
	docker compose down \
		pinn-postgres-auth  auth-postgres-port-forwarder \
		pinn-postgres-tasks  tasks-postgres-port-forwarder \
	 	redis kafka && \
	rm -rf ${PROJECT_ROOT}/out/auth/pgdata ${PROJECT_ROOT}/out/tasks/pgdata \
		${PROJECT_ROOT}/out/redis_data ${PROJECT_ROOT}/out/kafka && \
	echo "Файлы окружения очищены"

auth-postgres-port-forward:
	@docker compose up -d auth-postgres-port-forwarder

auth-postgres-port-close:
	docker compose down auth-postgres-port-forwarder

tasks-postgres-port-forward:
	@docker compose up -d tasks-postgres-port-forwarder

tasks-postgres-port-close:
	docker compose down tasks-postgres-port-forwarder

kafka-ui-up:
	@docker compose up -d kafka-ui

kafka-ui-down:
	@docker compose down kafka-ui

auth-goose-create:
	@if [ -z "$(name)" ]; then \
		echo "Отсутствует необходимый параметр name. Пример: make auth-goose-create name=init"; \
		exit 1; \
	fi; \
	docker compose run --rm \
		-e GOOSE_COMMAND=create \
		-e GOOSE_COMMAND_ARG="$(name) sql" \
		goose-auth

auth-goose-up:
	@docker compose run --rm goose-auth


tasks-goose-create:
	@if [ -z "$(name)" ]; then \
		echo "Отсутствует необходимый параметр name. Пример: make tasks-goose-create name=init"; \
		exit 1; \
	fi; \
	docker compose run --rm \
		-e GOOSE_COMMAND=create \
		-e GOOSE_COMMAND_ARG="$(name) sql" \
		goose-tasks

tasks-goose-up:
	@docker compose run --rm goose-tasks

swagger-gen:
	@if [ -z "$(service)" ]; then \
		echo "Отсутствует необходимый параметр service. Пример: make swagger-gen service=auth"; \
		exit 1; \
	fi; \
	docker compose run --rm \
		-v ${PROJECT_ROOT}:/code \
		swagger init \
		-g cmd/main.go \
		-o services/$(service)/docs \
		-dir services/$(service) \
		--parseInternal \
		--parseDependency \
		--parseDepth 2
#		--quiet

swagger-fmt:
	@if [ -z "$(service)" ]; then \
		echo "Отсутствует необходимый параметр service. Пример: make swagger-gen service=auth"; \
		exit 1; \
	fi; \
	@docker compose run --rm \
		-v ${PROJECT_ROOT}/services/$(service):/code \
		swagger fmt

mockery:
	@docker compose run --rm \
    		-v $(shell go env GOCACHE):/root/.cache/go-build \
    		-v $(shell go env GOMODCACHE):/go/pkg/mod \
    		-e GOCACHE=/root/.cache/go-build \
    		mockery


gateway-run:
	@docker compose up --build pinn-gateway

gateway-shutdown:
	@docker compose down pinn-gateway

auth-run:
	@docker compose up --build pinn-auth

auth-shutdown:
	@docker compose down pinn-auth

tasks-run:
	@docker compose up --build pinn-tasks

tasks-shutdown:
	@docker compose down pinn-tasks
