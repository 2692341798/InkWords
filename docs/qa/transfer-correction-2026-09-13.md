# SS-01 迁移题修订与历史合同准入检查

日期：2026-09-13。状态：修订提案通过当前 v10 离线门禁；真实任务入口拒绝，未落库。

## 修订内容

重新读取实际章节 workspace，批准和当前修订均为 r15：
`d35e4382-0253-4c6e-be05-0bb13b0b10e7`，章节版本 15。
本轮从该修订的 document.sample 准备内容，不使用旧输出文件替代当前母稿。

保留 TestMethodSpecific 的方法隔离示范，把“独立迁移”改成未有完整示范的新约束：
新增 tryAddRoute，组合现有查找与登记，只接受同一方法、完整路径的首次登记。重复
返回 false 并保留旧 handler；另一方法仍可登记。使用 bool 并解释 true/false，避免
在零基础题目中突然引入未说明的错误类型。与上轮草案相比，信号由错误改为布尔值，
“明确拒绝且不覆盖”的验收目的不变。

题面限定单线程、以斜杠开头且没有连续/末尾空段的路径及非 nil handler，不宣称是
生产冲突检查。新测试要求同时检查返回值和真实调用标记，包含首次接受、重复拒绝
且旧函数保留、另一方法、其它路径及缺失路径；父子路径变式检验“节点存在不等于已
登记函数”。明确先独立提交、再看分层提示和参考算法，旧六个测试不能证明新任务通过。

同步修改正文、LearningArc 的 independent_transfer、PracticeSet 的迁移题/变式/
参考答案/提示/评分。保留六项任务 ID 与 mode、其它练习及两份原 Go 代码字节。
CE-01 方法介绍改为平行句式，并保留首次 POST 的显式中文定义与后续模拟边界。
没有替读者写完整新方法、执行练习或给出掌握成绩。

## 实际检查

通过已部署 llm-stream 容器内 textbook-recheck 的 correction-stdin 模式执行；该
工具没有 provider、候选持久化或教学执行端口。原 receipt 哈希不变：
`475d183cea860eee93219a684074f3103c61f8480d32524dca531163333752e2`。

首次检查发现独立迁移正文未显式说明“独立”以及 POST 定义格式缺失；保留失败输出，
补清这两处后重检退出 0：quality.passed=true、contract=inkwords.sample-quality.v10、
failures=[]、provider_calls=0、candidate_persisted=false。16 项引用、3 项关键事实
覆盖保留；长段落等原有软建议仍存在，自动通过不能代替内容审阅。

原正文哈希：`sha256:d8c57b17ba1242d4b0f370225a40b273f8df88af60a89a9dc616ee6ce816804d`。
提案正文哈希：`sha256:2ae488275cd6f271a96edaf114165c243a1c37d2967e0c8fa29004a6c78294ca`。

随后只提交一次本地修订请求，明确 baseline 15/r15，原任务
`570eed05-42bc-49a4-9624-977d6d46d2e3`。实际 correct-sample 入口返回 400
INVALID_STATE，读取的完整章节 workspace 前后相同，候选没有登记、没有模型调用。

## 准入障碍与下一步实现约束

只读查询确认原失败任务 prompt v16、quality v9、蓝图 r4。当前代码只为修订保留 v15
提示例外，并仍要求当前质量版本 v10；因此真实历史 v16/v9 在来源检查处被拒绝。
即使只扩提示白名单，MatchesCorrectionBaseline 仍按整个蓝图/证据输入匹配，而当前
批准蓝图已是 r6。离线内容检查通过不表示符合服务器的候选来源与当前版本合同。

下一步需要显式的历史→当前候选迁移，而非改变旧任务/旧 receipt/旧质量报告：

1. 单独验证已支持的历史 prompt/quality 组合和原失败任务、receipt、输入哈希；历史
   兼容只用于只读来源准入，不能让旧任务重新调用 provider。
2. 新请求必须明确当前预检输入哈希、当前章节版本及父修订，服务端读取当前批准合同。
   保留原来源输入哈希与当前候选输入哈希两种身份，新候选使用当前 v10 门禁。
3. 若允许蓝图添加其它章节后迁移，必须逐字段验证目标章、读者、BookContract、
   StyleSheet、来源与证据不变；同时把新蓝图身份写入候选。不得仅忽略全局蓝图哈希。
   目标章、证据或合同发生变化时应明确拒绝，不自动拼接来源。
4. worker 重新核对原 receipt、当前请求与修订内容并运行全部当前门禁，保留原模型
   用量但本次 provider_call_count=0；core-api 最终持久化再检查当前版本、锁与 CAS。
5. 添加历史 v16/v9 真实形状、当前版本变化、证据变化、跨项目、未知旧版本、篡改双
   哈希、幂等与并发失败测试；通过后再部署并提交此提案，不能将本次离线通过当作迁移实现。

本轮没有产品代码或依赖变更，没有重跑无关产品测试。旧批准稿、冻结构建、自学性失败
记录均保留。新候选后仍需审阅、批准与新冻结，旧审阅/收据不得自动搬运为新结论。

证据目录：`output/real-acceptance/2026-09-13/transfer-correction/`：
`chapter-before.json`、`chapter-after.json`、`prepare_edit.py`、`correction-input.json`、
`body.diff`、`proposed-manuscript.md`、`before-proposal.json`、`before-recheck.log`、
`proposal.json`、`recheck.log`、`submission.json`、`submission-result.json`、
`source-contract-identity.json`、`verification.json`。
