# InkWords 可自学教材平台基线

日期：2026-09-02
资料夹具：`backend/services/llm-stream/app/projectcourse/testdata/textbook-gin-fixture/`
计划阶段：PR-00 / M0

## 目的与边界

该基线用于防止教材平台在功能增加后出现“内容更多、但更难自学或更不可信”的回归。它不是一份 Gin 教材成品，也不代表真实模型、真实 sandbox、目标读者试学或出版审校已经通过。

夹具只包含 Gin `v1.12.0` 的少量公开源码片段、固定提交、行号、内容 hash 与官方资料入口：

- 主资料：`https://github.com/gin-gonic/gin`，ref `v1.12.0`，固定 SHA `73726dc606796a025971fe451f0aa6f1b9b847f6`，MIT；
- 官方补充：Gin 官方文档、版本化的 Go Package Reference 与 Go 官方文档；
- 代表章节：零基础“路由如何抵达处理函数”概念章，以及“中间件链”实操章；
- 不镜像完整仓库，不下载资料，不执行 Gin 的构建、测试、安装脚本、hook 或服务。

源码片段仅用于验证证据定位合同。完整教材生成时必须重新使用当次导入的不可变 SourceSnapshot，而不是把夹具当作生产资料库。

## 自动验收

`TestTextbookBaselineFixturePinsGinEvidenceAndQualityContract` 验证：

- 主资料的仓库、tag、40 位 SHA 与 MIT 许可证固定；
- 官方资料仅来自 `gin-gonic.com`、`pkg.go.dev`、`go.dev` 的 HTTPS 地址；
- 每个证据都有 path、symbol、行区间、短片段和 SHA-256；
- 概念章和实操章都能回链到现有 evidence ID；
- 必须识别术语墙、循环定义、未解释缩写、隐藏前提、类比缺失边界、无证据事实、未验证可运行声明、缺恢复路径和伪造运行证据；
- 出版候选的正确性、自学性、结构、证据、代码、迁移能力、可编辑性评分门槛均为 3/4。

运行：

```bash
cd backend
GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/projectcourse \
  -run 'TestTextbookBaselineFixturePinsGinEvidenceAndQualityContract|TestOfflineInkWordsAcceptance' -count=1 -v
```

该命令不访问网络、不调用模型，也不会执行上游仓库。

## Token、输出与人工质量状态

当前离线基线模式为 `deterministic_fixture`：模型调用数为 0，输入 Token、输出 Token 和延迟均为 `not_collected`。这不是零成本生成的结论，而是当前阶段没有在未配置用户 API Key 的情况下进行真实 Provider 调用。

真实样章测量必须以相同 Gin SHA、相同 BookContract、StyleSheet 和 evidence pack 运行，并记录：

| 指标 | 状态 | 采集位置 |
| --- | --- | --- |
| provider/model | 待 PR-11 | Generation usage record |
| input/output/cache Token | 未采集 | Generation usage record |
| P50/P95 延迟、重试次数 | 未采集 | task stage metric |
| 概念章、实操章原始候选稿 | 未生成 | immutable candidate revision |
| 目标读者 4 分制评分 | 待试学 | ReaderTrial record |

不得用自动测试或开发者自评替代零基础读者试学；当前人工状态为 `pending_target_reader`。

## 已观察限制

以下限制来自当前实现，均不是“已解决”或“已验证”的替代说法：

| 编号 | 当前事实 | 后续计划 |
| --- | --- | --- |
| `LAB_VERIFICATION_DISABLED` | `TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED` 默认关闭；未经隔离执行器验证的教学制品只能标未验证。 | PR-12 |
| `DOCX_TEXT_FLATTENING` | DOCX 解析目前会移除 XML 标签，未保存段落、表格和样式结构。 | PR-07 |
| `BLOG_SERIES_PDF_ONLY` | PDF 导出围绕博客系列，不是教材母稿 AST 的整书构建。 | PR-15 |
| `NON_FSRS_REVIEW_PICKER` | 今日复习先选未复习内容，再选最久未复习内容，不是 FSRS。 | PR-14 |

## 质量比较规则

后续 PR 必须在同一夹具上比较质量，而不是只报告更低的 Token：

- 关键 claim 无证据、错误引用、伪造运行结果、未验证代码却声明可运行均为硬失败；
- 质量检测器只能报告风险，不能伪装成人工试学、同行评审或出版审核；
- 新样章的人工评分不能低于 3/4，且不得低于先前已批准的同类样章；
- Token 优化不得删除必要的前提、恢复步骤、反例、边界或证据；
- 真实 Provider、官网抓取、IDE 截图、sandbox 与出版排版校对都属于 opt-in 验收，未运行时必须明确展示未验证状态。
