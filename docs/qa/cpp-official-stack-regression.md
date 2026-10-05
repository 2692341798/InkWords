# C++ 官方资料栈回归

日期：2026-09-03

## 范围

本记录验证教材平台的受控抓取、资料检索和项目蓝图不依赖 Gin 或 Go 名称。它不导入、不执行或生成第二本 C++ 教材。

离线结构夹具以 `https://isocpp.org/get-started` 为入口、`https://isocpp.org/tour` 为同站深层页。夹具只检验 URL 边界、标题/标题层级与链接处理；不声称是上述官网在此日期的内容快照。

## 已验证的离线合同

| 合同 | 证据 |
| --- | --- |
| 同站入口与深层页会形成可追溯的两页清单，第三方链接被记为 `cross_origin` | `TestServiceCrawlKeepsBoundedCPlusPlusOfficialSiteManifest` |
| HTML 导航链接和嵌套标题会保留为解析事实，供清单和后续切分使用 | `TestExtractDocumentFactsHandlesNavigationAndNestedCPlusPlusHeadings` |
| 主资料和确认的官方 C++ 补充资料可被确定性检索；生成证据包仍必须含主资料快照 | `TestRetrieveEvidenceKeepsOfficialCPlusPlusSnapshotTraceable` |
| 含 CMake、`.cpp`、`.hpp` 和测试文件的项目清单可生成可验证的章节依赖和类型 | `TestPlanBlueprintAcceptsCPlusPlusProjectInventoryWithoutFrameworkBranching` |

运行：

```bash
cd backend
GOCACHE=/tmp/inkwords-go-build go test ./services/parser-service/domain/crawl -run '^TestServiceCrawlKeepsBoundedCPlusPlusOfficialSiteManifest$' -count=1
GOCACHE=/tmp/inkwords-go-build go test ./shared/platform/crawler -run '^TestExtractDocumentFactsHandlesNavigationAndNestedCPlusPlusHeadings$' -count=1
GOCACHE=/tmp/inkwords-go-build go test ./shared/kernel/textbook -run '^TestRetrieveEvidenceKeepsOfficialCPlusPlusSnapshotTraceable$' -count=1
GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/projectcourse -run '^TestPlanBlueprintAcceptsCPlusPlusProjectInventoryWithoutFrameworkBranching$' -count=1
```

## 2026-09-03 隔离环境真实抓取

在仅含开发身份的隔离 Compose 栈中，经本地网关向 parser-service 提交：

```json
{
  "entry_url": "https://isocpp.org/get-started",
  "allowed_path_prefixes": ["/get-started", "/tour"]
}
```

本次请求返回 `200`，清单状态为 `complete`，保留页面恰为：

- `https://isocpp.org/get-started`
- `https://isocpp.org/tour`

官网导航同时给出了两个尾斜杠别名。抓取器现用不改变请求 URL 的队列身份键去重，因此 `/get-started/` 与 `/tour/` 均被明确记录为 `duplicate`；相对链接仍按原始 URL 解析。实际响应还分别记录了 50 个 `cross_origin` 与 37 个 `path_outside_boundary` 拒绝项，未扩大到第三方站点或未授权的 isocpp 路径。

这是一次真实网络、路径边界和尾斜杠别名处理验收；它不是已在 core-api 保存的不可变 `SourceSnapshot`，也不证明 C++ 教材蓝图、AI 生成、人工审阅或导出通过。要形成教材输入，仍需由项目流程保存当次 SourceSnapshot；网页随后变化时必须重新抓取并创建新快照，不能复用本次结果。
