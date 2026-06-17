# CachexWireJaeger Makefile
#
# 一键启动 legoB 的 redis + jaeger 并运行 go test ./...
# 测试结束不会 down 服务,允许其他会话复用 legoB。

LEGOB_DIR      := /Users/hyx/work/gowork/src/legoB
COMPOSE_FILE   := $(LEGOB_DIR)/docker-compose.yml
READY_TIMEOUT  := 60
READY_INTERVAL := 2

.PHONY: test wait-redis wait-jaeger status down

test: up wait-redis wait-jaeger run-tests

up:
	docker compose -f $(COMPOSE_FILE) up -d redis jaeger

wait-redis:
	@echo "Waiting for redis to be ready (max $(READY_TIMEOUT)s)..."
	@for i in $$(seq 1 $$(( $(READY_TIMEOUT) / $(READY_INTERVAL) ))); do \
		if docker compose -f $(COMPOSE_FILE) exec -T redis redis-cli -a redis123456 ping 2>/dev/null | grep -q PONG; then \
			echo "redis ready"; \
			exit 0; \
		fi; \
		sleep $(READY_INTERVAL); \
	done; \
	echo "redis not ready after $(READY_TIMEOUT)s"; \
	docker compose -f $(COMPOSE_FILE) ps; \
	exit 1

wait-jaeger:
	@echo "Waiting for jaeger to be ready (max $(READY_TIMEOUT)s)..."
	@for i in $$(seq 1 $$(( $(READY_TIMEOUT) / $(READY_INTERVAL) ))); do \
		if curl -fsS http://localhost:16686/api/services >/dev/null 2>&1; then \
			echo "jaeger ready"; \
			exit 0; \
		fi; \
		sleep $(READY_INTERVAL); \
	done; \
	echo "jaeger not ready after $(READY_TIMEOUT)s"; \
	docker compose -f $(COMPOSE_FILE) ps; \
	exit 1

run-tests:
	# -count=1 disables Go's test result cache so every make test actually
	# re-executes the cases and emits fresh OTel spans into jaeger.
	go test -count=1 -race ./...

status:
	docker compose -f $(COMPOSE_FILE) ps

# 仅在需要本地完全清理时使用(注意:会影响其他复用 legoB 的会话)
down:
	docker compose -f $(COMPOSE_FILE) stop redis jaeger