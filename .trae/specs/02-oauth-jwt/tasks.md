# Tasks

- [x] Task 1: 安装必要依赖
  - [x] SubTask 1.1: 在 `backend` 目录下安装 `github.com/golang-jwt/jwt/v5` 和 `golang.org/x/oauth2`。
- [x] Task 2: 编写 JWT 工具与中间件
  - [x] SubTask 2.1: 创建 `backend/pkg/jwt/jwt.go`，实现 `GenerateToken(userID string)` 和 `ParseToken(tokenStr string)` 函数。
  - [x] SubTask 2.2: 创建 `backend/internal/middleware/auth.go`，实现 Gin 拦截器，从请求头中解析 `Authorization: Bearer <token>` 并将 `userID` 注入上下文。
- [x] Task 3: 实现 OAuth2 服务层
  - [x] SubTask 3.1: 创建 `backend/internal/service/auth.go`，配置 GitHub OAuth2 Config（从环境变量读取 `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` / `GITHUB_REDIRECT_URL`）。
  - [x] SubTask 3.2: 在 Service 中实现 `GetAuthURL(provider)` 返回第三方授权页面 URL。
  - [x] SubTask 3.3: 在 Service 中实现 `Callback(provider, code)`。负责向 GitHub 换取 Token -> 获取 User Profile -> 查询数据库中是否存在 `github_id` -> UPSERT 用户 -> 调用 JWT 工具生成 Token，并按接口规范组装 Response Data 格式。
- [x] Task 4: 实现 Auth Handler 与路由
  - [x] SubTask 4.1: 创建 `backend/internal/api/auth.go`，对接 Service，处理 `/api/v1/auth/oauth/:provider` 和 `/api/v1/auth/callback/:provider` 接口，返回标准的 `{code, message, data}` 响应。
  - [x] SubTask 4.2: 创建 `backend/internal/api/user.go`，实现获取个人配置的受保护接口 `/api/v1/user/profile`。
  - [x] SubTask 4.3: 修改 `backend/cmd/server/main.go` 注册上述路由组（公开的 `/auth` 与受中间件保护的 `/user`）。

# Task Dependencies
- [Task 2] 和 [Task 3] 并行，但均依赖 [Task 1]。
- [Task 4] 依赖 [Task 2] 和 [Task 3]。
