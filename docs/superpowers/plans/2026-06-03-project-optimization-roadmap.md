# InkWords Optimization Roadmap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不打断现有主流程的前提下，优先补齐 InkWords 在后端生命周期治理、系列生成一致性、前端流式渲染性能、环境可复现性和仓库治理上的高优先级短板。

**Architecture:** 本计划坚持最小改动原则，先修稳定性和可观测性，再做前端性能和工程化收口，不把大规模重构混入同一轮变更。后端以“生命周期可控 + 事务边界清晰”为主线，前端以“缩小订阅范围 + 批量刷新流式内容”为主线，部署侧以“默认可启动 + 文档强同步”为主线。

**Tech Stack:** Go 1.25+, Gin, GORM, PostgreSQL, React 18, Zustand, Vite, Tailwind CSS, shadcn/ui, Docker Compose, Nginx

---

## Scope

**In scope**
- 后端 `HTTP Server` 超时、优雅停机、请求取消传播治理
- 系列生成链路的事务边界、失败恢复与 token 记账一致性
- 前端 Zustand 订阅粒度优化与 SSE 流式内容批量提交
- Markdown 预览链路的性能与安全边界优化
- Docker Compose 启动可复现性、环境模板、文档与治理文件补齐

**Out of scope**
- 一次性完成所有 legacy service 的领域迁移
- 全量重写前端状态模型或引入新路由体系
- 本轮直接改动数据库 schema
- 本轮新增业务功能或 UI 大改版

## Priority Roadmap

- `P0`：后端生命周期治理、系列生成一致性、Compose 可复现性
- `P1`：前端流式渲染优化、Markdown 预览治理
- `P2`：仓库治理自动化、文档和 README 收口

## File Map

**后端生命周期**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/domain/stream/handler_test.go`
- Modify: `backend/internal/domain/stream/handler_error_test.go`
- Optional Create: `backend/cmd/server/main_test.go`

**后端系列生成一致性**
- Modify: `backend/internal/service/decomposition_generate.go`
- Modify: `backend/internal/service/decomposition_generate_persistence.go`
- Modify: `backend/internal/service/decomposition_generate_persist_test.go`
- Modify: `backend/internal/service/generator.go`

**前端状态与流式渲染**
- Modify: `frontend/src/pages/Generator.tsx`
- Modify: `frontend/src/pages/KnowledgeReview.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/components/generator/GeneratorStatus.tsx`
- Modify: `frontend/src/store/streamStore.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/hooks/usePolishStream.ts`
- Modify: `frontend/src/store/streamStore.test.ts`

**前端预览与交互**
- Modify: `frontend/src/pages/Editor.tsx`
- Modify: `frontend/src/components/editor/EditorBody.tsx`
- Modify: `frontend/src/components/MarkdownEngine.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/services/sse.ts`
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/components/sidebar/StreamOutlineSection.tsx`

**部署与治理**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Modify: `frontend/README.md`
- Create: `backend/.env.example`
- Optional Create: `.github/pull_request_template.md`
- Optional Create: `.github/workflows/ci.yml`

### Task 1: Backend Lifecycle and Request Cancellation

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/domain/stream/handler.go`
- Modify: `backend/internal/domain/stream/handler_test.go`
- Modify: `backend/internal/domain/stream/handler_error_test.go`
- Optional Create: `backend/cmd/server/main_test.go`

- [ ] **Step 1: 建立后端基线**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./cmd/server/... ./internal/domain/stream/... ./internal/transport/http/v1/...
```

Expected:
- 当前与启动、SSE、stream handler 相关测试通过，作为治理前基线。

- [ ] **Step 2: 将裸 `r.Run` 改为显式 `http.Server`**

Implementation snippet:

```go
server := &http.Server{
    Addr:              ":8080",
    Handler:           router,
    ReadTimeout:       15 * time.Second,
    ReadHeaderTimeout: 10 * time.Second,
    WriteTimeout:      0, // SSE 由 handler 自己控制 flush 生命周期
    IdleTimeout:       60 * time.Second,
}
```

Why:
- 先把 server 生命周期显式化，才能接入超时、信号监听和优雅停机。

- [ ] **Step 3: 补齐信号监听和优雅停机**

Implementation snippet:

```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

go func() {
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    _ = server.Shutdown(shutdownCtx)
}()
```

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./cmd/server/... ./internal/domain/stream/...
```

Expected:
- 启动链路编译通过。
- 现有 stream 测试无回归。

- [ ] **Step 4: 收回不必要的 `context.WithoutCancel`**

Implementation target:
- `Analyze`、`Scan`、`Continue` 默认使用 `c.Request.Context()`。
- 只有“明确要求断连后继续执行”的任务才允许脱离请求上下文，并且必须写注释说明原因。

Implementation snippet:

```go
requestContext := c.Request.Context()
if err := handler.service.Analyze(requestContext, request); err != nil {
    // write stable SSE error event
}
```

Verification:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -R "WithoutCancel" backend/cmd backend/internal/domain/stream || true
```

Expected:
- `stream` 主链路中不再残留默认的 `WithoutCancel` 用法，除非有明确注释说明。

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go \
  backend/internal/domain/stream/handler.go \
  backend/internal/domain/stream/handler_test.go \
  backend/internal/domain/stream/handler_error_test.go \
  backend/cmd/server/main_test.go
git commit -m "fix(backend): add graceful shutdown and request cancellation boundaries"
```

### Task 2: Series Generation Consistency and Token Accounting

**Files:**
- Modify: `backend/internal/service/decomposition_generate.go`
- Modify: `backend/internal/service/decomposition_generate_persistence.go`
- Modify: `backend/internal/service/decomposition_generate_persist_test.go`
- Modify: `backend/internal/service/generator.go`

- [ ] **Step 1: 建立系列生成回归基线**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service/... -run 'Decomposition|Generator|Persist' -count=1
```

Expected:
- 当前系列生成与持久化相关测试通过。

- [ ] **Step 2: 明确“草稿创建”和“章节回填”的事务边界**

Implementation target:
- 第一阶段：父节点检查、旧子节点清理、子章节草稿创建，放在一个事务里完成。
- 第二阶段：每个章节生成完成后单独回填，失败时保留草稿和错误状态，不回滚整个系列。

Implementation snippet:

```go
err := database.Transaction(func(tx *gorm.DB) error {
    if err := deleteExistingChildren(tx, seriesID); err != nil {
        return err
    }
    if err := createChapterDrafts(tx, seriesID, outline); err != nil {
        return err
    }
    return nil
})
```

Why:
- 这一步先保证“系列骨架”稳定存在，避免失败后历史树断裂。

- [ ] **Step 3: 统一 token 记账与错误落库策略**

Implementation target:
- 每个章节完成后都走统一的 `persistGeneratedChapter(...)`。
- `tokens_used` 更新失败不能静默吞掉，至少要记录结构化日志，并把章节状态标记为“内容成功但记账失败”或返回明确错误。

Implementation snippet:

```go
if err := updateTokenUsage(tx, blogID, usage); err != nil {
    return fmt.Errorf("update token usage for blog %s: %w", blogID, err)
}
```

- [ ] **Step 4: 增加异常恢复测试**

Test cases to add:
- 删除旧子节点后、创建新草稿前失败时，不应留下空系列
- 某一章生成失败时，其它章节和父节点仍保留
- token 记账失败时，测试能观测到稳定错误或日志分支

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./internal/service/... -run 'Decomposition|Generator|Persist' -count=1
```

Expected:
- 新增异常恢复用例通过。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/decomposition_generate.go \
  backend/internal/service/decomposition_generate_persistence.go \
  backend/internal/service/decomposition_generate_persist_test.go \
  backend/internal/service/generator.go
git commit -m "fix(backend): harden series generation persistence boundaries"
```

### Task 3: Frontend Store Subscription and Stream Batching

**Files:**
- Modify: `frontend/src/pages/Generator.tsx`
- Modify: `frontend/src/pages/KnowledgeReview.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Modify: `frontend/src/components/generator/GeneratorStatus.tsx`
- Modify: `frontend/src/store/streamStore.ts`
- Modify: `frontend/src/hooks/generator/useSeriesGenerator.ts`
- Modify: `frontend/src/hooks/usePolishStream.ts`
- Modify: `frontend/src/store/streamStore.test.ts`

- [ ] **Step 1: 建立前端构建与测试基线**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

Expected:
- 当前前端测试与构建通过。

- [ ] **Step 2: 把整包订阅改成 selector + shallow**

Implementation snippet:

```tsx
const { outline, isGenerating, chapterStatus } = useStreamStore(
  (state) => ({
    outline: state.outline,
    isGenerating: state.isGenerating,
    chapterStatus: state.chapterStatus,
  }),
  shallow,
)
```

Targets:
- `Generator.tsx`
- `KnowledgeReview.tsx`
- `Sidebar.tsx`
- `GeneratorStatus.tsx`

Why:
- 限定订阅面，避免无关字段变化触发整页重渲染。

- [ ] **Step 3: 给流式内容追加增加批量 flush**

Implementation snippet:

```ts
let pendingChunks: Record<number, string[]> = {}

const flushPendingChunks = () => {
  set((state) => {
    const nextContents = { ...state.chapterContents }
    for (const [sort, chunks] of Object.entries(pendingChunks)) {
      nextContents[Number(sort)] = (nextContents[Number(sort)] ?? '') + chunks.join('')
    }
    pendingChunks = {}
    return { chapterContents: nextContents }
  })
}
```

Design rule:
- 按 `requestAnimationFrame` 或 100~200ms 时间窗刷新一次。
- `useSeriesGenerator.ts` 与 `usePolishStream.ts` 统一使用同类批量策略，避免一个链路优化、另一个链路继续抖动。

- [ ] **Step 4: 增加流式状态回归测试**

Test cases to add:
- 高频 chunk 输入时，store 最终内容正确
- flush 前不会过早写入最终字符串
- `stop` 后 pending 队列被清理，不再继续写入

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand streamStore.test.ts
npm run build
```

Expected:
- 流式批量提交逻辑通过测试。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/Generator.tsx \
  frontend/src/pages/KnowledgeReview.tsx \
  frontend/src/components/Sidebar.tsx \
  frontend/src/components/generator/GeneratorStatus.tsx \
  frontend/src/store/streamStore.ts \
  frontend/src/hooks/generator/useSeriesGenerator.ts \
  frontend/src/hooks/usePolishStream.ts \
  frontend/src/store/streamStore.test.ts
git commit -m "perf(frontend): reduce store subscriptions and batch stream updates"
```

### Task 4: Markdown Preview, Auth Reactivity, and Blocking UI Cleanup

**Files:**
- Modify: `frontend/src/pages/Editor.tsx`
- Modify: `frontend/src/components/editor/EditorBody.tsx`
- Modify: `frontend/src/components/MarkdownEngine.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/services/sse.ts`
- Modify: `frontend/src/pages/Login.tsx`
- Modify: `frontend/src/components/sidebar/StreamOutlineSection.tsx`

- [ ] **Step 1: 把 Markdown 预览更新改为更温和的渲染节奏**

Implementation target:
- 对编辑器正文预览应用 `useDeferredValue` 或 debounce。
- 非活动面板不做重型 Mermaid 渲染。

Implementation snippet:

```tsx
const deferredContent = useDeferredValue(content)
return <MarkdownEngine content={deferredContent} />
```

- [ ] **Step 2: 收紧 Mermaid 注入边界**

Implementation target:
- 保持只渲染基础 Mermaid 语法。
- 用受控容器替代无边界的 `innerHTML` 注入，至少在渲染前清理非法 script/event handler。

Implementation snippet:

```ts
const { svg } = await mermaid.render(id, code)
container.replaceChildren(createRange().createContextualFragment(svg))
```

Why:
- 先降低注入风险，再考虑进一步抽离为独立渲染组件。

- [ ] **Step 3: 让鉴权状态变成响应式来源**

Implementation target:
- `App.tsx` 不再只在初始化时读取 token。
- 登录成功、退出登录、SSE 401 都统一驱动同一个 auth 状态源。

Implementation snippet:

```ts
window.addEventListener('storage', syncAuthState)
window.addEventListener('inkwords-auth-changed', syncAuthState as EventListener)
```

- [ ] **Step 4: 移除阻塞式 `alert/confirm/location.href`**

Implementation target:
- 用 `toast`、现有确认对话框组件和 store 驱动导航替代阻塞式浏览器 API。
- 所有面向用户的文案保持中文。

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
grep -R "alert(" frontend/src || true
grep -R "confirm(" frontend/src || true
grep -R "location.href" frontend/src || true
```

Expected:
- 核心业务代码不再残留阻塞式浏览器弹窗或跳转。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/Editor.tsx \
  frontend/src/components/editor/EditorBody.tsx \
  frontend/src/components/MarkdownEngine.tsx \
  frontend/src/App.tsx \
  frontend/src/services/sse.ts \
  frontend/src/pages/Login.tsx \
  frontend/src/components/sidebar/StreamOutlineSection.tsx
git commit -m "refactor(frontend): harden preview rendering and auth reactivity"
```

### Task 5: Docker Bootstrap, Docs, and Repo Governance

**Files:**
- Modify: `docker-compose.yml`
- Modify: `README.md`
- Modify: `frontend/README.md`
- Create: `backend/.env.example`
- Optional Create: `.github/pull_request_template.md`
- Optional Create: `.github/workflows/ci.yml`

- [ ] **Step 1: 建立部署基线**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
```

Expected:
- 明确当前 Compose 是否依赖外部环境变量才能渲染。

- [ ] **Step 2: 补齐 `backend/.env.example` 并收口默认启动方式**

Implementation target:
- 给 `DEEPSEEK_API_KEY`、`JWT_SECRET`、`DATABASE_URL`、`REDIS_URL`、`OBSIDIAN_VAULT_PATH` 等关键变量提供模板。
- README 中把“标准启动命令”和“标准重启命令”写成统一入口。

Example template:

```env
DEEPSEEK_API_KEY=
JWT_SECRET=change_me
DATABASE_URL=postgres://inkwords:inkwords@db:5432/inkwords?sslmode=disable
REDIS_URL=redis://redis:6379
OBSIDIAN_VAULT_PATH=/absolute/path/to/your/vault
```

- [ ] **Step 3: 固定镜像版本并减少漂移**

Implementation target:
- 把 `latest` 替换为明确版本或 digest。
- 保持前端为主入口 `http://localhost`，非必要不要默认暴露后端、Redis、Postgres 到宿主机。

- [ ] **Step 4: 补最小治理门禁**

Implementation target:
- 新增 PR 模板，要求填写验证方式、关键设计决策、已知限制。
- 可选新增最小 CI：前端 `build/test` + 后端 `go test ./...` + `docker compose config`。

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
```

Expected:
- Compose 可渲染。
- 文档与环境模板对新协作者可执行。

- [ ] **Step 5: Commit**

```bash
git add docker-compose.yml README.md frontend/README.md backend/.env.example .github/pull_request_template.md .github/workflows/ci.yml
git commit -m "chore(repo): improve docker bootstrap and project governance"
```

## Validation Matrix

- Backend validation:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./...
```

- Frontend validation:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --runInBand
npm run build
```

- Docker validation:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose config
docker compose down && docker compose up -d --build
```

- Manual verification:
- 打开 `http://localhost`
- 登录并检查验证码、登录态切换、退出登录
- 走一遍 `scan -> analyze -> generate` 主流程
- 中途断开生成请求，确认后台不再继续无边界运行
- 生成失败一个章节，确认系列父子结构仍保留
- 编辑器输入长文，确认预览不卡死、Mermaid 可正常渲染

## Risks and Rollback

- 风险：`WriteTimeout` 对 SSE 不当配置可能误杀长连接，因此需要显式说明为何保持为 `0` 或单独处理。
- 风险：前端批量 flush 若实现不当，会带来“最后一段丢失”或“停止后仍写入”的回归。
- 回滚策略：每个 Task 独立提交；若某项回归明显，直接按 Task 级别回滚，不混回其它优化。

## Self-Review

- Spec coverage: 该计划覆盖了上轮审查中最高优先级的稳定性、性能、部署和治理问题，没有把大规模架构重写混进同一轮。
- Placeholder scan: 无 `TODO`、`TBD`、`implement later` 等占位描述；每个 Task 都给出了目标文件、命令和验证方式。
- Type consistency: 使用的文件路径、模块名和现有仓库结构一致，未引入虚构目录。

Plan complete and saved to `docs/superpowers/plans/2026-06-03-project-optimization-roadmap.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using `executing-plans`, batch execution with checkpoints

**Which approach?**
