# 官方文档栈抓取器

## 2026-09-10：平台标签页解析 v3

新建任务仍使用 task/result v2，但解析器标识升级为 `inkwords.official-html.v3`。
标准 ARIA tab/tabpanel 的非活动内容连同平台标题进入快照，避免 Go 安装页漏掉 Mac 步骤。
旧 v1/v2 解析器继续按冻结版本重放，普通隐藏元素和脚本仍排除；见 `official-web-tab-content.md`。
以下 v2 说明为历史规则。

## 2026-09-06：正文解析 v2 与快照审计

新建官网导入任务使用 task/result v2，冻结 `parser_version=inkwords.official-html.v2`。
现有 `golang.org/x/net/html` 负责 DOM 构建；依次选 main、article、body，排除导航、侧栏、页脚和隐藏控件。
高亮代码按行容器拼接，保留空格、换行和语言；代码内出现反引号时使用更长围栏。
这仍是静态 HTML 提取，不执行网页脚本，也不声称恢复所有站点的视觉排版。

v1 冻结任务继续使用原 tokenizer。v2 内容身份由原始 `SnapshotInputHash` 和解析版本共同派生，
同一网页字节也不会复用 v1 的旧解析快照。历史快照不迁移、不覆写；回退需要保留可处理 v2
的 worker，或先终结 v2 在途任务，再将新任务入口回退，不能把 v2 静默降级成 v1。

完整成功 manifest（页面响应元数据、边界、预算、跳过决定）随任务结果与 snapshot limits JSONB 保存。
worker 重算原始 manifest hash；core 校验身份、版本、预算、完整状态和页面到文档的一一对应。
JSON 审计最多 8 MiB。原始 manifest hash 与解析后内容 hash 分开保存；无 schema/index 变更。
目前官网导入任务失败时仍只保存错误消息，未把 partial checkpoint 接入任务界面；
独立 crawl API 的 checkpoint 恢复不代表资料导入链路已支持恢复。任务重试会重新访问官网，
冻结的是范围和规则，不是尚未抓取的远端字节。

Go 教程真实导入发现 `.html` 别名跳转后重复收录同一 canonical URL。
抓取器现在按成功响应的最终 URL 保留一份页面；相同字节的别名写入 `duplicate_destination`，
已下载字节仍计入总预算。别名返回不同字节时拒绝完整导入。相对链接以最终 URL 为基准。

真实资料栈超过 500 段后，检索必须先按同一关键词归一化规则在 PostgreSQL 过滤，
再取最多 500 个匹配片段进入本地排序。使用参数化 `strpos`，查询中的 `%`、`_` 和引号
保持字面意义。较新快照优先进入候选窗口；主资料只在实际匹配后获得优先分。
旧快照仍可显式引用；当前检索尚未彻底消除多个历史快照之间的重复命中。
这不是向量检索或全局最优排序。现有 15,154 段真实资料的 `albums` 查询匹配 47 段，
EXPLAIN ANALYZE 为 19.623 ms，复用 `idx_source_chunks_document_ordinal`；未新增索引，
无额外索引写放大或存储成本。更大资料库仍需测量扫描成本后另行设计索引。

## 决策

抓取器以 `parser-service/domain/crawl` 的 `Policy`、`Service` 与 `Manifest` 为业务边界：入口 URL、允许主机与路径、深度、页数、单页/总字节、请求/总超时和并发上限必须明确。任何跳过、失败、robots 拒绝、跨域跳转或预算耗尽都会写入清单；未完成清单标记为 `partial`，并带下一个 URL 与可恢复的 `checkpoint_id`。

执行层通过 `Fetcher` port 注入。领域层不信任网页正文，正文只是教材证据候选，永远不能改变系统、开发或生成提示。

## 安全边界

- 只接受无用户信息的 HTTP(S) URL；拒绝 `file:`、`localhost`、私网、loopback、link-local 与特殊用途 IP。
- 每个候选链接和每个重定向都必须重新检查允许主机、路径前缀与 URL 参数；搜索、分页、会话和 token 参数默认跳过。
- 生产 `Fetcher` 必须在 DNS 解析后的连接层再次拒绝私网地址，以防 DNS rebinding；不能只在字符串层过滤 URL。
- robots.txt、`Retry-After`、限速和低并发是执行层的必选能力，不是最佳努力选项。

## Colly 选型结论

已调研 [gocolly/colly](https://github.com/gocolly/colly)：它采用 Apache-2.0，提供域名限流、正文大小、递归深度和 robots 支持。需要特别注意其默认 `IgnoreRobotsTxt` 为真，接入时必须显式关闭，且仍由 InkWords 的 `Policy` 负责 allow-list、SSRF、跳转审计与持久化 manifest。

本轮没有为使用 Colly 而增加依赖。取而代之的是标准库 HTTP 适配器：它关闭代理、连接前解析并拒绝私网地址、重定向前把目标交回 `Policy` 复核、缓存并执行 robots.txt 规则，且串行处理请求以稳定地遵守 `Retry-After`。这使本地夹具可验证 DNS rebinding、robots 与重定向逃逸；常规 CI 不访问互联网。

## 恢复语义

`parser-service` 将 partial manifest、原策略、待抓取队列和已处理 URL 以 0600 的 JSON 文件原子写入本地 `crawl-checkpoints` volume。恢复 API 只接收该 opaque `checkpoint_id`，并使用保存时的完整策略；它不能通过恢复请求扩大 host、路径或资源预算。请求取消或总超时时，正在请求的 URL 保留在队首，恢复后会重新请求；已完成页面则不会重复抓取。

checkpoint 只保存抓取元数据，不保存网页正文，也不会自动成为教材资料快照。预算耗尽仍是用户需要缩小范围或创建新策略的显式 partial 结果，而非偷偷放宽预算继续抓取。

## 条件再验证与去重边界

`PageManifest` 保存来源响应的 `ETag` 和 `Last-Modified`。`Service.Revalidate` 只接受完整清单、原始安全策略和有效的 `SnapshotInputHash`；它会对每一页发送条件请求，并继续执行 DNS、robots、重定向和 allow-list 检查。

只有所有页面都返回 `304 Not Modified` 时，服务才返回 `unchanged=true` 并保留相同的 `SnapshotInputHash`。任何页面返回新内容、失去 robots 许可、缺少安全的最终 URL 或无法再验证，都会 fail-closed，要求重新抓取并创建新快照；不会把过期正文伪装为当前资料。

这一步只完成 parser 端的可验证前提。`core-api` 仍需在官网资料导入路径持久化该输入哈希，并把它纳入 source-import / generation 的幂等键，才能宣称端到端地避免重复模型调用。在接通前，调用方只能据 `unchanged` 决定是否继续，不能自行复用可能陈旧的候选稿。
