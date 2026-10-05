FROM golang:1.26-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git tzdata
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/course-runner ./services/course-runner/cmd

# Why: the isolated textbook browser probe uses the same reviewed Playwright
# release as the frontend. The browser remains inside Bubblewrap; this image
# merely supplies the operator-owned Node/Chromium runtime and its fixed probe.
FROM mcr.microsoft.com/playwright:v1.62.1-noble
WORKDIR /app
ENV PATH="/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
RUN apt-get update && \
    apt-get install -y --no-install-recommends bubblewrap ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*
COPY --from=builder /usr/local/go /usr/local/go
COPY --from=builder /out/course-runner ./course-runner
COPY services/course-runner/playwright-probe/package.json services/course-runner/playwright-probe/package-lock.json ./playwright-probe/
RUN cd /app/playwright-probe && npm ci --omit=dev --ignore-scripts && npm cache clean --force
COPY services/course-runner/playwright-probe/probe.mjs ./playwright-probe/probe.mjs
RUN groupadd --system runner && useradd --system --gid runner --create-home runner && \
    mkdir -p /app/visual-assets && chown -R runner:runner /app
USER runner
EXPOSE 8080
CMD ["./course-runner"]
