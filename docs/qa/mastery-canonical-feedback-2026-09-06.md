# 统一评分判断与反馈投影

日期：2026-09-06。工程验证通过；后续 v6 真实模型验收已执行但未通过，候选未部署。
最新结果及推理选项实验见 [评分推理验收](mastery-reasoning-options-2026-09-06.md)。
下文保留同源投影实现阶段的原始验证记录。

## 一次诊断调用得到的新证据

新增 INKWORDS_REAL_ASSESSMENT_CASE 可只选择一个已知样本，并保留原样本索引和输入
身份。未知名称在调用前失败，run-scope.json 标记 diagnostic_subset；目标收据或
run-scope 已存在时拒绝覆盖。子集通过不能冒充完整验收通过。

本次只调用 procedural-only-answer 一次，使用原 v5 请求：
sha256:e687af9a70f96a3c759c3f6504ac60b9680f6e3782239c4f723c5ebe8325d9ba。
DeepSeek/deepseek-v4-flash 返回分数 3/3/2/3/3，结构和原分数门槛通过。输入 2898、
输出 1476、缓存 2816 Token，延迟 7461 ms，费用未知；其余样本没有调用。

同一请求此次通过，不能回填上一轮拒绝的具体原因。自动语义复核发现 completeness
和 clarity 的原始理由仍分别增加“查找不会临时创建处理函数”和“getValue 返回值处理”
作为完整性要求，而对应 decisions 均为 satisfied/gaps=[]。missing_points 又独立
生成了两个遗漏，只有因果缺口在 decisions 中出现。原结果保持，语义验收仍失败。

证据位于 output/real-acceptance/2026-09-06/decision-diagnostic-v1/。

## v6 只生成一份评分判断

v6 输入采用 inkwords.criterion-coverage.v2。Provider 只返回 judgments、next_hint 和
remediation；每个 judgment 包含原 criterion_id、score/status、作答引文/路径、来源、
gaps 和 unknown_reason。服务器从冻结 rubric 解析要求原文，不要求模型再复制一份。

评分说明和答对/遗漏/误解列表全部由 judgments 投影：

- satisfied 且无缺口：3/4 分使用固定说明，答对部分引用该冻结条目与同一来源。
- unsatisfied：0–2 分，说明逐字使用已验证 gap.reason；kind=omission 或 misconception
  决定进入遗漏或误解列表。不能另外生成一份独立扣分说明或遗漏。
- unknown：null，并保留有界 unknown_reason；同一工件运行证据缺失时仍不能给运行分。

gap.requirement_quote 仍必须来自本项冻结要求，单个原因最多 300 字，最多 5 个 gap。
重复/未知条目、引用和分数矛盾、额外旧格式字段、缺少 score 等情况均拒绝整份结果。
这保证几种显示内容同源，不能证明模型判断或缺口原因本身一定正确。

Result.Judgments 保存模型原判断；Result.Feedback/Decisions 是确定性投影。成功保存
和读回都会重放并对比，不能独立篡改投影。用户纠正仍单独追加，只影响有效反馈，不能
改写模型判断；运行证据未知与原作答不可变边界保持。旧 v1–v5 路径保留，v5 四份请求
hash 与改动前逐一相同。

v6 四份离线请求为 10558/10067/9727/9798 字节，均小于 32000；输出上限仍为
3000 Token。新格式比 v5 每份少 497 字节，此处是请求字节比较，不是实测 Token 节省。
未来真实入口需显式 INKWORDS_REAL_ASSESSMENT_JUDGMENT=approved，本轮未开启它调用。

## 已完成的验证

- 聚焦与全量 Go/架构通过；真实临时 PostgreSQL 的代码作答→评分→纠正→应用通过，
  原 judgments 保留、运行仍为 null。独立投影篡改拒绝测试通过。
- 用失败测试发现投影分数/建议与原判断共享可变指针；修正为独立副本后，修改投影不再
  改原判断或建议。缺口、错误类型、未知原因、伪引文和旧协议兼容测试通过。
- 对刚才的 v5 真收据做显式离线转换，原文件字节不变。v6 投影只保留一个有绑定的因果
  缺口，完整性/清晰度不再额外生成扣分理由；这是投影重放，不是新的 v6 模型输出。
- 71 文件、273 项前端测试通过，lint、TypeScript/Vite 构建和包体预算通过。
- 实际 Playwright 浏览器展开离线投影的原判断、缺口和冻结参考答案，截图已视觉检查，
  控制台无错误/警告。页面明确说明离线转换，无真实学习库写入、评分应用或人工审阅。

本地产物目录 output/real-acceptance/2026-09-06/canonical-grading-v1/，包含来源收据
hash、offline_projection_from_v5、Provider 调用 0 的重放文件、工程日志、请求预检、
浏览器快照和最终源码 hash。截图 output/playwright/canonical-grading-2026-09-06.png。

## 检查点

工作树默认新评分已组装 v6；当前服务仍为
sha256:489994ae1b9371835e9d28034645aefd8eab91d752121c615d6031a8270f2a05，healthy，
网关到期任务查询成功。本轮没有构建/部署 Docker 服务，没有迁移、依赖升级或远端写入。

下一步对固定四份答案执行一次有上限的 v6 真实验收，并逐项核对判断、缺口和来源；
离线转换不能替代它。PR-14、PR-16、真实学习者六维任务与延迟试学仍未完成，候选不能
随其他工作一并部署。
