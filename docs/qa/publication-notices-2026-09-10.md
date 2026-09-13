# 随稿分发声明与内容权利核对

日期：2026-09-10。审阅来源为用户委托 AI（Codex），不是真人试学、出版社或法律认证。

## 实现与冻结对象

页面新增“随稿分发声明”，声明必须关联当前批准章节或冻结教学工件。尚未添加的输入会阻止冻结，失败重试保留已填写内容。服务器只接受声明字段，不接受客户端选择批准修订或设置权利状态。

`inkwords.publication-notice.v1` 保存标题、精确声明文字、HTTPS 署名链接、整理者、作品引用、文字哈希与稳定 ID。来源链接仅作署名，不触发抓取。冻结时校验所属作品并将声明纳入输入哈希；声明有数量和字节预算。含声明的 AST 使用 v3，原 v1/v2 空声明构建继续兼容，不回填旧稿。

导出服务从冻结 AST 派生 Markdown、DOCX、PDF 和 ZIP。声明文字作为字面文本呈现，防止其中的 Markdown/HTML 改变正文结构。ZIP 增加根目录 `publication-notices.txt`，相关代码目录增加 `THIRD-PARTY-NOTICES.txt`。附带文件不是已验证教学工件的修改，不改变原三份文件及其运行收据。

- 新 BookBuild：`0f150bbe-b80a-4586-8997-320df2c35ae0`。
- manifest：`sha256:aad600431e1f82768ddf3b9926e4b8eb912dc9d99bf33530c9c9b06b0df22235`。
- 批准正文 r13：`1ee15e29-1304-4978-9e6f-d41421798c8b`。
- 原构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4` 保留；正文、16 项引用、原三份代码逐字节未变。质量/验证快照除捕获时间外一致。
- 真实浏览器填写三份声明，冻结 POST 201；重复冻结返回同一 ID。四种导出均由页面链接实际下载。

## 来源与权利判断

1. 固定 Gin 提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 的 [MIT LICENSE](https://github.com/gin-gonic/gin/blob/73726dc606796a025971fe451f0aa6f1b9b847f6/LICENSE) 已按原文随稿及随相关代码分发。版权人为 Manuel Martínez-Almeida。实际许可文件 SHA256 为 `03458b6d5828e1be1127ca2adf122572eb574fc47b56190c3b38203b8b2a98d0`。
2. [Go 网站版权说明](https://go.dev/copyright) 区分正文 CC BY 4.0 与代码 BSD。已针对冻结安装资料写明 The Go Authors / Google、原作品链接、中文翻译/改写/教学补充以及 [CC BY 4.0 条款](https://creativecommons.org/licenses/by/4.0/legalcode.en)，并保留 [Go 代码许可](https://go.dev/LICENSE)。正文不能仅凭仓库 LICENSE 归为 BSD。该页面的行号装饰从许可正文中去除，原网页另存为证据。
3. Apple 的冻结片段支持打开终端和提示符说明。稿件没有复制 Apple 截图、图标或整段译文，但 [Apple 网站条款](https://www.apple.com/legal/internet-services/terms/site.html) 没有给出通用出版许可。其具体使用依据仍为待确认项，来源说明不消除该问题。
4. 16 项引用均从实际 PostgreSQL 读取对应片段，存储哈希、引用哈希与重新计算的文字哈希全部相等。8 项 Gin、6 项 Go、2 项 Apple；初次按语义引用 ID 查询元数据为空，改用实际 chunk ID 完成核对，不能把该初次空结果解释为来源缺失。
5. 原教学实现与固定 Gin 三个源文件做有限比较，最长相同连续非空行只有两行通用语句/括号。委托 AI 审读也确认其静态分段结构与 Gin 压缩前缀实现不同。此比较不是著作权归属或不侵权证明，不按行数推断法律门槛。

新构建重新登记六条权利记录：教学代码及四项精确字体文件为限定用途 ready，正文 pending 缩小为 Apple 操作说明的具体出版依据。字体记录依据新 PDF 报告中的同四文件哈希核对后登记，不自动继承旧构建审批。整书仍不得晋级出版候选。

浏览器已保存八阶段各一条 `delegated_ai` 审阅：结构、技术、自学性、一致性、文字、版式在当前单章范围通过 3/4；权利 needs_revision 0/4；试学 not_assessed 0/4。真人记录为 0。记录绑新构建与新 manifest，未自动复制旧结论；复核依据与限制见 `saved-reviews.json`。

## 实际验证

- 后端全套 `go test ./...`：57 个包通过。其后新增 URL/控制字符拒绝、严格请求体和声明分页修正的定向检查通过。
- 真实 PostgreSQL 测试：冻结、精确重试、修改声明创建新构建、旧 manifest 不变、其它构建作品引用被拒绝；测试实际 PASS，非 SKIP。生产接口负例另外验证客户端指定批准修订、混入其它章节引用均返回 400 `INVALID_STATE`。
- 前端 76 个测试文件、302 项测试通过；lint、TypeScript 与构建通过。构建保留既有大分块提示。
- core-api/frontend 部署 `publication-notices-v1-20260910`，export-service 部署 `publication-notices-v3-20260910`。全部健康，本机 Nginx 入口成功。顺序部署保留实际知识库挂载、现有评分配置和沙箱约束；无新依赖、数据库迁移或 Provider 调用。
- 真实 ZIP 为 21 文件，所有清单哈希匹配。声明精确文字存在于 Markdown、DOCX XML、根声明及相关代码声明；PDF 提取文字扣除页脚后完整匹配。
- 服务端 PDF 共 19 页。第 2–15 页正文区域像素与原已审 r13 一致；目录、声明及后移的来源注释另行查看。首次声明标题跨页已修正；长许可按段分页，完整文字不丢失。
- DOCX 校样最终 23 页。首次默认校样因 Fontconfig 漏载中文字体失败，系统通用回退又选择了不合适字体；两份失败结果保留。使用前次校样已有的 Noto/FreeMono 字体、独立 `docx-fonts.conf` 后正常；没有安装字体或修改系统配置。查看全页联系表，末三页另看大图；最终前 20 页像素与已查看的 v2 校样相同，许可全文在渲染结果中可读且提取完整。该环境只用于 DOCX 本地 QA，不替代服务端 PDF，也不证明所有客户端的字体/分页。

证据目录：`output/real-acceptance/2026-09-10/content-rights-review/`。包括 `source-bindings.json`、`frozen-source-chunks.json`、`code-comparison.json`、`notice-drafts.json`、`freeze-result.json`、`saved-rights.json`、`projection-facts.json`、`docx-final-facts.json`、`verify-final.py`、测试/部署日志和实际四种下载。

最终 `review-bundle.zip` 为 751,182 字节，SHA256 `9e94860d6b317236fd22a8f6bddce260b8954f1cbc1ebefb0abe961d923a2b8a`；包含六权利与八委托审阅，和 core 当前账本完全一致。`final-facts.json` 保存完整核对结果，`browser-final.png` 显示真实页面的新构建与三份声明入口。

## 兼容、恢复与剩余范围

无需迁移；旧构建不写入声明，原批准正文不改动。恢复旧二进制时只能继续消费旧 AST，含声明的 v3 不能交给旧 exporter；需要同时恢复支持 v3 的 core/export 版本，不能删除声明或改写冻结 manifest 来绕过版本要求。

当前仍是一个批准样章。Apple 使用依据、独立读者试学、真实个人六维学习与延迟保持、其它读者代表稿及原计划剩余流程继续存在。声明与本次委托 AI 审阅不代替这些证据。
