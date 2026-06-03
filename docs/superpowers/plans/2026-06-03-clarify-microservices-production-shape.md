# Clarify Microservices Production Shape Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 明确 InkWords 的“生产形态”为 Docker Compose 多服务（`core-api/llm-stream/parser-service/export-service/review-service`）+ 前端 Nginx 单一入口；将 `cmd/server` 明确为本地/集成用的单体聚合入口，并避免在 Docker 镜像中默认/隐式启用。

**Architecture:** 继续使用单一后端镜像（同一 Dockerfile 构建多个二进制），但移除 `server` 二进制与默认启动入口，确保容器必须显式选择要运行的服务进程；对外入口仍为 `http://localhost`，由 Nginx 按路径分流。

**Tech Stack:** Go 1.25 + Gin + GORM + Docker Compose + Nginx

---

## Task 1: 让 Docker 镜像默认行为与“多服务形态”一致

**Files:**
- Modify: `backend/Dockerfile`

- [ ] **Step 1: builder stage 移除 server 二进制编译**

将 Dockerfile builder stage 中这行删除：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
```

- [ ] **Step 2: runtime stage 移除 server 二进制 COPY**

将 runtime stage 中这行删除：

```dockerfile
COPY --from=builder /app/server .
```

- [ ] **Step 3: runtime stage 默认 CMD 调整为 core-api**

将末尾默认启动从：

```dockerfile
CMD ["./server"]
```

改为：

```dockerfile
CMD ["./core-api"]
```

- [ ] **Step 4: Docker build 验证**

Run:

```bash
docker build -t inkwords-backend:microservices ./backend
```

Expected:
- build succeeds

---

## Task 2: 文档明确“生产多服务 / server 仅本地聚合”

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Architecture.md`

- [ ] **Step 1: README 更新“本地开发环境运行”说明**

在 `README.md` 的 “5.2 本地开发环境运行” 小节，保留 `cmd/server` 的用法，但明确：
- Docker Compose 是推荐入口（生产形态）
- `cmd/server` 是本地/集成用聚合入口，不作为 Compose 生产形态的一部分

建议将原文：

```md
go run ./cmd/server/main.go
```

替换为（示例）：

```md
go run ./cmd/server/main.go
```

并在同一段落补充两行说明（中文）：
- Docker Compose 生产形态已拆分为 `core-api/llm-stream/parser-service/export-service/review-service`，通过前端 Nginx 统一对外提供服务。
- `cmd/server` 是聚合入口，用于本地开发/集成调试；Docker 镜像默认不再启动该入口。

- [ ] **Step 2: Architecture 文档追加一条“生产形态澄清”的变更记录**

在 `.trae/documents/InkWords_Architecture.md` 的变更记录中追加一条（中文），说明：
- Docker 镜像默认 CMD 指向 `core-api`
- 镜像不再包含 `server` 二进制
- `cmd/server` 明确为本地/集成调试入口

---

## Task 3: Compose 冒烟验证（对外入口不变）

**Files:**
- None

- [ ] **Step 1: 一键重启验证**

Run:

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```

Expected:
- `docker compose ps` 显示 `core-api/llm-stream/parser-service/export-service/review-service/frontend` 均为 Up

- [ ] **Step 2: 网关健康检查**

Run:

```bash
curl -I http://localhost
```

Expected:
- 返回 `HTTP/1.1 200 OK`

---

## Self-Review Checklist

- [ ] `backend/Dockerfile` 不再构建/复制 `server` 二进制
- [ ] 镜像默认 CMD 不会隐式运行单体聚合入口
- [ ] `docker compose --env-file backend/.env up -d --build` 可正常拉起
- [ ] README 与 Architecture 对“生产形态 vs 本地聚合入口”描述一致
