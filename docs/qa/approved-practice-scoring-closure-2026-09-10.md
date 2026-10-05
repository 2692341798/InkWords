# 批准练习的评分、纠正与显式应用闭环

日期：2026-09-10。操作者：delegated_ai。承接 `approved-practice-assessment-2026-09-10.md`。

## 结果与验收范围

正式 programming r2 的解释作答和代码作答，在独立操作验收数据库中使用生产页面、
review 应用/Store/路由、真实 core 来源接口、先前真实 Go 运行收据完成：
预检 → 显式重评分 → 逐项反馈 → 原文/来源复核 → 两类追加纠正 → 显式应用 → 新浏览器恢复。
已部署 local-evaluation-v2。PR-14 的模型评分与可追溯纠正功能项可以勾选。

这证明功能闭环，不证明模型无需复核、不证明个人已掌握六维能力，不代替整书读者试学。
本轮仍查出两处内容边界问题并实际纠正。所有作答仍标记 AI 操作验收，代码来自批准示范，
独立完成=false、查看参考答案的事实不变；不生成个人学习或真人审校记录。

## 请求改进与兼容性

原失败为文字原文引用不匹配、代码评分引用无效证据 ID。新增显式
`MASTERY_ASSESSMENT_PROFILE=local-evaluation-v2`：

- judgments 的数量、criterion_id 枚举绑定当前冻结 rubric。
- 判断、提示和补救的 evidence_ids 枚举只包含输入中真正的 source/runtime ID。
  `assessment-answer`、`assessment-learner-code` 只标识待评数据，不进入来源词表。
- 提示模型保留精确别名，不自行添加 evidence- 前缀；文字引用须为连续原文，不拼接或省略。
- 原文逐字检查、来源归属、运行快照绑定、缺运行保持未知及其他输出门禁全部保留。

v2 仍为 deepseek-v4-flash，文字 low/6000、代码无推理/6000、60 秒时限、32,000 字节输入上限。
v1 和旧合同请求哈希保持原值；只在明确选择 v2 时改变 schema/指令，变化包含在 RequestHash。
无需迁移或索引。回退可恢复 v1 配置，保留所有新旧任务和纠正/应用；v2 输出仍用已有 v6 读取合同。
默认示例仍为空，不会自动发起调用。

## 相同输入的真实对照

恢复上一轮隔离 DB 导出后，两次新任务的完整 input 与上一轮逐字段相等，InputHash 相等，
RequestHash 不同。来源、作答、题目、时间、提示记录和运行收据未替换，未新增代码执行。

| 作答 | v1 实际结果 | v2 实际结果 | v2 输入/输出/缓存 Token | 耗时 |
| --- | --- | --- | --- | --- |
| 解释 | accuracy 原文引用拒绝 | 五项 4/4，合法引用 | 4,415 / 5,679 / 640 | 26,884 ms |
| 补全代码及同快照运行 | correctness 证据 ID 拒绝 | 五项 4/4，运行/测试引用真实收据与受测代码行 | 10,295 / 741 / 1,024 | 3,058 ms |

本轮恰好两次 Provider 调用，输入 14,710、输出 6,420、缓存字段合计 1,664；缓存属于 Provider
返回的用量字段，不额外加到输入总数，费用未知。两次预览分别 14,715 / 30,145 字节。
原两条失败任务完整保留，新任务通过 retry_of 关联：

- 解释新任务 `36f6d04e-4fe1-4de9-b677-5649cd5e3f5f`，前次 `4f2d343b-d113-47e7-8eaa-bf364ab3dd95`。
- 代码新任务 `24913a21-3867-4852-8bfb-90c5d85ffb20`，前次 `c8413af8-ac73-4bba-bd92-f260b429dbba`。
- 运行收据仍为 `1232ce4d-091e-4b19-8a15-51e6d38fb4e5`，作答快照仍为
  `sha256:fb2cc7178c02c1c26e2497cec595654b3c7713edd11c6652feaa2880e2203edc`。

测试专用 Port 包装器保存这两个非个人样本的结构化请求和正常模型返回，0600；不保存密钥
或 Provider 错误正文。该包装器不进入生产二进制，独立请求预算仍为两次。

## 实际纠正与应用

1. 解释作答声称“错误创建订单”，现有证据只能证明选错处理函数，未证明真实业务副作用。
   页面将 causality 从 4 改为 3，原因、原作答引文和冻结来源一并追加；原始模型五项 4 保留。
   应用记录 `f7e16c1e-c0e0-4e3a-a61f-bb05db781a1c`，sequence 1。
2. 代码补救建议将 Gin 概括为“循环结束后才写 handlers”，但冻结源码空树分支直接调用
   insertChild/return，其他分支也在循环内部处理/返回。页面修正补救文字，区分教学分段树
   与真实字节前缀树，保留分数、原始建议和同一来源。应用记录
   `b5f4a34d-7a1e-40e8-aa13-d0d8323af62a`，sequence 2。

两种纠正均标明 delegated_ai。修正后需另点“应用当前评分并更新复习安排”，没有再次调用模型。
旧解释作答应用时，后续代码自评仍影响当前安排；代码应用后，由原来的 complete 未通过安排
变为 reproduce 的低脚手架练习，仍保留独立性/提示依赖限制。due_at 从原作答时间推导，
没有伪造新作答时间、延迟保持间隔或掌握状态。

两条过期反馈 hash 的纠正请求都返回 409，保存状态逐字段不变。同一应用 ID 重发返回原 ID
与 sequence，不新增应用。两次原始作答数组与上一轮完全相等。

最初尝试在 CLI run-code 中监听请求导致工具 Session closed，未计通过。改用原生 CLI
requests 命令后，新浏览器打开页面成功恢复评分、纠正与已应用状态：只新增正常开始练习的
practice-sessions POST，没有 assessments/verifications POST。截图已实际查看。

## 部署、回归与清理

review-service 新镜像 `inkwords/review-service:grounded-identities-v1-20260910`，
实际 ID `sha256:ce538f7d2bdf22dfc72ff2b8fd7cc752c46c7b87d4a547c6ca888b6eb1122779`。
生产 profile=v2、模型正确、密钥存在、health=healthy，外部网关 mastery/due 返回 200。
部署前无正在评分的生产任务；沿用上一层 core 引用别名修复及其余全部运行配置。

聚焦失败测试先证实旧 schema 没有限定任务数量，修复后评分包通过。旧 v1 请求指纹检查发现
不应隐式修改既有配置，因此改为显式 v2 后，旧指纹与全部 review-service/架构回归均通过，
包含 PostgreSQL 持久化、纠正、应用、恢复及运行输入检查。前端产品代码未修改。

独立 DB 最终为 1 目标、2 原作答、4 评分任务（2 原失败、2 新成功）、2 纠正、2 应用、1 原运行。
生产三章节 workspace 与个人目标完整响应和上一轮 baseline 完全一致。
临时 DB 导出已保存为 0600，随后正常关闭 review、删除临时 DB、停止专用 runner。

证据目录：`output/real-acceptance/2026-09-10/approved-practice-scoring-v2/`。
截图：`output/playwright/approved-practice-scoring-v2/applied-restored.png`。
后续部署从本证据目录 `runtime-build/deploy.py` 续接，避免回退评分配置或前一轮 core 修复。
