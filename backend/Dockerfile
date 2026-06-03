# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Why: 当前环境下默认 Alpine CDN 比硬编码阿里云镜像更快，避免构建阶段在系统包下载上异常变慢。
RUN apk add --no-cache git tzdata

# 设置 Go 代理加速
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并构建
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.19

WORKDIR /app

# Why: Chromium/字体/PDF 运行时依赖本身很重，优先使用默认 Alpine CDN，减少镜像源导致的额外等待。
RUN apk add --no-cache \
    ca-certificates \
    chromium \
    font-noto-cjk \
    freetype \
    git \
    harfbuzz \
    nss \
    poppler-utils \
    ttf-freefont \
    tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 复制编译产物
COPY --from=builder /app/server .

# 暴露端口
EXPOSE 8080

# 启动服务
CMD ["./server"]
