QUALITY_TARGETS := tools-install fmt fmt-check lint test test-race check
# Quality checks do not depend on local service credentials.
ifneq ($(filter-out $(QUALITY_TARGETS),$(or $(MAKECMDGOALS),kafka)),)
include .env
export
endif

ifeq ($(OS),Windows_NT)
    PROJECT_ROOT := $(shell pwd -W 2>/dev/null || pwd)
    PROJECT_ROOT := $(subst \,/,$(PROJECT_ROOT))
    PROJECT_ROOT := $(shell echo $(PROJECT_ROOT) | sed 's|^/c/|C:/|')
else
    PROJECT_ROOT := $(shell pwd)
endif

export PROJECT_ROOT

.PHONY: ps \
	kafka \
	gateway-env-up gateway-env-down \
	auth-env-up auth-env-down \
	tasks-env-up tasks-env-down \
	solver-env-up solver-env-down \
	env-cleanup \
	auth-postgres-port-forward auth-postgres-port-close \
	tasks-postgres-port-forward tasks-postgres-port-close \
	kafka-ui-up kafka-ui-down \
	auth-goose-create auth-goose-up \
	tasks-goose-create tasks-goose-up \
	solver-goose-create solver-goose-up \
	swagger-gen swagger-fmt \
	mockery \
	gateway-run gateway-shutdown \
	auth-run auth-shutdown \
	tasks-run tasks-shutdown \
	solver-run solver-shutdown

kafka:
	@docker compose up kafka-init

ps:
	@docker ps

gateway-env-up:
	@docker compose up -d redis kafka

gateway-env-down:
	@docker compose down redis kafka

auth-env-up:
	@docker compose up -d pinn-postgres-auth redis kafka

auth-env-down:
	@docker compose down pinn-postgres-auth redis kafka

tasks-env-up:
	@docker compose up -d pinn-postgres-tasks redis kafka-init

tasks-env-down:
	@docker compose down pinn-postgres-tasks redis kafka

solver-env-up:
	@docker compose up -d kafka-init

solver-env-down:
	@docker compose down kafka

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

solver-goose-create:
	@if [ -z "$(name)" ]; then \
		echo "Отсутствует name. Пример: make solver-goose-create name=add_index"; \
		exit 1; \
	fi
	@docker compose run --rm --build --no-deps \
		-e GOOSE_COMMAND=create -e GOOSE_COMMAND_ARG="$(name) sql" goose-solver

solver-goose-up:
	@docker compose run --rm --build --no-deps goose-solver

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

solver-run:
	@docker compose up --build pinn-solver

solver-shutdown:
	@docker compose down pinn-solver

deploy:
	@docker compose up -d --build

undeploy:
	@docker compose down

.PHONY: $(QUALITY_TARGETS)
QUALITY_BIN := .tools/quality
QUALITY_SOURCES := $(wildcard pkg/cmd/quality/*.go)

$(QUALITY_BIN): $(QUALITY_SOURCES) pkg/go.mod Makefile
	@mkdir -p .tools
	@cd pkg && GOWORK=off go build -trimpath -ldflags="-s -w" -o ../$(QUALITY_BIN) ./cmd/quality

tools-install: $(QUALITY_BIN)
	@$(QUALITY_BIN) install

fmt: $(QUALITY_BIN)
	@$(QUALITY_BIN) fmt

fmt-check: $(QUALITY_BIN)
	@$(QUALITY_BIN) fmt-check

lint: $(QUALITY_BIN)
	@$(QUALITY_BIN) lint

test: $(QUALITY_BIN)
	@$(QUALITY_BIN) test $(TEST_ARGS)

test-race: $(QUALITY_BIN)
	@$(QUALITY_BIN) test -race $(if $(TEST_ARGS),$(TEST_ARGS),./pkg/... ./services/auth/... ./services/tasks/... ./services/solver/... ./services/gateway/...)

check: $(QUALITY_BIN)
	@$(QUALITY_BIN) check
