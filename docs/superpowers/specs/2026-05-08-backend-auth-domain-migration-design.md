# 后端 DDD 垂直切片渐进迁移（Auth Domain）设计文档

**类型**：Technical Design / Refactor Spec  
**范围**：后端目录升级（不改变对外 API 行为，不改 DB schema）  
**迁移领域**：Auth（register/login/oauth/captcha/bind-github）  

## 1. 背景与动机

在完成 `internal/domain/blog` 与 `internal/domain/user` 之后，认证域仍处在 `internal/api/auth.go` + `internal/service/auth.go` 的水平分层结构中，且 API 层直接依赖 `service.ErrEmailExistsBindRequired` 等实现细节。为统一后端结构、收敛依赖组装、并建立更清晰的测试边界，需要将 Auth 领域迁移为垂直切片。

## 2. 目标与非目标

### 2.1 目标
- 新增 `backend/internal/domain/auth` 领域切片目录（repo/service/handler/dto）
- `internal/api/auth.go` 薄化：保留路由绑定与方法名不变，内部转发到 domain handler
- `cmd/server/main.go` 统一完成依赖组装（repo -> service -> handler -> api）
- 保持对外行为不变：
  - 路由与请求/响应结构保持一致
  - OAuth 重定向参数、回调跳转逻辑、验证码生成/校验行为保持一致

### 2.2 非目标
- 不修改鉴权中间件协议（token 解析与 `user_id` 注入逻辑不动）
- 不在本次引入新的安全策略（例如随机 state、PKCE、刷新 token 等），仅做结构迁移
- 不改变错误码体系（继续保持现有 message 文案与状态码语义）

## 3. 总体方案（全迁移）

1. **domain/auth 落地 repo/service/handler/dto**
   - repo：围绕 `model.User` 做查询与写入（email/github_id、更新锁定字段、绑定 github_id 等）
   - service：迁移现有 `internal/service/auth.go` 的业务逻辑（OAuth、验证码、注册、登录、绑定）
   - handler：迁移 `internal/api/auth.go` 的 HTTP 适配逻辑（参数绑定、跳转与 JSON 输出）
2. **api 薄适配**
   - `internal/api/auth.go` 保留对外方法名，但内部全部调用 `authDomainHandler.*`
3. **DI 收口**
   - `cmd/server/main.go` 统一组装 `authRepo -> authService -> authHandler -> authAPI`
4. **回归验证**
   - 每个阶段执行 `cd backend && go test ./...`（至少保证编译与已有测试不回归）

## 4. 目录结构（目标态）

```
backend/internal/domain/
  auth/
    dto.go
    repository.go
    service.go
    handler.go
    handler_test.go
```

## 5. 边界定义

- Repository：
  - `GetByEmail` / `GetByGithubIDOrEmail` / `CreateUser`
  - `UpdateLoginFailureState`（failed attempts / locked_until）
  - `UpdateGithubBinding`（github_id/username/avatar_url）
- Service：
  - `GetAuthURL` / `HandleCallback`
  - `GenerateCaptcha` / `VerifyCaptcha`
  - `Register` / `Login` / `BindGithub`
- Handler：
  - `OAuthRedirect` / `OAuthCallback`
  - `Register` / `Login` / `BindGithub` / `GetCaptcha`

## 6. 验证与验收

- `cd backend && go test ./...` 通过
- 行为不变（至少满足）：
  - OAuthCallback：`bind_required` 与 `token` 的重定向行为不变
  - Login/Register：错误码与 message 语义不变
  - Captcha：返回 `{ captcha_id, image }` 不变

