# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 安装必要的系统依赖 (使用阿里云镜像加速)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache git tzdata

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

# 设置时区为上海，并安装 git 供拉取代码使用 (使用阿里云镜像加速)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache tzdata ca-certificates git && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 复制编译产物
COPY --from=builder /app/server .

# 暴露端口
EXPOSE 8080

# 启动服务
CMD ["./server"]