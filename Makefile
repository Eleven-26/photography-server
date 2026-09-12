.PHONY: run run-dev build test tidy docker-up docker-up-debug docker-down docker-build

# Go 构建标签：本地开发默认带 debug（编译基础设施调试接口 /test/redis|nats|es|mongo|jaeger
# 与 presentation/controller/test.go）；构建"纯净二进制"（等价生产镜像）用：make build GO_TAGS=
# 注意：只有 debug 标签在编译期摘除基础设施调试接口；/test/config/encrypt（密文生成）始终保留。
GO_TAGS ?= debug
GO_TAGS_FLAG = $(if $(GO_TAGS),-tags $(GO_TAGS),)

run:
	go run $(GO_TAGS_FLAG) ./cmd/server -c config/config.yaml

# 本地 dev：Go 进程不会自动读 .env（那是 docker compose 的能力），
# 这里手动 source 后启动，否则 config.yaml 的 nacos.username/password 为空 → 拉取配置 401。
run-dev:
	@if [ -f .env ]; then set -a; . ./.env; set +a; else echo "⚠️ 未找到 .env，按 config.yaml 原样启动"; fi; \
	go run $(GO_TAGS_FLAG) ./cmd/server -c config/config.yaml -p dev

build:
	go build $(GO_TAGS_FLAG) -o bin/photography-server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker compose build

docker-up:
	docker compose up -d

# 连带启动可选调试组件（Elasticsearch / MongoDB）——二者在 debug profile，默认 up 不启动
docker-up-debug:
	docker compose --profile debug up -d

docker-down:
	docker compose down
