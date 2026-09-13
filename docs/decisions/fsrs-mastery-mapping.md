# ADR：FSRS 与六维掌握度的职责边界

- 状态：接受
- 日期：2026-09-03
- 关联：教材平台 PR-14

## 决策

`go-fsrs/v4` 只计算下一次复习时间。InkWords 自己的六维评分仍根据正确性、独立性、提示、耗时、错误类别和信心度决定下一项练习是 explain、complete、reproduce、transfer、diagnose 还是 retain。

每次 `mastery_attempts` 只追加一条表现记录。写入 schedule 时，从最早尝试开始重放 FSRS 卡片：失败映射为 `Again`，带提示、低信心或超时的正确回答映射为 `Hard`，独立且稳定的正确回答映射为 `Good`。模型只提供逐项反馈；用户明确应用后，由确定性规则解释评分，FSRS 计算日期，原作答不改写。

## 版本与升级

`mastery_schedules.algorithm_version` 记录 `fsrs-v4-default-parameters`；它是可丢弃的查询投影，权威事实是追加式尝试日志。升级 FSRS、参数或评级映射时必须：

1. 创建新的明确算法版本；
2. 通过迁移或受控重放重建所有 schedule；
3. 保留原始 attempts，不原地修改；
4. 用固定历史 fixture 比较升级前后 due date，并在发布说明中解释变化。

因此，算法升级不会丢失学习证据，也不会改变六维任务类型的选择规则。

## 作答与自评证据（2026-09-05）

`mastery_attempts.answer` 保存用户提交的原始文本，最多 20,000 个 Unicode 字符，
不得把正文当指令、渲染为可执行 HTML 或执行其中代码。空值表示旧接口或历史记录没有
提交作答，不能从 correct 字段推断答案。自评结果仍按原调度算法解释，保存文本本身
不提高掌握等级；模型评分、逐项反馈和用户纠正尚未接入。

`GET /api/v1/mastery/objectives/:id` 在检查当前 workspace 后返回冻结 objective 与
最近 20 条 attempts；来源、rubric 和先修在作答前可见，关键点提示由用户主动查看。
`POST /api/v1/mastery/objectives/:id/attempts` 增加可选 answer，旧客户端保持兼容；
新学习界面要求填写正文并与自评一同提交，保存失败不清空表单。

迁移 `00027_mastery_attempt_answer.sql` 只增加文本列与长度约束，不增加索引：正文
不参与到期检索或筛选，写入/存储成本随实际作答长度增长。读取沿用 objective 主键及
objective_id 尝试索引。回退旧应用二进制可保留此列；若已有非空正文，Down 会拒绝
删除证据，需保留新 schema 或按备份恢复流程处理，不能强制丢弃作答。

## 逐项评分合同与人工纠正（2026-09-05，接入进行中）

`inkwords.mastery-assessment.v1` 的输入必须绑定 attempt、objective、批准 revision、
具体题目、作答、结构化 rubric 与选取的来源原文。现有 LearningArc 的阶段目标不是
逐项评分 rubric，success_evidence 标识也不是可供核对的来源正文，不能直接替换这些输入。
输入及请求各自有哈希，模型、输出 schema、指令或证据改变不能冒用原评分身份。

评分维度依任务类型固定：解释/保持检查准确、完整、因果、边界与清晰度；补全/复现/迁移
检查正确、完整、运行、测试与设计；诊断检查假设、定位、证据、根因与修复验证。
每项可按具体题目细化描述，不能省略必需维度。分数为 0–4，null 表示未知而非 0。
数字正分需要作答原文引文；所有评分及反馈需引用提供的证据 ID。没有与被评工件哈希
匹配的 VerificationRun 证据时，运行、测试及修复验证项必须未知。用户在正文中声称
“运行通过”没有执行证据权威。这里的自动校验能验证引用身份与文本一致性，不能替代
对语义判断正确性的人工/代表性模型评测。

Provider-neutral 适配器位于 `review-service/app/masteryassessment`，只使用共享
Generation Port，不导入 llm-stream 或 Provider SDK。每次至多调用一次，并发上限 1、
超时 60 秒、输出上限 3,000 Token；以请求 UTF-8 字节数不超过 32,000 作为保守输入门槛，
超过时拒绝且不截断。不自动重试或切换模型，未知用量保持未知，Provider 错误正文不外露。
无效反馈保留可用遥测但不成为有效评分；未知字段（包括模型自报 mastered）被拒绝。

`AssessmentCorrection` 记录纠正者、时间、理由、原输入哈希、前一反馈哈希及逐项改动。
重放产生新的有效视图，不覆盖自动反馈；陈旧、跨作答或重复纠正被拒绝。人工纠正也不能
伪造缺失的运行证据。未来存储层必须从认证 workspace 派生纠正者，并在事务中完成 CAS。

领域合同、适配器、权威输入、下述持久化评分任务/纠正事务和 UI 已接入并通过受控测试。
已将用户明确应用的评分重放到 FSRS 调度（下述 v31）。2026-09-06 三份代表文字作答的真实
Provider 测试已通过既定门槛；代码运行评分、真实学习者与纠正/应用验收仍未完成。

### 2026-09-06：四份因果对照与校准限制

引用纠正 UI 后续补齐：每项直接展开所引原文，纠正可重新选择本次冻结 evidence_ids，
不再强制沿用模型引用。运行数字分要求实际选中对应运行证据，保存失败保留选择；服务端
原有 CAS 与引文/来源验证不变。真实临时 PostgreSQL 证明纠正引用读回和原引用保留；
269 项前端测试及部署产物的受控浏览器流程通过。详见
`docs/qa/mastery-correction-evidence-2026-09-06.md`，不计作真实读者或模型语义验收。

后续尝试增加 rubric 范围和显式表达约束，共新增六次真实调用：第一批在第二例因果
上限失败后停止，第二批分数通过但仍有完整性理由越界。这两版候选指令未部署且最终
从生产代码撤回，仅在验收目录保存实验源码；当前代码和服务仍为下述首批版本。真实验收产物增加待语义复核状态，离线预检与真实
调用共用同一答案；不能将测试 PASS 当作理由准确。详情及失败结果见上述 QA 记录。

在下述三例暴露问题后，增加原流程答案与追加因果解释答案的对照，补入固定源码
`(methodTrees).get` 证据，并约束高因果分必须由作答引文本身说明依赖或改变条件的结果。
新指令进入请求 hash；真实验收入口改为四次串行调用，并在分数门槛失败前保存结果。
此次真实 DeepSeek 评分平均分依次为完整 4.0、流程 2.8、遗漏 0.6、误解 0.0；
完整/流程因果分分别为 4/2，共输入 8006、输出 3763 Token，无重试，费用未知。

固定对照的因果区分改善，但流程答案其他条目的理由新增未要求的未命中 handlers 细节，
仍需约束和复核评分理由。详细输入、运行、部署证据与限制见
`docs/qa/mastery-causal-calibration-2026-09-06.md`。没有真实学习库作答或掌握状态变更，
没有完成真实学习者代码评分、用户纠正/应用或延迟试学；PR-14 保持未完成。

### 2026-09-06：三份真实评分结果及校准限制（此前批次）

用户授权后，使用 DeepSeek/deepseek-v4-flash 对完整、遗漏、误解三份文字解释夹具各调用
一次，平均分分别为 4.0、0.6、0.0（满分 4）；遗漏样本指出 3 项遗漏，误解样本指出
3 项遗漏和 2 项误解。三次共输入 4,620、输出 2,659 Token，测试耗时 13.92 秒。
验证后的逐项反馈、答案引文、证据 ID、请求 hash 和遥测保存于
`output/real-acceptance/2026-09-06/mastery/`，均标识为 automated。

这证明当前模型在三份固定文字样本上通过预设合同和区分门槛，未证明六类任务或代码运行
评分准确。自动复核发现完整样本 causality=4 所引原句仅直接给出方法选树与路径查找的
先后顺序，理由却称已解释“为何”；仍需增加流程复述与因果解释的对照校准，不能把平均分
通过当成所有条目可靠。没有向真实学习库写入作答、纠正、评分应用或掌握状态。
缺少学习者工件/VerificationRun 的运行条目保持未知。PR-14 仍未完成。

### 2026-09-05：显式学习者 VerificationRun（review schema v33）

用户已明确接受冻结学习者 Go 文件的受限来源范围。预检只读，只有用户点击“明确开始
验证”才创建任务；同一 `request_id` 幂等，重试必须引用同一次作答的终态任务。任务只
向 course-runner 发送一次性引用，令牌明文不入库、不进公开 JSON，消费后不能再次解析。
course-runner 从 review-service 重读原作答、批准题目和完整计划，不读取 review 数据库。

计划和报告绑定固定 runner 镜像、实际 Go 工具链、离线 `go test -count=1 -mod=readonly
-trimpath ./...`、环境白名单、无网络、非 root、内存/PID/文件/输出/CPU/超时限制。缺少
`go.mod` 时只注入计划中列明并进入执行树 hash 的最小本地模块文件。临时目录只含冻结
字节和派生文件，完成后删除；引用不能携带路径、命令、环境、源码或执行结果。

v33 记录 queued/running 与 passed/failed/timed_out/cancelled/unavailable/runner_error/
interrupted 的不同语义。启动恢复只把未完成任务标为 interrupted，不自动重放；Down 在
存在运行证据时拒绝。报告写库前再次核对原输入、固定策略和状态，学习者自写测试通过也
不自动成为题目正确或掌握证据。评分 v3 只读消费同一作答最新的 passed、failed 或
timed_out 报告；输入 hash 绑定原快照和受限报告投影，输出为评分预算显式截断并保留原
字节数/截断标识。取消、runner_error、unavailable、interrupted 或不匹配报告均不成为
评分证据，运行类条目继续保持未知。模型调用前 UI 显示将使用的运行 ID 和状态。

2026-09-06，本机 course-runner 已通过双层 seccomp、固定离线 Go 编译/测试、非 root、
只读源码、宿主路径隐藏、禁外网、资源上限和嵌套 namespace 拒绝预检；能力接口返回
available，学习者开关已显式启用。预检只执行操作方固定代码，尚未运行真实学习者作答或
创建 VerificationRun。完成的自动合同和持久化链不能替代代表性真实模型评分、真实读者
练习或延迟 retain；PR-14 仍未完成。

### 2026-09-05：学习者源文件进入静态评分（评分合同 v2，schema 仍 v32）

本节记录 v32 当时的静态评分状态；后续来源范围已获用户确认并进入上面的 v33
显式运行生命周期。具体边界见 [学习者代码执行提案](learner-code-execution-scope.md)。

含代码作答现从持久化快照组装完整 `learner_artifact`，替代上一切片的临时 409
门禁。它只能由 review-service 按作答读取并校验 workspace、会话、批准 revision、
题目、时间与文件 hash；浏览器评分请求仍只传作答 ID 和已确认的预检 hash。
评分任务入库再次对照原 attempt 的 `learner_artifact_hash`；遗漏、换文件或跨身份
输入不会进入 Provider。应用重放也核对同一快照与原练习，不能把另一个工件的建议
应用到该次作答。学习者文件仍不属于教材母稿工件，不进入教材导出。

`AssessmentInput.ContractVersion()` 对无文件输入保持 `inkwords.mastery-assessment.v1`，
对带文件输入使用 v2。新增 JSON 字段为空时省略，旧输入、反馈和纠正 CAS 的固定
哈希回归保持一致；不能直接改全局 v1 常量。任务终态按自己的输入版本保存，失败
或取消但未调用模型的结果也保留对应版本。无需新增数据库列或迁移。

Provider 请求将源文件作为独立 `assessment-learner-code` 数据项发送一次，正文不
进入系统指令或任务元数据，不截断文件；原文件、注释和作答均无指令或执行权威。
v2 每个条目必须返回 `answer_path`：空字符串表示文字答案，否则必须是实际文件。
`answer_quote` 逐字匹配该文件原 content，不接受不存在文件、拼接引文或 JSON 转义
后的替代文本。文件数据项不能充当 `evidence_ids` 中的权威来源。

输入仍受完整请求 32,000 UTF-8 字节预算约束，超过时在预检和调用前拒绝，不自动
缩减代码、扩预算或补发调用。已保存的代码可以比模型预算大；保存能力与一次模型
请求容量不同，用户可保留代码并另开范围更小的练习。输出上限 3,000 Token、单次
调用、并发 1、60 秒超时、失败用量未知/已知的处理不变。

此版本仅静态评阅文件。`ArtifactHash` 必须为空，不接受教材参考实现的 runtime
证据；运行、测试和修复验证项均须 null。显式应用后仍由已有确定性映射将未知保持
为 unverified，不凭自称“测试通过”增加掌握证据。执行准入与 learner VerificationRun
尚未接入；PR-14 与真实读者验收保持未完成。

UI 展示文件路径和原文引文，纠正表单可选择文字或固定文件；切换位置清空待编辑
引文。纠正重放使用新值替换整条记录，避免空路径因 JSON omitempty 保留旧文件。
评分、纠正、应用后刷新保留原文件和模型结果，主动准备评分不调用模型。

隔离 PostgreSQL 覆盖从真实作答保存到权威输入、受控 Provider、任务终态、代码
纠正和显式应用的全链，拒绝换码/漏码请求。新的准入查询 EXPLAIN 复用
`idx_mastery_attempts_objective_time` 与 `uq_mastery_objectives_workspace_chapter_active`，
按文件 hash 过滤；不新增索引。存储成本增加任务和应用中的冻结代码 JSON（各次
受模型预算限制）。评分存储关闭含原文参数的 SQL 日志，避免写入错误泄漏代码。
旧二进制无法理解 v2 输入哈希；已有代码评分后应暂停评分/学习写入并保留 v32
schema，优先向前修复，不能丢弃评分或代码快照来回退。

### 2026-09-05：随原作答冻结代码（review schema v32，历史切片）

具体题目的 `POST .../attempts` 可同时提交 `code_files: [{path, content}]`。
代码必须与首次作答原子保存，不能事后附到旧练习并沿用过去的耗时、帮助或保持间隔。
客户端不能提交执行命令、磁盘目录、验证状态或文件哈希；未知字段被拒绝。
当前仅接收 `.go` 与根目录 `go.mod`，至少一个 Go 文件，最多 32 个文件、单文件
128 KiB、总计 256 KiB。路径必须为受限相对路径，拒绝重复、穿越、隐藏文件、依赖
目录及同一路径既是文件又是目录。内容须为非空 UTF-8，无 NUL；CRLF 与 BOM 不改写。

`inkwords.learner-artifact.v1` 冻结 workspace、objective、attempt、session、批准
revision、题目 ID/hash、练习维度、服务器保存时间和按路径排序的文件。快照哈希绑定
全部这些字段；它是提交身份，不是执行器的文件树 hash，也不证明测试成功。
含文件的提交使用 `inkwords.practice-submission.v2` 将原提交摘要与文件摘要绑定；
无文件提交保持 v29 原哈希。仅调整文件顺序的重试返回原结果；改动或移除已保存
文件返回 409，须开始新练习。保存快照失败会回滚作答、会话完成与调度。

`GET /api/v1/mastery/objectives/:id/attempts/:attemptID/code` 按当前 workspace 校验
目标、作答与完整快照身份；历史无文件返回空结果。普通最近作答只带快照哈希，
展开该次历史时才请求代码，按纯文本展示；丢失提交响应后可刷新恢复原字节。
快照入库使用参数化 SQL，并关闭该含原文写入的 SQL 错误日志，避免故障时泄漏源文件。

v32 增加 `mastery_attempts.learner_artifact_hash` 与只追加的
`mastery_learner_artifacts`。读取只用 attempt 主键，不另建全文或路径索引；隔离
PostgreSQL EXPLAIN 使用 `mastery_learner_artifacts_pkey`。成本为每次带代码作答
新增一条 JSONB 快照及主键记录。Down 在任何快照或非空 hash 存在时拒绝；回退应用
应暂停学习写入并保留 schema，优先向前修复，不可删除原代码证据。

本切片只负责原代码的持久化与读回。尚未接入服务端执行清单准入、学习者隔离执行、
VerificationRun 或模型对文件原文的引用。含代码的评分预检直接返回 409，界面显示
“代码验证与评分尚未接通”，避免把只读取 answer 的 v1 评分冒充代码评测。
历史文字作答的评分流程不变；不能拿教材示例执行结果当学习者的运行证据。

### 2026-09-05：明确应用评分与调度重放（review schema v31）

`POST /api/v1/mastery/objectives/:id/assessments/:jobID/apply` 只接受幂等决策 ID、
`expected_feedback_hash` 与可选 `previous_id`（此作答上一次应用的决策 ID）。
用户先逐项核对，再点击“应用当前评分并更新复习安排”；该操作不调用模型。
服务端从成功任务及纠正日志冻结本次有效输入/反馈，不接收浏览器自报的分数、
身份、练习时间或复习日期。自动反馈、纠正、评分应用和原作答分别保留。

映射版本为 `fsrs-v4-assessment-v1`，阈值是本地可审查的调度策略，不是模型可修改
参数或经过真实学习效果实验验证的结论：

- 所有评分条目至少 3/4 才派生为 passed；原独立性、提示、耗时、自信仍决定 Hard/Good。
- 任一已知条目小于 3/4 派生为 needs_practice，FSRS 使用 Again。
- 无已知不合格项但存在 null 时为 unverified：六维得分增量为 0，FSRS 使用 Hard；
  未知不当作零分或失败，也不构成掌握证据。运行项缺少本次工件证据时仍为 null。

v31 的 `mastery_assessment_applications` 保存连续 objective sequence、作答与任务
身份、应用者/时间、前一决策、版本及完整输入/反馈快照与 hash。重放将该作答的最新
已应用解释放回原练习位置；不会新增一次练习，也不把应用时间当作延迟回忆时间。
后来追加的纠正仅更新建议，必须重新应用才影响调度。UI 可核对当前有效评分、已经
应用的评分快照和当前安排；提交会话返回的历史安排明确标为“作答保存时的安排”。

应用事务先锁 objective，再锁 job，核对反馈 hash、前一应用及 workspace 后追加
事件，并在同一事务重建 schedule。普通作答提交也在同一 objective 锁内重读应用
事件，所以即使该作答在应用之前开始准备，也不会把 schedule 恢复成原自评结果。
重复请求返回当前持久化视图；已应用的旧请求在较新纠正/应用后重试，不重新生效。
原 attempt 的 correct、answer、提示、时间及 job 原始自动分数不改写。

历史读取按 attempted_at/created_at/id 固定排序。新索引用于 objective/sequence
重放和 objective/attempt/sequence DESC 的最新应用读取；隔离 PostgreSQL EXPLAIN
对后者使用 `idx_mastery_assessment_application_attempt` Index Scan。每次应用新增
一条快照及主键、顺序唯一、最新应用索引记录，并重放该目标历史；没有答案全文索引。
实际隔离测试覆盖并发同 ID 去重、旧版本/错误身份拒绝、纠正不自动应用、后续作答
保持评分解释、重放时间一致、原证据不变及非空 Down 拒绝。浏览器覆盖应用后刷新
恢复且 Provider 调用数不增加；这是受控评分夹具，尚非真实模型/读者验收。

v31 不改现有 schedule；没有应用记录的目标继续原算法，第一次应用才改为新版本。
Down 仅允许空应用表。已有应用后回退旧二进制时，必须暂停学习作答写入，避免旧
版本忽略应用日志重建调度；优先保留 schema 并向前修复，必要时按备份恢复流程
恢复整个一致时间点，不能只删除应用表或把 schedule 伪装成旧版本。

### 2026-09-05：持久化评分任务、恢复与用户纠正（review schema v30）

展开某条已保存作答时只读取评分状态；“准备模型评分”组装权威输入并检查预算，
“调用模型评分一次”才发起调用。请求必须包含随机 request_id 和预检返回的
input_hash/request_hash。提交重试复用 request_id；成功结果只在完整请求 hash 相同
时复用。请求 hash 覆盖 provider/model、指令、schema、证据、作答、rubric 和选项。
本次同时消除了指令允许空补救材料、而输出合同要求至少一条的矛盾；此变化进入
request hash，不冒用旧请求身份。

API 位于 `/api/v1/mastery/objectives/:id`，由稳定本地 workspace 解析身份：

- `GET /attempts/:attemptID/assessment-preview`：只读预检，无 Provider 调用。
- `GET /attempts/:attemptID/assessment`：最近一次评分，尚无任务时为 null。
- `POST /attempts/:attemptID/assessments`：提交 request_id、expected_input_hash、
  expected_request_hash；失败后的新调用必须显式指定 retry_of。
- `GET /assessments/:jobID`：恢复原输入、任务状态、遥测、自动反馈及纠正视图。
- `POST /assessments/:jobID/cancel`：先持久化取消，再中止本进程调用。
- `POST /assessments/:jobID/corrections`：提交纠正 ID、previous_hash、理由和逐项改动；可选 findings 用于追加纠正反馈与提示，规则见下节。
  身份与时间由服务端生成，未知请求字段被拒绝，最大请求体为 512 KiB。

任务状态为 running/succeeded/failed/cancelled/interrupted。输入、模型和请求身份先
保存，随后执行一次有时限的调用。原始结果和纠正日志分别保存；失败的模型正文不会
成为有效评分。丢失 HTTP 响应后读取既有任务即可恢复，不重新调用模型。模型结果
写库失败只重试写库三次，不重试模型；孤立任务或进程重启后标为执行结果未知，须
用户明确重试。Result 为 null 代表没有已知完成结果，不能展示为调用零次/零成本。
取消先于结果保存时不能再晋级成功，但可补存已知的调用次数与 Token 遥测。

一个 PostgreSQL session advisory lock 限定一个本地评分执行器，防止同时启动聚合
server 与 review-service 时互相中断活跃任务。它占用一个专用连接，连接丢失后不再
接收新调用；正常关闭释放锁，进程退出由数据库释放。任务创建另用短事务锁和一个
running 部分唯一索引控制一次在途调用；不把模型等待时间放进数据库事务。

v30 新增 mastery_assessment_jobs 和 mastery_assessment_corrections；作答、原始
自动反馈和自评调度不会被纠正覆盖。纠正在 job 行锁内检查有效反馈 hash，再按连续
sequence_no 追加；同一纠正重试只保存一次，陈旧版本返回 409。UI 保留失败草稿，
显示原始评分、用户纠正和引用原文；较晚返回的旧读取不会覆盖已保存的新纠正。
只有正在查看的评分任务进行有界轮询，关闭页面即停止；没有新增后台通知。

#### 2026-09-06：反馈与提示的追加式纠正

原接口只能改 criteria，导致不成立的 missing_points、misconceptions 或 next_hint
无法纠正。现在 CorrectionInput / AssessmentCorrection 增加可选 `findings`，其中
必须同时提供 `correct_points`、`missing_points`、`misconceptions`、`next_hint`、
`remediation` 五部分。前三组可显式传空数组以清除错误反馈；遗漏字段或 null 不代表
清除，必须拒绝。每条内容继续引用冻结 evidence_ids，next_hint 必填，remediation
至少一条，列表/文本大小沿用原反馈合同。

`changes: []` 配合完整 findings 可只改反馈而不改分数；changes 与 findings 都无内容
仍拒绝。findings 是当前五部分的完整替换快照，同一次请求也可携带逐项分数纠正。
服务端先深拷贝、重放再验证，以同一个 PreviousHash 保护整份反馈；相同请求 ID 必须
连 findings 一起逐值相同，才能按原成功决定重放。未知引用、过期 hash、跨任务/工作区
请求不能保存。更改提示不会生成 VerificationRun 或触发 Provider。

前端提供“纠正反馈与提示”，支持逐条编辑、添加、移除和冻结引用选择；保存失败保留
原请求 ID、原 hash 与草稿，不自动基于新版本覆盖。页面区分当前反馈、原始模型反馈和
每次用户纠正的内容。切换评分任务会重建编辑器，避免把旧任务草稿提交到新任务。
原有评分应用仍绑定完整反馈 hash：纠正不会自动改动已应用快照或复习安排。

该改动复用 correction_json，无新增表、索引或迁移。省略 findings 的旧记录序列化与
重放语义保持，原模型输入、反馈 hash 算法和 Provider schema 不变。部署先更新
review-service 再更新前端。已有 findings 纠正记录后，不能回退到不识别该字段的旧
重放代码；回滚功能入口时须保留兼容读取与重放实现，不能删除纠正历史来迁就旧版本。
这提供用户纠正能力，不意味着自动评分理由的语义校准已经通过。

#### 2026-09-06：冻结参考答案合同 v4（候选，尚未部署）

新组装的 AssessmentInput 从同一冻结 LearningProjection/PracticeTask 附加
task_reference {task_id, practice_content_hash, expected_answer}。它是已编写的参考答案，
不是独立来源、唯一正确实现或运行事实；字段最多 4000 个字符并计入完整请求预算。
准备请求仍只接受作答/任务身份，由服务器读取保存的题目；不允许客户端替换参考答案。
字段进入输入和请求 hash，随评分任务 input_json 冻结；纠正和应用继续使用原输入。

TaskReference 非空采用 inkwords.mastery-assessment.v4；空时仍使用既有 v1/v2/v3，
省略 JSON 字段保持旧 hash。代码的静态/运行判断根据实际匹配 ArtifactHash，不能仅用
合同版本选择。AttachLearnerVerification 还核对解析任务的参考答案和练习内容；应用
匹配保存作答的 task_id/content hash。参考答案不能作为 answer_quote 或 evidence_id。

无新表、索引或迁移；JSON 增量字段不能保证旧二进制向前兼容。产生 v4 记录后若回滚，
必须保留字段读取、哈希、合同选择与运行分支，不能丢字段重算身份或清理历史。
尚未有生产 v4 写入，本轮候选未部署：真实评分对照因流程答案被过度给因果分而失败。
工程合同通过不等于评分语义通过，详见 docs/qa/mastery-reference-binding-2026-09-06.md。

#### 2026-09-06：逐项要求覆盖 v5（候选，尚未部署）

新准备输入设置 decision_policy=inkwords.criterion-coverage.v1 并采用评分合同 v5；
旧空策略保持原 v1–v4 身份。此策略要求冻结参考答案存在，所有字段参与原输入及请求 hash。
未改变 PracticeTask/rubric 的持久化 schema，也没有重新生成批准题目或覆盖学习作答。

模型除了原反馈还返回 decisions，每个 criterion_id 恰好一个、requirement 必须等于
对应 description。satisfied 对应 3–4 且 gaps=[]；unsatisfied 对应 0–2 且至少一项
逐字绑定本项 description 的 requirement_quote/reason；unknown 对应 null 且 gaps=[]。
原引文、来源、运行校验同时生效。任何矛盾拒绝整份有效反馈，不改分、不自动重试。

Result.decisions 作为原模型判断保存，Finish 与成功读回都重验原判断；用户纠正只修改
有效反馈，仍可把原 satisfied 的分数纠正到 2，不重写模型历史或把缺失运行变为通过。
前端把原始判断与冻结参考答案作为按需展开的核对信息，不能当作当前纠正后的判断。

Result.rejection 保存有界规则码及合法 rubric ID，不带原作答或模型正文。无有效评分时
保留 Provider 用量但不输出汇总零分。无新迁移/索引；产生 v5 记录后回滚必须保留策略、
字段、哈希、原判断验证与纠正兼容读取，不能用丢字段或清理历史来适配旧二进制。
本轮无生产 v5 写入，候选未部署。结构约束只证明内部一致性；实际判断准确性仍需代表性
真实验收，当前第二例已拒绝，具体诊断待补证，参见 docs/qa/mastery-decision-coverage-2026-09-06.md。

#### 2026-09-06：同一判断派生评分反馈 v6

2026-09-10 新部署检查点：显式 `local-evaluation-v2` 在 v6 响应 schema 中冻结当前 rubric
编号及 source/runtime 引用编号，并指导文字连续原文引用。v1 请求哈希保持不变，模型、
推理/Token 预算与时限不变；新 RequestHash 包含词表变化。两份正式批准练习同输入复测
通过，真实页面追加逐项/建议纠正并显式应用，旧评分与作答保留，CAS/幂等/新浏览器恢复通过。
模型建议仍需复核，记录为 delegated_ai 操作验收，不计个人六维或真人试学。
无新迁移；回退配置不清理历史。详见 `docs/qa/approved-practice-scoring-closure-2026-09-10.md`。

部署检查点：v6 及行号引用已部署至本机 review-service/frontend。显式
`MASTERY_ASSESSMENT_PROFILE=local-evaluation-v1` 使用已评测的 deepseek-v4-flash：文字
low/6000，代码关闭推理/6000；代码附带运行报告仍按代码策略，适配器与任务均 60 秒。
示例默认留空，旧合同选项保持；未知 profile/不兼容模型在恢复前拒绝。预览不调用模型，
本次部署没有 Provider 调用或学习写入。见 docs/qa/mastery-profile-deployment-2026-09-06.md。
以下未部署表述保留为历史验收记录；真实学习者运行评分、纠正应用和人工试学仍待完成。

代码引用扩展：新 v6 代码请求使用 `inkwords.numbered-learner-code.v1`，模型返回
`answer_span` 的 `inkwords.code-line-range.v1` 单文件物理行范围，原选择保留在
judgments，feedback 引文由冻结文件确定性提取。旧无范围引文仍按原规则读取，文字
请求与 v1–v5 不变。越界、跨文件、范围与自写代码引文并存均拒绝。新未知代码运行项
在完全没有运行记录时可显式 `evidence_ids=[]`；其他评分仍必须有证据。前端允许更正
未知说明，不能升为有分数。两个真实静态样本、持久化/纠正/应用与浏览器通过，未
部署、没有运行学习者代码。字段无新迁移，但未来回退必须保留范围读取与重放能力。
详见 docs/qa/mastery-code-span-2026-09-06.md。

界面与代码检查点：三份真实反馈的五次独立 HTTP 浏览器纠正已通过有效 hash/刷新/
原文保留验证，未写真实成绩。静态代码真实评分两批首例均失败，尚无有效代码评分；
不改变引用规则。内层适配器超时/取消现保留安全分类，使任务层记录 timeout/cancelled，
不保留错误正文、不延长时限、不回填旧收据。全量 Go 通过，候选未部署。详见
docs/qa/mastery-code-grading-2026-09-06.md。

最新输入修复：v6 请求元数据不再包含空 answer/evidence，改为 answer_evidence_id 指向
实际作答；输入与原文不变，请求 hash 更新，旧 v1–v5 路径保持。显式任务输出预算
可在原 3000 默认之外设到最多 6000，预算变化进入预览/hash，生产尚未启用。最终
low + 6000 四例真实评分通过数值门槛，原文语义复核仍有三份待审查纠正提案。
提案已领域重放验证，不是人工决定或真实学习记录。曾引入的引文枚举限制因拒绝合法
原文而撤回；原连续引文校验保持。见 docs/qa/mastery-answer-binding-2026-09-06.md。

后续真实验收与推理设置：v6 关闭推理仍有因果过评分；high/low 各一次也未得到有效评分。
新增显式任务 Options.ReasoningEffort，只对 v6 生效并进入请求哈希/预览/结果；默认和
生产 bootstrap 保持原行为。无效输出保留已知 Token，失败诊断细分为安全规则码，
不持久化模型正文。原收据不回填新原因，未部署。见 docs/qa/mastery-reasoning-options-2026-09-06.md。

新输入策略 inkwords.criterion-coverage.v2 采用评分合同 v6，旧 v1–v5 输入路径保持。
Provider 输出仅含 judgments/next_hint/remediation。每个判断用 criterion_id 绑定冻结
rubric，带 score/status、answer_quote、可选代码路径、evidence_ids、gaps/unknown_reason。
服务器解析要求原文并从同一组 judgments 派生评分说明和答对、遗漏、误解；禁止模型
另外返回 criteria/decisions 或独立 findings，以避免两套评分内容偏离。

gaps.kind 为 omission/misconception，要求引文绑定本项，原因最多 300 字、每项最多
5 个；score/status/未知运行规则继承 v5。positive reason 使用固定评分含义，缺口说明
直接使用同一 gap.reason；下一步提示和补救材料保持来源约束。结构一致性不代表判断
一定正确，仍保留原文、来源、实际运行证据、用户纠正与语义验收。

Result.Judgments 为原模型记录，Feedback/Decisions 为独立副本的投影；Finish 和成功
读回重放后逐值核对。用户纠正有效反馈不会覆盖原判断，投影修改不会通过指针反向修改
原判断。字段复用现有 JSON，无新迁移/索引；回滚须保留 v6 输入识别、原判断读取与投影
重放能力，不得删除字段重算旧身份。本轮仅代码与离线验收，尚无生产 v6 写入或部署。
详见 docs/qa/mastery-canonical-feedback-2026-09-06.md。

索引包括 workspace/request_id 去重、单个 running 任务、workspace/objective/
attempt/created_at/id 的历史查询，以及 job/sequence_no 纠正顺序。真实 PostgreSQL
EXPLAIN 对最近任务查询使用 idx_mastery_assessment_attempt 的 Index Only Scan。
成本为每个任务的三条二级索引维护、状态转换时更新部分索引及一个专用执行器连接；
每条纠正增加主键和顺序索引记录。没有给 learner answer 正文建立索引。
非空 v30 Down 拒绝删除任务/纠正证据；回退应用保留 schema，需要退回旧结构时沿用
备份恢复流程。SQL CHECK 使用 COALESCE，终态缺少完成时间不能经 NULL 绕过。

Compose 使用 MASTERY_DEEPSEEK_API_KEY（可单独覆盖，默认取 DEEPSEEK_API_KEY）与
MASTERY_DEEPSEEK_MODEL（默认取 DEEPSEEK_MODEL）；独立命令需显式设置前者。评分
配置与旧复习功能的自动反馈配置分开，新增配置不启用旧入口的模型行为。未配置时
预检拒绝调用，密钥不进入预检、任务 JSON 或日志。受保护的 real-acceptance 工作流现
提供三样本评分质量门禁，只有 `INKWORDS_REAL_ASSESSMENT_ACCEPTANCE=approved` 才会运行，
默认测试会跳过且不联网；2026-09-06 已执行三份文字样本并通过预设门槛，实际结果与
校准限制见上节。费用与六类任务的整体评分准确性不能由该三样本推定。

### 2026-09-05：评分输入的权威读取链

core-api 新增只读接口
`GET /api/v1/textbook-projects/chapters/:chapterID/practice-evidence?revision_id=:revisionID&task_id=:taskID`。
必须显式给出批准 revision 和其中存在的具体题目。先核对 installation workspace、
章节归属及批准稿正文/题目合同，再复用固定来源解析器读取该章节已引用的快照，
只返回本题引用的原文。候选、其他章节/工作区、未知题目、缺失原文或混合主资料版本
均被拒绝；不从浏览器接收 rubric、参考来源或“运行通过”声明。

跨服务合同 `inkwords.practice-evidence.v1` 保留 workspace、chapter、revision、母稿
content hash 和 task ID，并为每段原文保留 EvidenceRef、SourceSnapshot、精确
excerpt hash。导入器的 chunk hash 与原文的 UTF-8 字节 hash 是独立字段，不能假设
其序列化编码相同。原文中的代码围栏和不可信指令按原样数据保留，不执行、不改写、
不截断。每题最多 40 条来源，每段最多 20,000 字符；评分器仍单独执行 32,000 请求
字节预算。HTTP 读取沿用固定 origin、10 秒超时、4 连接、3 MiB、无重定向及无重试。

review-service 的 `PrepareAssessmentInput` 只接收 workspace/objective/attempt 身份，
从已保存作答和冻结 PracticeSet 组装答案、题目/变式和五项 rubric。目标所有权与作答
绑定先于来源读取检查；返回来源集合必须与本题完全一致，母稿 hash/版本/题目不得
漂移。按冻结 EvidenceIDs 顺序组装，以保证相同输入 hash 稳定。没有已绑定的学习者
工件/VerificationRun 时 ArtifactHash 为空，来源一律标为 source；作答自称“测试通过”
不会制造 runtime 证据。这个方法本身不调用模型、不保存评分、不改变调度。

作答改为按 `id AND objective_id` 定位单条记录，不为评分读取全部历史。未新增迁移
或索引；隔离 PostgreSQL EXPLAIN 在小型夹具上选择现有
`idx_mastery_attempts_objective_time` Index Scan 并过滤 id，主键也可由优化器选择。
core 复用已有来源联查；没有新增写入、写放大或存储成本。实际测试覆盖历史批准稿
原文、排除无关来源、其他身份拒绝、截断/超大/错误/重定向响应，以及原文代码块保留。

## 母稿拥有具体六维题目（2026-09-05）

`inkwords.practice-set.v1` 在样章合同内保存六个不同 mode 的具体题目、变式、
参考答案、五项评分依据、三层提示、来源 ID，以及 retain 的最小间隔。结构校验
拒绝缺项、重复题目/模式、未知来源、错误运行证据标记和不足 24 小时的保持要求。
题目引用章节中的教学文件，不新增可执行代码块；运行状态仍由 VerificationRun 决定。

生成器将该结构确定性地渲染进母稿正文，整体参加 ContentHash；质量 v9 同时检查
结构及正文一致性。prompt v12 请求实际结构化题目，原有代码、来源及写作门禁不放宽。
总预算仍为 32,000 Token；六维任务使输出预留由 7,000 增至 12,000，输入上限降为
20,000，超预算不调用 Provider。尚无新合同的真实模型生成验收。

批准稿学习投影升级为 v2，原样投射 PracticeSet。旧 v1 可继续读取，但不能被新界面
冒充为已有具体六维题目。学习目标冻结 `approved-revision:chapter:revision` 身份；
会话按当前调度 mode 选择对应题目，章节或 revision 不一致时拒绝替换作答依据。
题目和答案不成为独立可编辑的学习副本，答案也不等于执行或掌握证据。

提示逐层显式展开，参考答案单独展开。界面记录已展示帮助的最低次数，不能手动降低；
看过答案则本次不能标为独立完成。这仍是当前会话的辅助使用与自评记录，不是服务端
认证的防作弊证据，刷新后不能据此声称没有看过帮助。

### 服务端批准依据与延迟约束

review-service 从配置的 `CORE_API_URL` 读取固定 revision 的批准投影。请求
`GET /api/v1/textbook-projects/chapters/:id/projections?revision_id=:revision` 可读取
历史批准稿；仍拒绝候选、草稿和其他 workspace。省略参数时沿用当前批准稿行为。
响应新增 workspace_id，调用方核对 workspace、chapter、revision、内容哈希及 v2 合同。
客户端由服务端固定 origin 初始化，最多 4 个连接、10 秒超时、3 MiB 响应，不跟随
重定向、不应用重试、不返回远端错误正文。Compose 默认 `http://core-api:8080`；
运行旧的聚合本地命令时可设置 `CORE_API_URL=http://127.0.0.1:8080`。

创建 `approved-revision:` 学习目标时，浏览器传入的 title、rubric、关键点与来源
不能代替批准依据；服务端校验并冻结 LearningProjection，记录独立 revision/hash
列。它只是不可编辑的批准稿读模型。重新打开学习会话直接读该冻结投影，后续批准
新稿不会改换原题。没有可信绑定的历史目标保留作答读回，但不能冒充新式具体题目任务。

新式作答必须包含 `practice_task_id` 与 `practice_content_hash`，均与目标及 skill
匹配；答案不能为空。保存时间取服务器时钟，浏览器 attempted_at 不能制造已过去的
时间。retain 提交必须距上一条已保存练习达到题目的 min_delay_hours；下一题为
retain 时也将 FSRS due date 向后限制到该时间。此版本标记为
`fsrs-v4-practice-delay-v1`；旧自由文本目标保留原算法标识和兼容接口。

掌握领域判定修正为比较“retain 作答时刻 − 上次练习时刻”；不能比较“当前时刻 −
retain 记录时刻”。同日即时复述即使保存数天也不能成为延迟证据，发生在学习之前或
未来的复习也不能证明保持。`IsMasteredAfter` 支持明确的至少 24 小时要求；自动判定
仍不替代真实运行证据或模型逐项评测。

迁移 v28 为 objective 增加冻结投影与 revision/hash，为 attempt 增加题目 ID/hash，
SQL CHECK 使用 COALESCE 拒绝 JSON 缺字段造成的 NULL 绕过。不新增索引；正文读取
沿用主键。并发追加在事务内锁定 objective，核对历史条数，拒绝基于旧历史生成的
schedule。真实 PostgreSQL EXPLAIN 显示计数复用 `idx_mastery_attempts_objective_time`
的 Index Only Scan；写入成本增加一次对象行锁和索引计数。同一目标会串行，不同目标
互不锁定。发生冲突时界面保留作答，由用户重试并重新计算。

Down 在任何冻结题目/作答绑定存在时拒绝删除证据。回退应用可保留 v28 列；如需回退
schema，必须使用备份恢复流程。当前真实学习库无目标和作答，不做假记录回填。

### 2026-09-05：持久化练习会话与帮助披露（review schema v29）

具体题目增加 `practice_session_id`。主动开始练习调用
`POST /api/v1/mastery/objectives/:id/practice-sessions`，请求为 `{ "skill": "explain" }`；
重复打开复用同一目标/维度尚未提交的会话。`GET .../practice-sessions/:sessionID`
恢复固定题目 ID/hash、开始时间、帮助计数和已提交的原始调度结果。
`POST .../practice-sessions/:sessionID/help` 接收 `{ "kind": "hint", "level": 1 }`
或 `{ "kind": "answer", "level": 0 }`。提示只能逐层展开，同一披露重复请求只记一次。
所有接口使用 installation workspace 验证目标所有权，不能由请求体指定身份。

页面只有收到保存成功的帮助记录后才展开内容。刷新恢复已展开层级、提示次数和
答案辅助状态；失败保留作答并允许重试。浏览器 sessionStorage 只保存待确认会话 ID，
不保存答案；收到成功结果后清除该 ID。响应截断后再次打开会话可读回原结果和历史
作答，不会先创建另一会话。浏览器存储不可用时仍能恢复服务端未提交会话，已提交
记录仍在历史中可读，但不能保证全页刷新自动定位丢失的完成响应。

作答事务锁定 objective，重新核对帮助总量和历史条数。服务端取保存时间与会话耗时，
提示计数不能低于已披露数量，看过参考答案强制 `independent=false`；其他会话中的
新帮助也视为已知学习接触，retain 必须重新满足题目的保持间隔。帮助只记录接触，
不直接增加 mastery 或 FSRS 分数；已显示的 due 建议可能早于新接触后的最早提交时间，
提交时仍以服务器约束为准。

会话、作答和调度在一个事务中完成。提交身份包含会话、答案、自评与题目依据，排除
浏览器时钟/耗时并统一空错误列表。相同提交（包括同时发出的请求）返回第一次保存的
结果，不重复追加或更新调度；改变已提交答案返回 409。完成后不能新增帮助，只能
重读已保存披露。内部提交 hash 和聚合计数不经 HTTP 返回。

v29 新增 `mastery_practice_sessions`、只追加的 `mastery_practice_help` 与作答会话外键。
`ux_mastery_open_practice_session(objective_id,skill)` 的部分唯一索引保证一次未完成
练习身份；`idx_mastery_sessions_objective(objective_id,id)` 支持汇总含历史会话的帮助；
`ux_mastery_attempt_session` 保证每个会话至多一个作答。真实隔离 PostgreSQL EXPLAIN
对恢复查询使用前一个 Index Scan，帮助联查对会话使用后一个 Bitmap Index Scan，
小型帮助表使用 Seq Scan/Hash Join。它们增加会话创建/完成及作答追加的索引维护和
存储成本；帮助主键还承担同一披露去重。没有全局行锁，不同目标互不串行。

Down 仅允许没有会话和绑定作答的空表；存在帮助/会话证据即拒绝。应用回退应保留
v29 schema；如需退回旧数据结构，沿用已验证的备份恢复流程，不能删除学习证据。
PostgreSQL 用例覆盖刷新、跳层拒绝、重复帮助、答案辅助、跨目标身份、延迟接触、
并发相同提交、旧历史冲突和回滚保护；浏览器覆盖帮助失败不展开、提示刷新恢复和
截断提交响应恢复。模型评分/纠正及真实读者验收仍未完成。
