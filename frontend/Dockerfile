# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

# 复制 package 文件
COPY package*.json ./

# 设置 npm 镜像加速并安装依赖
RUN npm config set registry https://registry.npmmirror.com && \
    npm install --legacy-peer-deps

# 复制源码并构建
COPY . .
RUN npm run build

# Stage 2: Nginx Runtime
FROM nginx:alpine

# 设置时区为上海 (使用阿里云镜像加速)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 移除默认的 Nginx 静态文件
RUN rm -rf /usr/share/nginx/html/*

# 复制自定义 Nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 复制构建产物到 Nginx 目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 启动 Nginx
CMD ["nginx", "-g", "daemon off;"]