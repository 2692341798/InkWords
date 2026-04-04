# 封装 DeepSeek 客户端与 SSE 实时推流 Spec

## Why
在墨言博客助手 (InkWords) 的核心功能中，我们需要将大模型生成的内容实时展示给用户，以缓解长文本生成带来的等待焦虑。因此，必须封装一个专用于调用 DeepSeek API 的客户端，并通过 Server-Sent Events (SSE) 技术建立起后端到前端的流式推流渲染通道。

## What Changes
- **后端**：新增 `internal/llm/deepseek.go` 封装对 DeepSeek Chat API 的请求。
- **后端**：新增 `internal/service/generator.go` 承载生成业务（组合 Prompt、调用 LLM、流式分发、持久化落库）。
- **后端**：新增 `internal/api/stream.go` 实现 `/api/v1/stream/generate` 的 SSE 接口。
- **前端**：在 `src/store/` 中增加负责存储流式文本的 Zustand store 模块。
- **前端**：新增 `src/hooks/useBlogStream.ts` 管理 EventSource 连接与状态流转。
- **前端**：引入 `react-markdown` 及 `rehype-mermaid` 构建核心渲染器 `MarkdownEngine` 与 `MermaidViewer`。

## Impact
- Affected specs: 博客生成核心流、流式文本渲染
- Affected code: 
  - `backend/internal/llm/*`
  - `backend/internal/service/generator.go`
  - `backend/internal/api/stream.go`
  - `frontend/src/store/stream.ts`
  - `frontend/src/hooks/useBlogStream.ts`
  - `frontend/src/components/MarkdownEngine.tsx`

## ADDED Requirements
### Requirement: 建立 DeepSeek 流式生成服务
系统 SHALL 通过 SSE 将 DeepSeek 模型的输出逐片 (Chunk) 推送给前端。
#### Scenario: 成功建立 SSE 流
- **WHEN** 前端携带 `task_id` 请求 `/api/v1/stream/generate`
- **THEN** 后端响应 `text/event-stream`，并不断发送 `event: chunk`，结束时发送 `event: done`。

### Requirement: 前端流式渲染与 Markdown 解析
前端 SHALL 能够接收 SSE 数据并在页面上实时渲染，包含代码高亮及无样式 Mermaid 图表。
#### Scenario: 实时渲染 Markdown 内容
- **WHEN** 前端接收到新的 `chunk` 数据
- **THEN** Zustand 状态自动拼接，Markdown 渲染器实时更新，Mermaid 图表在渲染时自动过滤自定义样式。
