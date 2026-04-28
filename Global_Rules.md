# Trae Solo 全局规则（合并版 · Vibe Coding 最佳实践）

> 适用范围：本仓库 / 本项目的默认工程与协作约束（前端 React、后端 Go、PostgreSQL、Docker Compose、Git 发布流程）。  
> 目标：更稳定、更可控、更少返工；同时兼顾安全、性能、可维护性与可验证交付。  
> 生成方式：基于现有 4 份规则文档“层级整合 + 统一表述”去重合并。  
> 更新时间：2026-04-28

---

## 目录

- [0. 最高优先级：全局原则（永远优先于其它规则）](#0-最高优先级全局原则永远优先于其它规则)
- [1. 术语与默认约定](#1-术语与默认约定)
- [2. 标准工作流（必须遵循）](#2-标准工作流必须遵循)
- [3. 架构与工程约束（Monorepo / 模块化 / 注释）](#3-架构与工程约束monorepo--模块化--注释)
- [4. 后端规范（Go + Gin + PostgreSQL）](#4-后端规范go--gin--postgresql)
- [5. 前端规范（React 18 + Vite + Tailwind + shadcn/ui + Zustand）](#5-前端规范react-18--vite--tailwind--shadcnui--zustand)
- [6. 容器化与环境一致性（Docker-First）](#6-容器化与环境一致性docker-first)
- [7. 数据库规则（PostgreSQL）](#7-数据库规则postgresql)
- [8. 文档即代码（Docs-as-Code）](#8-文档即代码docs-as-code)
- [9. Git 提交与发布规范（Conventional Commits + Tag）](#9-git-提交与发布规范conventional-commits--tag)
- [10. Obsidian 第二大脑（LLM Wiki Pattern）](#10-obsidian-第二大脑llm-wiki-pattern)
- [附录 A：来源与映射](#附录-a来源与映射)

---

## 0. 最高优先级：全局原则（永远优先于其它规则）

若本文件的不同条款存在“看似冲突”，一律按以下优先级理解与执行（从高到低）：

1. **先对齐再动手**：信息不足时必须先补齐需求与约束，通过询问确认，禁止凭感觉大改。  
2. **可验证优先**：任何“已完成/已修复/可上线”的结论必须附带可复现验证方式与结果。  
3. **最小改动原则**：默认做最小可行变更；与目标无关的重构/升级/格式化不得混入同一次变更。  
4. **安全默认开启**：默认拒绝不安全做法（明文密钥、拼接 SQL、缺鉴权、过度权限等）。  
5. **性能有预算**：性能敏感路径必须可度量（指标/基准/Explain），不清楚就先加观测与验证。  
6. **任务启动必须回声确认**：收到开发请求后，必须先回述理解（需求、范围、技术栈、约束）并明确将调用的 skills；得到确认后再进入 Kickoff（若用户明确要求“无需确认，直接开始”，可跳过，但必须在 Kickoff 中注明已跳过及理由）。

---

## 1. 术语与默认约定

- **验证**：可执行、可复现的证据之一：测试命令与结果、复现步骤与现象对比、截图/日志片段、SQL `EXPLAIN/EXPLAIN ANALYZE`、性能基准数据。  
- **关键路径**：用户高频/核心链路（登录、列表加载、核心 API、生成/发布等）。  
- **关键 SQL**：出现在关键路径、慢查询、或可能扫描大表/影响写入的 SQL（新增/修改都算）。  
- **最小改动**：优先让单次变更“可 review/可回滚”。当预计改动跨 3+ 模块、影响面不清、或涉及 DB + API + UI 同时改时，默认先拆分并请求确认。

---

## 2. 标准工作流（必须遵循）

### 2.1 需求回声（开始前必做）

在任何改动前（写代码/改配置/改文档），必须先输出“需求回声”，至少包含：

- 你理解的目标与问题
- 范围（做/不做）
- 约束与风险点（安全/兼容/性能/时间）
- 计划使用的 skills（按规则强制映射）
- 验证计划（如何证明完成）

### 2.2 Kickoff Brief（强制输出）

当需求回声得到确认后（或用户明确要求跳过确认后），必须输出：

```md
【Kickoff】
- 目标：
- 范围（做/不做）：
- 影响面（模块/API/表）：
- 计划改动（文件/目录/组件/包）：
- 验证计划（我将运行/提供）：
- 待确认问题：
- 已激活 skills：
```

### 2.3 小步迭代（一次只解决一个主题）

- 拆成可回滚的小步骤，避免“一把梭”。  
- 禁止“顺手重构”：与目标无关的格式化、重排、抽象升级不得混入同一变更。  
- 对关键路径/关键 SQL 的变更优先写测试或提供替代验证证据。

### 2.4 Closeout Brief（强制输出）

给出最终结论前必须输出：

```md
【Closeout】
- 变更摘要：
- 风险与回滚：
- 验证结果（实际执行/实际观察到）：
```

---

## 3. 架构与工程约束（Monorepo / 模块化 / 注释）

### 3.1 Monorepo 目录隔离（硬约束）

- **前端代码全部放入 `frontend/`**  
- **后端代码全部放入 `backend/`**  
- 严格隔离：不得跨目录引用（例如后端 import 前端、前端直接读取后端源码）。

### 3.2 模块化与单文件体积（强约束）

- **500 - 800 行警戒线**：单文件超过 500 行应主动提出拆分计划；**超过 800 行强制拦截**（拒绝继续堆砌）。  
- 优先“高内聚低耦合”：将相关逻辑靠近业务领域/组件边界，避免“万能工具包”。

### 3.3 业务逻辑注释（Why-first）

- 必须在关键代码块上方写清楚注释，重点解释 **为什么这么做（Why）**，而不是仅描述做了什么（What）。  

---

## 4. 后端规范（Go + Gin + PostgreSQL）

### 4.1 技术栈与目录结构

- Go 1.21+，Gin，GORM/pgx，PostgreSQL 14+  
- 目录结构遵循标准 Go 约定（例如 `cmd/server/`, `internal/api/`, `internal/service/`, `internal/parser/`, `internal/llm/`, `pkg/`）

### 4.2 依赖注入（DI）

- 业务逻辑必须使用依赖注入，保证模块低耦合、高内聚；接口隔离清晰，便于测试替换。

### 4.3 错误处理（分层、wrap、稳定输出）

- 对内保留根因：使用 wrap（例如 `%w`）保留链路。  
- 对外输出稳定错误码与可读信息，禁止吞错、禁止直接泄漏内部实现细节。

### 4.4 并发与流式通信（LLM 调用硬约束）

- 调用 LLM 大模型时必须使用 **Goroutine 池 + semaphore** 限制并发上限，避免 API 限流或内存溢出。  
- 使用 **SSE** 向前端推送生成进度与内容。

### 4.5 数据库交互边界

- 数据库仅用于结构化存储（例如博客历史记录、第三方平台授权信息等）。  
- 必须使用参数化/安全写法（GORM/pgx），禁止字符串拼接构造 SQL。  
- 妥善处理数据库连接生命周期（连接池、超时、关闭、重试策略等）。

### 4.6 Godoc 注释规范

- 所有公开（Public）的函数、结构体、接口必须编写标准 Godoc 注释。

---

## 5. 前端规范（React 18 + Vite + Tailwind + shadcn/ui + Zustand）

### 5.1 技术栈与组件规范

- React 18 + Vite + Tailwind CSS + shadcn/ui + Zustand  
- **只能使用函数式组件 + Hooks**，严禁类组件。

### 5.2 UI 与体验（极简阅读风）

- UI 强制 Tailwind + shadcn/ui，整体风格：**极简阅读风（浅色主导），对标 Notion 体验**。  
- **前端界面所有展示文本必须使用中文**（按钮、提示、错误信息、空状态等）。

### 5.3 逻辑复用与代码组织

- 可复用逻辑抽到自定义 Hooks：`frontend/src/hooks`  
- API 请求与服务调用抽到模块：`frontend/src/services`  
- 复杂全局状态放在 `store/`；避免把服务端数据当全局 state 纯搬运。

### 5.4 通信与跨域（本地开发）

- 本地开发通过 Vite proxy 解决跨域。  
- **流式请求必须使用 `@microsoft/fetch-event-source` 发起 POST**（以携带大文本 payload）。

### 5.5 React 性能与副作用

- 禁止在 render 阶段产生副作用；副作用必须在 effect 或事件处理里。  
- 列表超过阈值默认考虑虚拟列表。  
- 任何可能增加 bundle 的改动需要说明原因与收益。

### 5.6 JSDoc 注释规范

- 核心 Hooks、复杂组件必须包含 JSDoc 注释。

### 5.7 A11y 与主题

- 满足键盘交互、可见焦点、语义化 ARIA。  
- 所有组件必须支持 Light/Dark 并可用；主题切换策略统一。  
- Token 化优先：避免在业务代码中大量使用随意的任意值，优先 CSS 变量与 Tailwind theme 映射。

---

## 6. 容器化与环境一致性（Docker-First）

### 6.1 环境一致性优先

- 本地开发、测试、生产部署默认使用 Docker 与 Docker Compose，确保跨平台一致性。

### 6.2 一键拉起与重启（强约束）

- 必须支持一条命令一键拉起并互通：
  - `docker compose up -d --build`
- 如需应用新代码或重启，必须使用“一键重启”：
  - `docker compose down && docker compose up -d --build`

### 6.3 访问入口约束

- 后端不提供页面服务；前端由独立 Nginx 容器提供静态文件服务与 API 代理。  
- 项目启动后 **必须通过前端入口 `http://localhost` 访问应用**，严禁通过后端端口直接访问页面。

### 6.4 多阶段构建（Multi-Stage Build）

- Go 后端：`golang:1.21-alpine` 编译，`alpine` 或 `scratch` 运行，缩减镜像体积。  
- React 前端：`node:18-alpine` 构建产物，`nginx:alpine` 运行与代理。

### 6.5 网络、配置与安全

- 容器通过 Docker 内部网络通信（例如 `inkwords-network`）。  
- 后端连数据库必须使用服务名（例如 `postgres`），禁止写 `localhost`。  
- **绝不把敏感信息硬编码到 Dockerfile**；统一通过 `.env` 或 compose `environment` 注入。  
- PostgreSQL 必须挂载 Docker Volume 持久化数据。

---

## 7. 数据库规则（PostgreSQL）

- **必须参数化查询**；禁止字符串拼接 SQL。  
- 新增/修改关键 SQL 必须提供 `EXPLAIN` 或 `EXPLAIN ANALYZE`（至少在本地/测试环境）。  
- 新增索引需说明：读收益、写放大、存储成本。

---

## 8. 文档即代码（Docs-as-Code）

- **文档与代码同源**：架构、API、Runbook 必须进入仓库并版本化。  
- 当 API / DB / 环境变量发生变更时，必须同步更新对应文档。  

---

## 9. Git 提交与发布规范（Conventional Commits + Tag）

### 9.1 提交前强制检查（提交门禁）

在执行 `git commit` 并 push 到 GitHub 之前，必须：

1. **强制对比变更**：使用 `git diff`（或等效方式）明确本次修改与上一次提交/暂存区的区别，梳理核心逻辑与影响范围。  
2. **强制更新项目文档**：确保文档与最新业务逻辑/架构保持强同步。至少覆盖以下文件（路径以仓库实际为准）：
   - `.trae/documents/InkWords_API.md`
   - `.trae/documents/InkWords_Architecture.md`
   - `.trae/documents/InkWords_Conversation_Log.md`
   - `.trae/documents/InkWords_Database.md`
   - `.trae/documents/InkWords_Development_Plan_and_Log.md`
   - `.trae/documents/InkWords_PRD.md`
   - `README.md`

### 9.2 Commit Message 规范

- 严格遵循 **Conventional Commits（Angular 规范）**：`feat:`, `fix:`, `docs:`, `refactor:`, `chore:` 等。  
- Commit message 必须包含上下文：首行概括；空一行后详细写清 **Why（为什么）** 与 **What（改了什么）**，必要时中英文结合。  
- 原子化提交：一次提交只做一个独立主题，避免把不相关改动混在一起。

### 9.3 Tag 与推送流程

- 发布前先 `git tag` 查看当前版本号进度。  
- 遵循 Semantic Versioning：`Major.Minor.Patch` 打新 tag。  
- 推送代码与标签：`git push` 与 `git push --tags`（或等效流程），确保版本标记同步。

---

## 10. Obsidian 第二大脑（Karpathy LLM Wiki Pattern）

基于 [karpathy-llm-wiki](https://github.com/Astro-Han/karpathy-llm-wiki) 的工作流规范与 [claude-obsidian](https://github.com/AgriciDaniel/claude-obsidian) 最佳实践，当处理或维护 Obsidian 知识库时，必须遵循以下机制：

### 10.1 核心操作机制

1. **Ingest（知识摄入）**：将原始资料（网页/PDF/文本）存入 `.raw/` 目录，大模型自动提取实体与概念，编译为 `wiki/` 下的持久化 Markdown 页面，并建立交叉引用。
2. **Query（带源检索）**：回答问题时，必须先读取 `wiki/hot.md`（近期会话记忆）与 `wiki/index.md`（全局目录），并在回答中**引用（Cite）具体的 Wiki 页面**，严禁仅基于训练数据凭空生成。
3. **Lint（大脑体检）**：定期检查孤立页面（Orphans）、死链（Dead links）、知识空白（Gaps），并输出修复建议与报告。
4. **Autoresearch（自主研究）**：在遇到知识盲区时，执行“搜索 -> 抓取 -> 综合 -> 归档”的自主研究循环，并自动生成相关 Wiki 页面。

### 10.2 目录与文件架构

- **`.raw/`**：不可变（Immutable）的原始输入资料库。
- **`wiki/concepts/` 与 `wiki/entities/`**：提取的核心知识与实体卡片。
- **`wiki/sources/`**：原始资料的元数据卡片。
- **`wiki/index.md`**：全局主目录（Master catalog）。
- **`wiki/log.md`**：只追加（Append-only）的操作日志。
- **`wiki/hot.md`**：热缓存（Hot cache），用于跨会话保留近期上下文，无需反复回忆。
- **`wiki/meta/dashboard.base`**：基于 Obsidian Bases 插件的主控制台（替代 Dataview）。

### 10.3 页面结构（Frontmatter 规范）

```yaml
---
type: <source|entity|concept|domain|meta>
title: "标题"
created: YYYY-MM-DD
updated: YYYY-MM-DD
tags:
  - "#domain/领域名"
status: <seed|developing|mature|evergreen>
related:
  - "[[相关页面]]"
banner: "_attachments/images/your-image.png" # 可选：Notion 风格头图
---
```

### 10.4 维护与扩展约束

- **不可变原则**：严禁修改 `.raw/` 下的原始文件，大模型只负责生成和更新 `wiki/` 目录下的提炼卡片。
- **状态强同步**：每次 Ingest 或修改后，必须同步更新 `wiki/index.md`、在 `wiki/log.md` 追加记录，并覆盖更新 `wiki/hot.md`。
- **矛盾处理**：当发现不同来源的知识存在冲突时，使用 `[!contradiction]` Callout 进行标记并附上信息源。
- **可视化与 MCP**：支持通过 `/canvas` 组织视觉卡片，推荐配置 MCP（Local REST API 或 filesystem）以实现工具对 Vault 的直接读写。

---

## 附录 A：来源与映射

### A.1 来源文件

- `.trae/rules/global-vibe-coding-rules.md`
- `.trae/rules/vibe-coding-workflow.md`
- `.trae/rules/architecture-and-engineering.md`
- `.trae/rules/git-and-release.md`

### A.2 映射说明（便于追溯）

- **全局原则 / 术语 / 工作流**：来自 global-vibe-coding-rules + vibe-coding-workflow，并统一口径合并。  
- **Monorepo / 工程与注释 / Docker / 前后端技术栈**：来自 architecture-and-engineering，并按主题重排。  
- **提交与发布流程**：来自 git-and-release，并与“文档即代码”条款统一放置在交付链路附近。

