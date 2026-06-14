# Stage 1: Build the core API binary only.
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git tzdata

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/core-api ./services/core-api/cmd

# Stage 2: Runtime dependencies owned by core-api.
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates git poppler-utils tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /out/core-api ./core-api

EXPOSE 8080

CMD ["./core-api"]
