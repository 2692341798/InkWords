# 带图构建的技术与发展性审阅

日期：2026-09-13；执行者：Codex（delegated_ai）。当前构建：
`966e247b-0473-4d54-936c-c1fd30ae92be`。

## 结论与实际证据

技术审校通过 3/4，仅覆盖当前冻结单章。通过数据库只读参数化查询重新取得该章
16 个引用片段，逐项确认计算内容哈希、存储哈希、冻结引用哈希与定位一致。
复查固定提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 的 GET/handle/addRoute、
ServeHTTP/handleHTTPRequest/getValue、longestCommonPrefix 与节点分裂代码；
正文明确区分教学方法树/斜杠分段与生产 Gin 的前缀压缩、参数和重定向等边界。

母稿两段 Go 与 ZIP 的 `main.go`、`main_test.go` 原字节相同；既有收据
`38db421a-3978-4e0f-9726-6b6d53ea5b12` 绑定工件哈希、清单哈希、Go 1.26.8、
`go_test` 及退出码 0，2026-10-13 到期，当前未过期。保留 r15 直接父候选 r14 的
同源工件身份，没有补造 r15 工件。本轮没有执行教学代码或调用模型；收据不证明
全新 Mac 安装、`go run`、真实 HTTP 服务或学习者掌握。

发展性审校需要修改，2/4。当前批准 BookContract v1 要求 `concept`、`hands_on`
两类代表章；Blueprint v4 及批准章节集合只有一个 `concept`。章内有动手小节
不能替代已批准的 `hands_on` 代表章。`sampleProfileCoverage` 实现从批准合同与
当前批准蓝图下的修订计算覆盖，缺口是实际状态，不是历史标签。

这项发现不要求删减合同以使门禁变绿。后续应补充有目标、来源、先修关系和验收标准
的 hands_on 蓝图候选，完成生成、运行验证及显式样章审批，然后冻结并重新审阅。
现有单章可以继续作为已审校样，不能代称完整多章教材。

此外，冻结教案的三个采集点中，已登记素材只覆盖源码和测试布局，缺少
`sandbox-verification` 运行详情截图。本轮已请求将本地 screenshot 脚本授权扩展到
该浏览器窗口；未收到答复前没有换用采集技术，也没有把静态图片说成运行截图。
WebP 新依赖授权同样待答复。独立试学、个人六维和延迟保持证据仍缺。

## 真实保存与回读

通过真实工作台独立的“用户委托 AI 整书审阅”表单保存：

| 阶段 | 记录 ID | 版本 | 结果 |
| --- | --- | --- | --- |
| technical | `f0eb1d9f-03dc-458a-a5c8-0baf2893a53a` | 1 | pass、3/4 |
| developmental | `ffb65cb3-1b68-45bd-9e39-4d4c6623e9e6` | 1 | needs_revision、2/4 |

刷新并重新打开工作台后两条记录恢复，发展性具体阻断可见，出版候选禁用。
此前 layout v1 保留；总计三条 AI 记录、零真人记录。API 与 ZIP 的记录逐字一致。
24 项 ZIP 内容哈希及 manifest 哈希通过；冻结 AST、manifest、Markdown、视频教案
不变；新 PDF 与已查看版本仅日期字段不同。

API 仍有 14 项阻断，ZIP 仍有 16 项：技术缺失项消除，发展性缺失转为明确的未通过
和具体缺口，计数未下降不代表本轮没有进展。其余五阶段仍无审阅，权利、代表章、
截图与个人学习等待办保留。旧构建及其历史审阅未修改。

本轮没有产品代码、模板、依赖或部署变更，未重跑无关产品测试。实际验证是来源与
工件核对、收据范围/有效期检查、真实保存、刷新及 ZIP 回读。

证据目录：`output/real-acceptance/2026-09-13/media-content-review/`。
主要文件：`source-chunks.json`、`workspace.json`、`manuscript.md`、
`verify_content.py`、`content-evidence.json`、`editorial-after.json`、
`saved-review-verification.json`、`review-bundle.zip`。
ZIP SHA-256：`86419780b5613f74775603699476f724f1e6b1417ce97643800402cf98b3e212`。
