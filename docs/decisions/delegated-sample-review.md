# 用户委托 AI 样章审阅

2026-09-09 用户明确授权 Codex 审阅并执行批准决定。该授权允许执行编辑决定，
不把 AI 变为真人读者、人工同行评审或出版机构。

批准请求新增可选 `reviewer_kind` 与 `delegation_note`：

- 省略来源或 `human`：沿用 `inkwords.sample-human-review.v1`，不接受委托说明。
- `delegated_ai`：必须提供 8–1000 字的明确委托说明，保存
  `inkwords.sample-delegated-review.v1`；不接受其他来源类型。

两者均需八维评分逐项至少 3/4、8–2000 字审阅说明、当前质量合同通过、
当前模型目标匹配、编辑锁、候选与章节 CAS，并保留不可改写的决定记录。
授权说明是本地用户声明的审阅权限来源，不是外部机构认证。

为兼容历史数据，决定仍存于 `textbook_candidate_reviews.human_review_json`，
其 JSON 显式包含 reviewer_kind、delegation_note 和独立合同版本；不需要迁移。
委托批准只追加与候选正文相同的批准修订，created_by 保留 generation，
不把 AI 产物写成人工创作。旧人工批准流程和已有记录保持兼容。

界面只在服务端声明 `reviewer_kinds` 能力时提供委托选项，默认仍为本人审阅。
AI 评分、授权说明、按钮及决定记录均明确标识来源。整书出版审校和学习者掌握
记录不会由此自动完成；没有自动生成真人试学或延迟回忆证据。

回退时保留 JSON 与版本，不将 delegated_ai 重写为 human；旧客户端不能正确展示
该来源时，应先恢复支持此合同的客户端，再操作含该类审阅的项目。
