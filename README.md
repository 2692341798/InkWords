# 墨言博客助手 (InkWords)

## 1. 项目简介 (About)
**墨言博客助手 (InkWords)** 是一款基于 **DeepSeek API** 的智能化技术博客写作辅助平台。
该平台致力于解决技术人员“写文档难、拆解长篇教程繁琐”的痛点。通过强大的源码/文档解析与 LLM 逻辑重构能力，它能将复杂的本地文档或 Git 仓库（如官方教程、开源项目）快速转化为结构清晰、内容详实的系列技术博客。

## 2. 核心特性 (Features)
- 🚀 **多源输入与智能拆解**：支持本地文档（PDF/Word/Markdown）上传与 Git 仓库拉取。针对大于 5000 字的大项目，系统会自动规划大纲并并发拆解为系列博客。
- 👶 **面向新手 (小白友好)**：内置强约束 Prompt，遇到枯燥概念强制补充代码解释，确保读者能一步步独立复现文中的环境与逻辑。
- 📊 **原生图文渲染**：大量穿插无样式（无 `style` 污染）的 Mermaid 架构图、时序图，配合极简阅读风 UI，提供沉浸式的 Notion 级体验。
- 🔒 **极致安全**：源文件解析采用“阅后即焚”策略，不在服务器进行任何物理滞留。
- ✈️ **一键多发**：支持接入 OAuth2.0，将生成的博客一键分发至掘金、CSDN、知乎等第三方技术社区。

## 3. 技术栈与架构 (Tech Stack)
本项目采用前后端完全分离的 **Monorepo** 结构。
- **前端 (`frontend/`)**：
  - React 18 + Vite
  - Tailwind CSS + Shadcn UI (极简浅色阅读风)
  - Zustand (全局状态管理：Token、Markdown 实时内容)
  - React-Markdown + Rehype-Mermaid (拦截样式，原生渲染)
- **后端 (`backend/`)**：
  - Go (1.21+) + Gin 框架 (依赖注入分层架构)
  - SSE (Server-Sent Events) 打字机流式推送
  - GORM + PostgreSQL 14+ (数据持久化：快照、授权记录)
  - JWT & OAuth2.0 (系统内无状态鉴权与第三方平台授权)

## 4. 文档索引与 AI 协作指南 (AI Collaboration)
本项目深度拥抱 **Vibe Coding**（AI 辅助编程）理念。在项目根目录下，我们维护了 `.cursorrules` 文件作为全栈开发的核心护栏。

⚠️ **【重要提示给 AI 助手与开发者】**
在着手开发任何新功能、修改现有逻辑之前，**必须**先阅读 `.trae/documents/` 目录下的核心基准文档。严禁脱离文档上下文“闭门造车”：

- 📖 [1. 产品需求文档 (PRD)](.trae/documents/InkWords_PRD.md)
- 🏗️ [2. 项目架构文档 (Architecture)](.trae/documents/InkWords_Architecture.md)
- 💾 [3. 数据库设计文档 (Database)](.trae/documents/InkWords_Database.md)
- 🔌 [4. API 接口设计文档 (API)](.trae/documents/InkWords_API.md)

**Documentation as Code (代码与文档强同步)**：
当业务逻辑、表结构或接口路由发生变更时，AI 助手**必须在修改代码的同一个执行上下文中**同步更新上述四份基准文档。

## 5. 快速开始 (Quick Start)

### 5.1 环境要求
- **前端**：Node.js 18+ (`npm` 或 `yarn`)
- **后端**：Go 1.21+
- **数据库**：PostgreSQL 14+

### 5.2 启动后端
```bash
cd backend
# 复制并配置环境变量 (包含数据库连接与 DeepSeek API Key)
cp .env.example .env
# 安装依赖并运行服务
go mod tidy
go run ./cmd/server/main.go
```
*后端服务默认运行在 `http://localhost:8080`*

### 5.3 启动前端
```bash
cd frontend
# 安装依赖
npm install
# 启动开发服务器
npm run dev
```
*前端页面默认运行在 `http://localhost:5173`*
- 📅 [5. 开发计划与日志 (Plan & Log)](.trae/documents/InkWords_Development_Plan_and_Log.md)

- 💬 [6. AI 对话与决策摘要 (Conversation Log)](.trae/documents/InkWords_Conversation_Log.md)
