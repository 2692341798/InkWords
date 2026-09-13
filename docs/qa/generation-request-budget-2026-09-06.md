# 完整生成消息的预算检查

日期：2026-09-06。执行者：Codex；本轮 Provider 调用 0，候选写入 0。

## 修复

此前样章工作器只估算 system/task 指令与 Evidence.Content，遗漏 response schema、
证据 ID/快照/定位、JSON 转义以及适配器追加的安全说明。真实 programming 调用曾报告
11990 输入 Token，而旧 v15 检查估算 8298；这不是可忽略的完整性差异。

新增 SDK 无关的 `generation.InputMessage` / `BuildInputMessages`，把既有消息组装逻辑
原样移到共享合同层；DeepSeek 和 OpenAI 的非流式/流式端口仍使用同一组装路径。
系统指令、来源的不可信数据边界与实际消息字节结构保持，生成请求和 prompt v16 未变。
业务代码没有导入 Provider SDK，也没有跨服务读取其他服务源码。

`CheckRequestBudget` 统计整个渲染消息的 JSON，再计一份响应 schema，覆盖 OpenAI
同时携带的原生 structured-output 配置，另留 256 Token 包装余量。DeepSeek 只使用
JSON 模式时也保留后一份 schema 余量。工作器、真实冻结输入预检和离线夹具预检复用
该方法，超过输入额度就在缓存/调用之前拒绝，不截断正文或自动提高预算。

方法标识为 `utf8_rendered_messages_schema_reserve_v1`。它使用字符估算，包含额外预留，
不是精确 tokenizer，也不能保证实际用量绝不超过估算。core-api 的任务级入队粗估
`conservative_utf8_payload_plus_prompt_reserve_v1` 保持独立；页面上的约数不是最终账单。
真实供应商用量和费用来源保持原记录，不用本次估算倒改历史用量。

## 验证

- 先复现旧工作器在“仅正文恰好在预算内”的输入上仍调用 Provider；新检查在调用前拒绝。
- 只有 schema、定位、证据或快照 ID 增长而正文不变时仍正确拒绝；包含引号、换行、
  反斜杠、中文和 HTML 字符的证据原样留在数据区，JSON 转义纳入估算。
- 精确估算阈值与低 1 Token 的边界测试通过；无效 schema/请求/预算拒绝。
- Generation kernel、两个 Provider 适配器、样章模块及全量 Go/架构回归通过。

原两份真实冻结资料（每份 10 段）在当前 v16 输入下离线预检通过：

| 读者 | 完整消息估算输入 | 允许输入 | 预留输出 | 当前请求 hash |
| --- | ---: | ---: | ---: | --- |
| programming | 13232 | 20000 | 12000 | 215cea8654df0eb29bfcbab07273544e00787a37bb72201c75892b7e1957a2a6 |
| stack_familiar | 13237 | 20000 | 12000 | 59984bd034c16796cc937cc4b3556b5fbb17c16580e0b9a032e6bada937d482e |

这些是 v16 请求的估算，不能与旧 v15 的真实用量视为同一次模型调用的误差测试。
没有重复真实生成，原拒绝稿的源码机制与引用问题尚未修复，三类读者条件仍未完成。

llm-stream 已构建并部署，镜像为
`sha256:126f73aab1b2ddad3e87da8959da9f8547639249702dfceb44a04c670611fef6`，容器 healthy；
更新前 RabbitMQ 所有队列 ready/unacknowledged 为 0。Compose 渲染确认真实知识库
绑定保持。core-api 未重建；真实 Nginx 入口预检仍为 v16 与任务级粗估方法，章节回查
仍是 r7 candidate、原批准蓝图 r3。日志与回查 JSON 保存于
`output/real-acceptance/2026-09-06/request-budget-v1/`。

未改变请求、持久化格式、prompt/cache schema 或数据库；回退可恢复上一 llm-stream
镜像，旧章节无需迁移，但会恢复旧预算遗漏。其他服务及隔离设置未改动。
下一步回到原 programming 稿的前缀机制、
教学实现差异、排错恢复路径和来源修订，保留原稿及真实调用用量。
