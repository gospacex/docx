# DbxWireJaeger Makefile
#
# 一键启动 legoB 的 mysql + jaeger 并运行 go test -count=1 -race ./...
# 测试结束不会 down 服务,允许其他会话复用 legoB。
# legoB min profile 默认含 mysql + jaeger,本 Makefile 不需要 --profile。

LEGOB_DIR      := /Users/hyx/work/gowork/src/legoB
COMPOSE_FILE   := $(LEGOB_DIR)/docker-compose.yml
READY_TIMEOUT  := 60
READY_INTERVAL := 2

.PHONY: test wait-mysql wait-jaeger run-tests status down

test: up wait-mysql wait-jaeger run-tests

up:
	docker compose -f $(COMPOSE_FILE) up -d mysql jaeger

wait-mysql:
	@echo "Waiting for mysql to be ready (max $(READY_TIMEOUT)s)..."
	@for i in $$(seq 1 $$(( $(READY_TIMEOUT) / $(READY_INTERVAL) ))); do \
		status=$$(docker compose -f $(COMPOSE_FILE) ps --format json mysql 2>/dev/null | grep -o '"Health":"[^"]*"' | head -1); \
		if echo "$$status" | grep -q healthy; then \
			echo "mysql ready ($$status)"; \
			exit 0; \
		fi; \
		sleep $(READY_INTERVAL); \
	done; \
	echo "mysql not ready after $(READY_TIMEOUT)s"; \
	docker compose -f $(COMPOSE_FILE) ps mysql; \
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
	docker compose -f $(COMPOSE_FILE) ps jaeger; \
	exit 1

run-tests:
	# -count=1 禁用 Go 测试结果缓存,确保每次 make test 都真正重跑;
	# 5 个用例并向 jaeger 上报新 span;-race 启用数据竞争检测。
	GOWORK=off go test -count=1 -race ./...

status:
	docker compose -f $(COMPOSE_FILE) ps

# 仅在需要本地完全清理时使用(注意:会影响其他复用 legoB 的会话)
down:
	docker compose -f $(COMPOSE_FILE) stop mysql jaeger
