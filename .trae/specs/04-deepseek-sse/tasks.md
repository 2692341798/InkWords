# Tasks

- [x] Task 1: 封装后端 DeepSeek 客户端
  - [x] SubTask 1.1: 在 `backend/internal/llm/deepseek.go` 中，实现 HTTP Client 请求 DeepSeek API，开启 `stream=true`。
  - [x] SubTask 1.2: 封装针对 DeepSeek 返回 JSON Stream 的解析工具，将 Delta 内容提取出来。
- [x] Task 2: 实现后端 SSE 推流接口与业务
  - [x] SubTask 2.1: 在 `backend/internal/service/generator.go` 中实现核心生成流转（组装 Prompt、调用 LLM 并向 Channel/Callback 推送内容）。
  - [x] SubTask 2.2: 在 `backend/internal/api/stream.go` 中实现 Gin Handler，利用 `c.Stream` 监听 Service 层的数据并通过 SSE 发送 `event: chunk` 与 `event: done`。
  - [x] SubTask 2.3: 注册路由 `/api/v1/stream/generate`，确保其通过中间件的鉴权。
- [x] Task 3: 实现前端 SSE 通信与状态管理
  - [x] SubTask 3.1: 在 `frontend/src/store/streamStore.ts` 中创建 Zustand Store，用于存储 `content`、`isStreaming` 状态。
  - [x] SubTask 3.2: 编写 `frontend/src/hooks/useBlogStream.ts`，封装 `EventSource` 的连接、断开及对 `chunk`、`done` 事件的监听与 Store 更新。
- [x] Task 4: 实现前端 Markdown 与 Mermaid 实时渲染器
  - [x] SubTask 4.1: 安装 `react-markdown`、`rehype-mermaid` 及必要的语法高亮插件。
  - [x] SubTask 4.2: 在 `frontend/src/components/MarkdownEngine.tsx` 中编写渲染组件。
  - [x] SubTask 4.3: 配置 `rehype-mermaid` 以拦截并移除所有 `style` 或 `classDef` 样式属性，确保渲染出极简默认主题。

# Task Dependencies
- [Task 2] 依赖 [Task 1]
- [Task 3] 依赖 [Task 2] (用于联调验证)
- [Task 4] 可与 [Task 1, 2] 并行开发。
