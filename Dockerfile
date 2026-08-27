# ---------- 构建阶段 ----------
FROM golang:1.26-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags "-s -w" -o /out/photography-server ./cmd/server

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