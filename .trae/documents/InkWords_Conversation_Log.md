# 墨言博客助手 (InkWords) - AI 对话与决策摘要 (Conversation Log)
> **目的**：记录在 Vibe Coding 过程中，每一次核心对话的上下文、用户指令意图以及关键架构决策。以便在长周期的开发中，不论更换 AI 会话窗口还是重新梳理思路，都能快速找回项目背景。

---

## 2026-04-04 (项目规划与脚手架搭建)

### 对话 1：项目初步规划 (Plan Mode)
- **用户需求**：要求开发一个基于 DeepSeek API 的博客写作助手，前端使用 React 18，后端使用 Go，支持 Word/PDF/MD 及 Git 仓库解析，大项目自动拆解，Mermaid 图表无样式，并要求配置严格的工程 rules。
- **AI 动作**：在 Plan 模式下起草了初步的《开发计划与架构设计》。
- **决策/变更**：用户多次反馈修改意向，要求强调“小白友好、可复现、枯燥概念加代码”的核心生成逻辑。

### 对话 2：文档共创流程引入 (Doc Co-Authoring)
- **用户需求**：用户拒绝将所有计划写在一个文件里，要求采用“结构化共创流程”，逐个拆分出产品需求(PRD)、数据库、架构文档。
- **AI 动作**：启动 `doc-coauthoring` 技能，通过多轮提问，与用户依次完成了：
  1. **PRD**：确立 5000 字拆分限制、50MB 体积限制、阅后即焚策略。
  2. **Architecture**：确立 Monorepo、SSE 流推送、并发调度策略及 JWT+OAuth 体系。
  3. **Database**：确立了父子系列博客关联 (`parent_id`) 及雪花 ID 策略。
- **决策/变更**：
  - 架构设计过程中，用户发现缺少“账号体系”，临时决定引入 GitHub/Wechat 第三方 OAuth 一键登录。
  - 数据库选型阶段，用户要求**将 MySQL 替换为 PostgreSQL 14+**，并采用 `TEXT` 和 `UUID` 类型。

### 对话 3：补充核心基准文档与重构 Rules
- **用户需求**：发现缺少前后端 API 定义，要求补全。随后要求将 `.cursorrules` 按规则类型拆分，并强调自己极度偏好在 Vibe Coding 中使用**共创模式**。
- **AI 动作**：
  - 补充了第 4 份文档《API 接口规范文档》。
  - 将 `.cursorrules` 删除，拆分为 `.trae/rules/` 下的三个 Markdown 文件（架构规范、业务约束、Vibe协作规范）。
  - 在规则中确立了**“强制阅读上下文 (Context First)”**的第一铁律。

### 对话 4：开发计划与里程碑
- **用户需求**：基于已有文档，要求给出开发计划和每日开发日志。
- **AI 动作**：使用共创模式编写了《开发计划与日志》，划分了 MVP、Alpha、Beta、V1.0 四个里程碑，并对各模块进行了耗时预估和强单测约束。

### 对话 5：创建对话日志文件 
- **用户需求**：在 `.trae/documents/` 目录下添加一个文件，专门用于简单记录每次的对话。
- **AI 动作**：创建了本文件 (`InkWords_Conversation_Log.md`)，并回溯记录了今日从项目启动到脚手架搭建的完整对话脉络与关键决策点。

### 对话 6：完成 MVP 骨架与基建开发
- **用户需求**：要求继续按照开发计划执行。
- **AI 动作**：
  1. 使用 Spec 模式执行并完成了“基础架构与数据库搭建”(Task 1, 2)
  2. 使用 Spec 模式执行并完成了“第三方 OAuth2.0 与 JWT 签发”(Task 3)
  3. 使用 Spec 模式执行并完成了“基础 PDF/MD 解析器”(Task 4)
- **决策/变更**：
  - 遇到 PDF 流解析问题，决定采用生成临时文件然后 `defer os.Remove` 的策略，确保兼顾 `io.ReaderAt` 接口需求与“阅后即焚”的安全需求。

### 对话 7：封装 DeepSeek 与 SSE 流式推流通道 (当前)
- **用户需求**：继续开发“封装 DeepSeek 客户端，建立前后端 SSE 实时推流渲染通道”功能。
- **AI 动作**：
  1. 编写了前后端 `spec.md` 及相关任务，并执行开发。
  2. 封装了 `DeepSeekClient` 以及 Gin 的 SSE 推流接口 `/api/v1/stream/generate`。
  3. 前端集成了 Zustand、`fetch-event-source`，以及基于 `react-markdown` 的视图组件。
  4. 利用自建 Remark 插件移除了 Mermaid 图表代码块内的样式注入。
  5. 自动更新了 `InkWords_API.md`、`InkWords_Architecture.md` 和 `InkWords_PRD.md`，随后又将开发记录同步至 `InkWords_Development_Plan_and_Log.md` 和本文件。
- **决策/变更**：
  - 因为原生 EventSource 不支持 POST 请求，且基于 Ticket GET 状态拉取长文本设计过于复杂，**更改架构决策**为前端携带提取出的纯文本（`source_content`），直接向后端发起 `POST` 请求获取 SSE 数据流（使用 `@microsoft/fetch-event-source`）。

---
