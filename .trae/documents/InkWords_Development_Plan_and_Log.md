# 墨言博客助手 (InkWords) - 开发计划与日志
> **目标**：跟踪项目的核心开发模块、里程碑进度以及每日开发记录。

## 1. 里程碑划分 (Milestones)

### 阶段 1: MVP (核心单篇生成)
**目标**：跑通前后端最小核心闭环，完成单篇轻量级文档的智能转换。
- [x] 完成 Go + Gin + PostgreSQL 基础架构搭建与依赖注入。
- [x] 搭建前端 React 18 + Zustand + Tailwind 极简阅读风骨架。
- [x] 实现第三方（GitHub/WeChat）OAuth2 登录与 JWT 签发。
- [x] 实现基础 PDF/MD 文本解析器 (阅后即焚)。
- [ ] 封装 DeepSeek 客户端，建立前后端 SSE 实时推流渲染通道。

### 阶段 2: Alpha (大项目智能拆解)
**目标**：支持超长代码库的解析与系列博客的生成。
- 接入 Git 仓库拉取，实现代码文件过滤与提取规则。
- 实现大项目评估逻辑，开发“大纲规划 -> 并发调度生成 -> 拼接”的复杂调度机制。
- 前端深度定制 `react-markdown`，引入并严格控制 `rehype-mermaid` 实现无样式图表渲染。

### 阶段 3: Beta (历史记录与编辑)
**目标**：完善数据库持久化与用户创作体验。
- 完成 `blogs` 表的自引用查询与系列博客展示。
- 前端开发类似 Notion 的双栏 Markdown 二次编辑器。
- 支持自动保存、覆盖更新及文章导出 (MD/PDF)。

### 阶段 4: V1.0 (商业化与多端分发)
**目标**：打通流量分发与用户额度体系。
- 接入掘金、CSDN OpenAPI，实现一键授权发文。
- 上线用户用量统计（Tokens 消耗）及 `subscription_tier`（订阅会员）防刷限流系统。

## 2. 核心模块拆解与时间预估

| 模块类别 | 核心功能点 | 难度/风险 | 预计开发时间 |
| --- | --- | --- | --- |
| **Backend (Go)** | OAuth2 第三方授权与 JWT 签发 | 中 | 1.5 天 |
| | DocParser / GitFetcher 源码提取与过滤算法 | 高 (边缘格式多) | 2 天 |
| | DeepSeek API 封装与流式 SSE 转发管道 | 中 | 1.5 天 |
| | 大项目并发调度引擎 (Goroutine Pool) | 极高 (死锁风险) | 3 天 |
| | PostgreSQL GORM 模型映射与组合查询优化 | 低 | 1 天 |
| **Frontend (React)** | Shadcn UI 基础布局与 Zustand 状态挂载 | 低 | 1 天 |
| | 拖拽上传与仓库 URL 解析组件 | 中 | 1 天 |
| | Markdown 实时打字机渲染与 Mermaid 图表接管 | 高 (样式覆盖难) | 2.5 天 |
| | 类似 Notion 的二次编辑器与状态同步 | 中 | 2 天 |
| **Integration** | 第三方发文 OpenAPI 对接与 Token 加密 | 高 (各平台不一) | 3 天 |

## 3. 测试与联调计划
遵循 Vibe Coding **“小步迭代与强制验证”** 的铁律，在实际开发过程中，严禁越过测试环节强行合并代码。

### 3.1 后端单元测试 (Unit Testing)
- **重点目标**：`internal/parser` (Git与文档的提取清洗)、`internal/llm` (Prompt构建) 及 `internal/service` (大项目拆解算法与 Goroutine 调度)。
- **约束**：使用 Go 的内置 `testing` 框架和 `testify` 库。所有核心 Service 必须包含 Mock（如 `gomock`），特别是对 DeepSeek API 的 Mock 测试，确保在断网下仍能测试内部状态机的流转。

### 3.2 前端组件测试 (Component Testing)
- **重点目标**：Markdown 渲染器（验证是否准确剥离了 Mermaid 的样式 `style`）以及 Zustand 的状态变化。
- **约束**：引入 `Jest` 和 `React Testing Library`，为 `MermaidViewer` 和核心的 `useBlogStream` Hooks 编写隔离测试。

### 3.3 端到端联调测试 (E2E Integration)
- **重点目标**：打通上传 -> SSE 流接收 -> 渲染 -> 落库保存的完整闭环。
- **约束**：后端在每次 API 开发完成后，使用 Postman 导出或在代码中编写 `httptest` 进行路由联调验证；前后端联调阶段关注 50MB 超大文件的上传稳定性及流媒体推送时的断线重连（Resume）。

### 3.4 真实场景/效果验证 (Reader Testing)
- **目标**：验证 AI 的 Prompt 策略是否真的达到了“小白友好、可独立复现、图文并茂”的要求。
- **测试用例**：
  1. 上传一个极简的小脚本（如 100 行 Go 代码），验证生成的单篇内容是否丰富详实。
  2. 导入一个著名长篇开源项目或官方教程（如 React 官方教程），测试系统的“复杂度评估器”是否能准确切分为多个结构连贯的 5000 字篇章。

## 4. 每日开发日志 (Dev Log)
> 该区域将由 Vibe Coding 工程师（AI 助手）在每天/每次开发周期结束时，如实记录当天的完成事项、遇到的技术坑点及架构小规模调整。

### 2026-04-04 (规划与脚手架阶段)
- **开发模块**: [全栈架构与基准文档设计]
- **完成事项**:
  1. 完成产品需求文档、数据库、架构设计、API 接口这四份基准 Markdown。
  2. 确立了遵循“文档即代码”与“共创模式优先”的 `.trae/rules` 规范。
  3. 创建了 README.md 项目入口指引。
  4. 初始化了 `backend/` 目录下的 Go+Gin 项目骨架，跑通了 `/api/v1/ping` 健康检查。
- **踩坑记录 / 架构调整**: 
  - 架构设计阶段将数据库选型由 MySQL 修改为 PostgreSQL 14+，以利用其原生的 `UUIDv4` 和更优秀的 `TEXT` 存储及部分索引特性。
  - 在 PRD 与 API 阶段，发现缺少了用户注册体系，及时补充了 GitHub/Wechat 第三方 OAuth2.0 一键登录机制，并完善了数据库的 `subscription_tier` 和 `tokens_used` 字段。
- **遗留问题 (TODO)**: 
  - ~~尚未初始化 `frontend/` 目录的 React 骨架。~~ (已在后续开发中完成)
  - ~~后端的 `gorm` 数据库模型 (Entity) 暂未建立。~~ (已在后续开发中完成)

### 2026-04-04 (MVP - 基础骨架与鉴权模块)
- **开发模块**: [前端骨架初始化, 后端数据库模型, OAuth2 与 JWT 鉴权]
- **完成事项**:
  1. **前端**：在 `frontend/` 目录下初始化了基于 Vite 的 React 18 (TS) 项目，并成功集成了最新版的 Tailwind CSS v4、Shadcn UI 以及 Zustand 状态管理。
  2. **后端模型**：在 `backend/internal/model/` 目录下完成了基于 `GORM` 的 `User`, `Blog`, `OAuthToken` 实体模型定义，支持了 UUIDv4 主键生成、软删除，并在 `blogs` 实现了多字段组合索引。
  3. **后端鉴权**：集成了 `golang-jwt/jwt/v5` 与 `golang.org/x/oauth2`，编写了 `auth` 的 Middleware 拦截器。打通了 GitHub OAuth2 授权回调获取用户信息并 UPSERT 数据库记录的完整闭环流程。
  4. **版本控制**：项目成功梳理了根目录及子目录的 `.gitignore` 过滤规则，初始化 Git 仓库，并推送至 GitHub。
- **踩坑记录 / 架构调整**: 
  - Shadcn UI 最新版本（v4.x）默认使用并推荐了 Tailwind CSS v4（弃用旧版 tailwind.config.js），因此在前端安装时做了最新技术的适配，使用 `@tailwindcss/vite` 插件替代了旧版的 PostCSS 插件方案。
- **遗留问题 (TODO)**: 
  - ~~尚未实现核心的文件解析器（DocParser/GitFetcher）及与 DeepSeek API 的 SSE 流式交互管道。~~ (已部分实现基础 DocParser，待实现 GitFetcher 及 SSE 流式交互管道)

### 2026-04-04 (MVP - 基础解析器开发)
- **开发模块**: [后端文件解析器]
- **完成事项**:
  1. **文档解析器**: 在 `backend/internal/parser/` 中实现了 `DocParser`，支持解析 PDF 与 MD/TXT 纯文本格式。
  2. **安全策略 (阅后即焚)**: 实现了严格的临时文件处理机制，使用 `defer os.Remove(tempFile.Name())` 确保无论是上传的文件流还是内存数据，解析完成后立即物理销毁，不产生任何持久化实体。
  3. **单元测试**: 引入了 `github.com/jung-kurt/gofpdf` 生成测试 PDF，并完成了 `doc_parser_test.go`，覆盖了纯文本提取、PDF 解析与不支持格式的错误拦截。
- **踩坑记录 / 架构调整**: 
  - PDF 文本提取通常需要实现 `io.ReaderAt`，而用户上传的文件或网络流默认仅为 `io.Reader`。为解决此冲突并兼顾“阅后即焚”的原则，统一将流转存到 `os.CreateTemp` 的临时文件中再行解析，这样既获得了 `io.ReaderAt` 的能力，又能在函数结束时精准清空。
- **遗留问题 (TODO)**: 
  - 尚未封装 DeepSeek 客户端。
  - 需要建立前后端 SSE 实时推流渲染通道。