# 生成练习的固定运行证据策略

日期：2026-09-06。执行者：Codex；自动化工程验证，无人工审校或学习者作答。

## 变更与边界

真实 programming 拒绝稿暴露出重复决策：`PracticeRubricDimensions` 已固定哪些维度
需要运行证据，Provider 输出合同却仍要求模型填写同一布尔标记。样章生成现在只要求
每项 rubric 的 id 和题目相关 description，由生成边界在解码后、校验和渲染前根据
共享策略补入 `requires_runtime`。模型输出的相反布尔值不能改变系统规则。

仅 runtime、tests、fix_verification 要求运行证据；正文和题目中的运行结论仍需实际
VerificationRun，补入标记不会产生证据或把 unverified 改为已验证。

缺失/未知/重复维度、无效描述、任务或模式不完整、错误来源、延迟要求不足，以及
模型已渲染的练习与最终结构化合同不符，仍由原校验拒绝。模型评分描述的语义正确性
仍需独立审阅，非空检查不等于语义通过。没有改动持久化 PracticeSet v1、质量合同 v9、
既有收据重检、人工修订或批准边界；不会对旧章节批量补写字段。

Provider schema、形状示例和指令同步移除模型负责填写的布尔值；prompt/cache schema
升级为 v16。core-api 与 llm-stream 必须一同更新；旧版本任务仍按版本不匹配拒绝，
不静默改写排队输入。更新前真实 RabbitMQ 所有队列 ready/unacknowledged 均为 0。

## 原真实稿的离线对照

使用上轮固定收据 `7f9ec866508804bdc6d6670b054724da1cbb8c178419bfb360e6b230d75b00c5`，
将原章节完整深拷贝后应用新的生成策略并重新渲染练习；文件和内存中的原稿均保持。
这项对照没有把旧收据冒充新模型调用，也没有保存候选。

运行标记失败消除，仍有 6 项失败：命令代码来源缺失、node.addRoute 与
longestCommonPrefix 两处引用缺失、对应两处关键事实未引用、关键事实覆盖不完整。
原内容 hash 为 `0041e3c2424225e5629784948255133092ad7b90e948d9ba39f7506c233696e2`；
对照生成内容 hash 为 `111219bc9a4311dd777373d0c2082bcaefabdce36f6645442590e96373d33e73`。

最终报告保存在
`output/real-acceptance/2026-09-06/practice-runtime-policy-v16-checked/runtime-policy-comparison.json`，
origin 为 automated_offline_policy_comparison，ProviderCalls=0、CandidatePersisted=false。
原 v15 收据继续按原值重检并失败；内容机制缺口不因这次系统修复而消失。

## 验证

先运行失败测试，确认缺省和相反布尔值均被旧实现拒绝；实现后这两类输出通过固定策略，
九类内容/绑定错误仍失败。样章模块、共享教材合同及真实收据的离线对照通过。
全量 `GOCACHE=/tmp/inkwords-go-build go test ./...`（含架构）通过。针对本轮文件的
格式检查无新问题；计划文件第 3、4 行原有 Markdown 双空格换行保留。

core-api 与 llm-stream 顺序构建并部署成功，两个容器均 healthy。真实知识库绑定仍为
`/Users/huangqijun/Documents/obsidian_knowledge/knowledge`。镜像身份：

- core-api：`sha256:d8b0672a7a95b3b03dbc78b0b35e5044ecf6bb8a509a31575240da5de60b1ff4`。
- llm-stream：`sha256:a1a46107a811875673701c60309cd2d1967263bd5fb8e58929ea96836c4ac2bc`。

Nginx 本机入口的 generation-preflight 从 v15 更新到 v16，质量仍 v9、已选模型仍为
deepseek-v4-flash；输入身份随版本从 `cea8e438...` 变为 `3566c9d7...`。
前后完整预检与部署后工作台回查保存在上述 checked 目录中。
通过 CUA 打开真实本机页面，项目列表正常显示 Gin 从零自学教材、已有历史博客与
当前 0 项到期任务。该页面检查只证明入口与实际数据加载，不是生成流程或人工审校验收。
API 回查章节仍为 r7 candidate，current revision 为
`018e61ab-c48f-4eff-ae87-9aeca4a3209e`，原批准蓝图 r3 保持。

本轮无数据库迁移或持久化格式变化；回退需成对恢复两个服务镜像，继续拒绝版本不匹配
任务，不改写旧任务或章节。未更改 course-runner、review-service、前端和隔离设置。

本轮不重复真实模型调用。下一步仍需完成教学分段树与 Gin 压缩前缀树的机制对照，
并处理包含 schema/证据元数据的完整输入预算估算，再继续三类读者真实生成验收。
