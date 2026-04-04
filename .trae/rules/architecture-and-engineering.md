# 架构与工程规范

## 1. 整体架构
- **Monorepo 结构**：前端代码全部放入 `frontend/` 目录，后端代码全部放入 `backend/` 目录。严格隔离，不得交叉引用。

## 2. 后端规范 (Go + Gin + PostgreSQL)
- **技术栈**：Go 1.21+, Gin 框架, GORM / pgx, PostgreSQL 14+。
- **目录结构**：遵循标准 Go 目录结构（`cmd/server/`, `internal/api/`, `internal/service/`, `internal/parser/`, `internal/llm/`, `pkg/`）。
- **依赖注入**：业务逻辑必须使用依赖注入（Dependency Injection），保证模块低耦合、高内聚。
- **命名规范**：函数名、变量名必须具备极强的自解释性，符合 Effective Go 规范。
- **数据库交互**：
  - 数据库仅用于存储博客的历史记录、第三方平台（掘金/CSDN）授权信息等结构化数据。
  - 必须做好防 SQL 注入处理（推荐使用 GORM 或 pgx 的安全写法），并妥善处理数据库连接的生命周期。

## 3. 前端规范 (React 18)
- **技术栈**：React 18 + Vite + Tailwind CSS + Shadcn UI + Zustand。
- **组件规范**：全量使用函数式组件与 Hooks，严禁使用类组件。
- **UI 与体验**：组件库强制使用 Tailwind CSS 配合 Shadcn UI，保持“极简阅读风（浅色主导）”，对标 Notion 体验。**前端界面所有的文本展示必须使用中文**。
- **逻辑复用**：提取可复用的逻辑到自定义 Hooks (`src/hooks`)，分离 API 请求到单独的模块 (`src/services`)。
