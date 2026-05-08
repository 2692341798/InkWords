# Editor “润色（预览草稿）” Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在编辑器新增“润色”按钮（位于语音输入旁），通过新增 SSE 接口生成润色草稿并在右侧“润色预览”中展示，用户确认后才应用到正文。

**Architecture:** 前端新增“预览/润色预览”切换与草稿状态；后端新增 `POST /api/v1/blogs/:id/polish` SSE 接口调用 DeepSeek 流式生成。润色结果不落库，点击“应用”后才走现有 `updateBlog` 自动保存链路。

**Tech Stack:** React + Vite + Tailwind + shadcn/ui Button、`@microsoft/fetch-event-source`、Go + Gin、DeepSeek Chat Completions（stream=true）。

---

## Files & Responsibilities

**Backend**
- Modify: [main.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/cmd/server/main.go) — 注册新路由 `POST /api/v1/blogs/:id/polish`
- Modify: [stream.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A9%E6%89%8B/InkWords/backend/internal/api/stream.go) — 新增 `PolishBlogStreamHandler`
- Modify: [generator.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/service/generator.go) — 新增 `GeneratePolishDraftStream`（仅生成不落库）
- Test: [request_models_test.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/api/request_models_test.go) — 增加请求结构体字段/JSON tag 的回归测试

**Frontend**
- Create: `frontend/src/hooks/usePolishStream.ts` — 负责润色 SSE 拉流、取消、清空、状态互斥（包含 JSDoc）
- Create: `frontend/src/lib/polishDraft.ts` — 负责从润色草稿中提取正文（`---` 后内容）
- Test: `frontend/src/lib/polishDraft.test.ts` — 纯函数单测（vitest）
- Modify: [Editor.tsx](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/components/Editor.tsx) — 新增“润色”按钮、预览 Tab、润色预览操作区、互斥逻辑

---

### Task 1: Backend — 新增润色 SSE 接口

**Files:**
- Modify: [main.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/cmd/server/main.go)
- Modify: [stream.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/backend/internal/api/stream.go)
- Modify: [generator.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A9%E6%89%8B/InkWords/backend/internal/service/generator.go)
- Test: [request_models_test.go](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A9%E6%89%8B/InkWords/backend/internal/api/request_models_test.go)

- [ ] **Step 1: 在 stream.go 增加请求体结构体**

```go
type PolishRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}
```

- [ ] **Step 2: 为 PolishRequest 增加最小回归测试（字段与 json tag）**

```go
func TestPolishRequest_HasTitleAndContentFields(t *testing.T) {
	rt := reflect.TypeOf(PolishRequest{})
	field, ok := rt.FieldByName("Title")
	require.True(t, ok)
	assert.Equal(t, "title", field.Tag.Get("json"))

	field, ok = rt.FieldByName("Content")
	require.True(t, ok)
	assert.Equal(t, "content", field.Tag.Get("json"))
}
```

- [ ] **Step 3: 在 generator.go 增加 GeneratePolishDraftStream（不落库）**

实现要点：
- 复用 `DeepSeekClient.GenerateStream`
- 对输入内容做 rune 截断（上限 15,000,000）以规避 `invalid_request_error`
- Prompt 强制输出结构：先 `## 标题建议`（3 条）→ `---` → 正文 Markdown

```go
func (s *GeneratorService) GeneratePolishDraftStream(
	ctx context.Context,
	title string,
	content string,
	chunkChan chan<- string,
	errChan chan<- error,
) {
	// 与 GenerateBlogStream 类似：启动 LLM 流式调用，把 chunk 写入 chunkChan；不做 DB 保存
}
```

- [ ] **Step 4: 在 stream.go 增加 handler：PolishBlogStreamHandler**

实现要点：
- 复用鉴权与配额检查：与 `ContinueBlogStreamHandler` 相同（`api.userService.CheckQuota`）
- 校验 blogID 归属：读取 `user_id` + `:id`（用于权限确认；不做更新）
- SSE 输出事件：`chunk/done/error`，并增加 10s keepalive `ping`（与其它 handler 一致）

```go
func (api *StreamAPI) PolishBlogStreamHandler(c *gin.Context) {
	// 1) quota check
	// 2) parse blogID + userID
	// 3) bind json -> PolishRequest
	// 4) 开 goroutine 调用 generatorService.GeneratePolishDraftStream
	// 5) c.Stream: chunk/done/error + keepalive ping
}
```

- [ ] **Step 5: 在 main.go 注册路由**

把这行加到 blogGroup 中（紧挨着 continue）：

```go
blogGroup.POST("/:id/polish", streamAPI.PolishBlogStreamHandler)
```

- [ ] **Step 6: 运行后端测试**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/backend
go test ./... -run TestPolishRequest_HasTitleAndContentFields -v
```

Expected: PASS

- [ ] **Step 7: Commit（可选）**

```bash
git add backend/internal/api/stream.go backend/internal/service/generator.go backend/cmd/server/main.go backend/internal/api/request_models_test.go
git commit -m "feat: add blog polish SSE endpoint for editor preview"
```

---

### Task 2: Frontend — 抽象润色流式 Hook + 草稿解析函数

**Files:**
- Create: `frontend/src/hooks/usePolishStream.ts`
- Create: `frontend/src/lib/polishDraft.ts`
- Test: `frontend/src/lib/polishDraft.test.ts`

- [ ] **Step 1: 新增草稿解析函数 extractPolishedBody**

```ts
export function extractPolishedBody(draft: string) {
  const marker = '\n---\n'
  const idx = draft.indexOf(marker)
  if (idx === -1) return draft.trim()
  return draft.slice(idx + marker.length).trim()
}
```

- [ ] **Step 2: 为 extractPolishedBody 写 vitest 单测**

```ts
import { describe, expect, it } from 'vitest'
import { extractPolishedBody } from './polishDraft'

describe('extractPolishedBody', () => {
  it('returns full draft when marker missing', () => {
    expect(extractPolishedBody('hello')).toBe('hello')
  })

  it('returns content after marker', () => {
    const draft = '## 标题建议\n1. A\n---\n# 正文\n内容'
    expect(extractPolishedBody(draft)).toBe('# 正文\n内容')
  })
})
```

- [ ] **Step 3: 新增 usePolishStream（含 JSDoc）**

要求：
- 内部使用 `fetchEventSource` 发起 POST SSE，header 携带 `Authorization: Bearer <token>`
- 提供：`isPolishing`、`draft`、`start`、`cancelAndClear`
- 取消行为：abort + 清空 draft（用户已明确“直接清空”）

```ts
/**
 * 管理编辑器“润色（预览草稿）”的 SSE 拉流与取消逻辑，保证：
 * - 润色结果只在前端暂存，用户点“应用”前不覆盖正文
 * - 取消会中止请求并清空草稿，避免误用旧草稿
 */
export function usePolishStream() {}
```

- [ ] **Step 4: 运行前端单测**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm test -- --run src/lib/polishDraft.test.ts
```

Expected: PASS

- [ ] **Step 5: Commit（可选）**

```bash
git add frontend/src/hooks/usePolishStream.ts frontend/src/lib/polishDraft.ts frontend/src/lib/polishDraft.test.ts
git commit -m "feat: add polish stream hook and draft parser"
```

---

### Task 3: Frontend — Editor.tsx 接入润色按钮与预览切换

**Files:**
- Modify: [Editor.tsx](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/components/Editor.tsx)

- [ ] **Step 1: 在 Header 按钮区新增“润色”按钮**

要求：
- 放在“语音输入”旁边（同一行）
- 文案中文：`润色`
- disabled 规则：当 `isVoiceListening` 或 `isContinuing` 或 `isPolishing` 为 true 时禁用；润色中可显示 loading 状态

- [ ] **Step 2: 在预览区增加 Tab（预览 / 润色预览）**

要求：
- `activePreviewTab` 默认 `preview`
- 点击“润色”自动切换到 `polish`
- 润色预览渲染 `draft`；普通预览渲染 `content`

- [ ] **Step 3: 在“润色预览”中加入操作区**

按钮：
- `应用润色结果`：`setContent(extractPolishedBody(draft))`，并清空 draft、切回预览 tab
- `取消`：调用 `cancelAndClear`（abort + 清空），并切回预览 tab
- `重新润色`：清空 draft 后重新 start

- [ ] **Step 4: 做好互斥逻辑**

要求：
- 润色中：禁用语音输入与继续生成（并保持现有禁用逻辑一致）
- 语音输入/继续生成中：禁用润色

- [ ] **Step 5: 运行前端构建**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords/frontend
npm run build
```

Expected: PASS

- [ ] **Step 6: Commit（可选）**

```bash
git add frontend/src/components/Editor.tsx
git commit -m "feat: add polish preview tab and apply flow in editor"
```

---

### Task 4: E2E 手工验证（Docker-First）

- [ ] **Step 1: 一键重启（确保前后端最新）**

Run:

```bash
cd /Users/huangqijun/Documents/墨言博客助手/InkWords
docker compose down && docker compose up -d --build
```

- [ ] **Step 2: 手工验证清单（在 http://localhost）**

1. 进入编辑器页，确认“润色”按钮在“语音输入”旁边
2. 点击“润色”：右侧切换到“润色预览”，内容开始流式出现
3. 确认润色预览顶部出现“标题建议（3 个）”，标题输入框未被修改
4. 点击“取消”：停止生成并清空草稿，切回“预览”，正文不变
5. 再次点击“润色”并等待生成完成，点击“应用润色结果”：左侧正文被替换为润色正文（`---` 后内容）
6. 等待 2 秒自动保存，刷新页面后正文仍为润色后的版本
7. 润色/语音输入/继续生成互斥生效

---

## Plan Self-Review

- 覆盖 spec：按钮位置、预览切换、SSE 接口契约、标题建议不自动覆盖、取消直接清空、应用只写正文、互斥、DoD 均有对应任务。
- 无占位符：每个任务包含具体文件、关键代码骨架与命令。
- 命名一致：接口 `POST /api/v1/blogs/:id/polish`，事件 `chunk/done/error`，正文提取以 `\n---\n` 为分隔。

