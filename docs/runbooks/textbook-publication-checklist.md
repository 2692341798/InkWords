# 教材出版前清单

> 适用范围：InkWords 生成或整理的技术教材。此清单不是出版社要求的替代品；ISBN、CIP、版式模板、合同和交付格式必须以目标出版社的最新书面要求为准。

## 先分清两种导出

- **个人学习包**：当前系统支持。它包含已批准的单章 Markdown、代码工件、运行验证摘要、已登记截图、资产元数据以及 `manifest.sha256`。它可用于自学、录制视频和人工审稿，但不代表可以出版。
- **冻结构建（待审）**：系统可以冻结当前项目的 BookContract、StyleSheet 与已批准章节集合，并生成不可变 manifest；初始状态只能是 `ready_for_review`，不是出版候选。
- **冻结整书投影与审校包**：可从同一冻结 Build 下载 Markdown、DOCX、PDF，或下载包含三种投影、内容寻址的代码/已登记图片、权利记录、人工审校记录与 manifest 的 ZIP 审校包。缺失记录会继续作为阻断项；审校包本身不是出版批准。
- **出版候选**：必须由人显式执行晋级，并且当前冻结构建的自动检查、权利清单及八类人工审校记录全部通过。它只是 InkWords 内部预检状态，不代表出版社、ISBN 或 CIP 批准。

## 自动检查（系统必须保留证据）

- [ ] 只导出了指定章节的已批准修订；章节、修订、合同、样式和蓝图 ID 都出现在 `manifest.json`。
- [ ] `manifest.sha256` 能校验 `manifest.json`，且 ZIP 中的代码与截图能够按其 SHA-256 指纹读取。
- [ ] 每个需要运行证明的代码工件都有同一工件哈希、同一冻结 BookContract/StyleSheet、匹配工具链与 runner 输入且未过期的 `verified` 运行证据；只把工件状态列改成 `verified` 不能通过预检。
- [ ] 每个图片资产有替代文本、来源、生成方式和明确权利状态；权利不是 `ready` 则作为出版阻断项。
- [ ] 验证摘要不会把未验证、过期或被阻断的结果描述成“已运行”。

## 人工审校（出版候选的硬门槛）

- [ ] 发展性编辑：目标读者、学习目标、章节职责、范围和先修关系。
- [ ] 技术审校：事实、版本边界、代码、输出、故障路径和引用。
- [ ] 自学性审校：隐藏前提、术语负担、由浅入深的例子、练习梯度和失败恢复路径。
- [ ] 全书一致性审校：术语、跨章引用、项目状态、简化模型、代码版本与前后结论没有冲突。
- [ ] 文字编辑：术语一致、中文标点、单位、代码字体、图表题注与参考文献。
- [ ] 版式校对：目录、分页、孤行、代码换行、图像分辨率、替代文本、页眉页脚。
- [ ] 权利复核：代码许可证、图片和截图权利、引用、商标、个人信息、第三方素材。
- [ ] 读者试学：目标读者在没有作者口头补充的情况下完成代表章节；将卡点落实为修订记录。

## 整书构建前的冻结记录

- [ ] 在教材工作台点击“冻结待审构建”，或调用 `POST /api/v1/textbook-projects/:projectID/book-builds`；空请求/`{}` 保持兼容，也可只传 `publication_notices` 数组。服务端只读取当前已批准状态，不接受客户端指定章节修订或出版状态。
- [ ] 需要随稿提供版权/许可时，先在“随稿分发声明”填写并添加，或为每项提供 `title`、精确 `text`、HTTPS `source_url`、`prepared_by`、`subject_refs`。引用只允许本构建的 `chapter-revision:<UUID>` / `code-artifact:<UUID>`。最多 16 项、单项文字 24,000 字节、总文字 128,000 字节；来源链接只作署名。声明随输入哈希冻结（AST v3），精确重试复用，文字变化创建新构建；不得据此自动判权利 ready。
- [ ] 在 Markdown/DOCX/PDF 的“来源与分发声明”、ZIP 根 `publication-notices.txt` 及关联代码目录 `THIRD-PARTY-NOTICES.txt` 核对全文。代码随附许可文件属于导出投影，不改变原教学工件字节或验证清单；需要重新审阅新构建的权利和版面。
- [ ] 返回初始状态必须为 `ready_for_review`，并检查 `manifest_hash`、批准修订数、代码工件哈希和资产哈希。
- [ ] 对未变化的批准输入重复执行冻结时，必须返回同一 Build ID、`input_hash` 和 `manifest_hash`；`built_at` 只记录首次构建时间，不能导致重复待审构建。
- [ ] 在出版编辑工作台按 `required_rights_subjects` 逐项登记权利记录；章节批准修订、冻结代码工件及图片/截图资产都必须有作品类型一致且状态为 `ready` 的精确 `subject_ref`。记录与 Build 绑定且不可改写，笼统的“全书权利已确认”不能覆盖未登记对象。
- [ ] 真人审校必须明确记录结论和评分；不能把“已填写说明”当作通过。页面“追加真人审校记录”保留全部阶段与旧记录，复审只追加新版本。旧版说明保留但显示“未记录结论 / 未评分”；用户委托 AI 使用独立入口，不冒充真人。
- [ ] `POST /api/v1/textbook-projects/book-builds/:buildID/reviews` 使用 v2 写入字段：`id`（稳定 UUID）、`manifest_hash`、`expected_revision`（该阶段当前最大版本，无记录为 0，旧版说明为 1）、`stage`、`reviewer`、`verdict`、`score`、`scope`、`notes`、`evidence_refs`、`hard_failures`。服务端固定真人来源、分配记录时间；禁止客户端指定 actor、时间或 Build ID。旧 notes-only 写法返回 400。
- [ ] `pass` 需整数 3–4 分、至少一项证据、无阻断；`needs_revision` 需明确阻断发现；`not_assessed` 使用 0 占位但显示未评分。范围为 8–1000 字、说明为 8–4000 字，证据/阻断各最多 32 项且每项最多 500 字。记录使用 `inkwords.human-publication-review.v2`，绑定冻结 manifest。
- [ ] 精确重试保持同一 ID、原版本及内容，返回原记录；改变相同 ID 的内容、旧前序或不匹配 manifest 返回 409。刷新历史不会自动把未确认提交改绑到新版本；冲突后应核对新证据再修改草稿。已晋级构建只可精确重试，不可追加。权利补证后需追加当前清单的复审，历史失败不能由另一个来源的通过记录抹除。
- [ ] 确认预检无阻断后，由人点击“显式标记为出版候选”，或调用 `POST /api/v1/textbook-projects/book-builds/:buildID/publication-candidate`。服务端不接受客户端自选状态；缺少任一证据时请求失败并保持 `ready_for_review`。
- [ ] 记录项目 ID、全部已批准修订、BookContract、StyleSheet、Blueprint、资产哈希、代码工件哈希、工具版本和构建时间。
- [ ] 从同一 Canonical Book AST 生成 Markdown、DOCX、PDF 和 ZIP；可调用 `GET /api/v1/textbook-projects/book-builds/:buildID/export/{markdown|docx|pdf|review-bundle}`，后续格式不得重新查询当前章节或拼接旧博客 PDF。
- [ ] 新冻结输入 `inkwords.book-build-input.v4` 保存各批准修订的 `video_runbook` 与 `video_runbook_hash`，工具版本声明 `video_runbooks=inkwords.book-video-runbooks.v1`。检查 ZIP 的 `projections/video-runbooks.{json,md}` 与章节/修订/正文哈希一致并出现在文件清单；教案不拼入正文。无教案的新构建明确标识缺失，历史构建不从当前章节补取；导出不能将 manual_capture_pending/unverified 提升为已采集或已验证。
- [ ] DOCX 默认使用 export-service 镜像内已审校的 `assets/publishing/inkwords-reference.docx`，并由 `TEXTBOOK_PANDOC_BIN=/usr/bin/pandoc` 渲染；如覆盖参考文档，必须使用只读挂载并重新完成 DOCX 逐页检查。缺少其中任一配置时接口返回 `503 TEXTBOOK_BOOK_DOCX_UNAVAILABLE`。

  已核验的本地开发证据（2026-09-03）：Pandoc `3.8` 能使用该参考文档生成可打开的 OOXML DOCX；其 `--version` 输出声明为 free software，参考文档的 `word/styles.xml` 指定 `Noto Sans CJK SC` 与 `Noto Sans Mono CJK SC`。这只证明本机生成链路和样式声明；目标部署仍须记录字体替代和逐页校对结果。
- [ ] PDF 只在 `TEXTBOOK_CHROMIUM_BIN` 指向保留 Chromium 沙箱的可执行文件时可用。2026-09-04，macOS 上的 Playwright Chromium headless shell 已用同一 Canonical AST 生成真实 PDF；当前 Docker Desktop 中 export-service 仍因 Chromium 无法创建 sandbox namespace 保持未配置并返回 `503 TEXTBOOK_BOOK_PDF_UNAVAILABLE`。不得为了容器启动添加 `--no-sandbox`。
- [ ] 在目标运行环境对 DOCX 执行 Word/LibreOffice 逐页视觉检查，并对 PDF 留下页数、字体替换、代码折行、图片和目录问题的记录；本机真实 PDF smoke 只证明同源渲染链路，不能替代该项视觉验收。
- [ ] DOCX 客户端需具备模板声明的 Noto Sans CJK SC、Noto Sans Mono CJK SC 与 FreeMono；检查主题字体属性、缺失字体警告及短表格行跨页情况。安装用户字体须先取得具体授权。ZIP 的 `renderer_tool_versions.docx.docx_reference_sha256` 必须对应本次实际模板，模板更换后重新校样；2026-09-13 WPS 与 23 页独立渲染证据见 `docs/qa/docx-wps-fonts-2026-09-13.md`。
- [ ] 目标出版社要求的 ISBN/CIP、封面、合同和交付字段以“待确认”列出，绝不由系统伪造。

## 发布判定

只有自动检查全部通过、人工审校全部有明确记录、整书冻结完成并且版式校对完成后，才可将构建标为“出版候选”。任一项缺失时，保持“个人学习包”或“被阻断”状态，并说明缺失原因。
