FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git tzdata
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/course-runner ./services/course-runner/cmd

# Why: the isolated lab executor runs the generated Go tests inside bubblewrap;
# the runtime image therefore needs the Go toolchain while keeping the runner
# process itself unprivileged.
FROM alpine:3.19
WORKDIR /app
ENV PATH="/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
RUN apk add --no-cache ca-certificates tzdata bubblewrap && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /out/course-runner ./course-runner
RUN addgroup -S runner && adduser -S -G runner runner && \
    mkdir -p /app/course-artifacts && chown -R runner:runner /app
USER runner
EXPOSE 8080
CMD ["./course-runner"]
