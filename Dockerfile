# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOFLAGS=-mod=mod

# SkyWalking Go agent（skywalking-go）注入构建开关：
# SW_AGENT_ENABLE=true 时以 -toolexec 织入构建产物（agent 为 build context 内 build/agent/ 目录的本地
# 二进制，随 COPY build/agent/ 带入镜像，无需联网下载），服务走 SkyWalking native 通道（Horizon「原生」模式
# 可见、具备 OAP 拓扑/指标分析）。
# 默认 false —— 纯 OTel / Jaeger 版：不注入，保留的 OTel 埋点代码照常工作。
# 注意：注入构建会强制 -a 全量 rebuild（编译时间明显变长）；agent 版本需与 go.mod 依赖一致，
# 且 build/agent/ 下存在 skywalking-go-agent-${SW_AGENT_VERSION}-linux-amd64（从官方 bin.tgz 解压即可；
# 该二进制已 gitignore，目录由 .gitkeep 占位，缺失时注入分支会直接报文件不存在）。
# Go 构建标签（--build-arg 传入）：留空（默认）= 产出不含基础设施调试接口
# （/test/redis|nats|es|mongo|jaeger，源码见 internal/presentation/controller/test.go 的 debug 标签）；
# 需要容器内调试时传 GO_BUILD_TAGS=debug（compose 由 .env 的 GO_BUILD_TAGS 透传）。
# 注：/test/config/encrypt（配置密文生成）不受此影响，始终编译。
ARG GO_BUILD_TAGS=""

ARG SW_AGENT_ENABLE=false
ARG SW_AGENT_VERSION=0.7.0
ARG SW_AGENT_SERVICE=photography-server
ARG SW_AGENT_BACKEND=skywalking-oap:11800

# 依赖下载（缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制 Agent
COPY build/agent/ /app/build/agent/

# 复制源码
COPY . .

# 构建
RUN if [ "$SW_AGENT_ENABLE" = "true" ]; then \
      AGENT_BIN="/app/build/agent/skywalking-go-agent-${SW_AGENT_VERSION}-linux-amd64" && \
      chmod +x "${AGENT_BIN}" && \
      # 生成 agent.config：结构与官方 agent.default.yaml 一致（reporter 为顶层键）。
      # 值写成 ${ENV:default} 占位格式 —— 默认固化 ARG 传入值，运行期仍可被环境变量覆盖。
      printf 'agent:\n  service_name: ${SW_AGENT_NAME:%s}\n  sampler: ${SW_AGENT_SAMPLE:1}\nreporter:\n  grpc:\n    backend_service: ${SW_AGENT_REPORTER_GRPC_BACKEND_SERVICE:%s}\n' \
        "${SW_AGENT_SERVICE}" "${SW_AGENT_BACKEND}" > /tmp/agent.config && \
      echo ">>> building with local agent: ${AGENT_BIN}" && \
      go build -trimpath -ldflags "-s -w" ${GO_BUILD_TAGS:+-tags $GO_BUILD_TAGS} -toolexec="${AGENT_BIN} -config /tmp/agent.config" -a -o /out/photography-server ./cmd/server; \
    else \
      go build -trimpath -ldflags "-s -w" ${GO_BUILD_TAGS:+-tags $GO_BUILD_TAGS} -o /out/photography-server ./cmd/server; \
    fi

# ---------- 运行阶段 ----------
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && adduser -D -g '' appuser

WORKDIR /app

# 复制二进制
COPY --from=builder /out/photography-server /app/photography-server

# 创建上传目录并赋予权限
RUN mkdir -p /app/uploads && chown -R appuser:appuser /app

# 切换用户
USER appuser

EXPOSE 8080

# 启动命令（配置文件通过环境变量或挂载提供）
CMD ["/app/photography-server"]