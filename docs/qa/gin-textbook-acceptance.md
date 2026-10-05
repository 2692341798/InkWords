# PRD 17 验收台账：Gin 零基础教材

日期：2026-09-06

本台账逐项对应 PRD 17.1。`已验证`仅表示所列自动合同已经通过；`待真实验收`表示尚未在固定 Gin SourceSnapshot、真实浏览器或人工审校中完成，不能以代码存在替代。

最新状态见 [计划审计的 v15 真实 r6 与同项目源码补齐记录](textbook-plan-completion-audit.md)：
真实 DeepSeek 任务 `b278d9fa-af86-46c0-b901-cdbbe805a5b0` 已保存 r6 并自动登记未验证工件；
输入 5542/输出 7692/缓存 2048 Token，调用 1 次、45946 ms、重试 0、费用未知。
教学代码已逐层建树，但正文混淆了教学分段树与生产路径前缀机制，暂不建议批准。
三类读者的实际请求/worker/Provider 合同已补齐；离线基线也已分别生成不同读者起点、
场景、学习弧首步与练习要求，并以不同内容 hash 通过当前质量门禁。这不是三类真实章节验收。
当前运行版本为 prompt v15 / quality v9，真实候选拥有结构化六维题目，但题目内容仍需技术修正。
原项目已将同一固定提交的三个真实源码文件按 v3 声明级解析，旧快照保留，合计 9 文档/395
片段；Symbol/物理定位、额外证据传递、列表外选择已验证，八段源码提案为 14212 字节。
批准蓝图仍绑定原 6 段离线摘录，尚未将新提案批准为生成输入，不能直接用解析通过宣称
已完成真实源码驱动生成。原输入的预检估算仍为 10,191 / 20,000，输出预留 12,000。
review schema v33 已接入服务端批准题目冻结、作答 ID/hash 绑定、服务器保持间隔、
跨刷新帮助记录、会话提交幂等，以及显式模型评分任务、取消/重试和追加纠正。
明确应用的评分快照会事务性重放 FSRS，后续作答仍保留已应用解释。
代码可随首次作答原子冻结，按需读回原文件；截断响应后的刷新恢复已通过受控浏览器
验证。评分 v2 已将完整文件纳入静态评阅，逐项引文绑定文件路径，支持纠正和应用；
超过 32,000 字节请求预算时拒绝，不截断文件。缺少执行证据的评分条目保持未知。
目标/作答/会话/帮助/评分任务/纠正/评分应用/代码快照仍均为 0。评分、纠正及应用
恢复已通过受控浏览器验证。学习者代码的显式验证任务、一次性引用、固定离线策略、
取消/重试/中断恢复、评分 v3 运行证据消费和 UI 已实现。2026-09-06 本机固定 Go
沙箱预检已通过，学习者开关显式启用且能力接口 available；预检未读取真实学习者作答，
因此仍未取得真实学习者 VerificationRun。三份文字夹具的真实模型评分已有记录，因果评分
仍需校准，不能替代六模式真实学习者评测与读者验收。
下文逐日记录中的合同版本和“待授权”状态均属于当时快照。

| PRD 17.1 事项 | 当前状态 | 自动证据或记录位置 |
| --- | --- | --- |
| 资料清单、范围、版本、缺失项可见 | 部分：固定 Gin 离线夹具、固定提交源码导入，以及一次受边界限制的真实 Gin 官方文档快照已验证；完整官方资料栈与缺失项清单待验收 | `TestTextbookBaselineFixturePinsGinEvidenceAndQualityContract`；`backend/services/parser-service/domain/crawl/service_test.go`；本文件“Gin 固定提交源码导入回归”“Gin 官方文档导入回归” |
| 未批准蓝图不得批量生成 | 已验证 | `backend/services/core-api/domain/textbook/repository_test.go` |
| 样章质量骨架、可运行代码、真实输出、来源 | 部分：真实 DeepSeek 候选稿、固定 Gin 注册与请求查找证据、质量合同版本门禁已验证；教学工件仍标记未验证，真实运行输出待验收 | `backend/services/llm-stream/app/textbook/sample_chapter_test.go`；`backend/shared/kernel/textbook/runtime_observation_test.go`；本文件“DeepSeek 样章与新版证据蓝图回归” |
| 学习闭环并依表现调整 | 部分：样章合同、FSRS、具体六维练习、自评调度、帮助/作答恢复、评分/纠正/应用、v33 显式代码验证生命周期及评分 v3 运行证据消费已有自动验证；本机隔离预检后的真实学习者运行及真实模型/读者验收仍缺 | `backend/services/review-service/domain/mastery/practice_session_postgres_test.go`；`backend/services/review-service/domain/mastery/assessment_runtime_test.go`；`backend/services/review-service/infra/learnerverification/store_test.go`；`frontend/src/components/learning/AssessmentPanel.test.tsx` |
| BookContract、StyleSheet 与蓝图契约 | 已验证 | `backend/shared/kernel/textbook/contracts_test.go`；`backend/services/core-api/domain/textbook/repository_test.go` |
| 未批准样章不得批量生成 | 已验证 | `backend/services/core-api/domain/textbook/repository_test.go`；`backend/services/llm-stream/domain/stream/textbook_sample_consumer_test.go` |
| 自动终端/浏览器证据与 IDE 截图进入教材 | 部分：终端与受控 Playwright 执行、结构化证据、实际 Playwright/Chromium 版本、内容寻址截图及视觉资产绑定合同已通过；自动截图与人工 IDE 截图在 UI 中分开标识。Docker Desktop 上的真实受控运行与人工 IDE 截图仍待验收 | `backend/shared/kernel/textbook/asset_test.go`；`backend/services/course-runner/app/bootstrap/textbook_test.go`；`backend/services/course-runner/app/bootstrap/playwright_browser_test.go`；`frontend/src/features/verification/VerificationPanel.test.tsx`；`frontend/e2e/textbook-runtime-evidence.spec.ts` |
| 人工修改不会被重生成覆盖，候选稿驳回理由可追溯 | 已验证：驳回记录不可变且绑定候选稿哈希、质量合同和审阅 workspace；被驳回候选稿不能再应用 | `backend/services/core-api/domain/textbook/repository_test.go`；`frontend/src/features/textbook-projects/ChapterEditorPanel.test.tsx` |
| 冷启动试读、卡点与恢复记录 | 待真实验收 | 需目标读者或作者在干净环境留下试读记录 |
| 复述、补全、复现、迁移、诊断、延迟复习 | 部分：Objective、排程合同和任一到期模式的表现记录界面已验证；六种真实学习记录与延迟复习待验收 | `backend/services/review-service/domain/mastery/service_test.go`；`frontend/src/components/learning/MasterySession.tsx`；`frontend/e2e/mastery-session.spec.ts` |
| 到期任务仅在打开应用时出现 | 部分：前端按需读取实现存在；真实打开应用验收待补 | `frontend/src/hooks/useDueMasteryTasks.ts` |
| Markdown、DOCX、PDF、完整资产 ZIP | 部分：Markdown、DOCX、PDF、ZIP 的同源投影已在本机真实 smoke 通过；Docker Desktop renderer 和人工版式校对仍待完成 | `backend/services/export-service/domain/export/book_pdf_test.go`；`backend/services/export-service/domain/export/book_package_test.go`；`docs/runbooks/textbook-publication-checklist.md` |
| 整书术语、依赖、连续性、权利与编校预检 | 已验证为“阻断未满足项”，非出版通过 | `backend/services/export-service/domain/export/publication_preflight_test.go` |
| 前置内容、正文、附录、术语表、答案、参考文献、版本与出版社问题清单 | 部分：冻结整书 AST/审校包合同已验证；真实候选稿待验收 | `backend/shared/kernel/textbook/book_ast_test.go`；`backend/services/export-service/domain/export/book_package_test.go` |
| Token、缓存、费用可见 | 部分：真实 DeepSeek 候选稿已记录输入、输出、缓存 Token 与延迟；供应商未返回的费用仍保持未知 | `docs/qa/textbook-platform-baseline.md`；本文件“DeepSeek 样章与新版证据蓝图回归” |
| 断网或模型失败后本地资料、批准章节、学习记录可访问 | 部分：任务恢复与本地持久化合同已验证；断网端到端待验收 | `backend/services/llm-stream/domain/stream/textbook_sample_consumer_test.go` |

## 本轮明确未通过的运行时项

- 真实 isocpp.org 抓取：受控浏览器返回 HTTP 403；详见 `docs/qa/cpp-official-stack-regression.md`。
- Docker Desktop 中的真实 PDF 仍不可用：2026-09-03 在 `export-service` 容器内复测，服务以 `uid=10001(inkwords)` 运行、使用 `Chromium 124.0.6367.78 Alpine Linux`，对临时本地 HTML 的正常 headless PDF 命令返回 `Failed to move to new namespace ... Operation not permitted`，随后 zygote 以 `Trace/breakpoint trap` 终止，未生成 PDF。临时目录在命令退出时删除。2026-09-04 则在 macOS 本机通过 Playwright Chromium headless shell 成功执行 `TestChromiumBookPDFRendererRendersCanonicalASTWhenExplicitlyConfigured`：它从 Canonical Book AST 生成 `%PDF-` 文档，渲染日志为成功且含 Chromium 版本；`TestBookChromiumArgsRetainTheBrowserSandbox` 同时约束命令不传 `--no-sandbox`。这不替代 Docker 目标环境配置或人工逐页版式校对。
- 目标读者试读、IDE 截图、受控浏览器教学工件和人工出版审校：均尚未获相应环境或外部参与者的实际证据。六证据 DeepSeek v5 候选稿已经生成，但仍须由用户人工审阅；`frontend/e2e/textbook-runtime-evidence.spec.ts` 仅验证浏览器对受控证据响应的显示与人工采集边界，不是浏览器运行证据的采集结果。

后续验收必须为每一项补入当次固定 SourceSnapshot、测试命令/截图路径或人工签字记录，并且不得把本台账中的“部分”改写为“通过”。

## 2026-09-03 隔离 Compose 验证

本次使用全新 PostgreSQL volume 和本机 `18081` 网关端口启动了 core-api、llm-stream、parser-service、export-service、course-runner、review-service 与前端。所有后端服务及前端健康检查通过；网关 `GET /api/v1/ping` 和 `GET /api/v1/mastery/due` 分别真实连通 core-api 与 review-service。

该过程发现 `00015_mastery_objective_identity.sql` 曾被 core 迁移执行器错误纳入，空数据库会因 `mastery_objectives` 不存在而启动失败。迁移现明确标记为 review 角色，并以无缓存镜像、新 volume 重建验证：core-api 与 review-service 均健康，review-service 启动日志记录应用了 3 条 review 迁移。

同一临时数据库还验证了：一个六维 LearningObjective 创建后立即出现 `explain` 到期任务；同一冻结章节身份重试返回同一目标；记录表现后返回由六维策略与 FSRS 时间共同决定的下一任务。真实浏览器通过网关显示到期卡片、打开学习会话并提交一次结构化表现，控制台无错误。期间发现空队列曾序列化为 `tasks: null` 并使页面白屏；后端现固定输出 `tasks: []`，前端也兼容旧响应。

这只是合成的隔离学习目标和 fake Provider 环境，不是固定 Gin 官方 SourceSnapshot、真实模型输出、读者试学、IDE 截图、PDF 版式或人工审校的替代证据。

## Gin 固定提交源码导入回归

2026-09-03，在另一个全新、完成后可删除的 Compose volume 中，以本机网关 `18082` 建立 Gin 主资料项目，并导入从 `github.com/gin-gonic/gin` 固定提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 取得的 `routergroup.go`。导入使用 multipart 文件流；源码原始字节的 SHA-256 为 `e00c81913a4f1da0406ee0a53108ad51eb3d2c3f77d177f9696b31636d4a5b7c`。

真实结果如下：

- source-import parse task 可由同一显式开发身份轮询到 `succeeded`，而不是返回 403；这覆盖了空数据库下本地 workspace bridge 用户与 `DEV_AUTH_USER_ID` 的兼容绑定。
- parser-service 产出 1 个 `text/x-go` `SourceDocument`、40 个 Go 代码块；首块定位为 `routergroup.go` 第 1–3 行。
- core-api 将结果事务性落库。对应 `SourceSnapshot` 独立保存内容 hash 与 `resolved_version=73726dc606796a025971fe451f0aa6f1b9b847f6`；来源库仅显示文档元数据、路径和 chunk 数，不向浏览器返回全文。

这证明的是一个真实 Gin 固定提交文件的导入、解析、任务授权与持久化切片；它不证明完整 Gin 官方资料栈、三类读者生成、真实 Provider、读者试学或出版验收。

## Gin 来源资料浏览器回归与任务恢复

2026-09-03，在可删除的 Compose 隔离环境（本机网关 `18084`）中，浏览器创建了“Gin 路由源码导入验收”项目并进入资料工作台。此前空资料响应会让工作台读取 `null.length`；服务端现在初始化空 `sources`/`chapters` 数组，前端对历史空响应也归一化为数组。因此项目可以正常打开，而不是白屏。

同一项目随后通过固定提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 的公开 `gin.go` 文件完成真实导入：解析任务由 `queued` 进入 `succeeded`，写入 1 个 `text/x-go` 文档和 117 个带代码行定位的片段。浏览器刷新工作台后显示“已有 117 个可引用资料片段”“1 份文档”；检索“路由”返回最多 8 段主资料候选，并明确标注匹配理由与不会自动写入教材的边界。

这个过程还复现了 RabbitMQ 发布连接短暂不可用时的失败任务。恢复实现允许解析任务复用**同一冻结 payload**重试，既不重新读取文件，也不创建替代快照或变更固定 commit；后端领域测试和前端状态卡组件测试已覆盖该分支。

随后以重建后的 core-api 镜像在同一隔离环境对该失败任务执行一次重试。任务进入 `queued`、由 parser-service 消费并变为 `succeeded`；core-api 在读取完成结果时完成持久化。该项目资料库最终显示两份来自同一固定提交、但内容哈希不同的源码：`gin.go`（117 段）和 `routergroup.go`（40 段）。数据库也确认这两条 `SourceSnapshot` 共享 `resolved_version=73726dc606796a025971fe451f0aa6f1b9b847f6`，而各自保留不同 `content_hash`。这证明同一 Git 版本可安全导入多个独立文件，并且失败后的冻结重试不会覆盖或混淆已有快照；仍不构成完整官方文档栈、真实 Provider 或读者试学的验收。

## Gin 官方文档栈爬虫回归

2026-09-03，在同一可删除 Compose 环境中，对 `https://gin-gonic.com/en/docs/` 执行了受限的只读抓取，允许路径仅为 `/en/docs`。返回的 manifest 状态为 `complete`：保留 86 页，正文总量 7,283,550 字节，未触及 200 页或 32 MiB 预算。清单同时保留了 10,713 条被排除决定：其中 8,952 条为重复链接、1,296 条超出路径边界、452 条跨域、9 条 HTTP 404、2 条 HTTP 503、2 条不安全 URL。每个保留页面均含规范 URL、标题/标题树、内容哈希、HTTP 元数据和父链接。

这验证了 PR-06 的实际官方站抓取与“完整 manifest 不等于无限爬网”边界；随后同一能力已接入受确认来源的异步导入，结果见下节。抓取清单本身仍不等于教材的批准稿、运行时证据、真实 Provider 输出或读者试学。

## Gin 官方文档导入回归

2026-09-03，在可删除的 Compose 隔离环境（本机网关 `18084`）中，先登记并明确确认 `https://gin-gonic.com/en/docs/` 为官方补充来源，再以允许路径 `/en/docs` 创建官网资料导入任务。任务 `d3545a77-5492-4fd9-bb28-41483da50c6a` 由 `queued` 经 `running` 进入 `succeeded`，没有重试或错误。

核心服务只接受 parser-service 返回的完整 manifest 和已解析结构化页面：它重新校验 HTTPS、同一主机及路径边界，然后以 manifest hash 固定不可变快照。数据库确认该来源得到一个 `resolved_version=sha256:529a85b4539f480cde5dd33e23b7593fd37947f6c3f457d4eb5bcf68a3b9b39f` 的 `SourceSnapshot`、87 个 `text/html` `SourceDocument` 和 11,647 个结构化 `SourceChunk`。网页中的脚本、样式和嵌入内容不会作为教材证据进入分块；浏览器只获得文档元数据和片段计数。

工作台现在对已确认官网来源显示可编辑的“允许抓取的同站路径边界”和“抓取并导入官网资料”动作；该动作创建异步任务，完成前不把网页当成可引用证据。此验证证明受限官方站抓取到不可变快照的纵向链路，不证明所有 Gin 文档均无缺失，也不替代来源权利审查、教材批准、真实 Provider、读者试学或出版验收。

## Playwright 工作台复测

2026-09-03，使用 Playwright 经本机 `18084` 网关打开项目页并点击“打开工作台”。这是实际 Compose 前端与网关响应，不是组件 fixture：页面显示 2 个已登记来源、`https://gin-gonic.com/en/docs/` 的官方补充来源、允许路径边界 `/en/docs`、异步“抓取并导入官网资料”入口，以及“已有 11804 个可引用资料片段”和 89 份文档的资料库。该复测只验证浏览器中的资料工作台与已持久化数据可用；没有再次触发抓取任务，亦不把 UI 展示测试误记为教学代码的浏览器运行证据。

同日，`TestVerifiedBrowserRuntimeEvidenceRequiresCapturedPlaywrightBundle` 规定：任何标为 `verified` 的 `browser_page` 运行证据都必须保留受控本地页 URL 与最终 URL、截图引用、至少一条 DOM 断言、显式 console 收集结果和本地网络摘要；缺失任一项会被领域合同拒绝。既有 Chromium 工作台用例也已按该结构更新并通过。此项是防伪的持久化门禁，尚不是实际教学工件的浏览器执行器：当前 `course-runner` 只支持 Go 工件，且隔离预检未通过时仍保持 `unverified`。

后续实现已把受限 `browser_page` 命令、Gin 的 loopback 教学页、Bubblewrap 内 Playwright probe、独立浏览器运行证据和内容寻址 PNG 截图存储接入代码与 Compose 配置。已验证状态下，截图字节会先写入内容寻址视觉资产存储，再与同一 `RuntimeEvidence` 绑定为 `ManuscriptAsset`；该记录含证据身份、截图哈希、受控来源、用途及待核验权利状态，不能由未经验证的探针输出创建。2026-09-04 已在用户授权后成功构建新 `course-runner` 镜像：运行用户为 `runner`，Node 为 `v24.18.1`，Bubblewrap 为 `0.9.0`，并包含锁定的 Playwright probe、模块及浏览器目录。为避免错误启用，先执行了与服务启动完全相同、仅运行 `/bin/true` 的 Bubblewrap 预检；Docker Desktop 内核仍返回 `No permissions to create new namespace, likely because the kernel does not allow non-privileged user namespaces`。同日还评估了 macOS 15.7.3 的 `sandbox-exec`：最小 `/usr/bin/true` 预检可运行，但在禁止默认与网络、只允许临时目录写入的 profile 下，Google Chrome 152 的 headless PDF smoke 以 `SIGABRT` 退出且未创建 PDF。诊断显示 Crashpad 强制访问用户 Chrome 配置；即使 `HOME`、`TMPDIR` 与 `--user-data-dir` 均固定到受控临时目录也未改变结果。没有为让其启动而放宽用户配置访问；临时目录已删除。2026-09-04 又以同样的 deny-default、deny-network 和受控读取/临时写入边界启动 Playwright Chromium headless shell，其 `--version` 即以退出码 134 失败；因此不能把该 shell 在未受该 runner 隔离时能渲染 PDF 的事实，外推为课程运行器的安全执行证据。最后，用非 root 用户运行的独立 Playwright 容器也在 `--network none`、只读根文件系统、`--cap-drop ALL`、`no-new-privileges` 与临时 `/tmp` 下复测：Chromium 直接报告 `No usable sandbox` 并中止，没有生成 PDF；容器因 `--rm` 自动删除。两种替代运行器均不能在当时的默认配置下满足安全合同，因此没有创建任何已验证的浏览器页证据。

2026-09-06 的后续诊断确认失败点是 Docker 默认 seccomp，而非 user namespace 完全
不可用。受审外层 profile 与工件前内层 BPF 已使固定 Go 沙箱通过；学习者 profile 已
启用。教学工件及 Playwright browser_page 开关仍保持关闭，尚未运行其恶意 fixture 或
创建教材 RuntimeEvidence，所以上述历史浏览器结论没有被错误改写为已验证。

00:22 再次执行固定本地页面预检后，确认 Playwright 在未显式要求浏览器沙箱时会向最终
Chromium argv 加入 `--no-sandbox`。probe 已改为强制 `chromiumSandbox: true`，并在教材
开关打开时作为启动门禁。当前 Docker Desktop 中 Chromium 随后报告 sandbox 不可用，
所以 browser executor 不注入；Go executor 已用生产路径固定编译/测试预检通过，仍可逐
命令运行，浏览器命令和工件整体保持
unverified。本地教材 Go 开关已显式启用，core-api/course-runner healthy。真实库没有
批准代码工件，因而未保存任何运行证据；Gin 教材的 browser_page 验收仍未完成。

## 静态前端门禁复跑

2026-09-03，现有锁定依赖下依次执行了 `npm run lint`、`npm run deadcode`、`npm run test:coverage`、`npm run build` 和 `npm run check:bundle`，全部以退出码 0 完成。Vitest 报告 63 个测试文件、218 个测试通过；V8 报告 statements/lines 为 44.55%、branches 为 72.93%、functions 为 62.76%。该数值是当前测量结果，计划没有为其定义发布门槛，不能据此宣称测试覆盖充分。生产构建完成，入口压缩后为 154,634 bytes，符合现有包体检查脚本。Mermaid 安全回归中 JSDOM 缺少布局 API 的已知诊断输出仍出现，但对应 6 个防注入测试均通过；它不替代真实浏览器中的图表版式验证。

## 后端集成合同复跑

2026-09-04，执行 `GOCACHE=/tmp/inkwords-go-build go test -tags=integration ./integration -count=1 -v`，4 项合同全部通过：异步任务流水线、导出工件、Obsidian 适配器往返，以及压缩包路径穿越边界。这些是跨域合同测试，不代表真实 Provider、浏览器运行证据、PDF 排版或读者试学已经完成。

同日，`GOCACHE=/tmp/inkwords-go-build go test ./... -count=1` 全部通过，包括 core 与 review 的 PostgreSQL migration。review migration 容器原本创建 `inkwords_review_migration_test`，但 readiness probe 固定连向 `inkwords_migration_test`，使数据库已经就绪时仍被误判为超时；现 probe 由当前测试容器的数据库名生成。该修复提升的是迁移回归可靠性，不是生产数据迁移或备份恢复演练的替代证据。

## v12 真实生成超时回归

2026-09-06，用户确认预检后只提交了一次 foundation Gin 样章真实生成。冻结目标为
`deepseek / deepseek-v4-flash`，prompt schema 为 `inkwords.textbook.sample.v12`，质量合同为
`inkwords.sample-quality.v9`，证据 6 段，保守估算输入 `10,191 / 20,000` Token，预留输出
12,000 Token，确认哈希为
`sha256:1d8f74560f7cb0230957c6053865061b960f0da5139f22ef1a7d5ab28fa838b9`。任务
`ff0f6e35-c350-4351-be33-69564ff88d16` 在 45.02 秒后失败，错误为读取 Provider 响应时
`context deadline exceeded`。数据库确认该任务重试数为 0、候选修订为 0；没有完整响应用量时
不伪造 Token 或费用记录，Provider 是否对未完成请求计费保持未知。

失败时间与教材 DeepSeek 适配器写死的 45 秒边界一致。教材 worker 现从
`TEXTBOOK_GENERATION_REQUEST_TIMEOUT` 读取独立边界，默认 `15m`，只接受 `30s`–`30m`，并同时传给
DeepSeek/OpenAI 教材适配器；复习评分等其他调用仍保留原有超时。聚焦测试、全量 Go 测试、
Compose 配置和 diff 检查通过；重建后 `llm-stream` 容器实际读到 `15m` 且健康。没有自动
发起第二次 Provider 调用；v12 仍没有当前合格候选，必须另行确认同一冻结任务的显式重试。

同日补齐失败任务的真实重试门禁。`GET /api/v1/textbook-projects/tasks/:taskID` 只为可验证的
失败样章任务返回冻结重试摘要；任务 `ff0f6e35-c350-4351-be33-69564ff88d16` 的摘要与首次预检
一致：`deepseek / deepseek-v4-flash`、输入 `10,191 / 20,000` Token、输出预留 12,000 Token，
并绑定同一输入哈希。空确认和错误哈希的 POST 均返回 409，任务仍为 failed、重试 0。
重建后的真实浏览器先显示“查看重试影响”，展开后才显示“确认重试一次”、费用未知和
“成功后也只生成待审候选”；控制台 0 错误、0 警告。界面截图保存为
`output/playwright/generation-retry-confirmation.png`。验证没有点击最终确认，因此没有第二次
Provider 调用，也没有候选、审批或学习工件变化。

## 核心浏览器回归复跑

2026-09-04，执行 `npm run test:e2e`，Chromium 与 mobile-Chromium 合计 13 项核心 E2E 通过、1 项按测试定义跳过。通过范围包括工作台主导航与移动可达性、到期学习会话及服务端决定的下一任务、失败后的有界手动重试、候选稿未经明确批准不得导出，以及自动运行观察与 IDE 人工采集清单的边界。PR-13 版本真实性与自动截图标签补齐后，又定向复跑 `textbook-runtime-evidence.spec.ts`，Chromium 1/1 通过；重建镜像内核对到 Playwright 1.62.1 与 Chromium 151.0.7922.34。该套件使用本地 mock/合同数据验证浏览器交互；它不等于课程运行器已在受控 sandbox 中采集到真实教学页面证据。

同日，首次 `npm run test:e2e:smoke` 的 Chromium 用例通过；Firefox 和 WebKit 在启动前失败，原因是本机缺少其 Playwright 浏览器二进制，而非应用断言失败。失败运行创建的 Git 忽略 trace 目录已删除。

随后用户明确授权下载 Firefox 与 WebKit Playwright 浏览器。下载完成后重跑相同 smoke，Chromium、Firefox、WebKit 各 1 项 `@cross-browser` 工作台渲染与主视图导航断言通过（共 3 项，9.8 秒）。这证明该最小交互在三种浏览器引擎均可运行；它仍不替代课程运行器的受控 sandbox 运行证据或 PDF 排版验收。

## DeepSeek 样章与新版证据蓝图回归

2026-09-04，在用户明确配置并授权本机使用 DeepSeek API 后，真实生成任务 `b1a1e2c4-6cc5-40dc-af72-858a8a217d81` 由 `deepseek / deepseek-v4-flash` 完成候选稿 r1。冻结遥测记录输入 2,503 Token、输出 3,937 Token、缓存 1,024 Token、延迟 30,719 ms；费用字段没有供应商依据时继续保持未知。人工复核没有批准该稿：它把“请求到达时如何查找处理函数”建立在注册链证据上，并把直接调用 Gin 的示例误作“从零教学实现”。

为阻止 r1 被误用，当时的样章质量合同先升级为 `inkwords.sample-quality.v5`：候选稿必须分别引用 `RouterGroup.GET`、`RouterGroup.handle`、`Engine.addRoute`、`Engine.ServeHTTP`、`Engine.handleHTTPRequest` 与 `node.getValue` 六段固定提交证据；教学实现必须独立实现最小路由树与测试，不得导入 Gin，并明确非生产用途、遗漏边界与 `go test ./...` 验证方式。质量报告携带合同版本，缺失或旧版本的候选稿同时被前端与 core-api 拒绝应用。

本机 Compose 以真实 Obsidian vault 路径重建后，浏览器载入同一固定提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 下的 `routergroup.go`、`gin.go` 与 `tree.go` 三个不可变文件快照。数据库确认对应片段数为 3、2、1。浏览器创建蓝图 r2 草稿，保留同一关键事实并绑定全部六段证据；页面重载后关键事实与六个勾选项完整恢复。r1 仍为已批准的旧蓝图，r2 保持 `draft`，没有代替用户点击批准，也没有基于未批准 r2 发起新的付费生成。章节编辑器对旧候选稿显示“规则已过期”，应用按钮不可用。

用户明确批准后，蓝图 r2 已由本地审批接口变为 `approved`。随后只提交一次真实任务 `97fdd392-16dd-49d4-af86-ec9595625a84`，由 `deepseek / deepseek-v4-flash` 完成候选稿 r2：输入 2,939 Token、输出 3,863 Token、供应商调用 1 次、延迟 25,643 ms、本地缓存未命中、重试 0 次，费用在没有版本化定价依据时保持未知。结果绑定蓝图 r2、当前 BookContract/StyleSheet 与六证据包，`inkwords.sample-quality.v5` 硬门禁通过。

真实本机浏览器随后显示候选稿 r2、逐行差异、自动质量门禁、八维人工审阅提示、生成谱系和使用量。章节仍无 approved revision；未获取编辑锁时“应用候选稿”保持禁用。本轮没有替用户应用、批准或驳回候选稿。当前人工验收点改为核对六段证据、从零教学实现、测试、技术准确性与可自学性，再明确选择应用或驳回。教学工件仍标记 `unverified`，不能把自动质量门禁或真实 Provider 调用称为代码已经在受控 sandbox 中运行。

人工技术复核随即发现 v5 门禁仍漏掉两个硬问题：候选正文声称从零教学实现会按 method/path 匹配，但其 `addRoute` 与 `findRoute` 只接收 path；从空目录开始的步骤直接运行 `go test ./...`，没有先执行 `go mod init`。回归测试先复现旧门禁误放行，再把质量合同升到 `inkwords.sample-quality.v6`，分别要求两函数显式包含 method/path 参数，以及冷启动步骤初始化 Go module。Provider 请求同时消除写死“零基础”的偏差，为 foundation、programming、stack_familiar 输出不同读者任务目标；请求文本变更使 prompt schema 与缓存身份独立升到 `inkwords.textbook.sample.v7`。相关 Go 全量测试通过；随后补齐前端 `pending`/`streaming` 任务状态并区分服务端任务响应与仅含 `task_id` 的会话恢复引用后，前端 63 文件/229 测试、lint、构建和包体门禁通过。

顺序重建 core-api、llm-stream 和前端后，真实浏览器仍保留 v5 候选稿及其使用量以供审计，但明确显示“规则已过期”，应用按钮禁用。本次用户授权的一次 DeepSeek 调用已经使用；没有自动发起第二次付费调用。要得到可进入人工应用流程的新稿，需要用户另行授权一次 v6 生成。

同一真实项目的服务端阶段投影也已消除旧占位文案：当前样章阶段明确指出现有候选稿不符合当前合同，需要按 v6 重新生成；在没有批准修订时，验证、六维学习和冻结待审构建均为 `blocked`，而不是误报“进行中”或“尚未接入”。领域回归覆盖旧候选、当前候选和批准修订三种状态，重建 core-api 后由本机 API 与浏览器共同确认。

用户随后再次明确授权一次真实生成。任务 `3145fcae-69d0-4a9b-bf13-305531d438ee` 由 `deepseek / deepseek-v4-flash` 成功完成，并以 prompt schema `inkwords.textbook.sample.v7` 创建候选稿 r3（revision `23d36449-a668-4609-9d7d-ca3debae27b7`）。冻结遥测为输入 2,969 Token、输出 4,360 Token、供应商调用 1 次、延迟 27,165 ms、本地缓存未命中，费用继续保持未知。`inkwords.sample-quality.v6` 自动硬门禁通过，并给出“完整示范与机制对应较弱”和“多步调用适合流程图”两条人工软提示。

真实本机浏览器确认 r3 是当前候选，样章阶段显示“等待人工审阅”，章节仍没有 approved revision；未取得编辑锁时应用按钮禁用，验证、学习和出版保持 `blocked`。人工技术复核没有批准该稿：其 `main_test.go` 导入了 `net/http/httptest` 却未使用，按 Go 规则无法编译。这证明 v6 结构门禁已修复 method/path 与 `go mod init` 缺口，但仍不能替代编译级验证。候选继续标记 `unverified`；本轮没有应用、批准、驳回或发起额外 Provider 调用。

针对该缺口，新增回归先证明 v6 会放行未使用 import，再实现不执行候选代码的 Go 源码检查：两个 `go` 代码块必须分别形成可解析的 `package main` 源文件和测试文件；导入必须在对应文件中作为包限定符实际使用，点导入被拒绝。质量合同因此升级为 `inkwords.sample-quality.v7`，Provider 指令和缓存身份升级为 `inkwords.textbook.sample.v8`。该检查只覆盖语法、文件布局和 import 使用，不能表述为代码已经编译或运行；真实 `go test` 仍必须由受控教学工件运行记录证明。

全量 Go 回归与前端 63 文件/229 测试、lint、生产构建、包体预算通过后，core-api、llm-stream 和前端镜像已顺序重建。本机 11 个 Compose 服务无不健康项；真实 API 与浏览器均把 r3 标记为“规则已过期”并禁用应用，同时保留其正文、六证据谱系和 DeepSeek 使用量。样章阶段回到“可开始”，验证、学习和出版保持 `blocked`。没有自动驳回，也没有发起新的 Provider 调用。

用户再次明确批准后，本机只创建一次真实 v7 生成任务 `6a2244ec-9235-4f4a-af6e-fbbdee1ddba3`。任务在约 32.5 秒后被硬门禁拒绝，唯一失败为 `hands_on_procedure_missing: 打开`；没有写入候选修订，也没有代替用户批准或应用。现有失败持久化只保留脱敏错误文本，未保留这次被门禁拒绝输出的 Provider Token/延迟元数据，因此不伪造使用量。

该失败的其余 v7 检查均已通过，稿件已含“新建”和“执行”；旧风格探针却额外要求字面“打开”，与从空目录新建项目的有效步骤重复且过度词面化。回归先固定“只新建、不出现打开”仍应通过，再将规则改为“新建或打开至少一个，且必须执行”。这是兼容性变更，所以质量合同升到 `inkwords.sample-quality.v8`，Provider 指令与缓存身份升到 `inkwords.textbook.sample.v9`。修正和测试本身不会再次调用 Provider。

同日继续完成 PR-10 的软审阅合同：质量报告始终列出句子与段落负担、标题承诺、重复、语气、例子相关性、章节节奏、图示机会和练习梯度八个人工维度；确定性检测器只在命中可解释规则时附上风险代码、原文摘录和审阅问题。软提示不计分、不改变硬门禁 `passed`，即使没有命中也不能显示为人工审阅通过。章节编辑器单独展示这组提示，并以组件测试证明含软提示的当前质量合同候选稿仍可进入人工应用动作。

候选稿人工驳回已形成独立的 append-only 审阅记录：必须持有当前章节编辑锁并提交 8–2000 字符理由，记录冻结候选稿内容哈希、质量合同版本和本地审阅 workspace。同一理由重试幂等，既有理由不能改写；core-api 和章节编辑器同时阻止被驳回候选稿再次应用。该能力只保存用户实际作出的决定，不会自动替用户驳回现有 r1，也不把自动软提示冒充人工结论。

在批准 r2 之前的 Compose 重建检查中，迁移表 `textbook_candidate_reviews` 已存在且保持 0 行。真实浏览器显示人工驳回理由输入与禁用状态：未取得当前章节编辑锁时，即使输入有效长度理由也不能提交；旧候选稿继续显示“规则已过期”，应用按钮不可用。该检查当时控制台 0 错误，蓝图为 r1 `approved`、r2 `draft`，且没有写入驳回或触发 Provider 调用。此后发生的 r2 明确审批和单次 v5 生成以本节前两段的新记录为准。
