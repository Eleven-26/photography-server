.PHONY: run run-dev build test tidy docker-up docker-down docker-build

run:
	go run ./cmd/server -c config/config.yaml

# 本地 dev：Go 进程不会自动读 .env（那是 docker compose 的能力），
# 这里手动 source 后启动，否则 config.yaml 的 nacos.username/password 为空 → 拉取配置 401。
run-dev:
	@if [ -f .env ]; then set -a; . ./.env; set +a; else echo "⚠️ 未找到 .env，按 config.yaml 原样启动"; fi; \
	go run ./cmd/server -c config/config.yaml -p dev

build:
	go build -o bin/photography-server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
