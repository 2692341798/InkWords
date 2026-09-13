# 用户委托 AI 整书审阅

日期：2026-09-10。承接用户“需要人工审核的部分也交给你了，直接操作我的电脑去决策即可”的明确授权。

## 决定

冻结构建的八阶段审阅允许记录用户委托 AI 的实际结论；它与真人审阅、自动完整性检测分别存储。不能继续要求 AI 冒充真人来使用原来的 `automated=false` 接口，也不能因授权就跳过具体证据、权利核验或失败项。

新增 `inkwords.delegated-publication-review.v1`，字段包含冻结构建及 manifest hash、阶段、审阅版本、固定 `delegated_ai` 来源、审阅者、用户授权说明、实际范围、结论、0–4 分、证据引用、阻断发现和服务端时间。结论为 `pass`、`needs_revision` 或 `not_assessed`。通过必须至少 3 分、有证据且没有阻断发现；需要修改必须明确发现；未评估不声称评分或隐藏已知失败。

本接口是审阅记录，不会自动调用模型，也不会验证一个自然语言判断在事实层面正确。操作者必须先实际检查冻结内容与证据。来自生成会话的复核必须披露上下文，不得标记为隔离上下文的独立审阅；规范要求的独立 AI 审阅仍应只提供冻结母稿、BookContract、证据和 rubric。真实读者掌握、延迟保持、出版社与法律认可不由本接口产生。

## 状态与并发

- `POST /api/v1/textbook-projects/book-builds/:buildID/delegated-reviews` 接收客户端 UUID 和 `expected_revision`，不接受客户端 actor、构建 ID、完成时间或最终状态。
- 事务锁定当前工作区的构建，核对 manifest hash，再检查幂等请求及当前阶段版本。相同 ID/输入重试返回原记录；不同输入重用 ID、过期版本、manifest 不一致返回冲突；跨工作区返回未找到。
- `textbook_delegated_publication_reviews` 只追加审阅版本。旧结论保持原样；预检读取每阶段最新版本。最新“需要修改”继续阻断，即使曾经有通过记录。
- 合法 AI 通过可满足同阶段的内部审阅条件。所有权利主体和现有自动检查仍独立要求通过，最后晋级仍需显式操作。`publication_candidate` 是内部状态，不是出版认证。
- EditorialWorkspace 返回独立 `delegated_reviews` 与能力版本。旧真人表和 API 不改写；前端只在服务端声明支持时显示新表单。export-service 读取相同版本合同，审校包增加独立的 `delegated-ai-reviews.json`，原 `human-reviews.json` 不混入 AI 结论。

## 迁移与恢复

core 的 goose 00035 仅新增表。唯一索引 `(build_id, stage, revision)` 同时服务历史有序读取和阶段最新版本查询；另有 UUID 主键。每次追加写入两个 B-tree，记录数量随复审次数增长，没有额外重叠索引。真实 PostgreSQL `EXPLAIN` 已验证在禁用小表顺序扫描的测试条件下使用该复合索引；不据此声称生产延迟指标。

发布重建 core-api、export-service 与 frontend，保留原镜像 tag。应用回滚可保留新增表；Down 在存在记录时主动失败，避免丢失审阅证据。需要回滚数据库时先做经验证的备份恢复流程，不在此次执行破坏性回滚。

## 验证范围

契约覆盖缺失授权、伪造 actor、低分通过、通过仍含阻断、未评估隐藏失败、错误 manifest、最新版本覆盖判断、八阶段 AI 通过及权利依然阻断。真实 PostgreSQL 覆盖隔离、幂等、CAS、不可变历史、晋级拒绝和 EXPLAIN；HTTP 测试拒绝客户端伪造身份与时间；页面测试保留失败历史、重复提交身份和晋级后锁定。

真实本机操作及当前构建结论另见 `docs/qa/delegated-publication-review-2026-09-10.md`。
