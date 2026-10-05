# r13 出版权利范围审阅

日期：2026-09-10。来源：`delegated_ai`。结论：需要修改；没有授予第三方许可或确认出版就绪。

## 固定对象与分发事实

构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4`，manifest
`sha256:a744d724013b2ed9cf9bc7cd2d461fa7b9b03cca01258ac32fe7f047ae798e76`。
已直接读取现有审校 ZIP 和主服务 editorial：当前 2 个必需对象为 r13 正文
`1ee15e29-1304-4978-9e6f-d41421798c8b` 与教学代码工件
`e769de34-dfbc-5edd-b040-b7517c310d31`，权利项为 0。
包中有母稿、Markdown/DOCX/PDF、三个 Go 工件文件、验证/质量/审阅/预检/哈希清单，共 17 文件。
没有独立图片/截图文件或打包的字体文件。PDF 实际存在字体资源，不因 ZIP 无 font 目录而免除来源核对。

冻结的 16 个引用是 8 个 Gin 代码定位、6 个 Go 安装说明定位和 2 个 Apple Terminal 说明定位。
实际 manifest 中共有五个 snapshot ID；这些是引用次数和快照数，不是 16 个独立许可。
原文与教学代码不可因本次审阅直接认定用户独占所有权；需区分自写解释、生成/编辑记录与第三方引用。

## 许可核查事实与边界

- 实际 HTTP 200 取得固定 Gin SHA `73726dc606796a025971fe451f0aa6f1b9b847f6` 的 LICENSE，
  1091 字节、SHA-256 `03458b6d5828e1be1127ca2adf122572eb574fc47b56190c3b38203b8b2a98d0`。
  其 MIT 条款允许多种复用方式，并要求相关副本或实质部分保留版权及许可声明。
  当前分发稿含源码定位但没有完整许可声明；引用/改写的实际范围仍须核对，不能仅按 repo 当前分支许可放行。
  [固定版本许可](https://raw.githubusercontent.com/gin-gonic/gin/73726dc606796a025971fe451f0aa6f1b9b847f6/LICENSE)。
- Go website 仓库当前 LICENSE 明确列出源/二进制再分发条件及非背书限制；这不自动证明冻结网页
  中所有材料、商标和其它素材适用同一许可，仍需对应页及具体引用范围。
  [Go website LICENSE](https://github.com/golang/website/blob/master/LICENSE)。
- Apple 当前站点条款对内容复制、翻译、出版和商业分发有限制。本次未取得针对两段 Terminal
  材料的独立出版许可，也未作法律例外认定；应核对实际文字是否仅为独立操作描述/来源链接，或另需许可。
  [Apple 站点条款](https://www.apple.com/legal/internet-services/terms/site.html)。
- Noto 官方文档说明其字体使用 OFL；OFL FAQ 区分印刷使用、嵌入和独立分发，允许文档嵌入且
  不因此改变正文许可。须先证明实际字体身份，不能反推任意 PDF 都已满足条件。
  [Noto 字体说明](https://notofonts.github.io/noto-docs/website/use/)、[OFL FAQ](https://openfontlicense.org/ofl-faq/)。

以上是当前公开来源与本次取回字节的审閱记录，不是对历史授权或最终出版合法性的证明。
网页缓存无法读取固定 Gin LICENSE 后，直接请求官方 raw 地址成功，保留状态/hash。

## 字体检查

使用 PyMuPDF 检查原 PDF，得到 **213 个无字体名称的 Type3 字体对象**，原文件未改。
现有 CSS 指定 Noto Sans CJK SC 等正文候选，代码候选含 Noto Sans Mono/DejaVu Sans Mono/FreeMono。
当前容器 `fc-list` 可见 Noto CJK、Open Sans、FreeFont；安装清单和 CSS 不能证明历史 PDF 中
每个 Type3 字形来自哪个文件。当前未取得与导出 hash 绑定的实际字体使用记录，因此不登记字体 ready。
本次仅查元数据，不重复声称完成全页版面校对；原版面审阅保持其原范围。

## 发现的补证流程缺口

`textbook_rights_items` 唯一主体只允许一次不可变记录。现有仓库测试及代码确认不同内容的同主体
提交被拒绝；构建按同内容 input_hash 幂等返回原 ID。因而“先填 pending，再补证”缺少合法路径。
目前不向主构建写入不可修订的推测性 ready，也暂不创建会固定阻断的 pending 条目。
实际发现先追加到 delegated_ai 权利审阅；补证实现按
`docs/decisions/publication-rights-amendments.md` 继续。

可登记内容草案：正文记录生成/编辑来源及 Gin/Go/Apple 引用边界；代码记录每文件原创/改写范围和
相应许可/署名；字体记录实际渲染文件 hash、版本、许可及嵌入/独立分发方式。每项依据齐备后才评 ready。
读者试学与其它门禁不受本次审阅替代。

## 证据

`output/real-acceptance/2026-09-10/publication-rights-audit/` 中的 frozen-manifest、source-inventory、
pdf-fonts、license-fetch 与固定许可原文可重核；本地提取的 book.md 仅是审阅副本，未回写母稿。
现有 rights 不可变/幂等与工作区测试按真实临时 PostgreSQL 重跑；结果见 rights-current-contract.log。
没有 Provider 调用、代码执行或依赖安装，也没有发送许可申请或其它对外消息。

## 委托复核与原审校包核验

实际浏览器保存 rights v2：`4c3f5968-f924-4177-9e77-2f1c33d50650`，HTTP 201，
`delegated_ai / needs_revision / 0`。此前 8 条审阅完全保留。下载
`Gin-r13-权利范围复核审校包.zip`，603655 字节，SHA-256
`04ead9d8ddc4bba7666a5a0f1b4c105d79d22d3a090aa352b6e53227ee2bf435`。
17 文件及 manifest 哈希全部通过；冻结 manifest、Canonical Book AST、Markdown 与 Go 文件逐字节
相同。16 页 PDF 文本和页面内容流相同；DOCX 仅 `docProps/core.xml` 元数据不同。
包内 9 条委托记录、零真人记录、零权利项，preflight false。见 `final-facts.json`。

## 追加补证实现与验证

新增 `inkwords.rights-amendment.v1` 与 `inkwords.rights-ledger.v1`，core schema migration 36。
原 RightsItem 不改；补证绑定原件、构建 manifest、明确前序及版本，保存完整有效快照、实际审阅来源、
理由和证据引用。服务端锁定构建后执行前序 CAS，精确重试幂等；缺前序、分叉、跨工作区、跨对象、
改换 work_type、改写已存在 ID、在出版候选追加都拒绝。晋级后的精确重试只是读回原记录。

core 预检和 exporter 都使用共享解析器，按连续前序链选有效版本，不按时间猜测。ZIP 的
`rights.json` 为有效项；新增 `rights-ledger.json` 同时包含原件、全部补证和版本化选择规则。
补证晚于权利审阅时，旧审阅不能继续放行，需要重新进行权利审阅。现有委托审阅支持追加复审；
旧真人审阅仍是不可变单条记录，尚无独立真人复审版本入口，不能把 AI 复审冒充真人记录。

界面展示有效状态、原件和历史；已登记 pending 对象不再建议重复登记或无效地重新冻结。
补证表单固定读到的前序版本，失败保留输入和重试 ID；不会自动更新前序覆盖他人的后续证据。

实际临时 PostgreSQL 验证持久化、幂等/过期父版本、原件与构建不变、服务对象重建恢复及晋级锁定。
目标 SQL `WHERE build_id = ? ORDER BY base_item_id, revision` 的 EXPLAIN 使用复合唯一索引
Bitmap Index Scan；测试关闭 seqscan 仅用于确认索引可用，不声称小数据量默认规划器一定选它。
每次追加写 PK、构建/原件/版本唯一索引和前序唯一索引；无额外 current 指针或重叠 build 索引。
数据库已有补证时 Down 明确拒绝删除证据；回退应用时保留迁移和证据，旧应用不能继续进行出版晋级。

全量后端 `go test ./...`、架构、全量前端 301 项、lint/build 通过。
新增真实 exporter 数据库读回与 ZIP 合同测试通过共享解析器对照；导出构建测试首次缺少 fixture
input_hash，补齐后通过；早期 manifest 原始字节断言改为 JSONB 语义与 manifest hash 双重检查。
这两次失败保留原日志，没有修改生产证据来通过测试。

## 本机部署和真实验收结果

core/export 镜像 `rights-amendments-v1-20260910`，前端最终为 `rights-amendments-v2-20260910`。
顺序更新三个服务，core schema 36；保留真实 vault、review 模型和 profile、其它服务与沙箱配置。
最初前端 Docker 上下文忽略 dist，构建失败后使用单独的已生成 dist 上下文成功，未改项目 ignore。
镜像/health/网关资源逐字节校验见 `runtime-final.json`。Chromium 8 条与 bundle budget 通过。

真实浏览器先保存正文原件 `e8cdbeb5-6307-4bf4-80be-b5fb4f9bdb6d`、代码原件
`378b6663-96be-458b-b21f-40e6ab34e826`，两项 pending。
连续填写时 POST 先返回、工作台 GET 后返回导致第二项作品引用被旧表单 reset 清空；第二次当时
未通过表单校验、未写入。核对数据库只有第一条后补齐引用并提交，最终仅两条原件，无重复。
前端 v2 在保存期间禁用整个登记 fieldset；新增断言通过，避免在 reset 完成前填写下一项丢输入。

代码追加补证 `91e52605-fa3b-4bd0-830a-79613947b453` v1，实际浏览器 HTTP 201，
`delegated_ai`，具体固定 Gin MIT 原文哈希、证据引用和适用范围限制保存在独立记录中；仍 pending。
使用同一请求真实重试返回 201 和相同原记录；随机新请求 ID 带过期前序返回 409，历史仍只有一条。
原件、冻结 build、此前九条审阅完全不变。随后真实保存 rights v3
`0833e080-ef71-497f-98a5-271405df176a`：修复了补证流程，引用/许可/署名和字体仍需要修改。
最终共十条委托审阅，零真人审阅；刷新后两原件、一补证恢复可见，已查看浏览器截图。

真实新导出 `Gin-r13-权利补证与历史审校包.zip`：608178 字节，SHA-256
`00072d0be9144eb7f4fdcea573ff417816b4d2418e6aa6cde45249a9225f14bb`。
新增 `rights-ledger.json` 后为 18 文件，全部条目/manifest 哈希通过，权利原件、补证、有效版本与
core GET 完全相同。预检权利阻断相同；export 另保留既有 `explicit-publication-candidate` 状态检查，
因此完整 blocker 数组不声称逐字相同。两边 passed=false。旧冻结 manifest/AST/Markdown/代码逐字节
相同；PDF 16 页文本与内容流相同；DOCX 仅 core 元数据变化。见 `amendment-export-facts.json`。

当前仍需处理字体与输出哈希绑定、实际引用改写范围/分发声明、真人复审入口及原计划其它验收。
本切片没有 Provider 调用，没有写入个人学习成绩或伪造读者试学。
