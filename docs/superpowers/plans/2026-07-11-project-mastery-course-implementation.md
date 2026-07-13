# GitHub 项目精通课程生成模块实施计划

> 本计划对应设计文档：`docs/superpowers/specs/2026-07-11-project-mastery-course-design.md`。

**目标**：新增独立的 `project_mastery_course` 场景，把锁定 commit SHA 的 GitHub 仓库生成分卷、证据可追溯、包含累积式代码实验的项目精通课程。

**架构**：`core-api` 持有课程实体、蓝图审批和最终业务事实；`llm-stream` 执行仓库理解、蓝图规划、章节生成和质量门禁；`export-service` 打包课程代码；隔离验证器只运行系统生成的实验，不执行目标仓库。现有 generation task、RabbitMQ、SSE、task-only 持久化和 blogs 树继续复用。

**技术栈**：Go + Gin + GORM + PostgreSQL + RabbitMQ + Redis + React + Zustand + Vite + Vitest + Docker Compose + DeepSeek 客户端抽象。

**基准仓库**：`https://github.com/2692341798/InkWords`，验收开始时解析 `main` 并固化实际 SHA。

**明确不做**：首期不建设网页 IDE，不执行目标仓库，不替换现有 `beginner_walkthrough`，不允许用户编辑学习目标和源码证据，不自动提交、推送或创建 PR。

---

## 1. 交付阶段与发布门槛

| 阶段 | 可交付能力 | 发布门槛 |
| --- | --- | --- |
| Phase 0 | 基线、契约和评测夹具 | 现有生成回归测试不退化 |
| Phase 1 | 锁定 SHA、确定性仓库清单、知识图和课程蓝图 | InkWords 蓝图可生成，核心覆盖与引用可审计 |
| Phase 2 | 蓝图工作台、有限编辑和批准 | 版本冲突、依赖断裂、覆盖降级行为正确 |
| Phase 3 | 证据驱动的分类型章节生成 | 项目事实证据覆盖率与引用有效率 100% |
| Phase 4 | 累积实验、隔离验证和课程 ZIP | 验证实验通过率 100%，目标仓库未执行 |
| Phase 5 | dogfood、灰度、观测与回滚 | InkWords 自动验收和人工学习路径通过 |

每个阶段独立受功能开关保护。Phase 1/2 可以先上线蓝图预览，不必等待实验验证器完成。

---

## 2. 目标文件结构

以下是计划目标，不要求一次性创建全部文件。

```text
backend/
  shared/kernel/projectcourse/
    enums.go
    snapshot.go
    evidence.go
    blueprint.go
    coverage.go
    result.go
  services/core-api/domain/projectcourse/
    model.go
    dto.go
    repository.go
    service.go
    handler.go
    service_test.go
    repository_test.go
  services/core-api/transport/http/v1/routes.go
  services/llm-stream/app/projectcourse/
    service.go
    snapshot_service.go
    inventory.go
    semantic_analyzer.go
    analyzer_go.go
    analyzer_typescript.go
    analyzer_config.go
    knowledge_graph.go
    blueprint_planner.go
    coverage_validator.go
    official_sources.go
    evidence_pack.go
    claim_ledger.go
    chapter_contracts.go
    chapter_pipeline.go
    quality_gates.go
    lab_builder.go
  services/llm-stream/domain/stream/
    dto.go
    task_consumer.go
  services/export-service/domain/export/
    course_package.go
    course_package_test.go
  services/course-runner/
    cmd/main.go
    domain/verification/model.go
    domain/verification/consumer.go
    domain/verification/runner.go
    app/bootstrap/bootstrap.go
    Dockerfile
frontend/src/
  lib/projectCourse.ts
  services/projectCourse.ts
  store/projectCourseStore.ts
  hooks/projectCourse/
    useProjectCourseAnalyze.ts
    useProjectCourseBlueprint.ts
    useProjectCourseGenerate.ts
  components/project-course/
    ProjectCourseEntry.tsx
    AudienceLevelSelector.tsx
    BlueprintWorkspace.tsx
    VolumeTree.tsx
    ChapterRow.tsx
    CoveragePanel.tsx
    ApprovalBar.tsx
```

若实现过程中发现单个文件接近 500 行，应按解析器、合同、校验器或 handler/service 边界继续拆分，不把整个课程流水线堆入现有 `decomposition_quality.go`。

---

## 3. Task 0：建立基线、失败样例与固定评测夹具

**Files:**

- Create: `docs/qa/project-course-baseline.md`
- Create: `backend/services/llm-stream/app/projectcourse/testdata/inkwords-fixture/manifest.json`
- Create: `backend/services/llm-stream/app/projectcourse/testdata/known-failures.json`
- Modify: `docs/qa/e2e-testing.md`

- [x] **Step 1：记录现有生成链路基线**

记录当前 Git 分析的：

- 输入文件总数、实际保留数和排除原因；
- 大纲章节结构；
- 模型调用次数、token 和耗时；
- 现有文章中的失效路径、失效符号和不可运行示例；
- 当前 `origin/main` SHA 和评测日期。

- [x] **Step 2：建立最小离线仓库夹具**

夹具只保留 InkWords 的代表性结构和经过脱敏的短代码片段，覆盖：

- Go 服务入口、route、handler、service；
- RabbitMQ consumer；
- React hook 与 Zustand store；
- `docker-compose.yml`；
- README、runbook、测试和 script。

不要把完整仓库复制进 testdata。

- [x] **Step 3：固化已知失败断言**

至少包含：

- 旧路径 `backend/internal/service/...` 不得在新课程中作为当前源码路径出现；
- 引用的 symbol 必须存在；
- 当前 commit 下的内容 hash 必须匹配；
- README 声明、源码观察和模型推断必须使用不同置信度。

- [x] **Step 4：运行现有回归测试**

Run:

```bash
cd backend && go test ./services/llm-stream/... ./services/core-api/... ./services/export-service/...
cd frontend && npm test -- --run
```

Expected: PASS。把实际耗时和失败项写入 baseline；不要为了计划任务顺手修复无关失败。

---

## 4. Task 1：建立 Project Course 共享合同与新场景

**Files:**

- Create: `backend/shared/kernel/projectcourse/enums.go`
- Create: `backend/shared/kernel/projectcourse/snapshot.go`
- Create: `backend/shared/kernel/projectcourse/evidence.go`
- Create: `backend/shared/kernel/projectcourse/blueprint.go`
- Create: `backend/shared/kernel/projectcourse/coverage.go`
- Create: `backend/shared/kernel/projectcourse/result.go`
- Modify: `backend/shared/kernel/prompt/scenario_mode.go`
- Modify: `frontend/src/lib/scenarioMode.ts`
- Test: `backend/shared/kernel/projectcourse/*_test.go`
- Test: `frontend/src/lib/scenarioMode.test.ts`

- [x] **Step 1：先写枚举与 JSON 合同失败测试**

锁定：

- `project_mastery_course` 是独立场景；
- `foundation / programming / stack_familiar` 三种读者等级；
- 章节类型、证据置信度、课程状态、门禁结果的 JSON 值稳定；
- 未知枚举必须显式报错，不能静默回退到电子书场景。

- [x] **Step 2：定义不可变 SourceSnapshot**

要求 `resolved_commit_sha` 为必填 40 位十六进制 SHA；任何下游合同必须带 `course_id`、`blueprint_version` 和 SHA。

- [x] **Step 3：定义 EvidenceRef 与 Claim**

EvidenceRef 至少包含 path、symbol、line range、content hash。Claim 必须包含类型、置信度、证据 ID 和验证状态。

- [x] **Step 4：定义 Blueprint 与 Coverage**

蓝图包含 volumes、chapters、dependencies、learning outcomes、evidence IDs、lab spec 和 coverage matrix。对外更新 DTO 只暴露 title、sort、enabled。

- [x] **Step 5：运行合同测试**

Run:

```bash
cd backend && go test ./shared/kernel/projectcourse -v
cd frontend && npm test -- --run src/lib/scenarioMode.test.ts
```

Expected: PASS。

---

## 5. Task 2：重构为确定性的仓库快照与文件清单

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/snapshot_service.go`
- Create: `backend/services/llm-stream/app/projectcourse/inventory.go`
- Create: `backend/services/llm-stream/app/projectcourse/inventory_test.go`
- Modify: `backend/shared/platform/parser/git_fetcher.go`
- Modify: `backend/shared/platform/parser/git_fetcher_filter.go`
- Modify: `backend/shared/platform/parser/git_fetcher_git.go`
- Modify: `backend/shared/platform/parser/git_fetcher_github.go`
- Modify: `backend/shared/platform/parser/git_fetcher_types.go`

- [x] **Step 1：写失败测试复现非确定性截断**

对相同输入重复构建清单 100 次，要求路径、角色、顺序和纳入状态完全一致。

- [x] **Step 2：把“忽略”改为“分类与处置”**

为文档、示例、脚本、测试、generated、binary 建立 FileRole 与 disposition：

```text
covered | indexed | excluded
```

每个 excluded 项必须带机器可读 reason。

- [x] **Step 3：解析并锁定 commit SHA**

GitHub API 和 Git CLI 两条路径都返回 resolved SHA。禁止后续请求继续使用 `HEAD` 读取正文；统一使用固定 SHA。

- [x] **Step 4：稳定排序与预算**

按 role priority、normalized path、content hash 排序。超预算时保留完整元数据，只延迟正文读取；不得随机删除模块。

- [x] **Step 5：保护路径和大文件**

增加 path traversal、symlink、非 UTF-8、超大单文件、Git LFS pointer 和 submodule 的显式处置测试。

- [x] **Step 6：运行测试**

Run:

```bash
cd backend && go test ./shared/platform/parser ./services/llm-stream/app/projectcourse -run 'Snapshot|Inventory|Deterministic' -v
```

Expected: PASS；现有 Git 分析路径行为保持兼容。

---

## 6. Task 3：构建静态语义适配器和 Repository Knowledge Graph

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/semantic_analyzer.go`
- Create: `backend/services/llm-stream/app/projectcourse/analyzer_go.go`
- Create: `backend/services/llm-stream/app/projectcourse/analyzer_typescript.go`
- Create: `backend/services/llm-stream/app/projectcourse/analyzer_config.go`
- Create: `backend/services/llm-stream/app/projectcourse/knowledge_graph.go`
- Create: `backend/services/llm-stream/app/projectcourse/knowledge_graph_test.go`

- [x] **Step 1：定义适配器接口和确定性输出**

接口只接收文件内容和固定 SHA，不执行 build、test、install 或项目脚本。

- [x] **Step 2：实现 Go 适配器**

优先使用标准库 `go/parser`、`go/ast`、`go/token` 提取：

- package、imports；
- exported/unexported types、functions、methods、interfaces；
- 路由注册、构造函数和明显调用关系；
- 文件与 symbol 的精确行范围。

- [ ] **Step 3：选择 TypeScript/JavaScript 成熟解析方案**

在引入依赖前记录：许可证、维护状态、安全记录、二进制/CGO 要求、镜像体积和 React/TSX 支持。禁止用正则手写通用 TS/TSX parser。

验收最少提取 imports/exports、函数、组件、hooks 和调用引用。若选型未通过，先提供 `low_precision` 文件级适配器并阻止把关系推断写成确定事实。

- [x] **Step 4：实现配置适配器**

解析 `docker-compose.yml`、Dockerfile、Nginx、Go module 和 package manifest，提取服务拓扑、端口、依赖和技术版本证据。

- [x] **Step 5：构建知识图**

合并静态事实与受证据约束的 LLM 模块摘要。LLM 输出不得创建没有 EvidenceRef 的 symbol 或 relation。

- [x] **Step 6：建立主链路候选**

从入口、route、consumer、command、React event handler 等开始，生成到持久化、消息发布、外部 API 或 UI 状态的候选链路。

- [x] **Step 7：测试 InkWords 关键事实**

断言至少能识别 core-api route、llm-stream consumer、RabbitMQ 配置、React generator hook 和 Docker Compose 服务边界。

Run:

```bash
cd backend && go test ./services/llm-stream/app/projectcourse -run 'Analyzer|KnowledgeGraph|MainFlow' -v
```

Expected: PASS。

---

## 7. Task 4：新增 ProjectCourse 领域模型、持久化与迁移

**Files:**

- Create: `backend/services/core-api/domain/projectcourse/model.go`
- Create: `backend/services/core-api/domain/projectcourse/dto.go`
- Create: `backend/services/core-api/domain/projectcourse/repository.go`
- Create: `backend/services/core-api/domain/projectcourse/repository_test.go`
- Modify: `backend/internal/infra/db/db.go`
- Modify: `backend/db/init/00-create-review-db.sql` 或新增独立初始化 SQL（按现有初始化约定选择）
- Modify: `docs/runbooks/core-blog-task-boundary.md`

- [x] **Step 1：先写 repository 测试**

覆盖创建、按 owner 读取、蓝图版本 CAS 更新、批准后不可原地修改、跨用户不可读取。

- [x] **Step 2：新增 `project_courses` 模型**

实现设计文档中的字段，并增加：

- `(user_id, created_at desc)` 列表索引；
- `(repository_url, resolved_commit_sha)` 复用查询索引；
- `status` 检查约束；
- `blueprint_version > 0` 检查约束。

- [x] **Step 3：实现乐观锁更新**

更新条件必须包含：

```sql
WHERE id = $1 AND user_id = $2 AND blueprint_version = $3 AND status = 'draft'
```

受影响行数为 0 时返回稳定的 version conflict 或 invalid state 错误。

- [ ] **Step 4：验证关键 SQL**

在 PostgreSQL 测试环境运行：

```sql
EXPLAIN ANALYZE SELECT * FROM project_courses
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 20;
```

记录是否命中索引。说明索引带来的写放大和存储成本。

- [x] **Step 5：说明回滚策略**

功能开关关闭即可停止新写入；数据库回滚只删除尚未对外使用的 `project_courses` 表。已产生课程数据后不自动 drop，改用前向迁移。

- [x] **Step 6：运行测试**

Run:

```bash
cd backend && go test ./services/core-api/domain/projectcourse ./internal/infra/db -v
```

Expected: PASS。

---

## 8. Task 5：实现课程 API 与任务创建边界

**Files:**

- Create: `backend/services/core-api/domain/projectcourse/service.go`
- Create: `backend/services/core-api/domain/projectcourse/handler.go`
- Create: `backend/services/core-api/domain/projectcourse/service_test.go`
- Modify: `backend/services/core-api/transport/http/v1/routes.go`
- Modify: `backend/services/core-api/transport/http/v1/routes_test.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`
- Modify: `backend/services/core-api/domain/task/dto.go`
- Modify: `backend/services/core-api/domain/task/service.go`

- [x] **Step 1：写 API 契约测试**

覆盖创建、读取、更新蓝图、批准、生成、覆盖报告、质量报告和打包。

- [x] **Step 2：创建分析任务**

`POST /project-courses` 校验 URL/ref/audience，创建 course draft 和 `project_course_analyze` task，消息只携带 course ID、owner ID 和不可变请求字段。

- [x] **Step 3：限制蓝图更新字段**

服务端只应用 title、sort、enabled；若客户端提交 evidence、learning outcomes 或 coverage，返回 400，不做静默忽略。

- [x] **Step 4：批准前校验**

批准必须确认：无依赖断裂、蓝图版本匹配、覆盖报告已计算、至少一个启用章节。若禁用核心章节，允许批准但状态标记为 customized coverage。

- [x] **Step 5：创建生成任务**

生成任务固定 course ID、blueprint version、commit SHA。批准后的蓝图版本不可被后台读取为“最新版本”。

- [x] **Step 6：鉴权与错误码**

跨用户统一返回 404；版本冲突返回 409；状态冲突返回稳定业务错误码，不泄漏内部 SQL。

- [x] **Step 7：运行测试**

Run:

```bash
cd backend && go test ./services/core-api/domain/projectcourse ./services/core-api/transport/http/v1 -v
```

Expected: PASS。

---

## 9. Task 6：实现蓝图规划、分卷和覆盖校验

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/blueprint_planner.go`
- Create: `backend/services/llm-stream/app/projectcourse/coverage_validator.go`
- Create: `backend/services/llm-stream/app/projectcourse/blueprint_planner_test.go`
- Create: `backend/services/llm-stream/app/projectcourse/prompts/blueprint.go`
- Modify: `backend/services/llm-stream/domain/stream/task_consumer.go`
- Modify: `backend/services/llm-stream/domain/stream/dto.go`

- [x] **Step 1：先写 Blueprint 结构与门禁测试**

要求每个启用章节具备类型、学习成果、依赖、证据、技术主题和练习信息；项目地图章必须为第一卷前置。

- [x] **Step 2：实现依赖优先的分卷算法**

先根据知识图生成主题簇，再按主链路完整性、卷容量和累积实验检查点拆卷。不要直接按根目录一目录一章。

- [x] **Step 3：实现三层依赖分类**

- core technology：必须原理与实验覆盖；
- important library：必须用途、原因、替代方案和边界；
- ordinary dependency：只进索引。

- [x] **Step 4：计算全局 Coverage Matrix**

所有 module、main flow、technology、decision、file、exercise 都有 disposition 和 chapter IDs。

- [x] **Step 5：修复或阻断错误蓝图**

模型生成后先做确定性校验；缺证据、循环依赖、孤立核心模块时进行一次定向修复，仍失败则任务失败，不用空章节补位。

- [x] **Step 6：持久化 Analyze 任务结果**

结果包含 source snapshot、knowledge graph 摘要、blueprint、coverage 和成本估算。由 core-api 归并到 course 实体。

- [x] **Step 7：运行测试**

Run:

```bash
cd backend && go test ./services/llm-stream/app/projectcourse -run 'Blueprint|Volume|Coverage' -v
```

Expected: PASS。

---

## 10. Task 7：建设前端课程入口与蓝图工作台

**Files:**

- Create: `frontend/src/lib/projectCourse.ts`
- Create: `frontend/src/services/projectCourse.ts`
- Create: `frontend/src/store/projectCourseStore.ts`
- Create: `frontend/src/hooks/projectCourse/useProjectCourseAnalyze.ts`
- Create: `frontend/src/hooks/projectCourse/useProjectCourseBlueprint.ts`
- Create: `frontend/src/hooks/projectCourse/useProjectCourseGenerate.ts`
- Create: `frontend/src/components/project-course/ProjectCourseEntry.tsx`
- Create: `frontend/src/components/project-course/AudienceLevelSelector.tsx`
- Create: `frontend/src/components/project-course/BlueprintWorkspace.tsx`
- Create: `frontend/src/components/project-course/VolumeTree.tsx`
- Create: `frontend/src/components/project-course/ChapterRow.tsx`
- Create: `frontend/src/components/project-course/CoveragePanel.tsx`
- Create: `frontend/src/components/project-course/ApprovalBar.tsx`
- Modify: `frontend/src/pages/Generator.tsx` 或当前生成器入口文件

- [x] **Step 1：先写 store 与 request builder 测试**

覆盖 analyze/create、乐观版本、蓝图有限编辑、批准与 generation task 请求。

- [x] **Step 2：增加独立场景入口**

中文文案明确：完整仓库、质量优先、先生成蓝图、耗时和成本高于小白教程。

- [x] **Step 3：实现读者等级选择**

默认 `programming`，解释三个等级只改变脚手架深度，不改变事实和能力目标。

- [x] **Step 4：实现卷册树和有限编辑**

只允许：

- 编辑章节标题；
- 同卷或跨卷排序；
- 启用/禁用。

学习目标、EvidenceRef、练习和覆盖状态只读。

- [ ] **Step 5：展示依赖和覆盖影响**

禁用或移动章节后调用后端 dry-run 校验，显示断裂依赖、覆盖率下降和 complete/customized 状态。

- [x] **Step 6：处理版本冲突**

409 时刷新服务器蓝图并提示用户重新应用修改，不覆盖另一窗口或后台的新版本。

- [ ] **Step 7：浏览器验证**

验证桌面和移动端：长标题、数十章节、多卷折叠、键盘操作、拖拽替代按钮、焦点可见、错误与空状态。

- [x] **Step 8：运行测试**

Run:

```bash
cd frontend && npm test -- --run
cd frontend && npm run lint
cd frontend && npm run build
```

Expected: PASS。

---

## 11. Task 8：建立官方资料 Provider 与安全抓取

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/official_sources.go`
- Create: `backend/services/llm-stream/app/projectcourse/official_sources_test.go`
- Create: `backend/services/llm-stream/app/projectcourse/official_registry.go`
- Modify: `backend/shared/platform/cache/redis.go`
- Modify: `backend/.env.example`

- [x] **Step 1：定义 `OfficialSourceProvider`**

输入 technology、version evidence 和所需主题；输出 URL、版本范围、规范化正文、获取时间和 hash。

- [x] **Step 2：建立首期官方 registry**

覆盖 InkWords 核心技术：Go、Gin、React、Zustand、PostgreSQL、RabbitMQ、Redis、Docker Compose、Nginx。每个 provider 只允许官方域名。

- [x] **Step 3：实现 SSRF 防护**

测试拒绝：

- localhost、环回、RFC1918、链路本地和云元数据 IP；
- DNS rebinding 后解析到内网；
- 跨域重定向；
- 非 HTTP(S)、超大响应、非文本内容和超时响应。

- [x] **Step 4：版本匹配和缓存**

从 manifest 提取版本，缓存 key 包含 provider、URL、version、content hash。版本无法匹配时不声称精确版本支持。

- [x] **Step 5：Prompt 注入隔离**

官方网页正文只作为引用材料，放在明确分隔的数据块中；网页指令不得影响 system prompt。

- [x] **Step 6：运行测试**

Run:

```bash
cd backend && go test ./services/llm-stream/app/projectcourse -run 'OfficialSource|SSRF|VersionMatch' -v
```

Expected: PASS。

---

## 12. Task 9：实现证据包、Claim Ledger 与分类型章节生成

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/evidence_pack.go`
- Create: `backend/services/llm-stream/app/projectcourse/claim_ledger.go`
- Create: `backend/services/llm-stream/app/projectcourse/chapter_contracts.go`
- Create: `backend/services/llm-stream/app/projectcourse/chapter_pipeline.go`
- Create: `backend/services/llm-stream/app/projectcourse/quality_gates.go`
- Create: `backend/services/llm-stream/app/projectcourse/chapter_pipeline_test.go`
- Create: `backend/services/llm-stream/app/projectcourse/prompts/*.go`

- [x] **Step 1：为每种章节类型写合同测试**

不要复用一个万能 Prompt。每种类型有独立 required sections、Evidence 要求、练习要求和门禁。

- [x] **Step 2：构造最小 Evidence Pack**

根据 chapter evidence IDs 拉取具体源码片段和官方资料。超预算时优先保留 Claim 所需证据，不按字符串头部截断。

- [x] **Step 3：先生成 ClaimPlan**

正文前先列出拟表达的项目事实、原理事实、推断和证据映射。没有证据的项目事实在草稿前就被移除或改成待确认问题。

- [x] **Step 4：按读者等级生成教学结构**

等级只影响术语解释、步骤颗粒度、提示数量和延伸深度，不影响 Claim 和 Evidence。

- [x] **Step 5：草稿、事实审稿、教学审稿分离**

事实审稿优先使用确定性校验：路径、symbol、line range、hash、Claim 覆盖。模型审稿只处理无法用规则判断的语义一致性。

- [x] **Step 6：终稿重新注入 Evidence Pack**

禁止沿用现有“只给草稿和 review actions”的终稿模式。终稿后再次解析 Claim Ledger 并运行硬门禁。

- [x] **Step 7：代码块来源一致性**

项目源码代码块从 EvidenceRef 渲染；教学代码块从课程代码工件渲染。模型不得独立生成一份与 artifact 不同的同名代码。

- [x] **Step 8：软门禁与风险报告**

类比、节奏、重复和等级适配触发一次修复；仍失败则保留风险，不阻止事实正确的章节完成。

- [x] **Step 9：运行测试**

Run:

```bash
cd backend && go test ./services/llm-stream/app/projectcourse -run 'EvidencePack|Claim|Chapter|QualityGate' -v
```

Expected: PASS。

---

## 13. Task 10：生成累积式课程代码与自动测试

**Files:**

- Create: `backend/services/llm-stream/app/projectcourse/lab_builder.go`
- Create: `backend/services/llm-stream/app/projectcourse/lab_builder_test.go`
- Create: `backend/shared/kernel/projectcourse/lab.go`
- Modify: `backend/services/llm-stream/app/projectcourse/chapter_pipeline.go`

- [x] **Step 1：定义 LabManifest**

包含 language、toolchain version、allowed commands、starter、checkpoints、hints、solution、tests、resource limits 和 dependency graph。

- [x] **Step 2：约束简化边界**

生成前明确：保留哪些核心语言/协议/技术，删除哪些外围复杂度。更换核心语言视为合同失败。

- [x] **Step 3：按补丁构建检查点**

每章输出从 previous checkpoint 到 next checkpoint 的文件变化，服务端应用到单一 workspace，避免多个章节各自生成冲突的全量项目。

- [ ] **Step 4：生成测试优先**

先从学习成果生成可观察验收测试，再生成 solution；starter 应至少有一个预期失败测试，solution 必须全部通过。

- [x] **Step 5：生成分级提示**

Level 1 只提示方向，Level 2 指向模块或接口，Level 3 给骨架或关键伪代码；完整答案只在 solution。

- [ ] **Step 6：变式与故障任务**

每个核心技术至少生成一种变式或故障任务，并带独立测试，不把“复制最终代码”作为唯一通关方式。

- [x] **Step 7：运行纯结构测试**

Run:

```bash
cd backend && go test ./services/llm-stream/app/projectcourse -run 'Lab|Checkpoint|Hint|Exercise' -v
```

Expected: PASS。

---

## 14. Task 11：选型并实现隔离实验验证器

**Files:**

- Create: `docs/decisions/project-course-sandbox-selection.md`
- Create: `backend/services/course-runner/domain/verification/model.go`
- Create: `backend/services/course-runner/domain/verification/runner.go`
- Create: `backend/services/course-runner/domain/verification/consumer.go`
- Create: `backend/services/course-runner/app/bootstrap/bootstrap.go`
- Create: `backend/services/course-runner/cmd/main.go`
- Create: `backend/services/course-runner/Dockerfile`
- Create: `backend/services/course-runner/domain/verification/runner_test.go`
- Modify: `docker-compose.yml`
- Modify: `backend/.env.example`

- [x] **Step 1：完成成熟沙箱选型记录**

至少比较：独立远程沙箱、rootless 容器、gVisor、nsjail 等方案。记录许可证、维护状态、macOS/Linux 开发兼容、Compose 部署方式、网络隔离、资源限制、镜像体积和逃逸风险。

不得把挂载宿主 `docker.sock` 给业务容器作为默认方案。

- [x] **Step 2：定义 runner 窄接口**

```go
type Runner interface {
    Verify(ctx context.Context, request VerificationRequest) (VerificationResult, error)
}
```

request 只能引用已落盘且经过 manifest 校验的课程 artifact，不接受任意 shell 字符串。

- [x] **Step 3：实现命令模板白名单**

按 toolchain 映射固定 argv，例如 Go 只允许受控的 `go test ./...`。禁止 shell 展开、命令替换、重定向和任意环境变量继承。

- [x] **Step 4：实现隔离与资源限制**

必须验证：无网络、非 root、临时 workspace、只读工具链、CPU/内存/pids/timeout/output limit 生效。

- [x] **Step 5：恶意样例测试**

覆盖：读取宿主文件、访问网络、fork bomb、无限循环、超大输出、符号链接逃逸、写只读目录、读取环境密钥。

- [x] **Step 6：RabbitMQ 任务集成**

使用独立 queue 和 task subtype。验证结果回写结构化 task result；失败日志截断并清理敏感信息。

- [x] **Step 7：功能开关**

沙箱未完成安全验收前 `PROJECT_COURSE_LAB_VERIFICATION_ENABLED=false`。关闭时只能生成“未验证”产物，不能通过课程完整成功硬门禁。

- [ ] **Step 8：运行测试与安全冒烟**

Run:

```bash
cd backend && go test ./services/course-runner/... -v
docker compose --env-file backend/.env up -d --build course-runner rabbitmq
```

Expected: 单元测试和受控冒烟通过；容器无法访问外网和宿主敏感路径。

---

## 15. Task 12：课程 ZIP 打包与受控下载

**Files:**

- Create: `backend/services/export-service/domain/export/course_package.go`
- Create: `backend/services/export-service/domain/export/course_package_test.go`
- Modify: `backend/services/export-service/domain/export/consumer.go`
- Modify: `backend/services/export-service/infra/artifact/store.go`
- Modify: `backend/services/core-api/domain/task/download_handler.go`
- Modify: `frontend/src/services/generationTasks.ts` 或新增 course download service

- [x] **Step 1：写 ZIP 结构测试**

断言 README、manifest、starter、checkpoints、hints、solution、tests、coverage 都存在，路径无穿越且排序稳定。

- [x] **Step 2：生成 manifest**

记录 course ID、blueprint version、repo URL、commit SHA、toolchain、测试命令、验证结果和文件 hash。

- [x] **Step 3：只打包已验证工件**

若某个检查点验证失败，package task 失败并列出具体 checkpoint；不得用缺失目录凑出成功 ZIP。

- [x] **Step 4：受控下载与清理**

复用 task owner 鉴权。文件名净化、过期策略和不存在 artifact 的错误语义保持与现有下载接口一致。

- [x] **Step 5：运行测试**

Run:

```bash
cd backend && go test ./services/export-service/... ./services/core-api/domain/task -run 'CoursePackage|Download' -v
```

Expected: PASS。

---

## 16. Task 13：生成结果归并到 blogs 与课程实体

**Files:**

- Modify: `backend/services/llm-stream/domain/stream/generation_result.go`
- Modify: `backend/services/core-api/domain/task/generation_result.go`
- Modify: `backend/services/core-api/domain/task/result_persister.go`
- Modify: `backend/services/core-api/domain/task/generation_result_repository.go`
- Modify: `backend/services/core-api/domain/blog/persistence.go`
- Test: corresponding `*_test.go`

- [x] **Step 1：定义 result schema v1**

结果包含 course、volumes、chapters、quality report、coverage、blog mapping、lab artifact references 和 usage。每章有显式 succeeded/blocked/failed 状态。

- [x] **Step 2：保持 task-only 边界**

`llm-stream` 只写 task result；`core-api` 事务性写入博客树和课程状态。

- [x] **Step 3：映射分卷博客树**

若现有 blogs 只支持一层 parent/child，首期使用一个课程父博客，卷册信息写入章节 metadata 和导读目录；不要未经评审扩展任意深度树。

- [x] **Step 4：部分失败语义**

成功章节可以持久化，但存在硬失败章节时课程状态为 `partially_blocked`，不能标为完整成功。重试只替换目标章节。

- [x] **Step 5：幂等归并**

使用 course ID、blueprint version、chapter ID 做幂等键；重复消费不得重复创建博客或累计 token。

- [x] **Step 6：运行测试**

Run:

```bash
cd backend && go test ./services/core-api/domain/task ./services/core-api/domain/blog ./services/llm-stream/domain/stream -run 'ProjectCourse|Idempotent|Partial' -v
```

Expected: PASS。

---

## 17. Task 14：SSE、可恢复检查点与观测

**Files:**

- Modify: `backend/services/llm-stream/domain/stream/task_consumer.go`
- Modify: `backend/services/llm-stream/domain/stream/task_store.go`
- Modify: `backend/services/core-api/domain/task/handler.go`
- Modify: `frontend/src/store/projectCourseStore.ts`
- Modify: `frontend/src/components/project-course/BlueprintWorkspace.tsx`
- Modify: `docs/runbooks/microservices-smoke-check.md`

- [x] **Step 1：增加课程阶段事件**

事件带 course/volume/chapter/stage/sequence，中文消息只用于展示，前端逻辑使用稳定枚举。

- [ ] **Step 2：阶段检查点**

保存 snapshot、inventory、knowledge graph、blueprint、evidence pack、claim plan、draft、review、lab manifest 和 verification result 的完成状态与 hash。

- [ ] **Step 3：幂等重试**

输入 hash 未变化时复用已完成阶段；commit、blueprint version、chapter contract 或 evidence hash 变化时只使相关下游阶段失效。

- [ ] **Step 4：观测指标**

至少记录：

- 每阶段耗时与模型调用数；
- token 与 cache hit/miss；
- hard/soft gate 失败次数；
- unsupported/contradicted claims；
- 实验验证耗时和资源峰值；
- 模块/主链路/文件处置覆盖率；
- 章节重试和恢复命中率。

- [x] **Step 5：取消语义**

取消新模型请求与未开始实验；已写入的阶段检查点保留。取消不得把课程标为失败或删除已验证产物。

- [x] **Step 6：运行任务与 SSE 测试**

Run:

```bash
cd backend && go test ./services/llm-stream/domain/stream ./services/core-api/domain/task -run 'Course|Checkpoint|Cancel|SSE' -v
cd frontend && npm test -- --run src/store/projectCourseStore.test.ts
```

Expected: PASS。

---

## 18. Task 15：InkWords 端到端验收与人工 dogfood

**Files:**

- Create: `docs/qa/project-course-inkwords-report.md`
- Create: `backend/scripts/test_project_course_e2e.go`
- Modify: `docs/qa/e2e-testing.md`
- Modify: `README.md`

- [x] **Step 1：锁定验收 SHA**

运行时解析 `https://github.com/2692341798/InkWords` 的 `main`，把 SHA 写入报告、课程 manifest 和所有 EvidenceRef。

- [ ] **Step 2：生成三种读者等级蓝图**

比较章节事实和证据是否稳定，差异是否只体现在解释与脚手架深度。默认 `programming` 进入完整生成。

- [ ] **Step 3：审阅并批准蓝图**

只通过 UI 修改至少一个标题、一次排序和一个非核心章节启用状态，验证版本、依赖与覆盖变化。

- [ ] **Step 4：生成完整课程**

验证至少包含：项目地图、生产主链路、核心技术原理、源码深挖、取舍、实验、故障排查和综合挑战。

- [ ] **Step 5：自动硬门禁**

要求：

```text
核心模块映射率 = 100%
主链路映射率 = 100%
项目事实证据覆盖率 = 100%
引用路径/符号有效率 = 100%
计划验证实验通过率 = 100%
文件处置率 = 100%
```

- [ ] **Step 6：失败样例回归**

确认课程不再把旧路径、旧构造方法或无法运行的临时代码写成当前实现。

- [ ] **Step 7：人工完成一条累积路径**

测试者从 starter 开始，不先查看 solution：

1. 按章节完成至少三个检查点；
2. 只在卡住时逐级查看提示；
3. 完成一个变式任务；
4. 定位一个预置故障；
5. 运行全部测试；
6. 口头或书面解释关键技术的采用原因和替代方案。

- [ ] **Step 8：全量验证**

Run:

```bash
docker compose --env-file backend/.env up -d --build
cd backend && go test ./...
cd frontend && npm test -- --run
cd frontend && npm run lint
cd frontend && npm run build
```

Expected: 全部通过；任何环境原因导致的未验证项必须写入 QA 报告，不得伪造完成。

- [ ] **Step 9：文档同步**

更新 README、API、架构、运行方式、环境变量、沙箱安全边界和课程 ZIP 结构。

---

## 19. 发布顺序

### Release A：只读蓝图预览

- 开启 snapshot、inventory、knowledge graph、blueprint；
- 不开放正文生成和实验；
- 收集覆盖错误与用户对卷册结构的反馈。

### Release B：证据驱动文章

- 开放批准和分类型章节生成；
- 启用事实硬门禁；
- 课程代码仍标记为未验证，不提供最终 ZIP。

### Release C：实验与打包

- 沙箱安全验收通过后启用 lab verification；
- 开放 starter/checkpoints/hints/solution/tests ZIP；
- 完成 InkWords dogfood 后再将入口从实验功能提升为正式功能。

### 回滚

- 关闭 `PROJECT_COURSE_ENABLED` 隐藏入口并阻止新任务；
- 活动任务进入 cancelled，不删除 course、blog 或 artifact；
- 旧三种场景和旧系列生成继续工作；
- 数据库只做前向兼容，不对已有课程执行破坏性回滚。

---

## 20. 最终完成检查表

- [x] 独立场景存在且默认不影响旧链路。
- [x] 仓库 ref 在任务开始时解析为固定 SHA。
- [x] 文档、测试、示例和脚本进入确定性清单。
- [ ] Go、TS/TSX 和配置适配器满足 InkWords 基准。
- [x] 知识图、主链路和 EvidenceRef 可审计。
- [x] 超大仓库能分卷且保留全局覆盖清单。
- [x] 用户只能修改标题、顺序和启用状态。
- [x] 蓝图批准和生成绑定精确版本。
- [x] 每种章节类型使用独立合同。
- [x] 项目事实和核心原理来源通过硬门禁。
- [x] 终稿仍使用 Evidence Pack。
- [x] 累积实验包含 starter、检查点、提示、答案和测试。
- [x] 目标仓库从未被执行。
- [ ] 实验在成熟隔离方案中禁网、限时、限资源运行。
- [x] ZIP 受控下载且 manifest 锁定 SHA 与文件 hash。
- [x] task-only 结果由 core-api 幂等归并。
- [ ] InkWords 自动指标和人工 dogfood 均通过。
- [x] 全量 Go/前端测试、lint 和 build 通过。
- [x] README、API、架构和 runbook 已同步。
