.PHONY: run run-dev build test tidy docker-up docker-up-wait docker-up-debug docker-ps docker-logs-init docker-logs-backend docker-down docker-build

# Go 构建标签：本地开发默认带 debug（编译基础设施调试接口 /test/redis|nats|es|mongo|jaeger
# 与 presentation/controller/test.go）；构建"纯净二进制"（等价生产镜像）用：make build GO_TAGS=
# 注意：只有 debug 标签在编译期摘除基础设施调试接口；/test/config/encrypt（密文生成）始终保留。
GO_TAGS ?= debug
GO_TAGS_FLAG = $(if $(GO_TAGS),-tags $(GO_TAGS),)

run:
	go run $(GO_TAGS_FLAG) ./cmd/server -c config/config.yaml

# 本地 dev：Go 进程不会自动读 .env（那是 docker compose 的能力），
# 这里手动 source 后启动，否则 config.yaml 的 nacos.username/password 为空 → 拉取配置 401。
# ⚠️ .env 里的 APP_NACOS_SERVER_ADDR 是【容器内】地址（compose 网络里的 nacos:8848），
#     host 跑 Go 解析不了该主机名，故 source 之后强制覆盖为本机映射端口 127.0.0.1:${NACOS_PORT}。
run-dev:
	@if [ -f .env ]; then set -a; . ./.env; set +a; else echo "⚠️ 未找到 .env，按 config.yaml 原样启动"; fi; \
	export APP_NACOS_SERVER_ADDR="127.0.0.1:$${NACOS_PORT:-8848}"; \
	echo "→ Nacos 地址（host 视角）: $$APP_NACOS_SERVER_ADDR"; \
	go run $(GO_TAGS_FLAG) ./cmd/server -c config/config.yaml -p dev

build:
	go build $(GO_TAGS_FLAG) -o bin/photography-server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker compose build

# ===== 容器编排 =====
# 一键启动：首次部署与日常启动是同一条命令 —— compose 内部按依赖分层等待：
#   mysql(建库表) → redis / nats / nacos(就绪) → nacos-init(初始化管理员+发布配置，跑完即退)
#     → backend → frontend / h5 / wechat
# 前置条件（首次）：
#   1) 镜像已在别处 build 并 push（2C2G 机器上跑 build 必 OOM）；
#   2) .env 已按 .env.example 填好（尤其 MYSQL_ROOT_PASSWORD / APP_NACOS_PASSWORD / KEK）；
#   3) 挂载用到的模板文件已就位（nats.conf / jaeger config / horizon.yaml 等，见 docs/部署/）。
docker-up:
	docker compose up -d

# 等到所有 healthcheck 通过才返回（Compose v2 --wait）；适合脚本里判断"真的起完了"。
# 注意：它解决不了数据级依赖（配置是否发布），那部分由 nacos-init 容器承担。
docker-up-wait:
	docker compose up -d --wait

# 启动结果一览：-a 会带上 nacos-init 这类【已退出】的一次性容器（Exit 0 表示初始化成功）
docker-ps:
	docker compose ps -a

# 排查"Nacos 没管理员 / 配置没发布"时先看它（正常应看到登录成功 + 推送或跳过）
docker-logs-init:
	docker compose logs nacos-init

docker-logs-backend:
	docker compose logs -f backend

# 连带启动可选调试组件（Elasticsearch / MongoDB）——二者在 debug profile，默认 up 不启动
docker-up-debug:
	docker compose --profile debug up -d

docker-down:
	docker compose down
