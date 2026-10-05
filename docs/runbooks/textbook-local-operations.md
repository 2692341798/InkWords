# 教材平台本地操作 Runbook

本手册适用于单用户、本地 Docker Compose 环境。它不执行命令；尤其是恢复数据库、删除 volume 或调用真实模型前，必须由操作者确认目标环境和数据范围。

## 首次启动与 API Key

1. 复制 `backend/.env.example` 为只属于本机的 `backend/.env`，不要提交该文件。
2. 保持 `TEXTBOOK_GENERATION_PROVIDER=fake`，先完成离线导入、蓝图和候选稿合同检查。
3. 只有需要真实候选稿时，明确设置 `TEXTBOOK_GENERATION_PROVIDER=deepseek` 或 `openai`、对应的 `TEXTBOOK_STANDARD_MODEL` 与进程环境密钥。DeepSeek 的本机默认分层为 `TEXTBOOK_STANDARD_MODEL=deepseek-v4-flash` 和 `TEXTBOOK_CORE_MODEL=deepseek-v4-pro`；样章候选稿只走标准档，核心档只留给蓝图、核心机制与审校任务。教材的单次 Provider 请求默认使用 `TEXTBOOK_GENERATION_REQUEST_TIMEOUT=15m`，允许范围为 `30s`–`30m`；这个边界覆盖较长的结构化样章，不改变其他短任务的超时。密钥只放进 `backend/.env`/进程环境，绝不放入教材、截图、日志或工件。
4. 运行 `docker compose --env-file backend/.env up -d --build`，再以 `docker compose --env-file backend/.env ps` 和 `curl --fail http://localhost/api/v1/ping` 验证服务。
5. 在工作台点击“测试已配置连接”才会发出固定、小预算的 Provider 探测；连接成功不等于教材内容、证据或运行结果已验证。

真实教材生成的所有 Compose 命令（包括单服务 `build`、`up`、`restart`）都必须携带
`--env-file backend/.env`。若遗漏，Compose 会按安全默认值把新容器切回 `fake`，密钥也不会注入。
在每次已授权的付费生成之前，先用下列无密钥输出检查 core 与 worker 是否一致：

```bash
docker compose --env-file backend/.env exec -T core-api sh -lc \
  'printf "provider=%s model=%s\n" "$TEXTBOOK_GENERATION_PROVIDER" "$TEXTBOOK_STANDARD_MODEL"'
docker compose --env-file backend/.env exec -T llm-stream sh -lc \
  'printf "provider=%s model=%s\n" "$TEXTBOOK_GENERATION_PROVIDER" "$TEXTBOOK_STANDARD_MODEL"; test -n "$DEEPSEEK_API_KEY" && echo key=configured'
```

教材任务会把 provider/model 冻结进载荷和幂等哈希；worker 的实际配置若与任务不一致，
必须在网络调用前 fail-closed，不允许静默改用 fixture 或其他模型。
待审候选也必须与 core-api 当前 provider/model 目标一致才可应用；切换目标后，旧候选会保留供审计，
但项目阶段和章节编辑器都会要求重新生成。已经人工批准的修订不会仅因后续切换模型而失效。

在章节编辑器点击“生成候选教材稿”只执行只读预检，不会创建任务或调用 Provider。
预检会显示冻结的 provider/model、证据数、保守输入 Token 估算、预留输出预算、
缓存未知和费用未知。这些估算不是供应商账单。只有再点击“确认并创建任务”才入队；
POST 必须携带该预检的 `confirmed_input_hash`，core-api 会重新冻结当前输入并比对。
资料、蓝图、合同、章节版本或生成目标变化后，旧确认必须 fail-closed，不能入队。

真实样章任务失败后，重试也必须重新确认。任务查询只在冻结载荷仍能完整校验时返回
`retry_confirmation`，其中包括 provider/model、prompt/质量合同版本、与首次预检一致的
Token 估算和输入哈希，不返回证据正文。章节编辑器先显示“查看重试影响”，再显示
“确认重试一次”；只有第二个动作才把相同的 `confirmed_input_hash` 发送给服务端。
空哈希、旧哈希或无法重建摘要的任务返回 409，不重新入队。每次成功确认都可能形成一次
新的 Provider 调用和费用；没有版本化定价依据时，费用继续显示未知。

浏览器会话丢失后不需要从历史记录中手工复制 task ID。core schema v34 将样章任务的
章节身份从不可变载荷回填到 `job_tasks.textbook_chapter_id`；打开章节时，工作区接口按
当前 installation workspace 和 chapter 返回最新的样章任务，前端再把 ID 写回
sessionStorage。若页面没有恢复失败任务，先检查 core migration version 是否为 34、
目标任务的 `textbook_chapter_id` 是否存在，以及章节工作区响应中的 `latest_sample_task`；
不要直接写 sessionStorage、修改任务载荷或重投递 MQ。

候选稿不能仅凭“自动质量门禁通过”进入批准稿。操作者必须先取得当前章节编辑锁，
逐项填写“句子与段落负担、标题承诺、重复、语气、例子相关性、章节节奏、图示机会、
练习梯度”八维人工量表，每项至少 `3/4`，并填写 8–2000 字符的人工审阅说明，
再点击“我已人工审阅并批准”。服务端会把量表、说明、候选稿内容哈希与当前质量合同
作为不可变决定写入同一事务；缺项、低分、旧锁、内容漂移或已有批准/驳回决定都会拒绝。
自动软提示只用于定位复核点，不会替操作者评分，也不能冒充读者试学。
章节工作区会同时返回当前 `human_review_contract`；前端必须以其中的版本、维度、评分范围
和说明长度渲染表单，不能自行维护另一套当前规则。旧响应只能使用同版本兼容兜底，
最终提交仍由服务端按当前合同重新校验。

本机 UI 直接进入固定 installation workspace，不需要登录、JWT 或 GitHub OAuth。
旧博客与复习记录由服务端解析唯一的本地兼容 owner；若数据库存在多个活动用户或
无法建立唯一映射，服务会返回 503 而不是采用请求 header 中的身份。知识复习只读
`OBSIDIAN_VAULT_PATH` 的挂载内容，即使 Obsidian REST 插件未启动也能抽取题卡；
需要写入 Obsidian 的旧流程仍须配置 REST API。

若 Docker Desktop 内存不足以并行编译全部 Go 服务，逐个执行 `docker compose --env-file backend/.env build <service>`，全部成功后再执行 `docker compose --env-file backend/.env up -d --no-build`。不要把编译器因内存不足被终止误判为代码编译错误。前端网关会通过 Docker DNS 动态解析服务名；更新某个后端服务后仍应以 `curl --fail http://localhost/api/v1/textbook-projects` 验证实际反向代理，而不是只检查容器 healthcheck。

## 升级前备份

升级镜像、拉取代码或执行含 migration 的启动前，先停止写入并创建带时间戳的备份目录。以下命令中的路径必须替换为操作者确认过的空闲本地目录：

推荐从仓库根目录使用 fail-closed 工具；目标必须是仓库外的绝对空目录：

```bash
bash backend/scripts/backup_local.sh /absolute/path/to/inkwords-backups/2026-09-04T120000Z
```

工具会短暂停止当前运行的写服务，分别导出 core/review 数据库，以 `0700/0600`
保护目录和文件，在随机命名的临时数据库执行 `pg_restore --exit-on-error`，比对任务、
教材项目、掌握目标、旧复习会话和 migration version 计数，随后只删除该临时库并恢复
原服务。成功目录包含两个 custom-format dump、源/恢复 v2 快照和 SHA-256 manifest。
目录还包含本次 Compose 镜像清单，便于恢复时选择匹配版本。
它不会自动恢复或覆盖当前数据库；不要把该目录放进仓库或云同步目录。

若该备份将用于删除旧身份或 ProjectCourse schema，执行删除前必须再次运行只读预检：

```bash
docker compose --env-file backend/.env stop core-api llm-stream parser-service export-service review-service course-runner
bash backend/scripts/check_legacy_cleanup_ready.sh /absolute/path/to/inkwords-backups/2026-09-04T120000Z
```

预检要求六个写服务已经停止，并复核 manifest、两个 dump 的 SHA-256、恢复快照、
备份后任务/教材项目/博客/掌握目标/复习会话与轮次的行数和更新时间水位是否漂移，
以及 task/blog/review 的 workspace 归属、OAuth/ProjectCourse/user prompt settings
空表条件，并要求 core 恰为 v23、review 恰为 v21。它本身不停止服务，也不执行 `DROP`、`DELETE`、
migration 或 Provider 调用；任何条件不一致都会拒绝继续。若不继续执行获批准的
清理，应重新启动上述服务。

备份和预检共同读取 `backend/scripts/sql/backup_core_snapshot.sql` 与
`backend/scripts/sql/backup_review_snapshot.sql`，避免两条路径各自维护查询而产生漂移。

手工等价命令如下：

```bash
mkdir -p /absolute/path/to/inkwords-backups/2026-09-03
docker compose --env-file backend/.env exec -T db pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB" > /absolute/path/to/inkwords-backups/2026-09-03/core.dump
docker compose --env-file backend/.env exec -T review-service sh -lc 'pg_dump "$REVIEW_DATABASE_URL" -Fc' > /absolute/path/to/inkwords-backups/2026-09-03/review.dump
```

若 review 与 core 共用同一数据库，第二份导出不是必需的；若 `review-service` 镜像没有 `pg_dump`，应在已验证版本的 PostgreSQL 客户端容器中导出，而不是跳过备份。记录导出时间、Compose 镜像版本、目标库名和文件 SHA-256。完成后应在独立临时库执行一次 restore 演练；只生成 dump 不等于备份可恢复。

## 升级与失败恢复

1. 阅读 pending migration 与 `docs/decisions/versioned-database-migrations.md` 的 Down/恢复边界。
   `00015_mastery_objective_identity.sql` 会拒绝含有同一工作区、同一冻结章节身份的重复活动学习目标；这是防止双标签页重复创建的保护。出现该错误时，先按上节备份，再人工核对重复目标及其 append-only 尝试记录，不能删除记录或直接改 migration history。
   `00024_remove_legacy_identity.sql` 与 `00025_review_remove_legacy_identity.sql` 会永久删除已退役的账户表、ProjectCourse 表和旧 owner 字段；二者的 Down 始终拒绝执行。升级必须先停止六个写服务并通过上节预检。失败恢复只能使用通过恢复演练的备份及对应旧镜像。
   `00026_candidate_approval_review.sql` 为历史驳回表增加结构化人工批准证据；在尚未写入批准证据时可安全 Down，一旦存在批准决定或非空量表就会拒绝回滚，必须改用备份恢复。
   `00034_textbook_task_chapter_identity.sql` 增加可回填的任务章节投影和局部索引。它让章节
   恢复只读取同 workspace/chapter 的最新样章任务，避免运行时扫描、解析 JSONB；代价是每个
   样章任务多写一个外键列和一个局部索引项，占用 UUID、时间戳和主键的索引空间。Down 会
   删除该派生列和索引，但不可变 payload、任务结果和候选稿仍保留，之后浏览器状态丢失时将
   失去服务端按章发现能力。
2. 启动更新后的服务并检查 `ps`、`/api/v1/ping` 以及目标读写 smoke。migration 出错时不要重复启动或手改 schema。
3. 需要恢复时，先停止会写入数据库的服务，确认要覆盖的库就是预期本地库，再由操作者显式执行 `pg_restore`。恢复会覆盖目标数据库，不能在未确认数据范围时执行。
4. 对不可安全 Down 的 migration，恢复已验证的备份并使用对应的旧镜像，而不是删除表或篡改 migration history。

## 常见教材链路故障

| 现象 | 首先检查 | 不要做 |
| --- | --- | --- |
| Provider 未配置或连接失败 | provider 类型、模型名、对应进程环境密钥与稳定错误码 | 输出或提交密钥、原始 Provider 错误体 |
| 确认生成时提示预检已过期 | 重新打开生成前预检，确认资料、蓝图、章节版本与 provider/model | 绕过 `confirmed_input_hash`、重放旧 POST 或直接投递 MQ |
| 失败任务无法恢复或重试 | core schema 是否为 34、章节工作区是否返回 `latest_sample_task`、任务是否返回冻结 `retry_confirmation`，并重新核对模型、Token 预算和输入哈希 | 手写 sessionStorage、修改任务载荷、发送空确认、猜测哈希或绕过二次确认直接调用 Provider |
| 官方文档抓取不完整 | policy 的 host/path/预算、robots、manifest decision 和 checkpoint | 放宽跨域限制、忽略 robots、把 partial 当 complete |
| 教学代码未验证 | `TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED`、runner image digest 与 sandbox 预检 | 退回宿主机执行或为 Chromium/bubblewrap 添加不安全 sandbox 绕过 |
| PDF 不可用 | `TEXTBOOK_CHROMIUM_BIN` 和目标运行时 Chromium sandbox | 添加 `--no-sandbox` 或把单元测试当成 PDF 视觉验收 |
| 单个 API 在服务重建后返回 404 | 前端是否为含动态 Docker DNS 的当前镜像、`curl --fail http://localhost/api/v1/textbook-projects` | 仅因容器健康就忽略网关路径，或在 Nginx 中固化容器 IP |
| 手工章节似乎被生成覆盖 | revision status、锁定状态、候选 diff、expected version | 直接修改已批准修订或绕过 compare-and-swap |

相关说明：`docs/decisions/llm-provider-and-secret-storage.md`、`docs/decisions/official-document-crawler.md`、`docs/decisions/textbook-sandbox-selection.md`、`docs/runbooks/textbook-browser-video-evidence.md` 与 `docs/runbooks/textbook-publication-checklist.md`。
