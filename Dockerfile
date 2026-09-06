# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOFLAGS=-mod=mod

# SkyWalking Go agent（skywalking-go）注入构建开关：
# SW_AGENT_ENABLE=true 时以 -toolexec 织入构建产物（agent 为 build context 内 sw-agent/ 目录的本地
# 二进制，随 COPY . . 带入镜像，无需联网下载），服务走 SkyWalking native 通道（Horizon「原生」模式
# 可见、具备 OAP 拓扑/指标分析）。
# 默认 false —— 纯 OTel / Jaeger 版：不注入，保留的 OTel 埋点代码照常工作。
# 注意：注入构建会强制 -a 全量 rebuild（编译时间明显变长）；agent 版本需与 go.mod 依赖一致，
# 且 sw-agent/ 下存在 skywalking-go-agent-${SW_AGENT_VERSION}-linux-amd64（从官方 bin.tgz 解压即可）。
ARG SW_AGENT_ENABLE=false
ARG SW_AGENT_VERSION=0.7.0
ARG SW_AGENT_SERVICE=photography-server
ARG SW_AGENT_BACKEND=skywalking-oap:11800

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN if [ "$SW_AGENT_ENABLE" = "true" ]; then \
      AGENT_BIN="/app/sw-agent/skywalking-go-agent-${SW_AGENT_VERSION}-linux-amd64" && \
      chmod +x "${AGENT_BIN}" && \
      # 生成 agent.config：结构与官方 agent.default.yaml 一致（reporter 为顶层键）。
      # 值写成 ${ENV:default} 占位格式 —— 默认固化 ARG 传入值，运行期仍可被环境变量覆盖。
      printf 'agent:\n  service_name: ${SW_AGENT_NAME:%s}\n  sampler: ${SW_AGENT_SAMPLE:1}\nreporter:\n  grpc:\n    backend_service: ${SW_AGENT_REPORTER_GRPC_BACKEND_SERVICE:%s}\n' \
        "${SW_AGENT_SERVICE}" "${SW_AGENT_BACKEND}" > /tmp/agent.config && \
      echo ">>> building with local agent: ${AGENT_BIN}" && \
      go build -trimpath -ldflags "-s -w" -toolexec="${AGENT_BIN} -config /tmp/agent.config" -a -o /out/photography-server ./cmd/server; \
    else \
      go build -trimpath -ldflags "-s -w" -o /out/photography-server ./cmd/server; \
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