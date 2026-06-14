# Stage 1: Build the LLM streaming API and generation worker.
FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git tzdata

ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/llm-stream ./services/llm-stream/cmd

# Stage 2: Runtime dependencies owned by llm-stream.
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates git tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

COPY --from=builder /out/llm-stream ./llm-stream

EXPOSE 8080

CMD ["./llm-stream"]
