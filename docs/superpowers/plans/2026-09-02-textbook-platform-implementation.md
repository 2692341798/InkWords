# InkWords 可自学教材平台实施计划

> 对应产品需求：`.trae/documents/InkWords_PRD.md`
> 对应开发设计：`docs/superpowers/specs/2026-09-02-textbook-platform-development-design.md`
> 本计划基于 `main@a3de0c0` 制定，日期：2026-09-02。

> 2026-09-13 媒体检查点：三张冻结图片已嵌入 Markdown/DOCX/PDF，ZIP 24 项内容哈希通过。
> WPS 26 页实查图片，PDF 22 页；纯图片页页脚经原图/WPS 复查正常，已更正此前观察错误。
> 下一步继续 WebP 解码（依赖授权待答复）与剩余整书审校。
> 带图构建现已保存 layout v1 AI 审阅 3/4，刷新与 ZIP 回读一致；ZIP 16 项阻断仍保留。
> 证据见 `docs/qa/media-build-layout-review-2026-09-13.md`，其余阶段和原综合验收继续未完成。
> 内容审阅续：technical v1 AI 3/4；developmental v1 需修改 2/4，明确缺少 hands_on 代表章。
> 教案仍缺 sandbox-verification 截图（本地采集扩展授权待答复）。见 `docs/qa/media-build-content-review-2026-09-13.md`。
> 已新增 hands_on 空章与蓝图 r6 草稿（17 个来源片段）；原批准 r4/r15 不变，尚未生成或批准。
> 蓝图章节绑定校验已修复并本地部署：真实 PostgreSQL 红绿、后端全量、网关创建/旧稿批准拒绝通过，原状态不变。
> 已核对固定提交对应 Gin v1.12.0，并从缓存准备 31 模块离线包；校验器/组合接口与后端全量通过，尚未启用或运行。
> 组合工件独立预算、清单 pin、私有执行快照和冻结导出代码已完成；真实依赖树 2,198 文件存储/复制/导出夹具及后端全量通过。
> 本地检查器/工件修复工具已支持项目与主资料显式选择，真实空章只读预检、PostgreSQL 登记边界与后端全量通过。
> 已修复 hands_on 仍被强制生成概念路由树的问题：提示 v17/质量 v10 按冻结 profile 分流，文件/回环服务与选择中的 browser_page 合同完成。
> 全量回归及带浏览器观测的真实选择预检通过，见 `docs/qa/hands-on-generation-contract-2026-09-13.md`。
> 四服务现已部署并健康，权限/挂载等价、二进制哈希通过；runner 保持 Go 1.26.8。真实页面已代操作批准蓝图 r6，空章预检显示 v17/v10、17 段证据、输入 17,213 Token；未创建生成任务，等待一次有界调用授权。见 `docs/qa/hands-on-runtime-deploy-2026-09-13.md`。
> 新增当前冻结单章自学性 AI 审阅 2/4（迁移题重复完整示范，SS-01）、一致性与文字各 3/4；六条 AI、零真人，刷新/ZIP 24 项哈希通过。质量 v9 冻结报告在 v10 服务下另有过期阻断；下一步准备 SS-01 修订候选并重检，不改写旧报告。见 `docs/qa/media-self-study-review-2026-09-13.md`。
> SS-01 正文/学习阶段/练习整体修订提案已通过 v10 离线门禁，原代码和六项练习身份保留；真实 correct-sample 返回 400，完整章节状态不变。旧任务 v16/v9 与当前合同、r6 蓝图不兼容，下一步补显式双身份迁移与当前 CAS 检查，再登记候选。见 `docs/qa/transfer-correction-2026-09-13.md`。
> 原三项综合验收不勾选，详见 `docs/qa/frozen-images-rendering-2026-09-13.md`。成片录像不是必交付项。
> 迁移续：v6 显式绑定历史/当前双身份，旧章同源、当前 CAS 与 v10 重检、零调用通过；后端全量/真实 PostgreSQL 与前端 317 项通过。已部署并从真实任务恢复落 r16 候选，旧 r15/15 个修订/冻结构建保留；修正页面仍按 v9 判定的误报。下一步复审候选、显式批准与新冻结，原自学失败记录不搬用。见 `docs/qa/correction-contract-migration-2026-09-13.md`。

> 复审续：真实页面八维 AI 复审各 3/4，r16 已批准为 r17；独立沙箱验证 exit 0，三张原图同字节复用并明确来源。新构建 9c4dfc67-532a-46f6-906a-80a2a7876e67 绑定新稿/v10/本次收据/三图，16 个旧修订与旧构建保留。ZIP 24 项哈希通过、DOCX/PDF 各三图，PDF 21 页变化页 11/14 可读。新 WPS 校样与整书复审待做，不继承旧六条审阅；ZIP 17 项阻断与原综合项保留。见 `docs/qa/r16-candidate-review-2026-09-13.md`。

> WPS 续：明确打开 r17 新构建 DOCX，实际 27 页；迁移题、评分/提示、三图和末页定向检查可读，打开前后字节哈希相同。尚未逐页复核全篇，未登记整书版式通过；原综合项保留。见 `docs/qa/r17-wps-proof-2026-09-13.md`。

> 全页续：服务端 PDF 21 页与 WPS 本地打印校样 27 页均已完成编号总览检查和重点放大，保存 r17 新构建 layout v1 AI 3/4；刷新/API/ZIP 一致，24 项内容哈希与冻结内容不变通过。API 14、ZIP 16 阻断保留；其余七阶段及原综合验收未完成。见 `docs/qa/r17-layout-review-2026-09-13.md`。

> 内容复审续：r17 完整母稿、当前 16 个来源片段/两份代码/独立收据已核对；技术、自学性、一致性、文字各 AI 3/4，发展性需要修改 2/4（蓝图已规划，hands_on 批准稿仍缺）。六条 AI/零真人，真实刷新/API/ZIP 一致，24 项哈希与冻结内容不变；API 11、ZIP 13 阻断及原综合项保留。见 `docs/qa/r17-content-review-2026-09-13.md`。

## 1. 交付目标

把现有“项目精通课程”能力渐进演化为一个本地、单用户、桌面生产优先的可自学教材平台。最终交付不是若干互相分离的博客、课程和视频稿，而是一份有版本、有证据、有运行结果、可持续修订的教材母稿；博客、视频操作教案、学习任务、复习任务、Markdown、DOCX、PDF 和 ZIP 都由母稿投影生成。

首个标杆教材使用 Gin 主资料和官方资料完成。C++ 官方站仅用于验证第二种技术栈和官网文档栈的通用性，不在 Gin 纵向切片通过前并行开发。

### 1.1 最终成功标准

- 不登录即可在本机创建教材项目、配置 API Key、导入资料并继续上次工作。
- 支持 GitHub、官网文档栈以及 PDF、DOCX、Markdown、TXT、ZIP；每次导入形成不可变快照。
- 一个项目只能指定一份主资料，其余资料必须是已确认的官方补充资料。
- 可为零基础、有编程基础、熟悉技术栈三种读者生成不同版本。
- 每章回答“为什么需要、解决什么问题、有哪些实现、如何使用、为何有效、边界与取舍”。
- 每章遵循自然学习曲线，包含场景、受控类比、示范、渐退练习、独立迁移、复述与延迟复习。
- 所有关键事实可追溯到主资料或官方资料；代码来自固定快照或经过验证的教学工件。
- 人工修改的已批准版本不会被后台生成、重试或批处理覆盖。
- 蓝图和样章/风格探针分别审批；未通过审批不得批量生成整书。
- 复述、编码、复现、迁移、排错和延迟保持共同决定下一次任务类型与到期时间。
- 只在主动打开 InkWords 时显示到期任务，不运行通知守护进程。
- 能输出 Markdown、DOCX、PDF、ZIP；出版候选还必须通过权利、编校、版面和字段真实性预检。

### 1.2 明确不做

- 不做 SaaS、多租户、公共注册、服务器部署、团队协作、原生安装包，也不做桌面端/移动端分设备适配。
- 不建设网页 IDE，不执行被导入的目标仓库及其安装脚本、hook 或服务。
- 不自动发布书籍，不伪造 ISBN、CIP、出版社、专家审稿或官方认证。
- 不让联网搜索成为教材事实来源；联网只用于发现并确认官方资料。
- 首期不引入向量数据库；先用标题、层级、符号、关键词和固定范围做可解释检索。
- 不进行一次性全站重写，也不在功能 PR 中混入无关依赖升级。

## 2. 当前基线与改造策略

现有代码已有可复用基础：固定 Git SHA、Repository Knowledge Graph、EvidenceRef/Claim Ledger、蓝图审批、RabbitMQ 任务、SSE、task-only 持久化、课程 ZIP、PDF 导出、复习会话和架构测试。以下差距决定实施顺序：

| 当前实现 | 目标状态 | 处理方式 |
| --- | --- | --- |
| `ProjectCourse` 仍以 `UserID + repo URL/ref` 为核心 | 本地 workspace 下的教材项目和多资料快照 | 先加兼容层和新表，最后删除 auth，不做大爆炸重命名 |
| 章节正文最终仍落为 blog | 教材母稿是唯一内容真相 | 新建 revision/投影模型，Blog 仅是派生视图 |
| 蓝图批准后直接生成全课程 | 蓝图审批 → 样章/风格审批 → 按章/整书生成 | 增加双审批和候选稿状态机 |
| 官网资料只抓取单页 | 有边界、预算、清单和恢复点的文档栈抓取 | `CrawlerPort` + manifest + 同域规则 |
| PDF/DOCX/ZIP 被拍平成长字符串 | 保留文档、章节、页码、路径和代码块结构 | 返回 `SourceDocument + SourceChunk[]` |
| DeepSeek 类型泄漏到业务层 | DeepSeek/OpenAI 可替换生成端口 | 建立 `GenerationPort` 和统一 Usage |
| 复习按未复习/最久未复习选择 | 六维表现 + FSRS 到期队列 | 先事件模型，再接调度算法 |
| bubblewrap 在默认 Compose 中不能创建 namespace | 代码运行证据必须真实且 fail-closed | 先做隔离执行器 spike；不通过就保持“未验证” |
| PDF/ZIP 围绕 blog/course | Canonical Book AST 驱动所有导出 | 先建 AST，再接 Pandoc/Chromium |

### 2.1 渐进演化边界

1. 保留 `projectcourse` 现有公开合同，在新 `textbook` 合同成熟后通过适配器迁移调用方。
2. `core-api` 继续持有权威业务状态；worker 只消费任务、生成候选结果，不直接改写已批准正文。
3. `parser-service` 只负责获取与结构化；`llm-stream` 负责检索、规划与生成；`course-runner` 负责受控工件验证；`review-service` 负责学习事件和调度；`export-service` 负责不可变构建。
4. 先完成 Gin 单章纵向切片，再扩到完整资料栈和整本教材。
5. 迁移期允许旧 blog/review 数据通过固定本地 owner 读取，新表只写 `workspace_id`。

## 3. 执行协议

### 3.1 每个任务的标准循环

每个原子 PR 都按以下顺序执行：

1. 读取本任务涉及的实现、测试、相邻服务合同和最近层级 `AGENTS.md`。
2. 写一个能证明缺口的失败测试或固定离线夹具。
3. 做满足验收条件的最小实现；发现架构阻塞时先写 ADR，不顺手扩范围。
4. 运行目标测试，再运行服务级回归；跨服务合同变更必须跑集成测试。
5. 用 `git diff --check`、`git diff`、`git status` 审查，只暂存本任务文件。
6. 更新同 PR 的契约/运行文档和 QA 证据。
7. 创建 Conventional Commit；PR 描述写清 Why、What、验证、风险和回滚。
8. CI 全绿且人工门槛满足后再合并；下一任务从更新后的 `origin/main` 开始。

### 3.2 工作区保护

制定本计划时以下文件已有用户未提交改动，所有任务默认排除：

```text
backend/go.mod
backend/go.sum
frontend/package.json
frontend/package-lock.json
```

需要引入 Colly、goose、go-fsrs 或其他依赖时，必须先确认上述改动的归属和预期，再创建单独的 `chore(deps)` 原子提交。禁止使用 `git add .`。

### 3.3 完成定义

单个任务只有同时满足以下条件才能勾选完成：

- 失败测试先存在，随后因实现通过；
- 新增公开合同有序列化/反序列化和未知值测试；
- 数据迁移有 PostgreSQL 集成测试和回滚说明；
- 后台任务有幂等键、稳定错误码、进度和取消语义；
- UI 变更有 Vitest，核心流程有 Playwright 证据；
- 没有静默降级、没有伪造“已验证”、没有覆盖人工批准内容；
- 文档、配置样例和实际代码一致；
- 目标测试、全量回归和 CI 结果被记录在 PR 中。

## 4. 里程碑总览

| 里程碑 | 原子 PR | 可见交付物 | 依赖 | 规模 |
| --- | --- | --- | --- | --- |
| M0 基线护栏 | PR-00～PR-02 | 新合同、迁移机制、本地 workspace | 无 | M |
| M1 Gin 样章 | PR-03～PR-05 | 无登录创建项目并生成、审批、导出一章 | M0 | L |
| M2 完整资料栈 | PR-06～PR-08 | 官网/文件结构化导入、可解释检索、证据回链 | M1 | XL |
| M3 教材生产线 | PR-09～PR-11 | 双审批、质量门禁、Provider/Token、可靠恢复 | M2 | XL |
| M4 演示与运行证据 | PR-12～PR-13 | 可运行代码证据、截图资产和视频操作教案 | M3 | L |
| M5 掌握闭环 | PR-14 | 六维自适应学习和打开应用时的到期任务 | M3 | L |
| M6 出版构建 | PR-15～PR-16 | 整书审校、DOCX/PDF/ZIP、Gin dogfood | M4、M5 | XL |

规模表示相对工作量，不是日历承诺：S 约 1～2 个专注开发日，M 约 3～5 日，L 约 1～2 周，XL 应拆成多个小提交但保持一个业务里程碑。

关键路径：

```text
PR-00 → PR-01 → PR-02 → PR-03 → PR-04 → PR-05
                                      ↓
PR-06 → PR-07 → PR-08 → PR-09 → PR-10 → PR-11
                                      ├─→ PR-12 → PR-13 ─┐
                                      └─→ PR-14 ─────────┼─→ PR-15 → PR-16
```

## 5. M0：基线与架构护栏

### PR-00：固定教材质量、Token 与失败样例基线

**目标**：在改变业务前建立可比较基线，避免“功能更多但教材更差”。

**Files:**

- Create: `docs/qa/textbook-platform-baseline.md`
- Create: `backend/services/llm-stream/app/projectcourse/testdata/textbook-gin-fixture/manifest.json`
- Create: `backend/services/llm-stream/app/projectcourse/testdata/textbook-gin-fixture/expected-quality.json`
- Create: `backend/services/llm-stream/app/projectcourse/textbook_baseline_test.go`
- Modify: `docs/qa/e2e-testing.md`

**步骤：**

- [x] 固定 Gin 仓库 SHA、主资料清单、允许的官方 URL、代表性代码和 hash；夹具必须脱敏且足以离线运行。
- [x] 选取概念章和实操章，记录当前尚未进行真实模型调用、Token/延迟与目标读者评分均未采集的状态；不得伪造基线输出或人工评分。
- [x] 固化失败样例：术语墙、循环定义、未解释缩写、隐藏前提、类比越界、无证据 claim、不可运行代码、缺恢复路径、伪造运行截图。
- [x] 建立 4 分制人工评分表：正确性、自学性、结构、证据、代码、迁移能力、出版可编辑性；目标读者试学仍待后续 `ReaderTrial` 记录。
- [x] 记录当前 bubblewrap、DOCX、PDF 和复习选择器的真实限制。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/projectcourse -run 'TextbookBaseline|OfflineInkWordsAcceptance' -v
cd frontend && npm test -- --run
```

**退出条件**：离线夹具可重复运行；基线文件明确区分自动检查、人工判断和未验证项；无业务行为变化。

**提交**：`test(textbook): freeze gin manuscript quality baseline`

### PR-01：建立教材共享合同与兼容适配层

**目标**：用稳定领域语言承载教材，而不是继续给 `ProjectCourse` 和 blog 叠字段。

**Files:**

- Create: `backend/shared/kernel/textbook/enums.go`
- Create: `backend/shared/kernel/textbook/source.go`
- Create: `backend/shared/kernel/textbook/book_contract.go`
- Create: `backend/shared/kernel/textbook/style_sheet.go`
- Create: `backend/shared/kernel/textbook/manuscript.go`
- Create: `backend/shared/kernel/textbook/learning.go`
- Create: `backend/shared/kernel/textbook/quality.go`
- Create: `backend/shared/kernel/textbook/publication.go`
- Create: `backend/shared/kernel/textbook/events.go`
- Create: `backend/shared/kernel/textbook/contracts_test.go`
- Create: `backend/shared/kernel/projectcourse/textbook_adapter.go`
- Modify: `backend/services/architecture_test.go`

**先写测试：**

- [x] 三种 audience、七种 ChapterProfile、状态机、门禁结果和 JSON 值稳定。
- [x] BookContract/StyleSheet/Blueprint/ChapterRevision 都有独立 revision ID、version 和 hash。
- [x] `ScenarioFrame` 必须含真实问题、目标、类比、对应关系、失效边界和回到技术事实的桥。
- [x] `UnderstandingChain` 六问齐全；`LearningArc` 不能从 worked example 直接跳到无提示综合任务。
- [x] CodeArtifact、EvidenceRef、QualityAssessment、RightsItem、BookBuild 合同能拒绝缺失关键字段。
- [x] 旧 `projectcourse` 合同能经显式适配器映射，不允许 worker 直接 import peer service。

**实现约束：**

- 合同只表达业务语义，不 import GORM、Gin、DeepSeek、OpenAI、RabbitMQ 或具体文件解析器。
- JSONB 文档结构放合同；身份、状态、关系、hash、due_at 等可查询字段留给数据库列。
- 单文件接近 500 行即按领域拆分。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./shared/kernel/textbook ./shared/kernel/projectcourse ./services -run 'Textbook|Architecture' -v
```

**退出条件**：新合同完全由单元测试保护，旧功能无需修改即可继续运行。

**提交**：`feat(textbook): add versioned manuscript contracts`

### PR-02：引入版本化迁移与 LocalWorkspaceContext

**目标**：为单用户本地模式建立稳定身份边界，开始脱离生产 AutoMigrate，但暂不删除旧认证。

**Files:**

- Create: `backend/shared/platform/postgres/migrations/embed.go`
- Create: `backend/shared/platform/postgres/migrations/00001_local_workspace.sql`
- Create: `backend/shared/platform/postgres/migrations/migrations_test.go`
- Create: `backend/shared/kernel/httpx/local_workspace.go`
- Create: `backend/shared/kernel/httpx/local_workspace_test.go`
- Modify: `backend/shared/platform/postgres/core.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `docker-compose.yml`
- Create: `docs/decisions/versioned-database-migrations.md`

**步骤：**

- [x] 先用 PostgreSQL Testcontainers 写 migration up/down 和重复执行测试；SQLite 只用于快速 repository 单测。
- [x] 评估并单独引入 goose。其嵌入式 SQL migration 能随本地二进制/容器分发；版本选择、许可证和升级策略写入 ADR。
- [x] 新建唯一 `local_workspaces` 记录，使用稳定 UUID；middleware 注入 `workspace_id`，不信任客户端传入 owner。
- [x] 新表只使用 `workspace_id`；旧 user/blog/task/review 通过 `local_workspace_legacy_owner` 映射继续读取。
- [x] 默认本地路由先保持兼容认证；仅教材新路由使用 LocalWorkspaceContext。
- [x] 生产启动执行 goose；AutoMigrate 暂保留在测试和迁移期旧表路径，并打印可见告警。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./shared/platform/postgres/... ./shared/kernel/httpx/... ./services/core-api/... -run 'Migration|Workspace' -v
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault docker compose config
```

**退出条件**：全新数据库和既有数据库都能启动；重复 migration 无副作用；旧路由行为不变。

**提交**：`feat(platform): add local workspace and versioned migrations`

## 6. M1：Gin 样章纵向切片

### PR-03：建立教材项目、资料和章节修订权威模型

**目标**：让 `core-api` 能持久化最小教材项目，而不是把新状态塞入单个 `ProjectCourse` JSON。

**Files:**

- Create: `backend/services/core-api/domain/textbook/model.go`
- Create: `backend/services/core-api/domain/textbook/dto.go`
- Create: `backend/services/core-api/domain/textbook/repository.go`
- Create: `backend/services/core-api/domain/textbook/service.go`
- Create: `backend/services/core-api/domain/textbook/handler.go`
- Create: `backend/services/core-api/domain/textbook/*_test.go`
- Create: `backend/shared/platform/postgres/migrations/00002_textbook_core.sql`
- Modify: `backend/services/core-api/transport/http/v1/routes.go`
- Modify: `backend/services/core-api/app/bootstrap/bootstrap.go`

**最小表：**

```text
textbook_projects
textbook_sources
source_snapshots
book_contract_revisions
style_sheet_revisions
blueprint_revisions
textbook_chapters
chapter_revisions
chapter_locks
```

**步骤：**

- [x] repository 测试覆盖 workspace 隔离、线性 revision、CAS、批准后不可原地修改和软删除。
- [x] 一个项目最多一个 `primary` source；supplementary 必须 `official_confirmed=true`。
- [x] 章节人工编辑创建新 revision；批量任务只能创建 candidate，不能替换 approved revision。
- [x] 锁定采用短租约 + owner + version；过期可恢复，两个标签页同时应用只允许一个成功。
- [x] API 使用稳定错误码：`TEXTBOOK_VERSION_CONFLICT`、`PRIMARY_SOURCE_EXISTS`、`REVISION_LOCKED`、`INVALID_STATE`。
- [x] 为列表、状态、项目/章节 revision 查询补索引，并附真实 PostgreSQL `EXPLAIN`。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/core-api/domain/textbook ./services/core-api/transport/http/v1 -run Textbook -v
```

**退出条件**：并发和重试不能覆盖已批准版本；所有权威状态仅由 core-api 写入。

**提交**：`feat(textbook): persist projects and linear revisions`

### PR-04：重构前端为教材工作台外壳

**目标**：不重写全站设计系统，先让用户无登录进入本地教材项目并看到完整生产步骤。

**Files:**

- Create: `frontend/src/features/textbook-projects/`
- Create: `frontend/src/features/source-library/`
- Create: `frontend/src/features/manuscript/`
- Create: `frontend/src/features/generation/`
- Create: `frontend/src/services/textbook.ts`
- Create: `frontend/src/store/textbookStore.ts`
- Modify: `frontend/src/pages/ProjectCourse.tsx`
- Modify: `frontend/src/components/project-course/BlueprintWorkspace.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/Sidebar.tsx`
- Test: `frontend/src/features/**/*.test.tsx`

**步骤：**

- [x] HomeEntry 默认进入教材项目列表；旧登录页面仍可通过兼容入口访问，但不阻塞教材路由。
- [x] 项目向导分为：目标读者 → 主资料 → 官方补充资料 → BookContract → StyleSheet。前两步在创建前完成；创建成功后自动打开项目，从官方补充资料继续，并在同一工作台内完成契约与 StyleSheet，避免把尚不存在的项目伪装成可登记资料的状态。
- [x] 工作台展示资料、蓝图、样章审批、章节、验证、学习、出版七个阶段，状态来自后端而非前端猜测；蓝图与显式证据齐备后可创建候选样章任务，未接入的验证、学习和出版阶段仍明确显示为不可用。
- [x] 复用 Markdown editor；增加 revision、dirty、lock、候选稿谱系摘要、diff 和“应用候选稿”面板。
- [x] 通过键盘完成核心操作；错误信息给出失败原因、已保留内容和恢复动作。创建向导使用原生表单控件与按钮，项目草稿保留在本机，失败后提示修正并重试；章节、资料、契约和审批也继续使用可聚焦控件。
- [x] `ProjectCourse` 保持旧 URL 兼容，内部逐步投影为 TextbookProject。

**验证：**

```bash
cd frontend && npm test -- --run
cd frontend && npm run lint
cd frontend && npm run build
```

**退出条件**：无 API Key 时仍可创建和浏览空项目；刷新页面不会丢失本地状态；旧博客入口不退化。

**提交**：`feat(frontend): add local textbook workspace shell`

### PR-05：打通 Gin 零基础样章

**目标**：用现有固定 SHA、EvidencePack 和 fake provider，完成一章真实可审批母稿，尽早验证教材方向。

**Files:**

- Create: `backend/services/llm-stream/app/textbook/sample_chapter.go`
- Create: `backend/services/llm-stream/app/textbook/sample_chapter_test.go`
- Create: `backend/services/llm-stream/app/textbook/prompt_contract.go`
- Create: `backend/services/llm-stream/app/textbook/quality_gates.go`
- Create: `backend/services/llm-stream/app/textbook/fake_generation.go`
- Modify: `backend/services/llm-stream/app/projectcourse/chapter_pipeline.go`
- Modify: `backend/services/llm-stream/app/projectcourse/quality_gates.go`
- Modify: `backend/services/llm-stream/domain/stream/task_consumer.go`
- Modify: `backend/services/export-service/domain/export/course_package.go`
- Create: `frontend/e2e/textbook-sample-chapter.spec.ts`

**样章内容门禁：**

- [x] 选择 Gin 请求生命周期作为首个概念，用“GET 后出现 404、却不知道登记链在哪里”的排错场景提出问题。
- [x] `ScenarioFrame` 对餐厅菜单类比逐项映射方法/路径、处理函数链、方法路由树，并明确它不代表线程或逐请求登记模型。
- [x] Need、Problem、Alternatives、Usage、Mechanism、Observation、Tradeoffs 与解释/补全/从零实现/迁移/排错/延迟复述六类检查共同冻结在样章工件中。
- [x] 生成证据包只接受主资料或确认的官方资料；样章把每个关键 claim 映射到可见 evidence 引用，门禁计算并要求关键 claim 覆盖率 100%。
- [x] 最小示例代码可复制；隔离 runner 未接入时门禁要求明确“未验证”，并拒绝伪造终端输出、截图或运行结果。
- [x] 样章正文与结构化 LearningArc 都覆盖先验激活、问题体验、全貌预告、完整示范、引导练习、脚手架渐退、独立迁移、复述、延迟提取和按表现调整。
- [x] 样章先以白话说明登记材料，再引入精确术语，并明确本节只引入四个核心元素。
- [x] 门禁拒绝“显然、很简单、留给读者、顾名思义”等防自学表达；样章操作包含工具、路径、预期和恢复动作。
- [x] 单章 Markdown/ZIP 导出只沿本地工作区内的 `approved_revision_id` 读取；未批准、跨工作区或悬空指针均 fail-closed，ZIP 同时保存不可变修订的谱系 manifest。

**端到端路径：**

```text
创建项目 → 选择零基础 → 载入 Gin 离线快照 → 生成蓝图候选
→ 人工批准蓝图 → 生成样章候选 → 查看 diff/证据/质量报告
→ 人工批准样章 → 导出 Markdown/ZIP
```

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/textbook ./services/llm-stream/app/projectcourse ./services/export-service/... -run 'SampleChapter|Textbook' -v
cd frontend && npx playwright test e2e/textbook-sample-chapter.spec.ts --project=chromium
```

**退出条件**：用户无需口头补充即可按样章完成一次学习和练习；人工评分所有维度 ≥ 3/4；只有批准版本进入导出。

**提交**：`feat(textbook): deliver gin sample chapter slice`

## 7. M2：完整资料栈与可解释检索

### PR-06：建立有边界的官网文档栈抓取器

**目标**：自动抓取“整个文档栈”，但整个的含义由显式边界、预算和清单定义，而不是无限爬网。

**Files:**

- Create: `backend/services/parser-service/domain/crawl/port.go`
- Create: `backend/services/parser-service/domain/crawl/policy.go`
- Create: `backend/services/parser-service/domain/crawl/service.go`
- Create: `backend/services/parser-service/domain/crawl/manifest.go`
- Create: `backend/services/parser-service/domain/crawl/*_test.go`
- Create: `backend/shared/platform/crawler/colly_adapter.go`
- Create: `backend/shared/platform/crawler/colly_adapter_test.go`
- Modify: `backend/services/parser-service/app/bootstrap/bootstrap.go`
- Modify: `backend/services/parser-service/transport/http/v1/routes.go`
- Create: `docs/decisions/official-document-crawler.md`

**边界合同：**

- [x] 入口 URL、允许 host、允许 path prefix、最大深度、最大页面数、最大总字节、单页超时、总耗时为必填或有安全默认值；首期 HTTP 入口只允许缩小 path 边界。
- [x] 默认同源；跨域、下载文件、canonical 跳转和子域必须显式批准，且重定向目标在连接前重走 `Policy`。
- [x] 尊重 robots.txt、Retry-After、速率限制；首期串行抓取（并发上限为 1）以保证确定性排序和去重。
- [x] 阻止 localhost、私网、link-local、file scheme、DNS rebinding 和重定向逃逸；HTTP transport 在连接前重验 DNS 结果。
- [x] 过滤登录、搜索、日历、分页环和会话参数；保留 skipped/failed 原因。
- [x] 每页保存 URL、canonical URL、title、heading tree、content hash、抓取时间、HTTP 元数据和父链接。
- [x] 取消或总超时后把队列、已处理 URL、清单和原策略原子持久化；恢复只能重用保存策略，且已完成页面不重复抓取。
- [x] 同策略且远端内容未变化时，通过 HTTP 再验证、SourceSnapshot 输入哈希和 generation idempotency key 避免重复抓取及模型调用；远端任一页面变化即 fail-closed，绝不以过期缓存冒充当前资料。

**依赖决策**：优先评估 Colly。它提供并发/延迟、缓存、编码与 robots.txt 等能力，但业务边界、SSRF 防护和 manifest 仍由 InkWords 自己控制；许可证、版本和镜像影响写入 ADR。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/parser-service/domain/crawl ./shared/platform/crawler -v
```

另设显式 opt-in 的 Gin/go.dev 真实抓取测试；普通 CI 只使用本地 HTTP fixture，不能依赖互联网。

**退出条件**：Gin 官方文档栈生成完整 manifest；预算耗尽是可恢复的部分成功，不伪装为完整抓取。

**验收记录（2026-09-03）**：已在隔离 Compose 环境实际抓取 `https://gin-gonic.com/en/docs/`，路径严格限制为 `/en/docs`。manifest 为 `complete`，包含 86 页、7,283,550 字节；重复、跨域、超出路径边界、HTTP 404/503 与不安全 URL 都以可审计的 skipped decision 留存。详见 `docs/qa/gin-textbook-acceptance.md`。该记录仅完成爬虫出口条件，网页 manifest 投影为教材 SourceSnapshot/SourceDocument/SourceChunk 仍属于后续接入工作。

**提交**：`feat(parser): crawl bounded official documentation stacks`

### PR-07：把文件解析升级为结构化 SourceDocument

**目标**：保留 PDF、DOCX、Markdown、TXT、ZIP 的来源结构、定位信息和失败证据。

**Files:**

- Create: `backend/shared/kernel/textbook/document.go`
- Create: `backend/services/parser-service/domain/parse/structured_service.go`
- Create: `backend/shared/platform/parser/markdown_parser.go`
- Create: `backend/shared/platform/parser/text_parser.go`
- Create: `backend/shared/platform/parser/pdf_parser.go`
- Create: `backend/shared/platform/parser/docx_parser.go`
- Modify: `backend/shared/platform/parser/doc_parser.go`
- Modify: `backend/shared/platform/parser/archive_parser.go`
- Modify: `backend/services/parser-service/domain/parse/service.go`
- Create: `backend/shared/platform/postgres/migrations/00003_source_documents.sql`
- Test: `backend/shared/platform/parser/testdata/`

**步骤：**

- [x] 本地 PDF、Markdown、TXT、DOCX、ZIP 资料通过冻结的 parse task 导入；原始文件以内容寻址的本地 artifact 保存，任务仅携带 hash token，HTTP 使用 multipart 流式上传，单文件上限为 888 MiB。解析 worker 只返回 `SourceDocument + SourceChunk[]`，core-api 重新验证任务、来源、快照与定位后在事务内落库；ZIP/DOCX 另有独立的解压、条目数和单条资源预算，需一次性构建结构化索引的文本/可引用 PDF 输出限制为 64 MiB，超出时保留原始资料并提示拆分。相同来源字节重试复用 artifact，非法类型、扩展名或超上限文件被拒绝。
- [x] Git repository 主资料只接受 40 位固定 commit SHA；一次真实 Gin `routergroup.go` 导入证明 commit 版本、原始字节 hash、`text/x-go` 文档及逐行 code chunk 会经 parse task 后独立落入 `SourceSnapshot` 与来源库。开发认证在空本地数据库会与 workspace bridge owner 对齐，任务轮询不放宽跨身份访问。
- [x] 返回 `SourceDocument` 和 `SourceChunk[]`，保留 source ID、路径/URL、heading path、页码、段落、代码语言、字节范围和 hash；PDF 以页码而非虚假行号定位。
- [x] Markdown 保留 heading、code fence、link；TXT 以可解释段落切分；PDF 保留页码，空页不产生证据，乱码/扫描或无文本 PDF fail-closed 并要求 OCR 后重试。
- [x] DOCX 不再用简单 strip XML tags；先做解析器选型 spike，验证标题、列表、表格、代码样式、图片 alt/说明和顺序。
- [x] ZIP 设压缩前后大小、文件数、单文件大小、压缩比与嵌套压缩包上限；首期使用同步总耗时边界（调用上下文取消）而非伪造一个不会执行的计时器。
- [x] 拒绝路径穿越、绝对路径、symlink、加密包和 ZIP bomb；每个 entry 单独报告 keep/skip/fail，并以独立 `StructuredDocument` 保留压缩包内路径与定位。
- [x] 文档中的“忽略系统指令”等内容永远作为不可信资料，不进入系统/开发者提示层。
- [x] 低质量解析 fail-closed，提供“换格式/OCR 后重试”的恢复建议；首期不内置 OCR。PDF 页级结构已通过解析和 ZIP 回归测试，可进入结构化教材证据链。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./shared/platform/parser ./services/parser-service/domain/parse -run 'Structured|PDF|DOCX|Archive|Bomb|Traversal' -v
```

**退出条件**：任意 claim 能回到具体页面、标题或压缩包内路径；没有仅凭合并长字符串生成教材的路径。

**提交**：`refactor(parser): preserve document structure and provenance`

### PR-08：建立资料清单、官方确认与本地可解释检索

**目标**：按章节只送入最相关、可说明来源的资料，控制 Token 并避免整库全文提示。

**Files:**

- Create: `backend/services/llm-stream/app/textbook/source_catalog.go`
- Create: `backend/services/llm-stream/app/textbook/retrieval.go`
- Create: `backend/services/llm-stream/app/textbook/retrieval_test.go`
- Create: `backend/services/llm-stream/app/textbook/evidence_pack.go`
- Create: `backend/services/llm-stream/app/textbook/evidence_pack_test.go`
- Create: `backend/shared/platform/postgres/migrations/00004_source_chunks.sql`
- Create: `frontend/src/features/source-library/SourceManifestPanel.tsx`
- Create: `frontend/src/features/source-library/OfficialSourceApproval.tsx`

**步骤：**

- [x] 工作台展示主资料与已确认官方补充；来源角色、官方确认与用途由 core-api 保存，未确认来源不能进入教材证据集。当前产品规则不登记未知/非官方来源。
- [x] 用 heading、路径、关键词和主资料优先级进行确定性召回与重排；最多读取 500 个本地片段、返回 8 个候选，不调用模型或向浏览器发送全文。
- [x] 每次检索保存 query、候选、分数理由、最终 evidence IDs 和 input hash；相同项目、资料状态与查询复用同一检索记录。
- [x] 生成任务冻结前对人工映射的 evidence pack 施加 30,000 个源文本字符硬上限；超限返回可行动错误，绝不静默截断或发送整库。精确 provider token 估算、项目预算和费用统计仍属于 PR-11。
- [x] 主资料对项目事实有最高优先级；官方补充只用于概念和 API；冲突不得自动融合，必须生成冲突项。
- [x] 章节 evidence pack 达不到关键 claim 覆盖要求时停止生成并列出缺失资料：蓝图草稿可保存，但批准和任务冻结均要求作者为每章填写关键事实，并仅绑定本章已选 evidence；缺少或越界即返回 `CLAIM_COVERAGE_INCOMPLETE`。作者可按每条关键事实检索已固定的主资料/官方资料，查看候选片段、来源和理由；若无匹配结果，界面明确要求补充合规资料后重试，且不会自动选择或绑定 evidence。
- [x] 为未来 embedding 定义 `Retriever` port，但 Gin 评测证明关键词检索不足前不引入向量数据库。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/textbook -run 'Catalog|Retrieval|EvidencePack' -v
```

**退出条件**：相同快照、合同和 query 得到相同 evidence pack；缓存命中不调用模型；UI 可解释“为何引用这段”。

**提交**：`feat(textbook): add explainable evidence retrieval`

## 8. M3：教材母稿生产线

### PR-09：实现 BookContract、StyleSheet、蓝图与样章双审批

**目标**：在批量生成前锁定读者、教学结构、语言风格和代表性样章。

**Files:**

- Create: `backend/services/core-api/domain/textbook/approval_service.go`
- Create: `backend/services/llm-stream/app/textbook/book_contract.go`
- Create: `backend/services/llm-stream/app/textbook/style_probe.go`
- Create: `backend/services/llm-stream/app/textbook/blueprint.go`
- Create: `frontend/src/features/generation/BookContractEditor.tsx`
- Create: `frontend/src/features/generation/StyleSheetEditor.tsx`
- Create: `frontend/src/features/generation/ApprovalWorkspace.tsx`
- Modify: `frontend/src/components/project-course/BlueprintWorkspace.tsx`

**步骤：**

- [x] BookContract 明确读者已有知识、不可假设知识、学习成果、章节类型覆盖与出版目标；工具策略和代码验证仍由 StyleSheet/章节工件约束。
- [x] StyleSheet 以不可变 revision 固定中文语言、术语、代码、视觉、引用和禁用表达规则；更细的操作步骤、截图和录制字段在 PR-13 扩展。
- [x] 蓝图是 revision；服务端只允许绑定当前已批准的 BookContract/StyleSheet，只有批准版可冻结样章生成任务。
- [x] 样章集合覆盖主要 ChapterProfile，而不是只选最容易写的一章；风格探针同时覆盖概念解释和实操步骤：进度与章节阶段按当前批准 BookContract 的画像集合检查已批准样章覆盖，缺失画像时不开放后续章节阶段；样章质量报告分别记录概念解释与冷启动实操探针结果。
- [x] 样章质量报告和 diff 必须人工批准；合同任一关键 revision 改变使下游蓝图批准失效，并拒绝使用旧合同产生候选稿。跨阶段缓存失效仍由 PR-11 补齐。
- [x] UI 明确区分“自动检查通过”和“我已人工审阅并批准”；批准入口要求八维逐项 ≥ 3/4 和 8–2000 字符审阅说明，服务端以 `inkwords.sample-human-review.v1` 连同候选稿哈希、质量合同和本地 workspace 原子持久化，缺失或低分均 fail-closed。

**验证**：domain 状态机测试 + 两标签页 CAS 测试 + Playwright 双审批流程。

**退出条件**：没有样章批准就无法批量生成；变更合同后旧样章不能被误当作当前批准版。

**提交**：`feat(textbook): gate generation with blueprint and style approval`

### PR-10：实现章节候选稿、自然学习曲线与质量门禁

**目标**：把“通俗但专业、由浅入深、自顶向下”变成可执行合同和质量证据。

**Files:**

- Create: `backend/services/llm-stream/app/textbook/chapter_profiles.go`
- Create: `backend/services/llm-stream/app/textbook/learning_arc.go`
- Create: `backend/services/llm-stream/app/textbook/plain_language.go`
- Create: `backend/services/llm-stream/app/textbook/scenario_validator.go`
- Create: `backend/services/llm-stream/app/textbook/understanding_chain.go`
- Create: `backend/services/llm-stream/app/textbook/simplification_ledger.go`
- Create: `backend/services/llm-stream/app/textbook/chapter_pipeline.go`
- Create: `backend/services/llm-stream/app/textbook/chapter_pipeline_test.go`
- Create: `frontend/src/features/manuscript/CandidateDiffPanel.tsx`
- Create: `frontend/src/features/manuscript/QualityPanel.tsx`

**硬门禁：**

- [x] 关键事实无证据、代码来源不明、运行结果伪造、类比与机制矛盾、批准内容被覆盖。
- [x] Gin 样章门禁拒绝缺少工具/版本、位置或路径、预期结果或恢复动作的冷启动步骤；权限前提由具体技术栈 Runbook 在 PR-13 补充，且步骤完整不等于真实运行已验证。
- [x] 术语在定义前被用于解释自身，或连续缩写导致目标读者无法进入；质量门禁以保守的行级循环定义检测和重复缩写首现定义检测拒绝候选稿，代码块与正常后文提及不参与误判。
- [x] Gin 样章门禁拒绝 worked example 后直接跳独立综合任务：十个自然学习阶段必须按顺序出现且有实质正文，引导练习必须保留提示、脚手架渐退必须说明撤提示、独立迁移必须要求独立工作。面向任意 ChapterProfile 的结构化候选稿管线仍在本 PR 后续扩展。
- [x] 早期简化与后文事实冲突且未登记在 simplification ledger。

**软门禁与人工 rubric：**

- [x] 句子和段落负担、标题承诺、重复、语气、例子相关性、章节节奏、图示机会、练习梯度以固定人工审阅维度写入质量报告；确定性检测器为可定位风险附证据摘录和审阅问题。
- [x] 检测器只指出风险和证据，不计分、不改变硬门禁 `passed`，也不宣称能自动判断教材优劣；候选稿 UI 支持完成八维 ≥ 3/4 的人工量表与审阅说明后批准、继续修改或填写 8–2000 字符理由驳回。批准与驳回记录都绑定候选稿内容哈希、当前质量合同版本和本地审阅 workspace，决定不可改写；读者试学仍是外部验收项。

**候选稿规则：**

- [x] 每次生成创建 immutable candidate revision，保存 parent、contract hashes、evidence hash、prompt hash、provider usage 和质量报告。
- [x] 应用候选时做三方 diff；目标 revision 已变化则返回冲突，不覆盖。
- [x] 已批准母稿按需生成 BlogProjection、VideoRunbookProjection、LearningProjection；服务端仅接受当前 approved revision，投影不持久化为第二份正文，解析失败只返回错误且不影响母稿。前端视频教案也只展示批准稿的投影。
- [x] 按章重试只重跑失败阶段；任务以 `task_id + stage + input_hash` 幂等：样章候选生成将完成结果写入 `textbook_sample_phase` 检查点，重投递先按阶段键恢复该结果，已完成阶段不再调用模型；尚未完成的阶段才继续执行。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/textbook ./services/core-api/domain/textbook -run 'Chapter|LearningArc|PlainLanguage|Scenario|Candidate|Lock' -v
cd frontend && npm test -- --run
```

**退出条件**：人工编辑零覆盖事故；三种读者版本在先验、解释深度、练习脚手架上有可审计差异，而非只改提示词中的标签。

**提交**：`feat(textbook): generate evidence-bound self-study revisions`

### PR-11：抽象 Provider、Token 预算、缓存和可恢复任务

**目标**：支持用户自填 DeepSeek/OpenAI Key，并使成本、上下文和失败可见可控。

**Files:**

- Create: `backend/shared/kernel/generation/port.go`
- Create: `backend/shared/kernel/generation/usage.go`
- Create: `backend/shared/platform/llm/deepseek_adapter.go`
- Create: `backend/shared/platform/llm/openai_adapter.go`
- Create: `backend/services/llm-stream/app/textbook/model_policy.go`
- Create: `backend/services/llm-stream/app/textbook/token_budget.go`
- Create: `backend/services/llm-stream/app/textbook/generation_cache.go`
- Modify: `backend/shared/platform/llm/deepseek.go`
- Modify: `backend/services/llm-stream/app/bootstrap/bootstrap.go`
- Create: `frontend/src/features/settings/ModelProviderSettings.tsx`
- Create: `frontend/src/features/generation/UsagePanel.tsx`
- Create: `docs/decisions/llm-provider-and-secret-storage.md`

**步骤：**

- [x] `GenerationPort` 统一 structured output、stream、usage、finish reason、request ID、timeout、cancel 和 provider error。
- [x] 业务层不出现 DeepSeek/OpenAI SDK 类型；fake provider 是普通 CI 默认实现。
- [x] API Key 仅保存在本机安全存储或由进程环境注入；数据库只存 provider 配置引用和脱敏尾号，不回传完整 Key。
- [x] 连接测试由用户显式触发；日志、SSE、错误和导出物均不得包含 Key。
- [x] TaskModelPolicy 按任务复杂度选模型；高成本模型只用于蓝图、核心概念和审校，不用于确定性解析。
- [x] 生成前估算 token；超过项目/章节预算先压缩 evidence 或要求确认，不能静默截断关键资料。
- [x] 缓存键至少包含 source snapshot、audience、BookContract、StyleSheet、stage、evidence、prompt schema、provider/model。
  - 2026-09-05：补齐完整请求哈希与质量合同版本，章节蓝图、合同 revision、输出 schema/预算变化均失效；只缓存通过候选校验的响应。回归先复现失败响应污染缓存及跨章节复用，再证明显式后续调用可重新到达 Provider、成功结果仍可复用，无自动重试。
- [x] 保存 input/output/cache token、调用数、耗时和可选估算费用；供应商不返回 usage 时标 unknown，不猜数。
  - 2026-09-05：质量拒绝结果的 Provider、门禁版本、失败项与用量已在任务面板显示；缺失的单字段（含缓存 Token、耗时、本地缓存状态）独立显示未知。真实既有失败任务的浏览器只读核对已记录在计划审计中。
- [x] 合同测试覆盖两个 adapter 的同构结果、限流、超时、无 usage、截断、取消和无效 JSON。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./shared/kernel/generation ./shared/platform/llm ./services/llm-stream/app/textbook -run 'Provider|Usage|Budget|Cache|Resume' -v
```

真实 DeepSeek/OpenAI 测试只在 opt-in CI 或本机显式运行，使用很小的固定预算。

**退出条件**：相同离线夹具可由 fake provider 稳定运行；更换 provider 不改变领域合同；缓存节省量可解释。

**提交**：`refactor(llm): add provider-neutral generation pipeline`

## 9. M4：代码运行、截图和视频操作教案

### PR-12：选择并实现 fail-closed 隔离执行器

**目标**：对系统生成的教学工件提供真实运行证据，不通过提升容器权限掩盖隔离问题。

**Files:**

- Create: `docs/decisions/textbook-sandbox-selection.md`
- Modify: `backend/services/course-runner/domain/verification/model.go`
- Modify: `backend/services/course-runner/domain/verification/runner.go`
- Modify: `backend/services/course-runner/domain/verification/bubblewrap_executor.go`
- Modify: `backend/services/course-runner/domain/verification/consumer.go`
- Modify: `backend/services/course-runner/Dockerfile`
- Modify: `docker-compose.yml`
- Create: `backend/shared/platform/postgres/migrations/00010_textbook_runtime_evidence.sql`（实际序号跟随已落地的 core migration；涵盖 CodeArtifact、VerificationResult 与 Asset 表）

**Spike 必测方案：**

- Linux 主机原生 bubblewrap；
- 独立受限容器/容器运行器；
- macOS 本机的隔离替代方案；
- 无可用隔离时明确禁用验证。

**安全合同：**

- [x] 只运行 InkWords 生成且 hash 匹配的 CodeArtifact，绝不运行目标仓库。
- [x] 默认禁网、非 root、只读根文件系统、临时工作目录、CPU/内存/PID/文件大小/时间限制：bubblewrap 为实验创建独立网络 namespace、只读系统/工件挂载与受限 tmpfs，Compose 服务为非 root、只读根、无额外 capability 并限制 CPU/内存/PID。
- [x] 命令来自受支持技术栈模板，不接受任意 shell 字符串；stdout/stderr 有大小上限和脱敏。
- [x] 验证结果绑定 artifact hash、runner image digest、toolchain version 和 command manifest。
- [x] runner 不可用、隔离失败或证据过期时一律为 `unverified`；生成链路不得自行改成通过。

**验证**：单元测试、受控恶意 fixture、Compose 集成；记录 namespace、网络、资源限制和文件逃逸实测证据。

**退出条件**：Go 最小教材工件能在支持的本机环境真实通过；否则功能保持关闭且 UI 准确说明原因。

**提交**：`feat(runner): verify generated teaching artifacts safely`

### PR-13：生成运行证据、视觉资产与视频 Runbook

**目标**：让教材展示真实结果，让视频教案能直接放在副屏指导录制。

**Files:**

- Create: `backend/shared/kernel/textbook/asset.go`
- Create: `backend/shared/kernel/textbook/runbook.go`
- Create: `backend/services/llm-stream/app/textbook/tool_recommendation.go`
- Create: `backend/services/llm-stream/app/textbook/video_runbook.go`
- Create: `backend/services/export-service/domain/export/asset_manifest.go`
- Create: `frontend/src/features/verification/VerificationPanel.tsx`
- Create: `frontend/src/features/manuscript/VideoRunbookPanel.tsx`
- Create: `frontend/e2e/textbook-runtime-evidence.spec.ts`
- Extend: `backend/shared/platform/postgres/migrations/00010_textbook_runtime_evidence.sql`（Runbook 投影仍保存在不可变 revision document JSON；运行证据和 Asset 使用关联表）

**步骤：**

- [x] ToolRecommendation 根据技术栈与观察目标选择 Visual Studio 2022、GoLand 或浏览器开发者工具，并给出备选和理由；VS Code 仍作为轻量备选，不伪造 IDE 自动化能力。
- [x] RunbookStep 已包含工具版本、起始状态、精确操作、快捷键/菜单、输入、预期画面、旁白、截图点、失败恢复和完成信号；脚本会针对调用链、内存/资源采样、网络或性能面板调整观察步骤。候选稿将不可变 VideoRunbookProjection 写入 revision document JSON，章节工作台可只读显示副屏录制步骤与人工采集清单；未采集前固定标记为 `unverified`，不声称 IDE 自动化或视觉观察已经验证。
- [x] 自动证据合同与受控执行链覆盖终端输出和浏览器页面：终端保留脱敏结构化命令结果，浏览器保留本地 URL/最终 URL、DOM 断言、console、网络摘要、内容寻址截图及实际 Playwright/Chromium 版本；自动截图不会再标成“人工截图”。IDE 步骤仍只使用人工 capture checklist，不声称自动控制未接入的 IDE。当前 Docker Desktop 因不支持所需 user namespace 而保持功能关闭，所以这项实现完成不等于 Gin 样章已有真实运行证据。
- [x] 运行结果优先保存为结构化文本；只有布局、内存图、调用栈等视觉信息确有价值时才要求截图：运行记录保留结构化输出；新截图上传必须选定布局、内存图、调用栈、网络流或实际界面状态之一，旧截图只读标注为未分类，工作台和导出记录均展示该用途。
- [x] 资源占用、内存布局和调用链的结论必须绑定实际工具、版本、采样条件和原始证据，区分观察与解释：`RuntimeEvidence` 对已验证结果强制工具/版本、采样条件、原始引用及 `inkwords.runtime-observation.v1` 结构化观察；解释必须声明为来源说明或有限推断并指向观察字段。课程运行器仅记录直接观察，工作台明确显示观察与解释的边界。
- [x] 任何代码/合同/工具版本变化使相关 VerificationResult 和 Asset 过期：读取层根据不可变 manifest、当前已批准合同、运行器与验证输入派生失效原因；工作台和导出预检不再把带失效原因的证据或关联 Asset 当作当前事实。
- [x] 所有图片有 alt text、来源、权利状态、生成方式和在母稿中的稳定引用。

**退出条件**：Gin 样章至少包含一份真实终端或浏览器证据、一份 IDE 人工截图清单和一套可跟录的视频操作教案。

**提交**：`feat(textbook): derive runtime evidence and video runbooks`

## 10. M5：学习、复述与自适应复习

### PR-14：用六维表现和 FSRS 驱动到期任务

**目标**：让学习结果改变后续任务，而不只是给文章加“复习”按钮。

**Files:**

- Create: `backend/shared/kernel/textbook/mastery.go`
- Create: `backend/services/review-service/domain/mastery/objective.go`
- Create: `backend/services/review-service/domain/mastery/attempt.go`
- Create: `backend/services/review-service/domain/mastery/scoring.go`
- Create: `backend/services/review-service/domain/mastery/scheduler.go`
- Create: `backend/services/review-service/domain/mastery/*_test.go`
- Create: `backend/shared/platform/postgres/migrations/00007_mastery.sql`
- Modify: `backend/services/review-service/domain/review/picker.go`
- Modify: `backend/services/review-service/domain/review/model.go`
- Create: `frontend/src/features/learning/TodayQueue.tsx`
- Create: `frontend/src/features/learning/MasterySession.tsx`
- Modify: `frontend/src/App.tsx`
- Create: `docs/decisions/fsrs-mastery-mapping.md`

**六种任务：**

```text
explain      用自己的话复述原理
complete     补全关键代码
reproduce    从零实现最小版本
transfer     修改需求并迁移实现
diagnose     定位并修复故障
retain       若干天后延迟回忆与综合测试
```

**步骤：**

- [x] LearningObjective 绑定章节、目标行为、rubric、关键点、先修和证据引用；MasteryAttempt 只追加不覆盖，目标与尝试均由 review-service 的版本化 schema 持久化。
- [x] 领域层评分综合正确性、独立性、提示次数、耗时、错误类别和自信度；低自信或超时的答对不会升级为熟练。
- [x] 学习会话展示冻结目标、rubric、先修与来源，关键点提示显式计数；作答文本与自评一同追加保存，重新打开可读回最近 20 次作答。历史无正文记录保持为空，不补造答案。
- [x] 接入基于作答与 rubric 的模型逐项评分，给出答对、遗漏、误解、下一步提示和补救材料，并支持可追溯的用户纠正；当前自评不得替代该要求。
  - 2026-09-10 闭环验收：显式 local-evaluation-v2 绑定评分项/来源 ID 词表，旧 v1 指纹保持；正式 programming r2 两份相同冻结输入真实重评分成功，运行项绑定原真实收据。页面完成一条分数/原文纠正、一条源码机制建议纠正及两次显式应用；CAS 409、同 ID 幂等、原作答不变、新浏览器恢复且无模型/运行启动请求通过。全部 review-service/PostgreSQL/架构回归通过，新配置已部署，生产原状态不变。勾选此功能项；不计个人六维、真人试学或整书出版通过。见 `docs/qa/approved-practice-scoring-closure-2026-09-10.md`。
  - 2026-09-10 最新：真实页面从正式 programming r2 创建目标并保存两次 AI 验收作答，修复 practice-evidence 不识别批准引用别名的问题并部署 core practice-alias-v1。三份批准稿 18 题来源/哈希和五包 Go/PostgreSQL/架构通过。独立 DB 的代码快照真实隔离运行成功，模型输入绑定同一收据；两次真实 DeepSeek 调用分别因原文引用/证据编号被拒绝，0 纠正/应用，失败应用 409。生产原状态完全不变，临时 DB/runner 已清理。评分可靠性及实际纠正应用仍未通过，保持未勾选；见 `docs/qa/approved-practice-assessment-2026-09-10.md`。
  - 2026-09-10：生产验证路由/HTTP 客户端、应用、PostgreSQL Store 与原 Bubblewrap 在独立评测环境打通：正确/错误两例真实执行，重复 POST 同任务，领取重放 404，原输入/收据被评分层准确读回；每作答一条任务。相关七包/架构通过，临时数据库已删除，runner 已停止，主状态不变，无 Provider 或生产写入。输入仍为操作方夹具，批准练习与个人纠正应用尚待验收；保持未勾选。见 `docs/qa/learner-http-roundtrip-2026-09-10.md`。
  - 2026-09-10：补齐运行评分前的 UI 恢复缺口：历史读取失败不再误报未运行，独立时限、旧作答清理、串行轮询与取消竞态保护、丢响应同 request_id 重试、离线保留历史。前端 299 项、8 条 Chromium、真实浏览器故障恢复与零写入验证通过，已部署且实际资源 hash 一致。未调用 Provider 或写学习记录，生产完整链路仍待验收。见 `docs/qa/learner-verification-recovery-2026-09-10.md`。
  - 2026-09-10：两例操作方 Go 样本完成真实 v2 沙箱运行；首次运行评分缺代码引文被拒绝，补充受测代码/运行收据双绑定说明后，同输入收据的新评分正确 4/4、错误 0/0。共 3 次模型调用，原失败保留。前端 v1/v2 能力错配已修复；前端及评分服务部署完成，291 项前端、8 条 Chromium、相关 Go 全包/架构、网关与原状态一致性通过。未写个人记录，生产 claim/store 与实际纠正应用/学习验收仍待完成，保持未勾选。见 `docs/qa/learner-runtime-assessment-2026-09-10.md`。
  - 2026-09-06：已部署显式本地评分配置 local-evaluation-v1：文字 low/6000，代码关闭推理/6000，60 秒适配器；旧合同保持。预览请求与真实评测样本 hash 一致，Go/架构、顺序构建、网关与实际工作台检查通过，真实知识库与沙箱限制保持。无模型调用或学习记录写入。生产配置接入已完成；同作答真实运行评分、用户应用与读者验收仍缺，保持未勾选。见 `docs/qa/mastery-profile-deployment-2026-09-06.md`。
  - 2026-09-06：代码 v6 候选使用版本化行号范围从冻结文件解析引用，解决实际捕获的代码压平问题；未知且缺少运行证据的代码项可用空引用数组，不伪造来源。最终两次真实静态评分区分正确/错误实现（4/4/4 与 1/1/2），运行/测试未知。两次实际浏览器纠正、PostgreSQL 保存/纠正/应用、Go/架构及 274 项前端测试、lint/build/预算通过。本轮共 7 次 Provider 调用，失败原文/用量均按实际可得范围保存，未部署。仍待运行评分、生产配置与真实学习应用验收；见 `docs/qa/mastery-code-span-2026-09-06.md`。
  - 2026-09-06：三份真实反馈经生产组件与独立本机 HTTP 服务完成 5 次浏览器纠正，刷新读回、有效 hash 与原文保留通过；非生产数据库或人工学习记录。代码评分两批共 2 次真实调用均在正确实现首例失败，错误实现未发送；第二批定位到 correctness 引文门禁。已修复适配器超时分类，聚焦与全量 Go/架构通过，未放宽时限或引用。候选未部署，仍待代码及运行评分、生产纠正应用链路验收。详见 `docs/qa/mastery-code-grading-2026-09-06.md`。
  - 2026-09-06 最新：修复 v6 空 answer/evidence 元数据与真实作答并存的歧义；最终 low + 显式 6000 输出上限的四例真实评分通过数值门槛（平均 4.0/3.6/0.2/0，因果完整/流程为 4/2）。语义复核发现三处需纠正的建议、分数或理由，提案已通过领域重放与原文/过期 hash 保护，未应用到学习记录。过窄的引文片段枚举实验已撤回并归档。全量 Go/架构及实际 PostgreSQL 回归通过；尚待真实反馈前端纠正、代码评分代表性验收及部署，保持未完成。详见 `docs/qa/mastery-answer-binding-2026-09-06.md`。
  - 2026-09-06 最新：v6 原配置两次调用仍有因果过评分；high/low 推理各一次分别未得到有效反馈、被合同拒绝，各批失败后停止。新增任务级 low/high 选项，绑定请求哈希/预览/结果，保持旧默认；失败保留 Token 用量，后续可取得细分反馈拒绝码。全量 Go/架构回归通过，共 4 次真实调用，未部署或写入学习记录。详见 `docs/qa/mastery-reasoning-options-2026-09-06.md`；保持未完成，下一步先定位具体合同拒绝，不直接放松门禁或上线推理。
  - 2026-09-06：v5 单样本诊断保持原请求身份，分数门槛通过但独立理由/遗漏仍与判断偏离。v6 候选改为 judgments 唯一评分依据，派生说明和答对/遗漏/误解，重放验证并保留原判断与纠正。全量 Go/真实临时 PostgreSQL、273 项前端、lint、构建/包体、真收据离线投影与实际浏览器通过；v6 真实调用及部署尚未执行。详见 `docs/qa/mastery-canonical-feedback-2026-09-06.md`，本项保持未完成。
  - 2026-09-06：v5 候选新增逐项覆盖判断与冻结要求引用，检查判断/分数/缺口的一致性，持久化和读回重验，纠正保留原模型判断；界面可展开原判断及参考答案。全量 Go/真实 PostgreSQL、273 项前端、lint、构建/包体与实际浏览器通过。真实对照第二例被合同拒绝，后两例停止；已补后续有界拒绝诊断和未知分数不写零，未额外调用或部署。详见 `docs/qa/mastery-decision-coverage-2026-09-06.md`，不能把拒绝错误评分等同完整可用的语义验收。
  - 2026-09-06：补齐冻结 ExpectedAnswer 的 v4 输入/哈希/预算绑定，旧 v1/v2/v3 保持；聚焦与全量 Go、真实临时 PostgreSQL 通过。两次真实 DeepSeek 对照在流程答案因果 3 分处失败，后两例停止，候选实现未部署。已用实际浏览器对真实反馈执行受控纠正并保留原分数/引文/历史；不是人工审阅或真实学习库应用。详见 `docs/qa/mastery-reference-binding-2026-09-06.md`；语义验收仍未完成，不能随其他工作一起部署候选请求。
  - 2026-09-06：反馈纠正从 criteria 扩展至答对/遗漏/误解/提示/补救完整快照；支持清除不成立的反馈，原模型结果保持。真实 PostgreSQL 验证追加、幂等、过期 hash 和非法引用拒绝；全量 Go、前端 271 项测试、lint 和合成数据的实际浏览器编辑流程通过。前端构建与包体预算通过，review-service/前端均已部署 healthy，网关正常。详见 `docs/qa/mastery-findings-correction-2026-09-06.md`；不据此关闭语义校准、真实学习者与应用验收。
  - 2026-09-06 引用纠正补齐：逐项查看来源原文、在冻结依据中重选纠正引用并保留原模型引用已实现；运行数字评分须实际选择运行证据，失败保留草稿。真实临时 PostgreSQL 验证引用替换读回和旧引用保留，前端 269 项、lint、Docker 构建及部署前端的受控浏览器流程通过。无新增 Provider 调用或真实学习库写入。详见 `docs/qa/mastery-correction-evidence-2026-09-06.md`；模型理由语义校准、实际学习者纠正/应用和延迟试学仍缺，保持未完成。
  - 2026-09-06 后续评分范围实验：两批共 6 次真实调用保持四份原输入不变；v3 第二例因果过评分失败后停止，v4 分数门槛通过但完整性理由仍增加题目未要求的细节。候选指令未部署且最终从生产代码撤回，实验源码、原失败结果和自动语义复核均保留。留下验收入口改进：产物新增待语义复核状态，流程样本核对其余各项最低分，离线预检与实际输入一致。Go/架构、聚焦和 race 通过；不能据此宣称理由可靠。下一步需建立冻结要求到逐项判断的可核对覆盖关系，并验收真实纠正/应用，保持未完成。
  - 2026-09-06 后续校准：四次真实 DeepSeek 调用区分完整因果解释与正确流程复述，因果分为 4/2，另两例遗漏/误解保持低分；总输入 8006、输出 3763 Token。指令和四例门槛已更新，review-service 已部署，全量 Go 复查及 race 通过。逐项自动复核仍发现流程答案的完整性/清晰度理由新增题目未要求的细节，不能宣称整体评分已校准。详情见 `docs/qa/mastery-causal-calibration-2026-09-06.md`；真实学习者、代码评分、纠正/应用及延迟试学仍缺，保持未完成。
  - 2026-09-06：用户授权后的三份文字作答真实 DeepSeek 评分已执行并通过既定门槛：完整答案 4.0/4、遗漏答案 0.6/4（3 项遗漏）、误解答案 0/4（3 项遗漏、2 项误解）；共输入 4,620、输出 2,659 Token。逐项 JSON 见 `output/real-acceptance/2026-09-06/mastery/`。自动复核发现完整样本的因果评分把流程顺序当成“为何”的解释，仍需校准；本批次没有真实学习者作答、代码运行评分、用户纠正/应用和读者验收，因此保持未完成。当前不缺 Provider 授权。
  - 2026-09-05：已接入权威作答/题目/来源输入、单次调用适配器、v30 评分任务持久化与恢复、取消/显式重试、纠正 CAS 事务，以及逐项反馈、原始结果和纠正读回 UI。v31 追加明确应用的评分快照并事务性重放调度，后续作答也保留该解释；未知不当作失败或掌握证据。预检、纠正和应用均不调用模型，失联/重启不自动重跑，同请求只保存一次。尚缺学习者工件验证接入和代表性真实评分验收，保持未完成。详见 `docs/decisions/fsrs-mastery-mapping.md`。
  - v32 已将学习者 Go 源文件随首次作答原子冻结，绑定题目/会话/时间与哈希，支持幂等提交、丢失响应恢复及历史按需读回。代码执行和文件评分尚未接通，含代码作答暂拒绝模型评分，避免仅评文字。隔离 PostgreSQL 验证回滚、所有权、原字节与索引；浏览器验证真实文件选择及刷新读回。此进展不改变本项未完成状态。
  - 随后已接通评分 v2：权威文件快照进入受预算限制的模型输入，逐项引文绑定文件路径，支持原文纠正、应用和刷新恢复；保留 v1 输入及反馈 hash。含代码作答已解除临时评分拒绝，但仅作静态评阅，运行/测试/修复验证仍为未知。真实隔离 PostgreSQL 的作答→评分→纠正→应用通过；执行清单准入、学习者 VerificationRun 与真实评分验收仍缺。
  - 继续核对发现 `AGENTS.md:105` 和设计 10.1 只允许运行生成教学工件。学习者文件的执行属于来源范围扩展，具体方案见 `docs/decisions/learner-code-execution-scope.md`。原作答/题目/文件/运行器绑定的纯输入合同与服务端只读组装已实现；未接入路由、队列或执行器，启用范围待用户明确确认，本机隔离限制也仍需满足。
  - 用户已于 2026-09-05 明确接受该来源范围。review schema v33 现保存显式启动、幂等、取消、重试和中断终态；course-runner 只接收一次性冻结引用，重新向所有者解析原文件并用固定离线 `go test` 策略组装临时树。前端预览不会启动执行，只有明确点击才创建任务。评分 v3 只读取同一作答最新的 passed/failed/timed_out 报告，并在调用前展示运行 ID/状态；取消、运行器错误、不可用与中断不成为评分证据。全量回归、PostgreSQL、race、Compose 和真实浏览器网关通过；当前 Docker Desktop 的非特权 namespace 预检仍失败，开关保持关闭，未执行学习者代码。代表性真实模型评分与读者验收仍缺，因此本项保持未完成。
  - 2026-09-06：确认旧预检失败来自 Docker 默认 seccomp，而非 user namespace 完全不可用。course-runner 现使用受审外层 seccomp 让 Bubblewrap 建立 namespace，并在工件进程前装入内层 BPF 重新拒绝 mount/namespace 调用；容器仍保持非特权、只读、cap-drop all 和 no-new-privileges。固定 Go profile 升为 v2，并按 Go 1.26 实测把编译资源固定为 384 MiB/64 PID/64 MiB 单文件/30 秒 CPU。启动时生产执行器会编译运行操作方固定测试并验证嵌套 user namespace 被 EPERM 拒绝，外层 profile 由服务从只读挂载实际计算摘要并纳入运行器身份。新版 course-runner healthy，能力接口为 accepted=true/available=true；本地开关已显式启用。尚未执行学习者作答或产生 VerificationRun，代表性真实评分与读者验收仍缺，因此本项保持未完成。
  - 2026-09-06：补齐此前名存实亡的真实评分验收入口。受保护的 `real-acceptance` 工作流现在固定模型，并对完整答案、遗漏答案和误解答案各执行一次真实逐项评分；门禁核对证据合同、已知 Token 遥测、聚合分数、遗漏和误解，任一样本失败即停止尚未开始的调用，日志不保存密钥或原始响应。默认 Go 测试只验证三份冻结输入且明确跳过联网测试；尚未取得新的 Provider 授权并执行这 3 次调用，因此本项仍不勾选。
- [x] 领域层按六维弱项确定下一题：transfer/diagnose 的失败会优先重练对应维度，提示依赖安排低脚手架练习。
- [x] 将六维下一题策略映射到具体题目、变式和由弱到强的分层提示。
  - 2026-09-05：母稿六维 PracticeSet、候选质量 v9、批准稿 v2 投影与按调度 mode 展示题目/变式/三层提示均已实现。服务端冻结批准 revision，作答绑定任务 ID/hash，服务器约束保持间隔。v29 持久化练习会话与帮助披露，刷新恢复辅助状态；相同提交重试/并发只产生一条作答，丢失响应可恢复原结果，已完成会话不能改写。PostgreSQL、race、前端组件和浏览器恢复流程通过。此勾选只表示功能实现；新合同的真实生成和真实读者验收仍列于 PR-16。
- [x] 领域层评为掌握必须至少通过 explain、独立 reproduce/transfer、diagnose 和一次间隔至少 24 小时的 retain；当日即时成功不算长期掌握。
  - 2026-09-05：修正此前错误的“记录年龄”判定；现在必须在 retain 发生时已经达到距前次练习的间隔，保存数天不能把即时复述变成延迟证据。真实执行和用户表现仍须单独验收。
- [x] 评估并单独引入 go-fsrs；只让 FSRS 决定复习时间，任务类型仍由六维策略决定。
- [x] FSRS 状态可由 append-only review log 重放；算法或参数升级要有 migration/version。
- [x] 默认教材项目页及兼容首页在各自主动挂载时一次性请求 `/api/v1/mastery/due` 并展示到期任务；路由从稳定本地 workspace 解析身份，未打开应用时不推送、不常驻、不后台联网。
- [x] 旧 Obsidian 笔记复习通过适配器保留，逐步从 `UserID/note_path` 迁到 workspace/objective：原有 review session 和路由不变；显式迁移入口只读取已合格的笔记源，按稳定 `legacy-note:` 身份创建或复用 workspace objective，且只建立 explain/retain 起点，不把旧自由文本误报为六维掌握。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/review-service/domain/mastery ./services/review-service/domain/review -run 'Mastery|FSRS|Replay|Due|Adaptive' -v
cd frontend && npm test -- --run
```

**退出条件**：一次复述、编码或排错表现会确定性改变下一次时间和题型；日志重放得到相同状态；未通过延迟测试不显示“已掌握”。

**提交**：`feat(learning): adapt mastery tasks with fsrs scheduling`

## 11. M6：出版级整书构建与验收

### PR-15：建立 Canonical Book AST、全书审校和多格式导出

**目标**：从同一批准母稿构建可编辑教材，而不是分别拼 Markdown、HTML 和 PDF。

**Files:**

- Create: `backend/shared/kernel/textbook/book_ast.go`
- Create: `backend/services/export-service/domain/export/book_build.go`
- Create: `backend/services/export-service/domain/export/book_ast.go`
- Create: `backend/services/export-service/domain/export/markdown_book.go`
- Create: `backend/services/export-service/domain/export/docx_book.go`
- Create: `backend/services/export-service/domain/export/pdf_book.go`
- Create: `backend/services/export-service/domain/export/publication_preflight.go`
- Create: `backend/services/export-service/domain/export/*_test.go`
- Create: `backend/shared/platform/postgres/migrations/00008_publication.sql`
- Create: `frontend/src/features/publishing/EditorialWorkspace.tsx`
- Create: `frontend/src/features/publishing/BookBuildPanel.tsx`
- Create: `frontend/src/features/publishing/RightsPanel.tsx`
- Create: `assets/publishing/reference.docx`
- Create: `docs/runbooks/textbook-publication-checklist.md`

**审校流水线：**

1. 发展性编辑：目标读者、范围、全书结构和章节职责。
2. 技术审校：事实、代码、输出、证据、版本和边界。
3. 自学性审校：隐藏前提、术语负担、示范与练习梯度、恢复路径。
4. 全书一致性：术语、交叉引用、先修 DAG、贯穿项目代码状态和 simplification ledger。
5. 文字编辑：语法、标点、数字、单位、代码字体、图表题注和参考文献。
6. 排版校对：目录、分页、孤行、代码换行、图片分辨率、alt text 和页眉页脚。
7. 权利与合规预检：许可证、引用、图片、商标、个人信息、ISBN/CIP/出版社待确认字段。
8. 读者试学：目标读者在无口头补充下完成代表章节，记录卡点和修订。

**构建规则：**

- [x] BookBuild 固定 project、approved revision 集合、合同版本、资产 hash、工具版本和构建时间，不可变；导出端只读取 manifest 内嵌的 Canonical Book AST，不会追随随后修改的章节。
- [x] Markdown、DOCX、PDF、ZIP 从同一 AST 生成并共享章节/图片/代码/引用 ID：所有整书投影只接收冻结的 Canonical Book AST；包内保留章节、代码、资产与 manifest 的稳定 ID/哈希。
- [x] DOCX 优先采用 Pandoc `--reference-doc` 保留可编辑样式；引入前验证许可证、版本、中文字体和本地安装体验：已用 Pandoc 3.8 和内置 reference.docx 实际生成 OOXML，并记录其自由软件声明及 Noto CJK 样式声明。
- [x] PDF 由同一母稿构建，保存渲染日志；不能用旧 blog PDF 拼接冒充整书版式：2026-09-04 使用 Playwright Chromium headless shell，在未添加 `--no-sandbox` 的条件下实际渲染 Canonical Book AST；`BookPDFProjection` 保留渲染命令/状态日志和 Chromium 版本。Docker Desktop 容器仍因 namespace 限制保持未配置，逐页版式校对仍是人工门槛。
- [x] ZIP 包含母稿、投影、代码工件、验证摘要、资产、引用、权利清单、质量报告和 manifest hash：整书包写入 Canonical AST、Markdown/DOCX/PDF 投影、运行验证摘要、权利与质量记录、冻结 Build manifest 及逐文件哈希。
- [x] “个人学习导出”明确标为非出版候选；“出版候选”对权利、伪字段、硬门禁、人工审校和版面校对 fail-closed，缺少任一证据时审校包保留阻断项。
- [x] 自动规则与人工审校记录使用不同类型，自动结果不能充当人工同行评审；真实出版社要求以待确认清单呈现。

**验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/export-service/... -run 'BookAST|BookBuild|DOCX|PDF|Preflight|Rights' -v
cd frontend && npm test -- --run
```

另需人工打开 DOCX 和渲染后的 PDF 逐页校对，记录页数、字体替代、断行、图片和代码块问题。

**退出条件**：四种格式内容一致；出版候选没有伪造字段；权利不明或缺人工审校时明确阻断。

**提交**：`feat(export): build publication-ready textbook artifacts`

### PR-16：完整 Gin dogfood、C++ 回归、清理迁移桥梁

**目标**：用真实使用证明产品闭环，再删除已无调用的旧入口和认证代码。

**Files:**

- Create: `docs/qa/gin-textbook-acceptance.md`
- Create: `docs/qa/cpp-official-stack-regression.md`
- Create: `frontend/e2e/textbook-full-flow.spec.ts`
- Modify: `README.md`
- Modify: `frontend/README.md`
- Modify: `backend/services/architecture_test.go`
- Modify/Delete: 仅删除经 `rg`、测试和流量入口证明无调用的 auth/user/legacy project-course bridge 文件
- Create: `backend/shared/platform/postgres/migrations/00009_remove_legacy_identity.sql`

**Gin 完整验收：**

- [x] 真实固定 Gin SHA + 官方资料栈；三种 audience 各生成代表章节。
  - 2026-09-10 完成：stack_familiar 正式项目经五次导入、11 证据冻结、三契约批准，一次真实 DeepSeek 生成后局部零调用修订；独立 Go 隔离验证、delegated_ai 审阅批准 r2、新 19 文件冻结包均有真实证据。即时核对 foundation r13/programming r2/stack_familiar r2 的正式 audience、模型谱系、批准正文与冻结 AST；三份 Gin 引用全为 `73726dc606796a025971fe451f0aa6f1b9b847f6`，已有两项目不变。仅此代表章项完成，不代表无需编辑的一次生成质量、整本书或个人学习/出版验收。见 `docs/qa/stack-familiar-audience-project-2026-09-10.md`。
  - 2026-09-10：正式 programming 项目完成六次真实导入、14 证据冻结和三契约批准；一次 DeepSeek 生成被质量门禁拒绝后，零模型调用修订同步修正正文/代码/练习。r1 教学工件真实 Go 1.26.8 隔离测试通过，delegated_ai 八维审阅批准 r2。原 foundation 项目不变。旧章节 ZIP 空包故障已回归修复并部署 export-service，引用稿改走现有冻结构建；新 19 文件包哈希、母稿/代码字节、来源与收据通过。stack_familiar 正式代表章及其他真实验收仍缺，不勾选整项；见 `docs/qa/programming-audience-project-2026-09-10.md`。
  - 2026-09-10：受限 Fontconfig 集合与构建权利联验已部署；4 哈希文件/22 face、449 正文节点字符检查、前后稳定清单。实际发现并修复 SYSROOT 导致的中文方框，最终 16 页文本/像素一致，字体篡改负例在打印前拒绝。后端 57 包与最终导出/架构回归通过。生产 19 文件包哈希与原母稿/代码一致，PDF 字体阻断解除；rights v5 委托审阅撤下该缺口，正文/代码 2 pending、试学及原计划剩余验收继续，未晋级。见 `docs/qa/closed-pdf-font-profile-2026-09-10.md`。
  - 2026-09-10：正文字体观测和 AST/HTML/PDF 绑定已部署，真实 449 节点/5 字体名称/4 个文件，字体内许可与版本可追溯。4 个字体限定文档用途 ready 实际登记，正文/代码仍 pending；rights v4 需修改。19 文件 ZIP 哈希通过，16 页文本/像素与母稿/代码不变；后端全量 57 包、架构和实际沙箱渲染通过。仍缺打印页脚完整来源约束、正文权利及其它整书验收，见 `docs/qa/pdf-font-provenance-2026-09-10.md`。
  - 2026-09-10：权利范围实审取得固定 Gin MIT 原文并确认字体追溯缺口；追加补证/CAS/原件历史/共享 core-export 解析和 schema 36 已实现部署。实际登记两项 pending、代码补证 v1，201 幂等/409 过期前序通过；rights v3 委托复审保持需修改。18 文件新 ZIP 哈希通过，母稿/代码/PDF 不变。后端全量与架构、301 项前端、lint/build/bundle、Chromium 8 条通过。字体绑定、素材权利落实、真人复审与完整验收继续，见 `docs/qa/publication-rights-audit-2026-09-10.md`。
  - 2026-09-10：质量快照与审校报告汇总已实现部署，真实页面新冻结 `fb19cf7e-b952-4388-90a4-53b57a35acf4`，重复操作幂等；原 v9 报告、四条建议及人工复核要求固定保存。真实 PostgreSQL 与相关模块回归通过；完整架构仍被既有 pkg/jwt 文件阻断。新 ZIP 17 文件哈希通过，章节/代码/投影不变，新构建显式复核八项 delegated_ai，权利需修改、试学未评估，preflight false。未勾选整体验收，见 `docs/qa/frozen-book-quality-2026-09-10.md`。
  - 2026-09-10：已补新构建的冻结运行验证快照，ZIP 给出完整收据及冻结/导出时判断，后来的运行不能升级旧构建。真实 PostgreSQL、相关后端与架构检查通过；core/export 已部署，页面创建 `b2687220-106a-4ddd-bc8d-b31dfd18e157`，重复点击复用。新包 17 文件哈希通过，r13 内容不变，重新显式记录八项委托审阅，旧构建不动。权利、试学与其它完整验收未完成；整书质量报告快照仍缺。见 `docs/qa/frozen-book-verification-2026-09-10.md`。
  - 2026-09-10：r13 单章 PDF 16 页、DOCX 21 页逐页校样与一致性检查完成；浏览器保存 8 条委托 AI v1 记录，六项限当前单章范围通过，权利需修改、独立试学未评估。新审校包 17 文件/全部哈希通过且包含记录，母稿不变，preflight false。确认包内 verification-summary 尚无冻结运行快照，需要补版本化工件/输入绑定，不能修改旧构建补造证据；权利、学习和其他整体验收继续。见 `docs/qa/r13-book-review-2026-09-10.md`。
  - 2026-09-10：补入 Go/Apple 官方准备资料，修复并部署官方 HTML v3 平台标签页解析；批准 r4 蓝图的 16 段证据。真实 foundation 生成一次，质量拒绝后零 Provider 局部纠正，修复安装入口、测试与练习矛盾。新代码隔离测试通过，用户委托 AI 审阅批准为 r11，新冻结构建与 Markdown 实际输出成功。其余 audience、冷读者、权利、PDF/ZIP 与媒体仍未完成；见 `docs/qa/foundation-setup-evidence-2026-09-10.md`。
  - 2026-09-10：r11 DOCX 已真实导出并完成 18 页逐页校样；增加可点击目录、页码、脚注分隔及保留原代码的显示分段，部署 export-service v5。9 个目录目标、34 条脚注与代码逐字一致性通过；整书版式仍缺真实 PDF/ZIP，详见 `docs/qa/docx-book-layout-2026-09-10.md`。
  - 2026-09-10：export-service 的独立 Chromium sandbox 配置与非 root 工件 reader group 修复已部署，r11 PDF/ZIP 均实际 HTTP 200，ZIP 全部 17 文件哈希通过。PDF 现可渲染，但目录、页码与分页尚未达标，继续版式工作；课程 browser_page 沙箱未改变。见 `docs/qa/textbook-pdf-zip-runtime-2026-09-10.md`。
  - 2026-09-10：PDF 目录/页码/代码分页已实现并部署，真实 14 页全部逐页复核，9 个目录目标与原代码完整性通过。浏览器追加 layout v3，单章范围委托 AI 通过 3/4；新 ZIP 17 文件及全部哈希通过，包含该记录。DOCX/PDF 版式缺口已解决，母稿与冻结 manifest 不变，其余审校、权利和真人学习仍缺，完整验收保持未完成。见 `docs/qa/textbook-pdf-layout-2026-09-10.md`。
  - 2026-09-10：r11 内容复审记录三项需要修改，纠正类比与步骤指向，补零基础分步读法、状态表和提示梯度，代码与 16 来源不变。零模型候选 r12 经新清单真实隔离测试后，由浏览器委托 AI 批准为 r13；新冻结构建及 Markdown 已输出，旧 r11 保留。新稿排版/整书复审、权利和学习闭环仍需继续，不能继承旧构建通过结论；另需补工件验证的 UI 启动入口。见 `docs/qa/r11-editorial-refinement-2026-09-10.md`。
  - 2026-09-06：stack_familiar 原拒绝稿完成机制、边界与六维练习修订；两次 v9 离线门禁通过且结果一致，新增模型调用 0。独立工件在既有 Go 1.26.8/Bubblewrap 中通过 go_test，约 8.03 秒，保存实际容器配置、清单和代码绑定。main.go 保持，测试补齐两个 health 返回断言；没有执行迁移题或学习者作答，也未向主项目写候选，r7/r3 保持。详见 `docs/qa/stack-familiar-rejected-correction-2026-09-06.md`，完整读者验收仍未完成。
  - 2026-09-06：此前 stack_familiar 真实 v16 调用完成，12006 输入/8526 输出 Token、49.25 秒。固定运行标记与 3 条事实/10 段来源覆盖通过，HTTP/POST 缩写门禁拒绝；静态复核另发现无状态协议与服务内存混淆、编码任务语义不成立等问题。该生成阶段无重试、候选或代码运行。原始收据详见 `docs/qa/stack-familiar-real-generation-2026-09-06.md`。
  - 2026-09-06：programming 修订稿的原 Go 文件已在固定 Go 1.26.8/Bubblewrap 中真实通过 go_test，8.42 秒；单独保存正文/清单/代码树/镜像绑定与原始输出，不使用 r7 证据、不写入主项目。新增 opt-in 验收入口及 Runner/架构回归通过。其他 audience 和完整读者验收仍待完成。
  - 2026-09-06：programming 原真实拒绝稿完成内容修订，补齐前缀分裂、教学边界、状态码与六维任务；两次离线门禁通过且结果一致，教学 Go 文件保持。Provider 调用 0，未执行代码、未写入主项目。该评测身份没有原主项目 Task，不借用旧任务或 r7 运行证据。详见 `docs/qa/programming-rejected-correction-2026-09-06.md`，本项仍未完成。
  - 2026-09-06：完整消息预算检查补齐响应结构、证据元数据、JSON 转义和包装余量；两个适配器与工作器共用组装逻辑。原两份冻结 v16 请求离线估算 13232/13237 输入 Token，均在额度内；聚焦和全量 Go/架构通过。无新模型调用，内容拒绝项不变。详见 `docs/qa/generation-request-budget-2026-09-06.md`。
  - 2026-09-06：针对真实拒绝稿修复系统固定 rubric 标记的重复生成责任；v16 由系统补入运行证据要求，保留任务、维度、来源、延迟和正文绑定校验。原稿离线对照仍有 6 项内容失败，Provider 调用 0，无新候选。详见 `docs/qa/practice-runtime-policy-2026-09-06.md`，本项保持未完成。
  - 2026-09-06：真实生成验收入口改为必须显式提供冻结 audience 请求文件，校验全部输入、模型/读者身份、完整快照和原文 hash 后再调用；不再默认发送简化源码夹具。质量失败也保留完整请求、拒绝稿和用量，输出目录不能覆盖旧结果。以 8 段完整 Gin 源码和两段已导入官网正文准备 programming/stack_familiar 独立评测契约；首份真实调用 53063 ms、11990 输入/8575 输出 Token，被 rubric 运行标记、命令来源及前缀机制引用门禁拒绝，第二份未调用。离线工具复现失败，无新候选或母稿修改。模块/全量 Go/架构通过；详见 `docs/qa/gin-audience-frozen-generation-2026-09-06.md`，保持未完成。
  - 2026-09-06：已打通 Go 声明/Symbol/物理行号、额外已选证据保留、符号检索和列表外显式选择；资料导入 v3 按版本化输入与冻结快照幂等，旧快照保留。真实三文件重解析全部成功、115 个新片段逐字节/hash 校验，重复导入不新增，页面 9 文档/395 片段；八段源码提案为 14212 字节。全量 Go、前端 257 项及随后聚焦 8 项、lint/build、真实 PostgreSQL 和浏览器选择通过；尚未替换批准蓝图或修正 r6，本轮无 Provider 调用。不勾选本项。
  - 2026-09-06：v15 经真实界面调用一次并保存 r6（5542 输入/7692 输出/2048 缓存 Token，45946 ms），自动登记工件。代码已逐层建树，但正文与练习混淆教学分段树和生产前缀机制、方法隔离测试仍不足，暂不批准。向原项目真实导入同提交 routergroup.go/gin.go/tree.go，3 任务全部成功，文件 hash 与入库一致，页面 6 文档/280 片段，原蓝图不变。下一步先打通 Go 源码 Symbol 定位、完整证据选择和固定样章别名筛选，再以真实来源局部修正；详见计划审计最新快照。其他 audience、官方完整资料栈与人工验收仍缺。
  - 2026-09-06：拒绝稿私有留存和零 Provider 调用的离线重检工具已部署，自动回归覆盖不可变、哈希、权限、配额及全部门禁重跑。随后真实 v14 任务 retry=2 在 38,923 ms 通过硬门禁并保存 foundation 候选 r5（输入 5,243/输出 6,321/缓存 5,120 Token）。自动技术复核发现平面映射被称为路由树，仍需修正；工件因合同哈希前缀不一致未登记。另两类读者、真实运行和人工审阅仍缺，不勾选本项。
  - 2026-09-06：新增失败缩写有界诊断（原稿行号、正文/练习区域、次数、最多 160 字片段）并接通任务结果和失败 UI；排除代码围栏，隐藏凭据标记行，不保存完整拒绝稿。全量 Go、前端 12 项、lint/build 通过；旧 v14 任务没有保存正文，不能事后补造定位，真实样章仍未通过。
  - 同日通过界面对 v14 冻结任务重试一次：48,491 ms，输入 5,243/输出 7,313/缓存 5,120 Token。真实诊断定位到正文第 181 行独立迁移任务首次使用 POST 未释义（10 次）；同时新增“显然”和未经验证运行表述失败，未保存候选。新诊断已在真实界面显示，但整章生成稳定性仍不合格；下一步需要有界拒绝内容留存与局部修正/重检，不能以诊断能力代替样章验收。
  - 2026-09-06：修复文件布局按注释/字符串误分类，以及行内代码/加粗/斜体导致缩写首次释义误拒绝。prompt v14 明确 main 入口、真实测试与邻接边界说明；先复现失败，再通过聚焦与全量 Go 回归并更新本机 core-api/llm-stream。经界面创建真实任务 `67a71e66-de82-45f2-ba62-f61a05343aa7`，6 段冻结证据、foundation，40,729 ms 返回，输入 5,243/输出 6,696/缓存 1,280 Token。仅剩 POST 首次释义一项硬失败，未保存候选、未重试；需补首次出现定位并针对性修复，其他 audience 与人工验收仍未完成。
  - 2026-09-06 10:30：经实际界面创建 v13/v9 基础样章任务 `56958bf2-f4a5-466b-9f67-1a78856b92c9`，38,333 ms 返回，输入 5,104、输出 6,427、缓存 1,280 Token。练习文本错误未再出现，但 POST 释义、教学实现边界和 Go 文件布局门禁仍拒绝，候选未保存，重试 0；其余 audience 未继续。真实调用授权已存在，下一步应修复生成质量并保留真实审阅边界。
  - 2026-09-06 10:24：用户授权实际电脑操作后，通过界面重试基础样章一次。DeepSeek 在 39,797 ms 返回（输入 4,844、输出 6,590、缓存 4,736 Token），被练习文本和第三代码块来源门禁拒绝，未保存候选。已补字段级错误诊断及输出约束，prompt v13 在本机上线、全量 Go 通过；后续调用按失败即停止暂停，v13 仍待真实生成验证，不勾选本项。
- [ ] 完成双审批、按章恢复、候选 diff、人工修改保护、代码验证、截图和视频 Runbook。
  - 2026-09-13 候选续：零模型调用产生 r14，经页面独立 Go 1.26.8 隔离验证成功；候选教案预览/版本区分及长哈希换行修复，312 项/lint/build 和真实切换刷新通过。页面以 delegated_ai 完成显式应用，生成批准 r15，正式投影返回同源教案，r13 保留。静态素材登记、整书新冻结及三项综合验收仍待完成。见 `docs/qa/runbook-candidate-review-2026-09-13.md`。
  - 2026-09-13 视频续：教案与教学工件共用源码解析，从实际文件派生符号、行号与哈希，不再写死测试名。后端全量回归通过，r13 只读预览在真实 VS Code 1.137.0 中准确定位 TestMethodSpecific:49。未部署新 llm-stream、未改批准稿或形成新候选；下一步候选落库与工作台/媒体复核，综合项继续未完成。见 `docs/qa/runbook-source-binding-2026-09-13.md`。
  - 2026-09-13 续：实际页面上传历史 r6 第四次成功截图，原图哈希/修订/证据均匹配，重复提交幂等、刷新恢复通过；保持 pending/unverified，不冒充当前稿验证。修复部分样章覆盖时误报无批准母稿的阶段分支，真实 PostgreSQL/textbook/架构回归及部署后页面通过。当前视频投影仍缺本章精确断点/输入/状态，下一切片补可跟录性，不勾选综合项。见 `docs/qa/project-progress-and-media-upload-2026-09-13.md`。
  - 2026-09-13：截图登记改为显式选择运行记录并绑定同一修订，过滤孤立记录、刷新移除后禁止提交；旧实现三项回归失败，修复后前端 312 项/lint/build 通过。新前端上线，真实页面 r6/r12 切换、工件/哈希回读及刷新未选择通过；未实际上传媒体，不勾选综合项。见 `docs/qa/visual-asset-revision-binding-2026-09-13.md`。
  - 2026-09-10：独立验证尝试已部署 schema 38/core v2/runner 与 frontend v1；request_id、前序 CAS、退出回执和不可覆盖历史贯通。真实页面完成历史 r6 的取消后新建成功、成功后重验并运行中取消、再重跑成功；旧任务/证据保留，刷新 0 次 POST/retry，主稿/批准/构建不变。后端 57 包、PostgreSQL 并发/race、前端全量 308 项及到期入口联验 15 项、lint/build 通过。移动窄视口原侧栏遮挡已如实记录；不是全项完成。见 `docs/qa/teaching-verification-attempts-2026-09-10.md`。
  - 2026-09-10：r6 工件权限经受限哈希恢复，原任务实际重试通过但旧工具链收据不可直接沿用；新增绑定当前工具链的独立历史工件。runner 运行中取消、原子证据/终态及真实终态响应已部署，真实 Go/Bubblewrap 执行中取消后进程退出、0 证据、刷新恢复同一任务；后端 57 包、race、PostgreSQL 两向竞态通过。主稿/批准/冻结构建不变。取消后重新运行与到期重验证仍需新尝试生命周期，完整项继续未完成。见 `docs/qa/teaching-verification-cancellation-2026-09-10.md`。
  - 2026-09-10：教学代码面板现可显式启动、恢复任务、查询进度、重试失败及取消活动任务。新增只读 GET 与来源修订标记已部署；真实浏览器恢复当前成功任务，启动历史 r6 后显示目录权限失败，刷新不重复创建任务，重试一次沿用原 ID。主稿未变；真实取消、取消后恢复、到期重新验证与历史目录问题仍缺，不勾选整项。见 `docs/qa/textbook-verification-ui-2026-09-10.md`。
  - 2026-09-06：真实 PostgreSQL 复现并修复生成修订解析器的裸合同 hash 输出；单任务恢复工具绑定 task/revision/content hash，默认预览，apply 只登记派生工件。真实 r5 恢复两次后只有 1 份同 ID/hash 的 unverified 工件，源文件与母稿逐字节一致；仍为 5 份修订、人工审阅 0、运行证据 0。全量 Go/架构通过。r5 教学机制缺陷及后续批准、运行和媒体验收仍未完成。
  - 2026-09-06：修复样章投影使用固定登记示例而非当前母稿代码的问题。现在校验已保存 revision/hash，以 Goldmark 提取明确标识的 main.go/main_test.go 原字节，工件 ID/树 hash 随内容变化；来源或布局不明确时不生成替代工件。只补固定 go.mod 和 go_test，不补造浏览器页面。运行报告的 Go 版本改由运行器提供，不再照抄 manifest 声明。负例、同源性、全量 Go/架构回归通过；真实库工件仍为 0，故本条是代码缺陷修复，不是教材运行、截图或人工验收完成。
- [ ] 完成 explain/complete/reproduce/transfer/diagnose/retain 一次学习闭环。
- [ ] 构建 Markdown/DOCX/PDF/ZIP 并完成试学、技术审校和版面校对记录。
  - 2026-09-13：图片内部字体补证 v1 已复用权利追加链实现、部署；绑定冻结图片 ID/hash 与明确字体权利版本，缺失、pending、过期及未支持载体仍拒绝。后端 57 包、前端 319 项、真实 PostgreSQL、页面未选字体拒绝、API 错哈希 400 与 ZIP 24 项哈希通过。当前五项权利 pending/八条 AI 不变，API 16/ZIP 18 阻断；实际截图字体来源、hands_on 和试学仍缺，本项不勾选。见 `docs/qa/asset-font-review-2026-09-13.md`。
  - 2026-09-13 审阅续：当前 r15 的 19 页 PDF 全部逐页检查；16 引用、两代码文件及隔离收据核对完成。真实表单新增 layout/technical 两条 delegated_ai v1、3/4，刷新与审校 ZIP 一致；21 项内容哈希通过，出版阻断 13→11、真人记录仍为 0。其余阶段、权利和试学未完成，见 `docs/qa/r15-layout-technical-review-2026-09-13.md`。
  - 2026-09-13 冻结续：批准 r15 冻结为 `66e5f4a4-00ac-449e-82c1-1647d8dfd248`，修复教案在整书冻结中丢失的问题，ZIP 随附同版本 JSON/Markdown 教案；旧构建不动态补取。后端全量/架构检查通过，四格式 HTTP 200、ZIP 21 项内容哈希通过、19 页 PDF 完成 AI 初步版面检查；真实页面链接和重复冻结幂等通过。仍缺新构建权利补证、真人试学及审校，不勾选综合项。见 `docs/qa/book-runbook-freeze-export-2026-09-13.md`。
  - 2026-09-13 WPS 续：修复主题字体、过期字体表及短表格跨页；用户授权安装 5 个本地字体后 WPS 无缺字警告。23 页独立渲染全部检查，WPS 标题/代码/第 8–9 页表格/文末实查通过；ZIP 追加实际渲染工具与模板哈希，export/架构回归通过。保持综合项未完成，见 `docs/qa/docx-wps-fonts-2026-09-13.md`。
  - 2026-09-10：真人复审 v2 已部署至 core/export/frontend，schema 37。明确评分/结论/证据、冻结稿绑定、CAS 与只追加历史；旧说明不自动算通过，八阶段均可复审。真实 PostgreSQL 迁移保护与导出历史、API 拒绝负例和页面验证通过；后端 57 包、前端 306 项、lint/build 通过。主构建未变，21 文件 ZIP 哈希通过、19 页 PDF 文字/像素一致；八委托 AI、0 真人，未新增个人学习，不勾选完整验收。见 `docs/qa/human-review-revisions-2026-09-10.md`。
  - 2026-09-10：三份来源/分发声明通过真实页面冻结为新构建 `0f150bbe-b80a-4586-8997-320df2c35ae0`，AST v3 派生四格式与代码许可文件；精确重试复用，正文/16 引用/三代码不变。生产 21 文件 ZIP 哈希通过，PDF 19 页及指定本地字体 DOCX 23 页复核；六权利中五 ready、一正文 pending（Apple 具体出版依据）。新构建八条委托 AI 为六 pass、rights needs_revision、reader_trial not_assessed，0 真人/个人学习新增；完整项保持未完成。后端 57 包、前端 302 项与 lint/build 通过，详见 `docs/qa/publication-notices-2026-09-10.md`。
  - 2026-09-10：批准 r9 的 v2 BookBuild 已真实导出 Markdown/DOCX，19 页 DOCX 校样逐页查看；新增用户委托 AI 整书审阅合同和独立历史，浏览器实际保存八阶段结论，当前单章范围四项通过，自学性/版式/权利需要修改，试学未评估。真人记录与学习作答仍为 0，PDF/ZIP 服务能力仍未通过，未晋级出版候选，本项保持未完成。授权无需再次确认；下一切片补零基础安装入口及剩余证据。详见 `docs/qa/delegated-publication-review-2026-09-10.md`。
- [x] 逐项核对 PRD 第 17 节；台账为每项附自动证据、文件或明确的人工/运行时缺口，未将部分合同伪报为端到端通过。

2026-09-05 继续执行更新：三类 audience 的请求校验、Gin 六证据转换和 Provider
输出合同已补齐，离线夹具仍只支持真实写作目标为零基础的固定正文；prompt schema
为 v11、质量合同仍为 v8。17:18 最新一次已授权真实任务使用 prompt v11，被 POST
释义、教学实现边界标识和 Go 未使用导入检查拒绝，未产生候选、未重试；
输入 4,071/输出 4,373 Token、调用 1 次、29,610 ms 已持久化，见
`docs/qa/textbook-plan-completion-audit.md`。以上四项依然需要真实内容和人工记录，
不能因合同测试通过而勾选。

2026-09-05 23:41 继续执行更新：固定 Gin SHA 和六证据不变，离线 fixture 已为
foundation、programming、stack_familiar 生成三份真实差异化的读者起点、场景、
学习弧首步和练习要求，且 audience、内容 hash 与 fixture prompt hash 均不同；三个
版本通过当前 v9 门禁及 SampleTaskRunner 序列化回归。它用于真实调用前验证三类冻结
输入，不替代 Provider 生成、人工批准或读者试学，本项继续保持未勾选。

2026-09-06 继续补齐真实验收入口：受保护的 `real-acceptance` 工作流新增独立
`run_textbook_generation` 开关，默认关闭；明确启用后才按 foundation、programming、
stack_familiar 顺序各调用一次，前一读者版本失败就停止后续调用。三份候选必须分别通过
当前硬门禁、保留 `manual_review_required` 和 `runtime_verification=unverified`，并具有不同
请求/正文 hash；Markdown、结构化报告和 Token/延迟遥测作为工作流 artifact 保存。普通
测试已证明三份冻结请求不同且联网测试默认跳过。本轮未获调用授权、没有执行这 3 次真实
生成，因此本项仍保持未勾选。

2026-09-06 00:22 继续执行更新：真实最终 argv 证明 Playwright 在旧 probe 下会自动加入
`--no-sandbox`，因此旧的源码参数检查不足以形成浏览器证据。probe 已强制
`chromiumSandbox: true`、补齐固定离线 Go 页面环境，并在教材验证启用时成为启动预检。
本机临时打开开关后，页面启动但 Chromium sandbox 不可用。执行器现按 manifest 顺序
保留已通过的 Go 命令事实，browser_page 无执行器时记录 unverified，工件整体不升级；
本地教材 Go 开关已在 core-api/course-runner 打开且两者 healthy，学习者固定 Go profile
仍 available。真实库没有批准代码工件、截图、RuntimeEvidence 或任务，所以代码验证、
截图和视频 Runbook 项继续保持未勾选。

2026-09-06 00:36 继续执行更新：用户确认后只提交一次 foundation v12/v9
真实 DeepSeek 任务 `ff0f6e35-c350-4351-be33-69564ff88d16`。任务在 45.02 秒时读取响应
超时，重试 0、候选修订 0，无完整 Provider 用量可落库。教材 Provider 请求现独立使用
`TEXTBOOK_GENERATION_REQUEST_TIMEOUT`，默认 `15m`、限制 `30s`–`30m`，不改变其他调用的
45 秒边界。全量 Go、Compose 配置与 diff 检查通过，重建后 `llm-stream` healthy 且实际
读取 `15m`。未自动重试或应用候选；三类 audience 真实生成仍未验收，本项保持未勾选。

2026-09-06 浏览器后续诊断：在固定操作方页面上去除 Go 专用内层 BPF 后，
Chromium sandbox 仍不可用。私有 `/proc` 挂载被当前非特权容器拒绝，空 `/proc`
缺少 `uid_map`，只读绑定 `/proc` 又无法写入 `uid_map`，最终稳定复现
`No usable sandbox`。probe 已将此失败归类为 `chromium_sandbox_unavailable`，人工截图与
录屏的批准修订、工件/媒体哈希、工具版本、观察边界和恢复记录已写入
`docs/runbooks/textbook-browser-video-evidence.md`。当前仍没有实际自动截图或人工媒体，本项保持未勾选。

2026-09-06 失败任务重试门禁补强：样章任务只有在服务端能够重新校验完整冻结载荷时才
返回 `retry_confirmation`，POST 重试必须携带同一输入哈希；空哈希和错误哈希均以 409
拒绝。前端将单击重试拆成“查看重试影响”和“确认重试一次”，展示模型、合同版本、
与首次预检一致的 `10,191 / 20,000` 输入 Token、12,000 输出预留、费用未知及待审候选边界。
真实失败任务经重建后的浏览器验证，控制台 0 错误、0 警告，重试数仍为 0；未点击最终
确认，未产生第二次 Provider 调用。该门禁只让下一次决定可审阅，不完成真实生成验收。

2026-09-06 按章恢复补强：core schema v34 为样章任务增加受外键约束的
`textbook_chapter_id`，从不可变任务载荷回填历史关系，并以
`workspace_id + textbook_chapter_id + created_at DESC + id DESC` 的局部索引支持确定性查找
最新任务。真实库 9/9 个样章任务完成回填；目标查询经 PostgreSQL `EXPLAIN` 使用
`idx_job_tasks_textbook_chapter_created`。章节工作区现返回服务端 `latest_sample_task`；清空
浏览器 `sessionStorage` 并重载后，真实页面恢复任务
`ff0f6e35-c350-4351-be33-69564ff88d16`，重新显示失败原因和二次重试确认，再把任务 ID
写回本地缓存。控制台 0 错误、0 警告，最终确认未点击，任务仍为 `failed`、重试 0。
这完成了失败任务在浏览器状态丢失后的服务端发现能力；本项还包含候选批准、diff、代码
验证和真实媒体证据，因此继续保持未勾选。

**C++ 通用性回归：**

- [x] 用 isocpp.org/get-started 和 tour 官方页面测试不同站点结构、代码语言和工具推荐：2026-09-03 受控真实抓取返回恰两页，尾斜杠导航别名按队列身份去重，跨域和路径越界链接被拒绝；离线合同覆盖 C++ 项目清单、证据检索和 `cpp → Visual Studio 2022` 的人工采集 Runbook。
- [x] 只做抓取、清单、检索和蓝图回归；离线 C++ 结构夹具覆盖受控清单、跨域拒绝、检索主资料门禁和项目蓝图，未扩成第二本教材。真实官网抓取已在明确 opt-in 的隔离环境中完成，详见 `docs/qa/cpp-official-stack-regression.md`。

**最终清理：**

2026-09-10：实际全量检查发现旧 JWT 包和带“2”的历史副本仍在源码树；经无调用与内容
对照后，18 文件完整归档并完成隔离恢复验证，未覆盖独立变体。类型导出、v24 迁移夹具
及过期 E2E 夹具修正后，后端全量 57 包、integration 合同、前端 289 测试、lint、deadcode、
build、bundle 与 Chromium 核心 8 条通过。真实构建/审阅未变；此检查不替代 PR-16 的内容、
试学、学习和权利验收。见 `docs/qa/legacy-cleanup-and-full-validation-2026-09-10.md`。

2026-09-05 11:57 实测更新：用户指定备份父目录后，已在
`/Users/huangqijun/Documents/墨言博客助手/InkWords-backup-20260905-115739`
完成真实 core/review dump、随机临时库恢复、v2 聚合快照比对和 SHA-256 复核。
临时库已清理，11 个本机服务恢复正常。下列历史备注中“未获得备份目录、未执行恢复”
已被本记录取代；具体删除 migration、删除前预检及恢复方案仍待实施。

- [x] 统计 auth/login/user/OAuth/JWT/captcha 和 `UserID` 所有调用；逐域迁移且确认无调用后再删除。2026-09-05 已完成运行时迁移并应用 core v24/review v25；剩余关键词仅来自历史 migration、空库兼容 bootstrap、清理预检和防回流测试，不构成运行时身份边界。
- [x] 破坏性 schema migration 前生成本地备份并提供恢复步骤；无法安全 Down 的 migration 写清备份恢复。真实 v2 双库备份位于 `/Users/huangqijun/Documents/墨言博客助手/InkWords-backup-20260905-115739`，两库随机临时库恢复、聚合快照和 SHA-256 均通过；v24/v25 Down 明确拒绝并指向该恢复路径。
- [x] 删除旧 `ProjectCourse` 仅在所有 API/UI/MQ/导出调用都已迁移后进行；否则保留后台兼容适配器。真实库在表、任务、事件均为 0 后由 v24 删除 `project_courses`；59 篇博客、26 个任务及 4 个复习会话保留，六个新版服务和幂等重启验证通过。
- [x] 更新一键本地启动、首次 API Key 设置、备份/恢复、升级和故障排查文档；Runbook 明确 provider opt-in、备份恢复确认和 sandbox fail-closed 边界。

**全量验证：**

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./... -count=1
cd backend && GOCACHE=/tmp/inkwords-go-build go test -tags=integration ./integration
cd frontend && npm run lint
cd frontend && npm run deadcode
cd frontend && npm run test:coverage
cd frontend && npm run build
cd frontend && npm run check:bundle
OBSIDIAN_VAULT_PATH=/tmp/obsidian-vault docker compose config
```

Compose E2E 还必须通过服务健康、gateway、任务恢复、Playwright 核心流程和断网读取已批准教材。

**退出条件**：PRD 验收矩阵全部有证据或明确未交付；旧桥梁只在调用为零后删除；本地备份可恢复。

**提交**：`refactor(platform): complete local textbook migration`

## 12. 测试与证据矩阵

| 风险 | 最低测试层级 | 必备证据 |
| --- | --- | --- |
| 关键事实幻觉 | domain + pipeline | claim 到 EvidenceRef 100% 可回链 |
| 代码不可运行 | runner integration | artifact hash、命令、环境、stdout/stderr、exit code |
| 资料栈不完整 | crawler contract | manifest、skip/fail 原因、预算状态 |
| Token 浪费 | pipeline benchmark | stage/provider/cache/input/output token 对比 |
| 人工稿被覆盖 | repository concurrency | CAS 冲突与三方 diff 测试 |
| “防自学”文本 | detector + reader trial | 风险清单、人工 rubric、目标读者卡点 |
| 学习曲线断裂 | domain + sample review | LearningArc 状态和练习梯度 |
| 复习调度错误 | property/replay tests | 相同日志重放相同 card/due |
| 导出不一致 | golden + visual QA | AST ID、格式 diff、DOCX/PDF 页面检查 |
| 权利/出版误导 | preflight | RightsItem 和待确认字段清单 |
| 本地升级破坏数据 | migration integration | backup、up/down 或恢复演练 |

### 12.1 默认 CI

- Go 单元、合同、架构和离线 pipeline 测试；
- PostgreSQL migration 集成测试；
- React/Vitest、lint、deadcode、coverage、build、bundle budget；
- Docker Compose config 和本地 fixture smoke；
- fake provider、fake crawler、fake runner；
- 金丝雀导出物的结构和 hash。

### 12.2 显式 opt-in 验收

- 真实 DeepSeek/OpenAI 调用；
- 真实 Gin、go.dev、isocpp.org 抓取；
- 真实 Docker/主机 sandbox；
- 真实 IDE 人工截图；
- DOCX/PDF 人工版面校对；
- 目标读者试学。

CI 不得因为真实服务不可用而伪造通过；opt-in 未执行就显示“未验证”。

## 13. Token 优化验收

Token 优化是架构约束，不是最后的性能清理：

- 解析、分类、hash、引用校验、术语检查、diff 和导出尽量确定性完成，不调用模型。
- 全资料只在建立索引时处理一次；生成阶段使用 chapter evidence pack，不重复发送整库。
- 蓝图先于正文，claim plan 先于 prose；证据不足时早停，避免生成后返工。
- 共享 BookContract、StyleSheet 和术语表使用稳定 hash；缓存按阶段失效，而非任一编辑导致全书重跑。
- 对章节独立 checkpoint；失败只重跑当前阶段。
- 小模型用于抽取、分类和格式修复；高质量模型用于结构、核心解释和审校。
- 统计 cache read/write token；若供应商没有对应字段则分开记录本地缓存节省估算，不能冒充账单数据。

每个里程碑在相同 Gin 离线夹具上比较：模型调用数、input/output/cache token、P50/P95 耗时、失败重试次数和样章人工质量。质量下降时不得用 Token 节省抵消。

## 14. 风险与止损条件

| 风险 | 早期信号 | 止损/降级 |
| --- | --- | --- |
| 范围再次膨胀 | Gin 样章未通过就开发设备专项适配或第二本书 | 冻结新入口，只完成当前纵向切片 |
| 合同过度抽象 | 新包很多但没有端到端样章 | 以 PR-05 为强制架构验收点，删除无消费者抽象 |
| 官网抓取不可控 | manifest 不稳定、重复 URL、跨域膨胀 | 降级为 sitemap/手选入口，保留显式预算 |
| DOCX 解析质量不足 | 标题/代码/表格次序丢失 | 标记低质量并要求 Markdown/PDF 替代，不静默继续 |
| 模型成本失控 | 单章多次发送整库或无 cache 命中 | 阻断生成并展示预算诊断 |
| sandbox 无法安全落地 | 需要 privileged 容器或禁用关键隔离 | 保持 runner 关闭，发布“未验证代码”而非放宽安全 |
| 自动质量分虚高 | detector 通过但试学者无法完成 | 人工试学是发布硬门槛，调整规则和夹具 |
| 出版权利不清 | 来源仅标“网上公开” | 只允许个人学习导出，阻断出版候选 |
| 迁移伤害旧数据 | down/恢复演练失败 | 暂停删除旧表和 auth，保留兼容读取 |

## 15. 第一批执行顺序

开始开发时只领取一个 PR，推荐顺序如下：

1. PR-00：冻结 Gin 教材质量和 Token 基线。
2. PR-01：建立 `shared/kernel/textbook` 合同与架构测试。
3. PR-02：单独引入 migration 机制和 LocalWorkspaceContext。
4. PR-03：持久化最小教材项目与 revision。
5. PR-04：建立无登录教材工作台。
6. PR-05：交付第一章可读、可审、可导出的 Gin 教材。

PR-05 是第一个“继续/调整/停止”决策点。只有样章被实际阅读并确认达到通俗、专业、可跟练的标准，才进入 M2；若不达标，优先修合同、风格和生成链路，不提前建设全书基础设施。

## 16. 依赖选型检查点

以下依赖是候选，不代表计划文件授权安装：

| 能力 | 候选 | 采用前必须验证 |
| --- | --- | --- |
| 官网抓取 | [Colly](https://github.com/gocolly/colly) | 许可证、维护状态、robots、限速、SSRF 包装、缓存和 deterministic fixture |
| 数据迁移 | [goose](https://github.com/pressly/goose) | embedded SQL、PostgreSQL、up/down、启动失败语义、镜像体积 |
| 间隔重复 | [go-fsrs](https://github.com/open-spaced-repetition/go-fsrs) | 算法版本、日志重放、参数迁移、时区、可解释 due |
| DOCX 构建 | [Pandoc](https://pandoc.org/MANUAL.html) | reference.docx、中文字体、代码样式、交叉引用、安装包体积、失败恢复 |
| 集成测试 | testcontainers-go | 本机 Docker 兼容、CI 耗时、镜像固定、清理可靠性 |

每项依赖采用前先提交 ADR/选型测试，再提交最小依赖变更；不以“成熟项目”替代 InkWords 自己的安全边界和领域合同。

## 17. 计划维护规则

- 合并一个 PR 后更新对应 checkbox、实际提交、测试结果和偏差原因。
- 若实现文件与计划不同，以架构边界和验收条件为准，并在 PR 中更新本计划，不能悄悄漂移。
- 新需求先判断是否影响 PRD 成功标准；非关键能力进入 backlog，不插入当前关键路径。
- 每个里程碑结束做一次短复盘：质量是否提升、Token 是否下降、复杂度是否增加、哪些抽象应该删除。
- 只有 PRD 或开发设计发生实质变化时才重排里程碑；普通实现细节在对应原子 PR 内决策。
