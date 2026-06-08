# InkWords 后端真实服务拆分设计

**类型**：Technical Design / Backend Architecture  
**范围**：后端真实服务拆分（第一阶段聚焦目录与代码归属，不改变对外行为）  
**日期**：2026-06-03  
**状态**：待评审

## 1. 背景

InkWords 当前生产形态已经是 Docker Compose 多服务：

- `core-api`
- `llm-stream`
- `parser-service`
- `export-service`
- `review-service`

并由前端 Nginx 作为单一公开入口，对外保持 `http://localhost` 与 `/api/*` 路径不变。

但代码组织层面仍然保留了明显的“模块化单体 + 多入口部署”特征：

- 多个 `cmd/*` 服务入口共享 `internal/domain/*`
- 多个服务共享 `internal/service/*`
- 路由注册集中在共享 `internal/transport/http/v1`
- 基础设施组件集中在共享 `internal/infra/*`

这会导致目录结构无法直接表达服务边界，进一步放大以下问题：

- 新代码的归属不清，容易继续写回共享目录
- 服务自治很弱，后续独立演进和回滚成本高
- `core-api` 与 `llm-stream` 等服务之间仍通过共享业务代码深度耦合

## 2. 目标与非目标

### 2.1 目标

- 让 `backend/` 的目录结构直接表达“真实服务拆分”
- 服务尽量拥有自己的 `app / domain / infra / transport`
- 共享层收敛到极薄，只保留真正稳定的基础能力与协议契约
- 保持现有对外 API URL、数据库 schema、Docker 单入口访问方式不变
- 采用可回滚的阶段式迁移，而不是一次性全仓大搬家

### 2.2 非目标

- 本阶段不改变前端交互和前端目录结构
- 本阶段不改变 Nginx 对外公开入口与代理规则
- 本阶段不强制拆库或引入多 `go.mod`
- 本阶段不顺手优化业务逻辑、响应格式或鉴权策略

## 3. 设计原则

- **服务拥有业务**：带业务语义的代码必须落到对应服务，不再放到共享目录
- **共享层极薄**：共享层只允许放基础能力与稳定契约
- **先立边界，再做去重**：早期允许少量复制，优先换取清晰归属
- **最小行为改动**：目录先变，行为后变
- **逐批迁移**：每次只迁一个服务主题，确保编译、测试和容器验证可执行

## 4. 目标目录结构

```text
backend/
  services/
    core-api/
      cmd/
        main.go
      app/
        bootstrap/
        config/
      domain/
        auth/
        user/
        blog/
        task/
      infra/
        db/
        mq/
        cache/
        obsidian/
      transport/
        http/
          middleware/
          v1/
    llm-stream/
      cmd/
        main.go
      app/
        bootstrap/
        config/
      domain/
        stream/
        generation/
      infra/
        db/
        mq/
        llm/
      transport/
        http/
          middleware/
          v1/
    parser-service/
      cmd/
        main.go
      app/
      domain/
        parse/
      infra/
        db/
        mq/
        parser/
        git/
      transport/
        http/
          middleware/
          v1/
    export-service/
      cmd/
        main.go
      app/
      domain/
        export/
      infra/
        db/
        mq/
        renderer/
        obsidian/
        artifact/
      transport/
        http/
          middleware/
          v1/
    review-service/
      cmd/
        main.go
      app/
      domain/
        review/
      infra/
        db/
        wiki/
      transport/
        http/
          middleware/
          v1/
  shared/
    kernel/
      auth/
      errors/
      httpx/
      response/
      tracing/
    platform/
      postgres/
      rabbitmq/
      redis/
  migrations/
  scripts/
```

## 5. 边界定义

### 5.1 `services/<service>`

每个服务目录是该服务的事实拥有者，内部按职责再分层：

- `cmd/`：服务主入口
- `app/`：依赖装配、配置读取、服务启动与生命周期控制
- `domain/`：该服务的业务领域代码
- `infra/`：该服务访问外部系统的实现
- `transport/`：该服务对外暴露的 HTTP/SSE 路由和中间件

### 5.2 `shared/`

`shared/` 只允许两类内容：

1. 完全不带业务语义的基础能力  
2. 多服务必须一致的稳定契约

允许共享的示例：

- JWT 校验基础件
- 通用错误模型
- 通用响应包
- 请求 ID、结构化日志等中间件底座
- PostgreSQL / RabbitMQ / Redis 连接工厂
- 任务消息 schema

不允许共享的示例：

- `blog service`
- `review service`
- `stream service`
- 解析编排
- 导出编排
- 带业务名词的 repository / handler / use case

为什么要这样做：
如果共享包开始理解 `blog`、`review`、`stream`、`parse` 之类的业务语义，它就已经不再是共享基础件，而应该回到对应服务。

## 6. 当前目录到目标目录的映射

### 6.1 服务入口

- `backend/cmd/core-api` -> `backend/services/core-api/cmd/main.go`
- `backend/cmd/llm-stream` -> `backend/services/llm-stream/cmd/main.go`
- `backend/cmd/parser-service` -> `backend/services/parser-service/cmd/main.go`
- `backend/cmd/export-service` -> `backend/services/export-service/cmd/main.go`
- `backend/cmd/review-service` -> `backend/services/review-service/cmd/main.go`

### 6.2 领域代码

- `backend/internal/domain/auth` -> `backend/services/core-api/domain/auth`
- `backend/internal/domain/user` -> `backend/services/core-api/domain/user`
- `backend/internal/domain/blog` -> `backend/services/core-api/domain/blog`
- `backend/internal/domain/task` -> `backend/services/core-api/domain/task`
- `backend/internal/domain/stream` -> `backend/services/llm-stream/domain/stream`
- `backend/internal/domain/fileparse` -> `backend/services/parser-service/domain/parse`
- `backend/internal/domain/review` -> `backend/services/review-service/domain/review`

### 6.3 共享业务编排

下列目录不应继续以“共享业务核心”的方式长期存在，而应拆分归位：

- `backend/internal/service/generator*`
- `backend/internal/service/decomposition*`
- `backend/internal/service/blog.go`
- `backend/internal/service/pdf_export.go`
- `backend/internal/service/obsidian*`

建议归位方向：

- 生成与分析相关 -> `llm-stream/domain/generation`
- 博客与任务写入相关 -> `core-api/domain/blog` 与 `core-api/domain/task`
- PDF/Obsidian 导出相关 -> `export-service/domain/export` 与 `export-service/infra/*`

### 6.4 Transport 与 Infra

- `backend/internal/transport/http/v1` -> 各服务 `transport/http/v1`
- `backend/internal/transport/http/middleware` -> 先复制到各服务，稳定后再下沉到 `shared/kernel/httpx`
- `backend/internal/infra/db` -> 各服务 `infra/db`，随后将通用连接工厂收敛到 `shared/platform/postgres`
- `backend/internal/infra/mq` -> 各服务 `infra/mq`，随后将通用连接工厂收敛到 `shared/platform/rabbitmq`
- `backend/internal/infra/cache` -> 有需要的服务本地 `infra/cache`
- `backend/internal/infra/llm` -> `llm-stream/infra/llm`
- `backend/internal/infra/parser` -> `parser-service/infra/parser`

## 7. 迁移分期

### Phase 1：建立新骨架

- 创建 `backend/services/*` 目录
- 为每个服务建立 `cmd / app / domain / infra / transport` 空骨架
- 先把现有 `cmd/*` 入口复制到新位置
- 暂时允许新入口引用旧包，优先把目录骨架立起来

### Phase 2：迁移服务装配

- 把当前 `main.go` 中的大量初始化逻辑收敛到各自 `app/bootstrap`
- 每个服务只保留一个装配根，避免 `cmd/main.go` 再次膨胀

### Phase 3：迁移边界清晰的服务

优先迁移：

- `review-service`
- `parser-service`
- `export-service`

原因：

- 三者业务边界清晰
- 与 `core-api / llm-stream` 的深度共享较少
- 可以先验证“真实服务拆分”模式是否跑通

### Phase 4：拆解共享业务核心

重点处理：

- `generator`
- `decomposition`
- 任务编排与博客写入边界

目标：

- 让 `core-api` 与 `llm-stream` 不再共享主要业务核心
- 只通过协议契约和最小基础层协作

### Phase 5：收敛 shared

- 识别迁移后仍残留在旧目录或 shared 的业务代码
- 继续回收到具体服务
- 让 `shared` 只剩真正稳定的基础件

### Phase 6：删除旧结构

- 删除旧 `backend/internal/domain`
- 删除旧 `backend/internal/service`
- 删除旧共享 `transport`
- 评估 `cmd/server` 是否保留为本地聚合调试入口，或退役

## 8. 第一批实际落地范围

第一批建议只迁移以下三个服务：

- `review-service`
- `parser-service`
- `export-service`

暂不在第一批彻底迁移：

- `core-api`
- `llm-stream`

原因：

- 这两个服务目前共享 `generator / decomposition / task / blog` 编排最深
- 如果在第一批一起大拆，风险显著上升
- 先迁三类边界清晰服务，可以更快跑通目录模型、验证 Compose、沉淀 shared 准入规则

## 9. 验证方式

### 9.1 编译与测试

- `cd backend && go test ./...`

### 9.2 容器验证

- `docker compose --env-file backend/.env down`
- `docker compose --env-file backend/.env up -d --build`

### 9.3 冒烟检查

- `http://localhost`
- `GET /api/v1/ping`
- review 基础接口可访问
- parser 任务创建与基础通路可验证
- export 任务创建与基础通路可验证

## 10. 风险与回滚

### 风险

- 一次性移动过多目录导致 import 改动过大
- `core-api` 与 `llm-stream` 共享编排拆分时出现职责漂移
- 过早抽 shared 反而重新制造“万金油目录”

### 回滚策略

- 每次只迁一个服务主题
- 旧目录与新目录短期并存
- 新目录通过测试和 Compose 冒烟后，再删除旧目录
- 如果某一批次失败，只回滚该服务相关改动

## 11. 设计结论

InkWords 后端应从“共享 `internal` 的多入口部署”升级为“`services/*` 主导的真实服务拆分目录结构”。

本设计选择：

- 服务自治优先
- 极薄共享层
- 后端优先
- 第一批先迁 `review-service / parser-service / export-service`

待第一批落地验证通过后，再进入 `core-api / llm-stream` 的深层拆分阶段。
