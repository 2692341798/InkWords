# 后端 DDD 垂直切片渐进迁移（Blog Domain）设计文档

**类型**：Technical Design / Refactor Spec  
**范围**：后端目录升级（不改变对外 API 行为，不改 DB schema）  
**首批迁移领域**：Blog（包含列表/草稿/更新/导出）  

## 1. 背景与动机

当前后端仍以 `internal/api/*` + `internal/service/*` 的水平分层为主。随着功能增加（导出 PDF/ZIP、Obsidian 同步、SSE 相关能力等），API 层与 Service 层容易出现职责漂移，导致：
- 单领域逻辑分散在多个目录，阅读成本高
- 依赖注入边界不清晰，测试替身替换成本高
- 未来新增领域（例如掘金/CSDN 发布）容易把 `service/` 堆成“万金油目录”

目标是对齐工程规范中“垂直切片与高内聚”的约束，在不引入大规模行为改动的前提下，逐步迁移到按领域组织的结构。

## 2. 目标与非目标

### 2.1 目标
- 新增 `internal/domain/blog/` 垂直切片目录
- 明确三层边界：
  - `handler`：HTTP/Gin 适配层（参数绑定、鉴权上下文读取、错误转换、输出）
  - `service`：领域服务层（业务规则与编排）
  - `repository`：数据访问层（GORM 实现，参数化查询，错误 wrap）
- `internal/api` 保留为薄路由层/适配层：尽量只负责路由注册与调用 domain handler
- 保持现有 API 路由、响应结构、状态码与行为不变

### 2.2 非目标
- 不做一次性全量迁移（auth/user/stream/project 等后续再做）
- 不修改数据库表结构与迁移脚本
- 不在本次顺手做大规模重构（例如统一错误码体系、统一 response 封装），除非与迁移直接冲突

## 3. 总体方案

采用“渐进式迁移”：
1. **新增 domain/blog**：先把 Blog 相关 handler/service/repository 建出来，并写单元测试覆盖关键路径。
2. **API 层薄适配**：`internal/api/blog_*.go` 改为调用 `domain/blog` 的 handler；尽量不再直接 new service。
3. **依赖组装收口**：在 `cmd/server` 的初始化阶段完成依赖组装（repo -> service -> handler -> route）。

## 4. 目录结构（目标态）

```
backend/internal/
  domain/
    blog/
      handler.go
      service.go
      repository.go
      dto.go
```

说明：
- `dto.go` 放 HTTP 与 service 之间共享的请求/响应结构（仅 blog 域）
- `repository.go` 内定义 `BlogRepository` 接口与 `GormBlogRepository` 默认实现

## 5. Blog Domain 边界定义

### 5.1 Handler（HTTP 适配）
职责：
- 从 Gin Context 读取 `user_id`、path/query/body
- 将错误转换为对外稳定返回（不泄漏内部细节）
- 调用 `BlogService` 完成业务动作

### 5.2 Service（业务编排）
职责：
- 组合 BlogRepository 与其它依赖（如导出器、Obsidian exporter 等）
- 保持函数签名可测试（依赖通过构造函数注入）

### 5.3 Repository（数据访问）
职责：
- 所有 DB 调用收敛在 repo，禁止在 service/handler 里直接 `db.DB...`
- 必须参数化查询（GORM）
- 对内保留根因（wrap），对外由 handler 转换

## 6. 迁移策略与回滚

### 6.1 迁移步骤（第一批 Blog）
- 新增 `internal/domain/blog` 并实现与现有 BlogAPI 对齐的能力
- 修改 `internal/api` 中 Blog 相关路由处理器，改为调用 domain handler
- 保留旧 service 实现作为过渡期依赖，待 domain service 完全覆盖后再删除旧路径（后续批次）

### 6.2 回滚策略
- 迁移过程中每次只改一个路由组（例如先 `GET /api/v1/blogs`），确保可以按文件级回滚
- 若出现回归，直接恢复 `internal/api` 旧实现即可（domain 目录可保留不启用）

## 7. 验证与验收标准

- `cd backend && go test ./...` 必须通过
- 对外接口不变：API 路由、响应 JSON 结构、状态码不变（以现有测试与手工冒烟为准）
- 新增的 domain 代码具备单测（至少覆盖：鉴权缺失、参数错误、repo 返回错误、正常路径）

## 8. 后续扩展

完成 Blog 迁移后，按同样模式推进：
- `domain/user`（profile/stats）
- `domain/auth`（登录注册）
- `domain/stream`（SSE 生成/分析）
- `domain/project`（扫描与解析）

