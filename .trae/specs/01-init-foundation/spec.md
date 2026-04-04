# Init Foundation Spec

## Why
项目目前已经初始化了后端的基础 Go+Gin 骨架，并跑通了健康检查。但是根据《开发计划与日志》中的遗留问题（TODO），尚未初始化 `frontend/` 目录的 React 骨架，同时后端的 `gorm` 数据库模型 (Entity) 也暂未建立。这是完成 MVP（最小核心闭环）的基础，也是目前开发计划中的“第一个功能”。

## What Changes
- 在 `frontend/` 目录下初始化 React 18 + Vite + Tailwind CSS + Shadcn UI + Zustand 前端骨架。
- 在 `backend/internal/model/` 目录下建立基于 PostgreSQL 的 GORM 数据模型（User, Blog, OAuthToken）。

## Impact
- Affected specs: MVP 阶段的基础架构搭建。
- Affected code: 
  - `frontend/` (新增)
  - `backend/internal/model/` (新增)
  - `backend/go.mod` (更新，添加 gorm 等依赖)

## ADDED Requirements
### Requirement: 初始化前端 React 骨架
The system SHALL provide 一个基于 Vite 初始化的 React 18 前端项目，集成 Tailwind CSS 和 Shadcn UI，并包含状态管理 Zustand 的基础配置，代码结构符合规范。

#### Scenario: 成功启动前端
- **WHEN** 开发者运行 `npm run dev`
- **THEN** 前端服务正常启动并能访问基础页面。

### Requirement: 建立后端 GORM 模型
The system SHALL provide 按照 `InkWords_Database.md` 设计规范编写的 GORM 模型（Entity），包括主键策略（UUIDv4）、软删除、以及相应的表结构（users, blogs, oauth_tokens）。

#### Scenario: 成功迁移数据库模型
- **WHEN** 开发者调用数据库自动迁移函数
- **THEN** PostgreSQL 数据库中成功创建 `users`, `blogs`, `oauth_tokens` 表，并包含对应的索引。
