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
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app
COPY --from=builder /out/photography-server /app/photography-server
COPY config/config.yaml /app/config/config.yaml
RUN mkdir -p /app/uploads

EXPOSE 8080
CMD ["/app/photography-server", "-c", "/app/config/config.yaml", "-p", "prod"]