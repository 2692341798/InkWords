# OAuth2 & JWT Authentication Spec

## Why
根据《开发计划与日志》中 MVP 阶段的第二个目标，项目需要实现基于 GitHub/WeChat 的第三方 OAuth2 登录与 JWT 签发。这是识别用户身份、进行计费额度扣减以及保障后续 API 安全的核心基础模块。

## What Changes
- 添加基于 `golang-jwt/jwt/v5` 的 JWT 生成与解析工具。
- 增加基于 `golang.org/x/oauth2` 的 GitHub OAuth2 登录处理流程。
- 实现与 GitHub API 交互以获取 User Profile，并在本地 PostgreSQL 的 `users` 表中同步/创建对应的用户记录的 Service 层逻辑。
- 添加 Gin 的 JWT 鉴权中间件（`internal/middleware/auth.go`）。
- 注册并实现 `InkWords_API.md` 中约定的 `/api/v1/auth/oauth/:provider` 和 `/api/v1/auth/callback/:provider` 路由接口。

## Impact
- Affected specs: 满足 `InkWords_API.md` 中定义的 Auth API 接口以及鉴权头 `Authorization: Bearer <JWT_TOKEN>` 规范。
- Affected code:
  - `backend/pkg/jwt` (新增)
  - `backend/internal/middleware` (新增)
  - `backend/internal/api` (新增 auth handler)
  - `backend/internal/service` (新增 auth logic)
  - `backend/cmd/server/main.go` (更新并注册路由)

## ADDED Requirements
### Requirement: GitHub OAuth2 登录与 JWT 签发
The system SHALL provide 完整的第三方登录流程，能够通过 GitHub OAuth 授权获取用户信息，在数据库中自动注册或更新对应的用户数据，随后向前端签发带有过期时间的 JWT Token。

#### Scenario: 成功授权与登录
- **WHEN** 用户访问 `/api/v1/auth/oauth/github` 并在浏览器完成授权回调到 `/api/v1/auth/callback/github?code=xxx`
- **THEN** 后端通过 `code` 换取 GitHub Access Token 和 User Profile，在 `users` 表中 UPSERT（不存在则创建，存在则更新）该用户，最终返回包含 `token` (JWT) 与 `user` 对象（如 `id`, `username`, `avatar_url`, `subscription_tier` 等）的标准 JSON 响应。

#### Scenario: 请求受保护的接口
- **WHEN** 客户端在 Request Headers 中携带 `Authorization: Bearer <有效_JWT_TOKEN>` 请求 `/api/v1/user/profile`
- **THEN** 系统解析并校验 Token 的合法性，从上下文中提取 `user_id` 并在数据库中查询当前用户信息及配额并返回。