# 05-project-decomposition Spec

## Why
在 MVP 阶段完成了轻量级单篇文档的转换，但在实际开发与学习场景中，开发者往往需要处理庞大的官方教程或完整的 GitHub/Gitee 源码仓库。为了帮助用户更轻松地理解并复现大型项目，需要系统支持解析 Git 仓库，自动评估复杂度并智能拆解为多篇（单篇<5000字）小白友好的系列博客。

## What Changes
- 新增 Git 仓库拉取与文件过滤模块 (`GitFetcher`)，支持纯文本源码的提取。
- 实现大项目评估逻辑，针对超长文本调用 LLM 生成“系列大纲”。
- 实现基于 Goroutine 的并发调度生成引擎，按大纲并行生成多个篇章。
- 更新 API 路由，支持两步走流程：1) 提交仓库 URL 获取大纲；2) 确认大纲后开始并发生成与 SSE 推流。

## Impact
- Affected specs: MVP 阶段的基础解析器与 SSE 流式生成通道。
- Affected code:
  - `backend/internal/parser/` (新增 `git_fetcher.go`)
  - `backend/internal/service/` (新增 `decomposition.go` 与调度逻辑)
  - `backend/internal/api/` (新增获取大纲与确认生成路由)

## ADDED Requirements
### Requirement: Git 仓库解析与过滤
系统 SHALL 支持用户输入 Git 仓库 URL，克隆至临时目录，过滤非必要文件（如 `node_modules`, `.git`, 二进制文件），并提取源码文本。提取完成后 MUST 立即删除本地克隆目录（阅后即焚）。

### Requirement: 大纲规划与评估
当提取的源码文本量超过阈值（预估会生成>5000字的内容），系统 SHALL 自动调用 LLM 进行项目结构评估，并返回包含多个篇章（如基础篇、架构篇）的大纲 JSON 给前端。

### Requirement: 并发调度生成与落库
在用户确认大纲后，系统 SHALL 启动 Goroutine 池，针对每个篇章的上下文并发调用 DeepSeek API 进行生成，并将生成的每篇文章正确设置 `ParentID` 和 `ChapterSort` 落库。
