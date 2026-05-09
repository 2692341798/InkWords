# 项目文件整理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按 InkWords 工程规范清理仓库中的空壳文件与大文件/产物目录，并同步更新关键文档，确保仓库结构清晰、可维护、可复现。

**Architecture:** 以“最小改动”为原则，不做大规模架构迁移；优先移除误导性文件与不应进入 Git 的产物，并将必要产出归档到 docs 下。

**Tech Stack:** Go + Gin + PostgreSQL；React + Vite + Tailwind + shadcn/ui；Docker Compose。

---

## 文件结构（本次会触达的路径）

**后端**
- 删除空壳文件：
  - `backend/internal/api/blog.go`
  - `backend/internal/api/stream.go`
- 保持 `backend/internal/api/` 继续使用同一 package（不拆子包，避免 import 级联改动）。

**前端**
- 保持现有 `frontend/src/components/editor/*`、`frontend/src/components/sidebar/*` 的拆分成果。

**仓库减肥**
- 继续确保二进制与 PDF 不进入 Git（通过 `.gitignore` + 从 Git 追踪中移除）。
- 清理 `dogfood-output/`（将有价值内容归档到 `docs/qa/`，截图类产物不追踪）。

**文档同步**
- 更新：
  - `README.md`
  - `.trae/documents/InkWords_Architecture.md`
  - `.trae/documents/InkWords_Development_Plan_and_Log.md`

---

### Task 1: 删除空壳文件（避免误导）

**Files:**
- Delete: `backend/internal/api/blog.go`
- Delete: `backend/internal/api/stream.go`

- [ ] **Step 1: 确认文件为空壳**

Run:
```bash
sed -n '1,40p' backend/internal/api/blog.go
sed -n '1,40p' backend/internal/api/stream.go
```
Expected: 仅包含 `package api`（或非常少的内容），无实际逻辑。

- [ ] **Step 2: 删除文件**

Run:
```bash
rm backend/internal/api/blog.go backend/internal/api/stream.go
```

- [ ] **Step 3: 验证后端测试**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

---

### Task 2: 仓库减肥（dogfood-output 归档 + 忽略产物）

**Files:**
- Modify: `.gitignore`
- Create: `docs/qa/dogfood/issue-001-report.md`（如存在 report）
- Delete (optional): `dogfood-output/**`（按实际追踪情况）

- [ ] **Step 1: 列出当前 dogfood-output 被 Git 追踪的文件**

Run:
```bash
git ls-files dogfood-output | cat
```

- [ ] **Step 2: 归档 report.md（如存在）到 docs/qa**

Run:
```bash
mkdir -p docs/qa/dogfood
```
Then copy内容到 `docs/qa/dogfood/issue-001-report.md` 并从 `dogfood-output/` 移除该 report。

- [ ] **Step 3: 将截图/调试产物从 Git 追踪中移除并忽略**

Run:
```bash
git rm -r --cached dogfood-output/screenshots || true
```

Update `.gitignore`:
- ignore `dogfood-output/` 或至少忽略 `dogfood-output/screenshots/`

- [ ] **Step 4: 验证仓库体积扫描（Top 30）**

Run:
```bash
find . -type f \( -path './.git/*' -o -path './frontend/node_modules/*' -o -path './frontend/dist/*' \) -prune -false -o -type f -print0 | xargs -0 stat -f "%z %N" | sort -nr | head -n 30
```
Expected: 不再出现被追踪的巨大二进制；dogfood-output 截图不在追踪列表中。

---

### Task 3: 文档同步（结构与约束对齐）

**Files:**
- Modify: `README.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: README 补充“哪些不应提交进仓库”与“获取方式”**

Add/Update sections:
- 大文件/构建产物策略（backend 二进制、pdf、dogfood-output）
- 运行入口与 Docker Compose 约束（`http://localhost`）

- [ ] **Step 2: Architecture 文档更新仓库结构现状**

确保包含：
- 前后端目录隔离
- 单文件行数警戒线（500/800）
- 产物目录与忽略规则

- [ ] **Step 3: Development Log 追加本次整理记录**

记录：
- 删除空壳文件
- 移除大文件追踪
- dogfood-output 归档策略

---

### Task 4: 全量验证（不引入行为变更）

- [ ] **Step 1: 后端**

Run:
```bash
cd backend && go test ./...
```
Expected: PASS

- [ ] **Step 2: 前端**

Run:
```bash
cd frontend && npm run build
```
Expected: PASS

- [ ] **Step 3: 工作区状态检查**

Run:
```bash
git status --porcelain
```
Expected: 仅包含本次整理产生的变更；不出现大型二进制回流到追踪。

