# 2026-09-06 官网资料栈真实导入与检索

本记录来自本机实际浏览器操作、联网抓取、正式任务 API、PostgreSQL 与容器运行。
这些是自动化操作证据，不是人工编辑审校、读者试学或出版授权。
项目：`4db4fb0a-35db-431c-b0b3-fe695c3677e5`。本轮新增模型调用为 0。

## 实际结果

| 范围 | 任务 | 快照 | 文档 / 片段 |
| --- | --- | --- | --- |
| Gin 官网 `/en/docs`，旧解析 | `5027fc50-48e4-48bc-a729-5141cbda4fff` | `9b0a6961-7325-4e93-9cab-b39bcf15925f` | 87 / 11647 |
| Gin 官网 `/en/docs`，v2 正文解析 | `0749fefa-0e77-4f05-adc6-52d228e5bd94` | `38ae962f-cb37-4818-9412-b34c388ede46` | 87 / 1947 |
| Go 官方教程 `/doc/tutorial`，v2 | `244710cb-2cee-41b8-9ed8-ed95dc0526c7` | `b54e3131-f1f2-489b-a162-068795504af9` | 18 / 1165 |

来源入口分别是 [Gin Documentation](https://gin-gonic.com/en/docs/) 与
[Go Tutorials](https://go.dev/doc/tutorial/)。只访问已确认同站路径，未扩展到第三方链接。
新的两个资料栈合计 105 文档 / 3112 片段；加上原有源码和保留的旧解析快照，
实际项目资料库显示 201 文档 / 15154 片段。不能把总数视为去重后的新资料量。

Gin v2 任务 14:05:37–14:06:05 成功，抓取 7,357,709 字节。正文探针 `Docs`、`Blog`、
`On this page` 的独立片段数为 0；168 段标注 Go，其中 163 段保留多行代码。
首页完整 `main` 示例的缩进、空行、URL、章节标题路径均保留。
抓取清单明确记录 87 kept、9054 duplicate、1311 path_outside_boundary、456 cross_origin、
10 http_404、2 unsafe_url。`complete` 表示已处理限定队列，不表示失效链接有正文。

Go 首次任务在结构化校验时失败，未入库。正式 crawl 诊断得到 29 条记录，
其中 11 个最终 URL 重复（`.html` 别名跳转）。补齐最终地址去重及相对链接基准后，
通过页面“按原范围重新抓取”重试同一任务，retry_count=1，14:11:16–14:11:24 成功。
最终 18 个 canonical URL 唯一，720,255 字节；1 个 duplicate_destination，
其余队列重复在请求前跳过。页面未标注代码语言，因此语言为空，不自动猜测；
162 个片段保留换行。本轮未执行导入的上游示例。

## 身份与不可变性

v2 任务固定 `parser_version=inkwords.official-html.v2`；旧 v1 任务仍使用原 HTML tokenizer。
新快照内容 hash 同时绑定网页清单 hash 和解析版本，不能复用旧解析快照。
成功的完整 crawl manifest 同时保存在 task result 与 snapshot limits JSONB。

- Gin 原始 manifest：`sha256:5bdebf0ca2b3c2bce344055436ff2013308796f1ac95059c8806510f99cfb4fa`
- Gin 解析内容身份：`sha256:859b5b26d0f93a7c745bf3e1350dd919406a4fea72626024299e5e9ab0db2233`
- Go 原始 manifest：`sha256:d9ff85f03f9e5697ee712fa30482cd6282ad10e0c16a1d4fff11a4ae9129baea`
- Go 解析内容身份：`sha256:4ccef5c5ac86707372ee832922c56a5a4dfe5b8f3afa1daa4809c4962c0cf3a8`

原候选 r7、批准蓝图 r3 和既有运行证据没有被这次导入替换或批准。

## 真实检索缺陷与修复

浏览器查询 `albums` 首先错误返回了 8 段无关主资料：数据库先取最老 500 段，
且主资料即使没有关键词命中也得 2 分。修复后先参数化匹配，再取最多 500 段并本地排序，
无匹配不得加主资料分。数据库测试覆盖 502 个旧片段之后的目标、字面引号/通配符和无匹配。

同一界面复测返回 Go 官方的 REST API/Gin 与数据库教程，均显示 `正文关键词:albums`。
检索记录 `7efdf2c0-ec8b-4829-9b1a-bde6011782c7`：47 候选 / 8 选中，
全部绑定 Go 新快照。这里只建议证据，没有自动修改蓝图。
真实 EXPLAIN ANALYZE 19.623 ms，47 行；复用已有文档片段索引，无 schema/index 变更。

## 验证与边界

- 最终全量 Go 与架构检查通过：`/tmp/inkwords-official-retrieval-final-go.log`。
- 真实 PostgreSQL 回归通过：v1 保留、v2 JSONB 审计、幂等重放、版本篡改拒收及 >500 段检索。
- 前端全量 267 项：266 通过，文件遍历网关测试一次超过 5 秒；单独重跑网关与导入面板 11 项全部通过。
- 前端 lint 通过；Docker TypeScript/Vite 构建通过；core-api、parser-service、frontend 顺序构建并实际部署。
- 修改范围 `git diff --check` 通过；全工作区检查曾因 `mmap failed: Operation canceled` 未完成，不冒称全工作区干净。
- 原始结果和查询计划位于 `output/real-acceptance/2026-09-06/official-stack/`。

剩余：partial checkpoint 尚未接入官网导入任务界面；旧快照间重复命中尚未彻底消除；
Go 整个语言手册/标准库不在此次教程边界；静态 HTML 提取不承诺视觉结构完全还原。
资料权利审查、三类读者真实章节、人工审阅、学习闭环和出版审校仍未完成。
