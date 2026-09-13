# 真实冻结资料的分读者样章验收

日期：2026-09-06。执行者：Codex；本轮包含真实模型调用，仅保留评测产物，无母稿批准或真实读者试学。

## 验收入口修复

发现原真实生成测试调用 `sampleGenerationRequest(t)`，使用简化源码和占位快照 hash，
并仅改 audience 字段，不能支持“真实固定资料生成”的结论。入口现在必须显式提供
`INKWORDS_REAL_TEXTBOOK_REQUEST_DIR`，读取每个所选 audience 的同名 JSON 文件。
不自动修改 audience、契约身份或模型；缺失、身份不符、正文 hash 不符、占位快照 hash、
非固定 Git SHA、未知 JSON 字段或超出 4 MiB 都在调用前拒绝。所有选中请求先完成预算
预检，任一个输入不合格都不能先花费前一份调用。

原离线夹具测试仍只证明合同形状，不再供真实入口默认使用。真实输出目录必须为空，
避免覆盖原验收证据。调用返回后先保存完整请求、质量报告、正文与可用用量，再检查
验收门槛；失败时后续 audience 不启动，原始供应商错误正文不进入产物。
网络/解码失败可能没有可用的 SampleGeneration 用量，缺失不能解释成零次调用或零成本。
工作流增加 `textbook_request_dir` 输入；没有发起远端 workflow。

## 本轮冻结输入

从原真实 r3 收据提取 8 段完整源码，固定 Gin commit
`73726dc606796a025971fe451f0aa6f1b9b847f6`。保留 GET、handle、Engine.addRoute、
ServeHTTP、handleHTTPRequest、node.getValue、node.addRoute、longestCommonPrefix
的来源 ID、物理行号和逐字节 hash。再加入已真实导入的两段官方正文：

- Gin `https://gin-gonic.com/en/docs/routing/http-method/`，快照
  `38ae962f-cb37-4818-9412-b34c388ede46`，片段 `chunk-26f949afea44fcdd81a73ba8`。
- Go `https://go.dev/doc/tutorial/web-service-gin`，快照
  `b54e3131-f1f2-489b-a162-068795504af9`，片段 `chunk-827ddba3f6156b90f662f489`。

两段均说明方法和路径共同选择处理函数；本轮实时只读 API 确认 ID、文档与定位仍存在，
正文与 hash 从原成功导入结果提取并重新计算核对。没有重新抓取上游，也没有执行导入源码。

programming 与 stack_familiar 使用独立 `evaluation-` 项目、契约、章节身份，明确不同
已知知识与不可假设知识，契约内容重新计算 hash。它们是评测输入，不是假称原项目批准
过的修订；未写入数据库。原 r3/r7 不变。
输入保存在 `output/real-acceptance/2026-09-06/audience-frozen-v1/requests/`，每份
10 段证据，预检估算分别 8298 / 8303 Token，可用输入 20000、输出上限 12000。

## 真实调用与拒绝结果

采用已配置 DeepSeek / `deepseek-v4-flash`，每次至多 15 分钟，无自动重试或备用模型。
最多计划两次串行调用；第一份 programming 在 53063 ms 返回后被质量门禁拒绝，
stack_familiar 未调用。实际共一次：输入 11990、输出 8575 Token，缓存用量未报告，
费用未知。

programming 输入请求 hash 为
`sha256:cdb5fd2b6b288b111ffd82bbf47b04f6867f8e4d6775a18b051eb4824345c65a`，
正文 hash 为 `sha256:0041e3c2424225e5629784948255133092ad7b90e948d9ba39f7506c233696e2`。
原始结果、拒绝正文、冻结请求和日志保存在 `audience-frozen-v1/results/` 及 `run.log`。
没有产生可批准候选。主要失败为：

- complete/transfer 的 correctness、completeness 被错误标为需要运行证据；reproduce
  的 runtime/tests 反而不要求运行。当前合同正确拒绝这些不一致。
- 第一段项目初始化命令没有代码来源标记。
- 正文没有引用 node.addRoute 与 longestCommonPrefix，结构化压缩路径前缀事实的
  关键证据也未落实到正文，覆盖不完整。

Codex 进一步静态复核发现，正文仅带读者走到方法树选择及 getValue，未讲清要求的压缩
前缀分裂；教学程序按斜杠分段，必须明确与生产机制的映射及差别，不能只补上两个引用。
排错路径还建议删除整个练习目录重做，不适合作为默认恢复步骤。这些文字没有被执行。
尚未执行生成代码，文中的预期输出仍为未验证。

用同一请求与结果构造兼容既有离线工具的私有拒绝收据，hash 为
`7f9ec866508804bdc6d6670b054724da1cbb8c178419bfb360e6b230d75b00c5`。
`textbook-recheck` 在不调用模型、不创建候选、不运行代码的情况下复现同样的质量失败；
报告为 automated_offline_recheck / ProviderCalls=0 / CandidatePersisted=false。

本次实际输入 11990 大于预检估算 8298，仍低于 20000 的预算。源码核对发现预检只累计
指令与证据正文，而 DeepSeek 适配器还发送 response_schema 和证据元数据 JSON；这是
后续预算覆盖缺口，不能把当前估算宣称为严格上界或计费用量。

## 验证与下一步

冻结输入、身份拒绝、正文 hash、完整快照 digest、失败结果留存、旧离线夹具和预检测试
通过；样章模块回归、全量 Go/架构测试通过。严格输入检查补齐后，原两份请求再次通过
离线预检，未重复真实调用。实际项目只读回查仍为 r7 候选
`018e61ab-c48f-4eff-ae87-9aeca4a3209e`、批准蓝图 r3。未构建/重启产品服务，无迁移、
依赖更新、前端修改或远端写入。

下一步应把固定 rubric 运行标记交回系统策略管理，保留缺失条目/错误描述的质量检查；
完善按完整 Provider 请求估算预算；内容修订须完成前缀机制与教学版本边界，保留原稿
及用量，再重跑门禁和隔离验证。不能通过忽略失败来启动余下调用或勾选三类读者验收。
