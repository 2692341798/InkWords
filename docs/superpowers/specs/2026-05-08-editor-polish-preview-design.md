# InkWords 编辑器「润色（预览草稿）」设计规格（方案 2）

## 1. 背景

InkWords 当前编辑器 [Editor.tsx](file:///Users/huangqijun/Documents/%E5%A2%A8%E8%A8%80%E5%8D%9A%E5%AE%A2%E5%8A%A9%E6%89%8B/InkWords/frontend/src/components/Editor.tsx) 已具备：

- 双栏编辑与实时预览（正文 `textarea` + 右侧 `MarkdownEngine`）
- 2 秒防抖自动保存（`updateBlog`）
- AI 继续生成（SSE + `@microsoft/fetch-event-source`）
- 语音输入（浏览器 SpeechRecognition）

现在希望对“自己写的博客”提供一个「润色」按钮：把当前正文交给 AI，输出结构清晰、逻辑严密、补充示例代码与解释的高质量博客；并且采用“先预览对比，再决定是否应用”的交互（避免误覆盖原稿）。

参考（知识库）：
- [[concepts/前端组件体系：Editor 与 Markdown 渲染]]

## 2. 目标（要做）

- 在编辑器页新增「润色」入口，位置在「语音输入」按钮旁边。
- 润色流程为“预览草稿”模式：先在右侧预览区展示润色结果，用户确认后再应用到正文。
- 润色输出包含：
  - 3 个标题建议（不自动覆盖标题输入框）
  - 完整 Markdown 正文：结构清晰（H1-H4）、逻辑严密、补充配套示例代码与解释、必要时给出可复现步骤
- 使用 SSE 流式返回润色草稿，保证大文本输出体验稳定。

## 3. 非目标（不做）

- 不做“选中段落润色 / 局部润色”（本期只做全文润色）。
- 不做逐行 Diff 高亮（仅做预览切换）。
- 不引入前端直连大模型（API Key 不能下发到浏览器）。
- 润色结果在用户未点击“应用”前不落库，不影响当前博客历史版本。

## 4. 方案选择

选择方案 2：新增「润色（预览草稿）」SSE 接口。

核心原则：
- 生成与落库解耦：后端只负责生成流式草稿；前端暂存草稿；用户确认后才写回博客正文并触发既有自动保存。
- 避免竞态：润色/语音输入/继续生成互斥，避免多来源同时写 `content`。

## 5. 交互与 UI

### 5.1 按钮位置

编辑器 Header 右侧按钮区（与「语音输入」「继续生成」「导出/同步」同一行）新增：
- 按钮：润色（建议图标：`Sparkles` 或 `Wand2`）
- 放置在「语音输入」按钮旁边（紧邻，保持一致的 `variant="outline" size="sm"` 风格）

### 5.2 预览区切换

右侧预览区新增 Tab：
- `预览`：当前正文渲染（现状）
- `润色预览`：润色草稿渲染（新增）

点击「润色」后：
- 自动切换到 `润色预览`
- 开始 SSE 拉流，逐步填充润色草稿

### 5.3 草稿操作

在 `润色预览` 中提供操作按钮：
- 应用润色结果：把润色草稿正文写回编辑器正文（`content`），但不自动改标题
- 取消：停止生成（若在生成中），并保留现有正文不变
- 重新润色：清空润色草稿并重新发起润色

### 5.4 状态互斥

- 润色进行中：禁用「语音输入」「继续生成」
- 语音输入进行中：禁用「润色」「继续生成」
- 继续生成进行中：禁用「润色」「语音输入」

## 6. 后端接口契约

### 6.1 路由

- `POST /api/v1/blogs/:id/polish`
- 认证：需要 Bearer Token（复用现有 auth middleware）
- 响应：`text/event-stream`

### 6.2 请求体（JSON）

```json
{
  "title": "当前博客标题（可空）",
  "content": "当前博客正文（Markdown）"
}
```

备注：
- `:id` 用于权限校验与后续可观测性（不用于落库修改）

### 6.3 SSE 事件

- `chunk`：增量文本（Markdown 片段）
- `done`：结束标记（`[DONE]`）
- `error`：错误信息（中文，可直接 toast）

事件命名与现有继续生成保持一致，前端可复用 `fetchEventSource` 的处理结构。

## 7. Prompt 设计（润色输出规范）

### 7.1 系统约束

- 输出必须是中文（解释、步骤、提示等）。
- Mermaid 图表代码块不得包含 `style` / `classDef` / `linkStyle` 等自定义样式关键字。
- Mermaid 节点文本含特殊字符（如括号、幂符号）必须使用双引号包裹，例如 `A["O(1)"]`。

### 7.2 输出结构约定（便于前端展示）

润色输出按以下顺序生成：
1. `## 标题建议`：列出 3 个标题备选（编号列表）
2. `---` 分隔线
3. 正文：完整 Markdown 博客（H1-H4 结构）

前端策略：
- 先不解析结构，直接在 `润色预览` 渲染整段 Markdown。
- “应用润色结果”时，默认把分隔线后的正文部分写回 `content`；标题建议不写回 `title`。

## 8. 前端状态与数据流

### 8.1 新增状态

在 Editor 组件内新增（或抽到 hook）：
- `isPolishing: boolean`
- `polishedDraft: string`（润色草稿全文，包含标题建议 + 分隔线 + 正文）
- `activePreviewTab: 'preview' | 'polish'`
- `polishAbortController: AbortController | null`（用于取消）

### 8.2 流式接收

使用 `@microsoft/fetch-event-source` 以 POST 请求发起 SSE：
- 请求 body：`{ title, content }`
- 事件处理：
  - `chunk`：`polishedDraft += msg.data`
  - `done`：`isPolishing = false`
  - `error`：toast 错误并 `isPolishing = false`

### 8.3 应用逻辑

点击「应用润色结果」时：
- 从 `polishedDraft` 中提取“分隔线后的正文部分”（若提取失败则退化为使用全文）
- `setContent(extractedBody)`
- 保持 `title` 不变
- 触发现有自动保存机制（2 秒防抖）写回后端

## 9. 错误处理与降级

- SSE 连接失败：toast 提示“润色失败，请稍后重试”，不修改正文。
- 生成中取消：前端 abort 请求，保留当前已生成的 `polishedDraft`，正文不变。
- 权限/配额不足：后端返回明确错误信息（PaymentRequired/Unauthorized），前端 toast 透传。

## 10. 验收标准（DoD）

- 编辑器页出现「润色」按钮，且位于「语音输入」按钮旁边。
- 点击「润色」后自动切换到 `润色预览`，并能流式看到润色稿生成。
- 润色稿顶部包含 3 个标题建议，且不会自动修改标题输入框。
- 点击「应用润色结果」后，正文被替换为润色稿正文，且 2 秒后自动保存生效（刷新页面内容仍为润色后的正文）。
- 润色/语音输入/继续生成三者互斥，不会同时写入正文。
- Mermaid 规则仍被遵守（至少不会额外注入自定义样式关键字）。

## 11. 影响范围（预估文件）

前端：
- `frontend/src/components/Editor.tsx`（新增按钮、Tab、SSE 拉流与草稿状态）
- `frontend/src/services/*`（新增 polish API 调用封装，若现有项目对请求层有统一封装则复用）
- `frontend/src/hooks/*`（可选：抽 `usePolishStream`，与 `useBlogStream` 保持风格一致）

后端：
- `backend/internal/api/blog.go` 或 `backend/internal/api/stream.go`（新增路由 handler，建议放 stream.go 以复用 SSE 模式）
- `backend/internal/service/*`（新增 polish service，复用 `llm.DeepSeekClient.GenerateStream`）

