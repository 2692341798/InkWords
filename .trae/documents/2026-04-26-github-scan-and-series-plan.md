# GitHub 仓库解析重构与系列文章生成 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 GitHub 解析从单步流重构为“扫描->勾选模块->生成系列文章”的两步流。

**Architecture:** 
1. 新增 `Scan` 接口，拉取代码，提取核心目录，调用 AI 总结，返回卡片信息。
2. 改造前端页面，增加多选模块步骤。
3. 改造 `Analyze` 接口，支持批量生成单篇博客，并串联生成导读文章。

**Tech Stack:** Go, Gin, React, Tailwind, SSE

---

### Task 1: 增加系列文章数据库支持与后端模型
**Files:**
- Modify: `backend/internal/model/blog.go`

- [ ] **Step 1: 修改 Blog 模型**
```go
// 在 Blog struct 中新增 ParentID
type Blog struct {
    // ... 现有字段
    ParentID *uint `json:"parent_id" gorm:"index"` // 指向导读文章的ID，为空则是独立文章或导读文章
    IsSeries bool  `json:"is_series" gorm:"default:false"` // 是否为系列导读文章
}
```

- [ ] **Step 2: Commit**
```bash
git add backend/internal/model/blog.go
git commit -m "feat(model): add ParentID and IsSeries to Blog model"
```

### Task 2: 后端实现预扫描 (Pre-scan) 逻辑
**Files:**
- Modify: `backend/internal/service/decomposition.go` (或其他合适的服务文件)
- Modify: `backend/internal/api/project.go` (新增路由)

- [ ] **Step 1: 实现 ScanService 核心逻辑**
在服务层实现 `ScanProjectModules(ctx, githubURL)` 方法：
1. 调用 `git clone`。
2. 遍历一级目录，过滤 `.git`, `docs` 等。
3. 使用 Goroutine Pool 调用 LLM 为每个目录生成简介。
4. 返回 `[]ModuleCard`。

- [ ] **Step 2: 添加 API 路由与 Controller**
在 `backend/internal/api/project.go` 中新增 `ScanGithubRepo` handler，并绑定到 `POST /api/v1/project/scan` 路由。

- [ ] **Step 3: 测试与 Commit**
编写单测验证 Scan 逻辑，确保目录过滤和 LLM 调用正常。

### Task 3: 前端交互改造 - 新增卡片选择页面
**Files:**
- Modify: `frontend/src/services/api.ts`
- Modify: `frontend/src/components/GithubParser.tsx` (或对应表单组件)

- [ ] **Step 1: 增加 Scan API 调用**
在 `api.ts` 中新增 `scanGithubRepo` 方法。

- [ ] **Step 2: 增加状态与卡片渲染**
组件中增加 `step` 状态（1: 输入URL, 2: 模块选择, 3: 生成中）。
实现模块卡片网格布局，支持多选，显示目录名与简介。

- [ ] **Step 3: Commit**
```bash
git add frontend/src/services/api.ts frontend/src/components/GithubParser.tsx
git commit -m "feat(frontend): implement module selection step for github parsing"
```

### Task 4: 后端改造 Analyze 接口支持模块选择与系列导读
**Files:**
- Modify: `backend/internal/api/stream.go`
- Modify: `backend/internal/service/generator.go`

- [ ] **Step 1: 更新 Analyze 接口入参**
接收 `SelectedModules []string`。

- [ ] **Step 2: 调整流式生成逻辑**
1. 遍历 `SelectedModules`，依次（或并发）生成单篇博客，保存时带上关联标记。
2. 所有单篇生成完毕后，触发“生成导读文章”逻辑，保存导读文章并更新前面单篇的 `ParentID`。
3. 通过 SSE 推送各阶段进度。

- [ ] **Step 3: 测试与 Commit**
使用实际仓库测试整个链路，验证 SSE 消息流。

### Task 5: 验证与文档更新
**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: 运行本地验证**
执行 `docker compose down && docker compose up -d --build`，在浏览器中测试从 URL 输入到模块勾选再到最终生成的全流程。

- [ ] **Step 2: 更新项目文档**
根据全局规则，更新所有关联文档的架构、API及开发日志。
