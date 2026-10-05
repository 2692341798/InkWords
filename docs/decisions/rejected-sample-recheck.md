# 拒绝样章留存与离线重检

2026-09-06。真实 v14 整章重试曾消除旧失败却引入新的硬失败。只保留错误码和短片段无法
复用已付费生成的内容，因此增加服务自有的拒绝稿诊断存储，不把失败结果写进成功缓存。

每份 `inkwords.rejected-sample.v1` 保存冻结的生成输入、来源片段、结构化章节、原始质量
结果和 Provider 遥测；不含 Provider 请求头或配置密钥。文件按完整 JSON 字节的 SHA-256
只追加写入，目录 0700、文件 0600，单份最多 2 MiB、单个运行器最多 32 份。达到容量或
写入失败时不删除历史记录，任务返回 `rejected_draft_storage=unavailable`；保存成功只
在安全失败结果中返回哈希，不返回整篇拒绝稿。磁盘部分写入会被完整性校验拒绝，不能
覆盖已有文件修复；由操作方检查后处理该诊断目录。当前部署只有一个写入进程。

Compose 的 `rejected-drafts` 卷仅挂载 llm-stream，实际私有目录为
`/app/rejected-drafts/private`。它不挂载给 core-api、导出或执行服务，不属于可批准候选，
不会在普通重试中被自动重放。关闭留存可移除 `TEXTBOOK_REJECTED_DRAFTS_DIR` 配置；保留
卷中现有文件即可回退应用，无数据库迁移。不要使用 `docker compose down -v` 回退。

镜像内 `textbook-recheck` 仅离线读取一份哈希匹配的诊断记录，默认检查原稿；加
`-markdown-stdin` 时从 stdin 接收最多 1 MiB 的替换 Markdown。结构化练习、来源、蓝图
关键事实和身份仍取原记录，运行全部当前门禁，不执行教学代码或调用 Provider。输出注明
`automated_offline_recheck`、原/新正文 hash、ProviderCalls=0、CandidatePersisted=false；
失败退出码为 1，检查通过为 0。通过也不能直接应用为母稿。

```bash
docker compose --env-file backend/.env exec -T llm-stream \
  /app/textbook-recheck -hash <失败任务返回的64位哈希>

docker compose --env-file backend/.env exec -T llm-stream \
  /app/textbook-recheck -hash <同一哈希> -markdown-stdin < /绝对路径/局部修正.md
```

正文替换必须保留由 PracticeSet 渲染的练习区；直接更改练习答案而不更新结构化合同会被
母稿一致性门禁拒绝。

## 结构化局部修订

`-correction-stdin` 接收最多 1 MiB 的 `inkwords.sample-correction.v1` JSON；与
`-markdown-stdin` 互斥。输入必须提供原收据完整字节的 64 位 hash、原正文 hash、修订
理由以及 markdown_body / scenario / understanding / learning_arc / practice_set。
markdown_body 不含六维练习区：程序从 PracticeSet 重新渲染一次，避免正文答案与合同
分别修改。章节 ID、读者、标题、来源、关键事实和原 Provider 信息只能继承原收据，
不接受客户端覆盖；六道练习保留原 ID 和 mode，其他字段重新通过当前质量合同验证。
JSON 拒绝未知字段和尾随文档，原收据身份/hash 不匹配时不输出可用提案。

输出是带完整修订章节的私有提案，origin/generation_mode 为 automated_local_correction，
runtime_verification 固定 unverified，ProviderCalls=0、CandidatePersisted=false。
原 Provider 用量保存在单独 original_usage 中，不把上一轮调用费用或 Token 记作新调用；
另记录原收据/正文 hash、冻结请求 hash、修订输入 hash 和修订理由。自动质量检查仍须
全部执行，包括冻结来源、蓝图关键事实、运行表述和延迟练习。失败也可输出供检查的
提案，但退出 1 且 quality.passed=false；结构/身份错误不输出提案。

```bash
docker compose --env-file backend/.env exec -T llm-stream \
  /app/textbook-recheck -hash <同一哈希> -correction-stdin < /绝对路径/结构化修订.json
```

## 候选提交与兼容边界

失败任务面板可选择结构化修订 JSON，并显式提交到
`POST /api/v1/textbook-projects/:projectID/chapters/:chapterID/correct-sample`，
请求为 `{original_task_id, edit}`。接口拒绝未知字段，最多接收 1 MiB 修订数据和请求包装，
不能接受客户端提供的质量结论、来源、用量或 workspace 身份。

core-api 读取同工作区的原失败任务，核对留存收据及当前批准输入。异步工作器读取私有
收据，重算冻结请求 hash，并复用上述全部修订门禁。持久化时再次锁定原任务、核对
原 Prompt/usage 和当前批准合同，通过章节版本 CAS 追加 candidate。按新任务行锁串行
处理重复投递，同任务同内容返回已有候选；输入改变或原任务已重试则拒绝保存。
查询使用现有 job_tasks 主键和 generation_task_id 索引，无新增索引或 schema 迁移。

普通生成任务/结果继续使用 v3/v1；局部修订使用 v4/v2 和显式 Correction 合同，旧工作器
会在 Provider 调用前拒绝 v4。先部署支持新合同的 core-api 与 llm-stream，再提交修订。
回退时保留 v4 任务及原收据，不能降版本后当作普通 Provider 任务重新执行。

2026-09-09 增补：留存原稿允许当前 prompt 和 v15，但质量合同仍必须为当前版本。
这一兼容仅用于零 Provider 修订；普通生成仍拒绝 v15。原收据、原任务和原输入哈希不改写。
若章节已推进，调用者必须在 `edit.baseline` 明确提供
`{expected_chapter_version, parent_revision_id}`，与提交时章节完全匹配；省略时仍拒绝跨版本修订。
该修订使用任务 v5、结果 v2，`correction.source_baseline` 保留原任务版本和父修订，
顶层版本与父修订绑定当前追加目标。除 prompt 兼容差异外，来源、模型、蓝图、读者、
BookContract 与 StyleSheet 必须逐项一致。落库再次检查完整基线并通过 CAS 追加候选，
原有人工稿、批准稿和历史候选保持不变。旧工作器拒绝 v5；回退前应排空该类任务，
不可改写其版本后重试。无 schema 或索引变化。

2026-09-13 增补：显式合同迁移使用任务 v6、结果 v2。只读来源额外支持历史
`inkwords.textbook.sample.v16` / `inkwords.sample-quality.v9` 这一固定组合；原失败报告
必须仍为 v9、failed 且有失败项。普通生成和 v4/v5 修订均不因该白名单接受 v16/v9。

调用者在 `edit.target_input_hash` 提供当前 generation-preflight 的完整输入哈希，
并同时提供 `edit.baseline`。core-api 从当前批准输入构建候选任务，顶层仍为当前
v17/v10；`correction.migration` 保留原 prompt、quality 和完整 blueprint，
`source_baseline` 保留旧章节 CAS。移除修订并还原这些字段后必须得到原输入哈希；
仅移除修订则必须得到调用者确认的当前输入哈希。两种身份均参与任务哈希。

迁移只允许蓝图追加新章/单元：已有单元标题和顺序、所有已有章节的每个字段及所属
单元必须保持一致。目标章、读者、模型、BookContract、StyleSheet、来源或证据
改变均拒绝，不自动拼接上下文。排队时深拷贝所有嵌套输入，完成时再按完整当前输入
和章节 CAS 检查；即使后来只改了新增章，也必须重新预检，旧任务不能写入。

worker 仍从原私有回执重建冻结请求，重新执行当前质量门禁，provenance 额外记录
`target_input_hash`；原报告和模型用量保持来源身份，本次调用数为 0。迁移只追加
candidate，不取得章节批准锁或自动应用。批准阶段原有锁、内容哈希和 CAS 规则不变。
先部署 core-api 与 llm-stream，再提交 v6；回退前排空任务，保留历史记录，禁止把 v6
降为普通生成任务重试。无数据库 schema、索引或依赖变更。

created_by 继续用 generation 表示 AI 产物；document_json.correction.origin 与 sample 的
generation_mode 明确为 automated_local_correction，避免把自动修订归为 manual。
原模型字段保留来源，原用量单独放在 correction.original_usage_json，本次记录调用数 0。
界面分别显示自动局部修订、原模型及原用量；修订重试明确不调用模型。
离线通过或新候选均不等于新 Provider 生成、人工批准、代码运行或出版验收。
