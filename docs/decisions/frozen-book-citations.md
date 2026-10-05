# 冻结整书引用与 Pandoc 样式（2026-09-10）

真实 r9 DOCX 校样暴露两处问题：正文仍显示内部 `[evidence:alias]`，参考模板缺少 Pandoc 的 Table、Compact 等样式，LibreOffice 中表格变成纵向文字并挤压前文。

## 数据与输出

- Canonical Book AST v2 为每章冻结引用别名、EvidenceRef 和 SourceSnapshot。CreateBookBuild 在项目事务中，通过批准修订的 `document_json.evidence_aliases` 解析引用，仅查询当前项目的不可变来源；冻结内容参与 input_hash 和 manifest_hash。
- Markdown、DOCX、HTML/PDF 共用章节脚注投影，固定 GitHub SHA、文件、符号及行号。代码块、行内代码、转义标记不改写；批准 Markdown 和修订内容哈希不变。导出服务不查实时来源表。
- v1 无引用构建继续可读；缺少冻结引用的旧构建导出明确失败，必须新建构建。不得补写旧 manifest。
- 引用只要求实际引用的来源合法，不要求每章脚注包含主资料；生成任务的主资料约束保持不变。
- 来源元数据进行 Markdown 转义。只有 HTTPS 来源可成为链接；本地或无安全链接的来源以快照 ID/哈希呈现。

## 模板与恢复

`complete_reference_styles.py INPUT OUTPUT` 从本机 Pandoc 默认参考文档补入缺失样式，保留原有中文正文、标题等样式和其它 ZIP 部件。新增的代码样式使用运行镜像已有的 FreeMono 9pt，并允许长行换行，避免缺少 Consolas 时退为比例字体。使用受控 Python + lxml，不联网。此次原模板已留在本轮验收产物 `exports/reference-before-styles.docx`。

没有数据库迁移或新索引。引用查询沿用 source_chunks 主键和项目归属连接；真实 PostgreSQL 测试记录 EXPLAIN。回退服务镜像可恢复旧行为，但旧服务不应渲染 v2；保留冻结构建并使用兼容 v2 的导出服务。PDF 仍要求 Chromium 默认沙箱可用，本改动没有放宽隔离。
