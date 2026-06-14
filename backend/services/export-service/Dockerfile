# Stage 1: Build the export API and export worker.
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git tzdata

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/export-service ./services/export-service/cmd

# Stage 2: Runtime dependencies owned by export-service.
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    chromium \
    font-noto-cjk \
    freetype \
    harfbuzz \
    nss \
    ttf-freefont \
    tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /out/export-service ./export-service

EXPOSE 8080

CMD ["./export-service"]
