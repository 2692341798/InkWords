# 后端 DDD 垂直切片渐进迁移（User Domain）设计文档

**类型**：Technical Design / Refactor Spec  
**范围**：后端目录升级（不改变对外 API 行为，不改 DB schema）  
**迁移领域**：User（profile/update/avatar/stats）  

## 1. 背景与动机

在完成 `internal/domain/blog` 领域切片后，下一步将用户域从 `internal/api/user.go` + `internal/service/user.go` 的水平分层迁移为垂直切片，以统一后端的工程结构与测试边界。

## 2. 目标与非目标

### 2.1 目标
- 新增 `backend/internal/domain/user` 垂直切片目录（repo/service/handler/dto）
- `internal/api/user.go` 薄化：保留路由绑定与方法名不变，内部转发到 domain handler
- 保持对外接口不变：
  - 路由：`/api/v1/user/profile`、`/api/v1/user/avatar`、`/api/v1/user/stats`
  - JSON 返回结构：`{ code, message, data }` 与原语义一致
- `cmd/server/main.go` 统一完成依赖组装（repo -> service -> handler -> api）

### 2.2 非目标
- 不改数据库表结构
- 不改鉴权中间件协议（仍从 Gin Context 读取 `user_id`）
- 不统一重写所有错误码/错误消息，仅保证行为不回归

## 3. 总体方案

延续 Blog Domain 的渐进式迁移：
1. 新增 `domain/user` 骨架与最小单测
2. 将 `internal/api/user.go` 的 4 个 handler 逐个替换为 domain handler
3. 依赖注入收口到 `cmd/server/main.go`

## 4. 目录结构（目标态）

```
backend/internal/domain/
  user/
    dto.go
    repository.go
    service.go
    handler.go
    handler_test.go
```

## 5. 边界定义

- Repository：只负责 `model.User` 与统计所需的 `model.Blog` 查询（GORM）
- Service：封装业务规则（用户名长度校验、额度默认值、统计聚合等）
- Handler：HTTP 参数绑定、鉴权上下文读取、响应包装与错误映射（保持原结构）

## 6. 验证与验收

- `cd backend && go test ./...` 通过
- 新增 handler 单测至少覆盖：
  - 未授权（缺少 user_id）
  - profile 正常返回
  - update profile 参数错误（空/长度不合法）

