# r15 整书教案冻结与四格式导出验收

日期：2026-09-13。执行者：Codex（delegated_ai）。依据用户继续开发计划、更新知识库和直接操作电脑的授权；本报告不作为真人试学、出版社审核或权利许可结论。

## 修复的缺口

批准 r15 的视频教案已与实际教学源码同源，但原 BookBuild 只冻结正文等数据，未保存 document.video_runbook；离开当前工作台后，审校包无法重现该教案。

现在 core-api 在原冻结事务中读取批准修订的教案，校验并保存教案值、SHA-256、章节/修订/正文身份。输入格式升级为 `inkwords.book-build-input.v4`，工具版本声明 `video_runbooks=inkwords.book-video-runbooks.v1`。只有教案改变也会产生新的冻结输入，原构建保持不变；没有新增数据库表、迁移或索引。

export-service 只读取冻结清单，审校 ZIP 新增 `projections/video-runbooks.json` 和 `projections/video-runbooks.md`，纳入原有文件哈希清单。正文仍是唯一教材内容源，教案作为随附投影，不拼入正文。旧构建没有冻结教案时明确返回不可用说明，禁止从当前章节补取。未声明版本、错配身份、重复章节/修订或哈希不符均拒绝导出。

## 实际冻结与导出

- 新构建：`66e5f4a4-00ac-449e-82c1-1647d8dfd248`，`ready_for_review`，Canonical Book AST v3。
- 批准修订：r15 `d35e4382-0253-4c6e-be05-0bb13b0b10e7`。
- 教案哈希：`sha256:e16239faf3059ede49dc4bc56ad83ba4a666643e248ec9dda95d474ac656f5a1`。
- 旧 r13 构建 `0f150bbe-b80a-4586-8997-320df2c35ae0` 保持冻结。沿用来源声明的原文与出处前，比较新旧正文和教学工件字节；只将声明绑定改到相同内容的新修订/工件，没有继承权利通过状态。
- 四个正式导出接口均 HTTP 200；文件保存于 `output/pdf/Gin-r15-66e5f4a4/`。
- ZIP 共 23 个条目，21 个内容文件的 SHA-256 全部匹配清单；其中 book.md 与独立 Markdown 字节相同。
- 随附教案绑定 r15，包含实际 `TestMethodSpecific` 第 49 行定位；仍为 unverified、manual_capture_pending。导出不将代码运行通过推断为媒体已采集。

| 文件 | 字节数 | SHA-256 |
| --- | ---: | --- |
| book.md | 38761 | `8b666a0a61898ffbc52c7bcdad2a9cb1ecdb66594ae2aa249e5548608ac0426b` |
| book.docx | 45740 | `ce94e793ae10dccfd059d7ddddfc53343a4f6c1c7d9df8001f684c5792891ea9` |
| book.pdf | 1366562 | `120abbeb78c87490b480e1e5551a550be61ea25932c5d95ba0eb5b65952e2f04` |
| review-bundle.zip | 747623 | `9d6605f6cd73d8cbc6eb2fc73e2bbc2c2b4a3a17ea8db7bb652846f61bd4d303` |

## 测试与真实页面

真实 PostgreSQL 冻结回归在旧实现先失败（缺少教案快照），修复后通过。新增合同及导出测试覆盖完整教案、显式缺失、历史构建、篡改哈希、版本声明、重复与跨章绑定、教案变化重新冻结、旧清单不变。共享合同、core-api、export-service 和架构包通过，随后 `GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...` 后端全量通过。

保留各容器原 Compose 文件链和环境配置，顺序部署 `inkwords/core-api:book-runbooks-v1-20260913`、`inkwords/export-service:book-runbooks-v1-20260913`，两者健康。未安装依赖、调用模型、削弱沙箱或提交 Git。

真实浏览器刷新后，工作台四个导出链接都绑定新构建。再次点击“冻结待审构建”仍返回 `66e5f4a4-00ac-449e-82c1-1647d8dfd248`，没有重复构建。

PDF 为 19 页：已渲染全部页面并检查整册联系表，另放大核对第 5、7、9、18、19 页的表格、代码、声明和引用，未观察到明显文字截断、重叠或缺字。保留页间代码块和声明断点；第 15、17 页留白较多，后续出版校样仍需逐页精校。这是 AI 初步版面检查，不宣称真人版式审校通过。DOCX 结构含 240 段、2 表；本轮未做 Word/LibreOffice 逐页视觉验收。

## 剩余验收

新构建仍非出版候选，导出时有 13 项预检阻断。新构建的权利补证、八阶段审校、真实冷读者和个人六维学习证据不能从旧构建或 AI 执行结果自动继承。当前静态素材仍待持久登记；只有 concept 代表章批准，hands_on 样章尚缺，批量生成门禁继续生效。

原计划三项综合要求保持未勾选。证据目录：`output/real-acceptance/2026-09-13/book-r15/`，含先失败/后通过日志、全量回归、冻结响应、下载哈希与 19 页渲染图。
