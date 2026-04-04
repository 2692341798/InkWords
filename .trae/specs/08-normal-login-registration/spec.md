# Normal Login and Registration Spec

## Why
目前系统仅支持 GitHub OAuth 登录，但由于 GitHub OAuth 应用配置不完整（如 Client ID 无效），导致用户点击登录后会跳转到 GitHub 的 404 错误页面，完全阻塞了用户体验和测试流程。为了保证系统的核心流程畅通，需要提供原生的“邮箱/密码”登录与注册功能，并使用高质量的前端设计提升页面美感。

## What Changes
- **新增**：数据库 User 模型新增 `PasswordHash` 字段，用于存储加密后的密码。
- **新增**：后端提供 `/api/v1/auth/register` 接口处理用户注册，并对密码进行 bcrypt 加密。
- **新增**：后端提供 `/api/v1/auth/login` 接口处理账号密码登录，验证成功后返回 JWT Token。
- **修改**：使用 `frontend-design` 技能彻底重构 `frontend/src/components/Login.tsx`，设计一个极简、精致且具有高度审美体验的登录/注册切换表单。

## Impact
- Affected specs: 登录认证流程
- Affected code:
  - `backend/internal/model/user.go`
  - `backend/internal/service/auth.go`
  - `backend/internal/api/auth.go`
  - `backend/cmd/server/main.go` (注册新路由)
  - `frontend/src/components/Login.tsx`

## ADDED Requirements
### Requirement: 账号密码注册
用户能够通过邮箱、用户名和密码注册新账号。
#### Scenario: 成功注册
- **WHEN** 用户在注册表单填写有效的邮箱、用户名和密码并提交
- **THEN** 后端加密密码并创建用户，前端收到成功响应后可直接登录或跳转。

### Requirement: 账号密码登录
用户能够通过已注册的邮箱和密码登录。
#### Scenario: 成功登录
- **WHEN** 用户输入正确的邮箱和密码并提交
- **THEN** 后端校验通过并下发 JWT Token，前端保存 Token 并进入工作台。

## MODIFIED Requirements
### Requirement: 登录页面 UI
登录页面需提供极高品质的设计体验（符合 `frontend-design` 的高级审美要求）。
#### Scenario: 查看登录页面
- **WHEN** 未登录用户访问系统
- **THEN** 显示精心设计的登录/注册界面，包含平滑的模式切换动画、考究的排版和细节装饰。