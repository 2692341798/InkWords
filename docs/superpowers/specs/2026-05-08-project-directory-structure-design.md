# InkWords 目录结构工程化调整（目标态设计）

**类型**：Technical Design / Engineering Spec  
**范围**：目录结构目标态与迁移路线图（仅规划，不移动代码、不改行为）  
**原则**：最小改动、边界清晰、可渐进迁移、可回滚  

## 1. 背景与动机

InkWords 已采用 Monorepo 结构（`frontend/` + `backend/`），但后端同时存在：
- 历史的水平分层：`internal/api/*`、`internal/service/*`
- 渐进式垂直切片：`internal/domain/*`

当两个体系并存但边界不清时，容易出现：
- 同一业务逻辑分散在多个目录，理解与定位成本高
- `internal/service` 演变为“万金油目录”，领域边界被稀释
- API/HTTP 适配层承担业务编排，难以测试与复用

本设计目标是明确“领域层 / 传输层 / 基础设施层”的职责边界，并提供一个可按批次推进的迁移路线图，避免一次性大重构。

## 2. 目标与非目标

### 2.1 目标

- 统一后端目录语义：以 `domain/transport/infra` 三分法表达职责边界
- 保持现有对外 API 行为不变（路由、响应结构、状态码、鉴权策略）
- 保持现有 DB schema 不变
- 给出“当前目录 → 目标目录”的映射与迁移步骤（按阶段推进）

### 2.2 非目标

- 不在本阶段移动/重命名任何目录或文件
- 不引入新的 DI 框架（保持当前 `cmd/server` 组装方式）
- 不顺手做无关的重构（统一错误码、response 封装、日志体系等）

## 3. 现状快照（以仓库当前形态为准）

### 3.1 后端（关键目录）

- `backend/cmd/server/`：应用入口
- `backend/internal/domain/*`：已存在领域切片（auth/blog/project/stream/user）
- `backend/internal/api/*`：路由与部分 HTTP 处理
- `backend/internal/service/*`：偏“应用服务/业务编排”集合
- `backend/internal/db`、`backend/internal/cache`、`backend/internal/llm`、`backend/internal/parser`：基础设施与外部适配

### 3.2 前端（关键目录）

前端已具备较好的拆分：`components/`、`hooks/`、`store/`、`lib/`。仍可进一步明确“页面级组件”与“请求层（services）”的边界。

## 4. 目标目录结构（Target State）

### 4.1 后端（backend/）

```
backend/
  cmd/
    server/
      main.go
  internal/
    domain/
      auth/
      blog/
      project/
      stream/
      user/
    transport/
      http/
        middleware/
        v1/
    infra/
      cache/
      db/
      export/
      llm/
      parser/
  pkg/
```

#### 4.1.1 边界定义

- `internal/domain/<domain>/`：领域垂直切片（handler/service/repository/dto 等按领域自组织）
  - 领域层只关心业务规则与用例编排，不直接承担“HTTP/Gin 路由注册”
- `internal/transport/http/`：HTTP 传输适配
  - 职责：路由注册、版本化路由组、Gin 中间件挂载、参数绑定/校验、统一错误转换、SSE 写出
  - 不承载领域业务编排（业务逻辑应在 domain 中）
- `internal/infra/`：基础设施与外部系统适配（能力组件）
  - 职责：DB/Redis 初始化、LLM 客户端、文档解析/Git 获取器、导出器等
  - 原则：尽量“可被 domain 复用”，不产生领域耦合
- `pkg/`：跨模块可复用公共包（谨慎放置，避免变成杂物间）

### 4.2 前端（frontend/）

```
frontend/
  src/
    pages/
    components/
    hooks/
    services/
    store/
    lib/
```

#### 4.2.1 边界定义

- `pages/`：路由级页面组件（例如 Dashboard/Editor/Generator/Login）
- `components/`：可复用组件（包含业务组件与 ui 基础组件）
- `services/`：所有 API/SSE 调用入口（hooks 只做编排，不直接散落请求细节）
- `hooks/`：副作用、状态编排、与 UI 的粘合层
- `store/`：Zustand 全局状态
- `lib/`：纯函数、可测试工具、与框架弱耦合逻辑

## 5. 当前目录到目标目录的映射（Mapping）

### 5.1 后端映射

- `internal/api/*` → `internal/transport/http/v1/*`（最终态）
  - 迁移策略：先“语义约束变薄”，再按批次移动与重命名
- `internal/middleware/*` → `internal/transport/http/middleware/*`
- `internal/db` → `internal/infra/db`
- `internal/cache` → `internal/infra/cache`
- `internal/llm` → `internal/infra/llm`
- `internal/parser` → `internal/infra/parser`
- `internal/service/*` → 优先归位到 `internal/domain/<domain>/*`
  - 若确实跨多个 domain 的“用例编排”需要保留统一入口，再评估是否引入 `internal/app/`，但必须具备清晰边界与严格准入（避免新万金油）

### 5.2 前端映射

- `src/components/*.tsx` 中的“页面级组件” → `src/pages/*`
- 现有 `src/hooks/*` 保持，但逐步把请求细节收敛到 `src/services/*`

## 6. 迁移路线图（按批次、可回滚）

### Phase 0：冻结目标态与准入规则（本阶段）

- 产出本设计文档并在架构文档中引用
- 明确准入：新增代码优先落在 `domain/`、`transport/http/`、`infra/` 的目标边界内

### Phase 1：先“变薄”，再“搬家”（后端）

- 约束 `internal/api`：只做路由与适配，不再 new service，不再承载业务编排
- `cmd/server` 作为组装根：repo → service → handler → transport/http 注册

### Phase 2：收敛 service（后端）

- 将 `internal/service/*` 按领域归位到 `internal/domain/<domain>/`（一次一个领域）
- 每个领域迁移要求：`go test ./...` 通过，且对外 API 行为一致

### Phase 3：正式引入 transport/infra 目录（后端）

- 新增 `internal/transport/http`、`internal/infra` 目录
- 按映射逐步移动与改 import（一次一个子目录），保持可回滚

### Phase 4：前端页面与请求层归位

- 新增 `src/pages`、`src/services`
- 将页面级组件从 `components/` 根部迁移至 `pages/`
- 将 SSE/REST 请求入口从 hooks/components 收敛至 services

## 7. 验证与验收标准（结构调整的 DoD）

### 7.1 结构验收（无需改业务）

- 目录边界清晰：新增文件能判断应落点（domain/transport/infra/pages/services）
- 文档同步：架构文档中记录目标态与迁移策略

### 7.2 迁移验收（进入实际迁移阶段后使用）

- 后端：`cd backend && go test ./...` 全通过
- 前端：`cd frontend && npm test`（如已有）/ `npm run build` 通过
- Docker：`docker compose down && docker compose up -d --build` 冒烟通过（通过 `http://localhost` 访问）

## 8. 风险与回滚策略

- 风险：大规模移动导致 import 改动面过大、merge 冲突增加
  - 策略：按领域/按目录小步迁移；每次只动一个路由组或一个目录映射项
- 回滚：保持“文件级回滚”可行
  - 任何阶段出现回归，恢复到上一个批次即可；目标目录允许存在但暂不启用

