# DOCX 目录与页码由导出层生成

日期：2026-09-10。

实际 r11 校样缺少目录、页码，且书名与章节都使用 Heading 1。导出继续消费冻结 Canonical Book AST，不改母稿或 Markdown 投影；DOCX 的固定 Pandoc Lua filter 将投影首标题设为 Title，从其余一、二级标题生成可点击目录，再插入正文分页。书名不进入章节目录。

使用 Pandoc 已解析并去重的标题标识，避免另写中文 slug 或重复标题规则。目录是导出时生成的静态内部链接，打开即有内容；它不显示猜测的目标页码，也不依赖用户先更新一个空目录域。离线修改 DOCX 标题后需同步修改目录，或回到母稿重新导出。页脚采用 Word PAGE 域，各渲染器自行分页计算，实际校样逐页核对。

官方依据：Pandoc 的 [目录选项](https://pandoc.org/demo/example33/3.3-general-writer-options.html) 对 DOCX 生成目录指令而非预先排好的内容；[reference-doc](https://www.pandoc.org/demo/example33/3.4-options-affecting-specific-writers.html) 可提供样式和页眉页脚。原生目录探测的 OOXML 只有 TOC 指令，没有目录条目，因此本次选择即用的链接目录。

模板修正黑色无下划线的书名、TOC 层级缩进、PAGE 页脚、短脚注段落的同页保持及 9pt 脚注字号。现有中文/代码字体、页面尺寸和正文不变。`assets/publishing/refine_reference_layout.py` 可在已有本地模板上重放这些修改，无下载；修改前备份保存在本轮 QA 目录。

直接相邻的脚注标记之间增加上标逗号，避免连续编号看成一个数字；原来有空格分隔的标记保留。超过 30 行的代码块按已有空行优先切成最多 30 行的显示段，配合 SourceCode 同页保持减少孤立括号。空行归入后一段，避免 Pandoc 丢弃段尾换行；回归测试和真实 r11 导出均逐字重组比对（包括空行）。这是展示分段，不改变母稿、代码工件或执行清单。分段不是语法解析，极长行和其它语言仍需实际渲染检查。

运行版本记录增加 `docx_layout=inkwords.docx-layout.v2`，不改既有 BookBuild manifest。所有渲染变更通过实际 Pandoc 测试与实际网关导出验证；导航测试包含重复中文标题、三个不同锚点及目标存在性。无新依赖、数据库迁移或前端变化。

实际部署 `inkwords/export-service:book-layout-v5-20260910`；r11 DOCX 校样 18 页全部逐页查看，9 个目录目标有效，34 条脚注保留，7 个代码显示段重组后与修改前完全相同。见 `docs/qa/docx-book-layout-2026-09-10.md`。

本切片不解决浏览器 PDF 沙箱条件或完整出版排版。练习列表可正常跨页；Word 与其它渲染器可能重新分页，不能以本次 LibreOffice 校样替代服务端 PDF 或所有阅读器验收。
