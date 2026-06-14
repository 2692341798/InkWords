# Stage 1: Build the review service binary only.
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git tzdata

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/review-service ./services/review-service/cmd

# Stage 2: Runtime dependencies owned by review-service.
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /out/review-service ./review-service

EXPOSE 8080

CMD ["./review-service"]
