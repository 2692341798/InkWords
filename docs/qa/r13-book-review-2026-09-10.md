# r13 单章构建审阅与实际导出验收

日期：2026-09-10。来源：用户委托 AI（Codex）。本记录不代表独立冷读者、真人同行审校、学习掌握或出版社认证。

## 对象与实际结果

本次继续现有开发计划，审阅当前冻结 BookBuild `70aadfa4-32d1-4c87-b235-e76ca5678197`，manifest 为 `sha256:be7d2e46427bef674ead12e42a8091d3c3723d0ee73811988f32061a6a10fa3f`。构建只含一个 foundation 章节，批准母稿 r13 `1ee15e29-1304-4978-9e6f-d41421798c8b`；正文 SHA256 `d8c57b17ba1242d4b0f370225a40b273f8df88af60a89a9dc616ee6ce816804d`，16 项来源。

真实网关导出审校包成功，取出同源 PDF、DOCX、Markdown 和 Canonical Book AST。没有修改母稿、代码、蓝图、旧 r11 审阅或运行器配置。本切片新增 Provider 调用 0，没有执行教学代码或写入学习者作答。

证据目录：`output/real-acceptance/2026-09-10/r13-book-review/`。

| 文件 | 大小 | SHA256 |
| --- | --- | --- |
| Gin-r13-分步自学校样.pdf | 1,118,042 字节 | `280f8e189364938dbdb7312d1f5edbed7716bd2a2a634e74169fab87d1604a2a` |
| Gin-r13-分步自学校样.docx | 42,146 字节 | `086cf3c512405294eaafcc89cbc303d8ff3a1461875a7a18e0443c995b1e6ec5` |
| Gin-r13-分步自学校样.md | 32,819 字节 | `93914cbc1002db5f7cc222372a5657e014f50b0087a72cf934ddae5f66b4188b` |
| Gin-r13-审校记录包.zip | 593,794 字节 | `e05335323e9af2b7e408a435d84fdbda512ae0d60dd212f331210494e64cb94d` |

## 版面和一致性检查

服务 PDF 的 16 页全部逐页查看，中文、两张新表格、长代码、来源注释和页脚未见缺字、裁切或重叠。9 个目录链接在实际 PDF 中均有有效目标；每页页码与总页数正确。

DOCX 经技能包的 `render_docx.py` 和 bundled LibreOffice 渲染。初次漏加载已有中文字体配置，得到缺字的 13 页，保留为失败 QA 中间件，不能当作有效校样。以 `FONTCONFIG_FILE=output/real-acceptance/2026-09-09/delegated-review-v1/exports/fonts.conf` 加载已有 Noto CJK/FreeMono 后，同一份 DOCX 正常渲染为 21 页；本地文档字节未变，没有安装字体、修改系统配置或改写导出内容。`docx-pages-cjk/page-1.png` 至 `page-21.png` 全部逐页查看。

DOCX 的 9 个内部目录目标存在，34 个脚注引用均有对应正文，渲染脚注顺序正确；OOXML 的内部脚注 ID 不要求连续。两张表的列对应和文本换行正常。7 个代码显示段重组后，在 Pandoc 将制表符展开为 4 列空格的规则下，与原两段 Go 代码完全一致；不能把它说成原始字节完全不变。原始代码文件与冻结正文代码逐项一致，Markdown 与先前 r13 投影逐字相同。

Word 长脚注占较大页幅，部分练习列表正常跨页，代码引言与代码可能隔页；作为可编辑单章校样接受 3/4，仍有精修空间。不同 Word/LibreOffice 版本可能重新分页，本次只验证上述实际渲染环境。`verify_exports.py`、`render-facts.json` 和渲染日志保存可复现机器检查与视觉检查范围。

## 实际审阅记录

按当前 r13 逐段审查正文、代码、题目、答案与投影，没有继承旧 r11 记录。真实浏览器逐项填写委托审阅表单，8 次 POST 均返回 201；随后 API 和页面读回核对成功。所有记录均为 `reviewer_kind=delegated_ai`、v1。

| 阶段 | 结论 | 记录 ID |
| --- | --- | --- |
| 发展性 | 单章范围通过，3/4 | `c4d9d9ac-7923-4929-8dab-d09530f6d698` |
| 技术 | 已检查实现与解释通过，3/4 | `91f086f8-14ee-42b8-9e6a-5b8e39659e6f` |
| 自学性 | 教学设计审查通过，3/4 | `1970faf0-dd94-4d30-8468-27d83c41d089` |
| 一致性 | 当前单章及投影通过，3/4 | `ca98f102-52df-4c3d-97b1-54a3b35f281d` |
| 文字 | 单章复审通过，3/4 | `119fc9bf-4c11-425d-922d-6c8e38c70e84` |
| 版式 | 本次 PDF/DOCX 校样通过，3/4 | `a534c4e6-ab30-4398-882a-529d3d4d13eb` |
| 权利 | 需要修改，0/4 | `50c6b9b1-8c60-4fb1-8319-1a9087d58d12` |
| 读者试学 | 尚未评估，不评分 | `235f9af1-54d7-424f-ba0a-bdbed2743fc8` |

技术复核只读检查现存新工件 `e769de34-dfbc-5edd-b040-b7517c310d31` 的 verified 收据；对应任务 `34037fcf-690f-484a-a53a-f4edcd241f29`，Go 1.26.8/Bubblewrap，绑定 r12 与当前同代码树。它不证明全新 Mac 安装、主函数 go run 输出、HTTP 服务、性能或学习者表现。

两项必需权利主体（r13 prose、教学 code）均无记录；公开官方资料不等于已获得复用许可。没有独立冷读者试学、真实学习者六维表现或至少 24 小时延迟保持证据。委托 AI 内容审阅不能填造这些事实。真人审阅 0、权利记录 0，preflight 仍 false、构建仍 ready_for_review；没有执行出版晋级。

`browser-reviews-1.log`、`browser-reviews-2.log`、`editorial-after.json` 和 `browser-readback.log` 保存操作与回读证据；页面截图 `output/playwright/r13-book-reviews.png` 已查看。

## 审校包回读和后续开发缺口

记录后再次真实 GET 导出 `Gin-r13-审校记录包.zip`：17 个文件 ZIP CRC、全部内容哈希及 `manifest.sha256` 校验通过。包内 8 条委托记录 ID 与 API 一致，`publication_candidate=false`，保留权利和试学阻断。两次导出之间冻结 AST、Markdown、构建 manifest 完全相同；PDF 逐页文本和内容流相同，DOCX 仅 `docProps/core.xml` 时间元数据变化。详见 `final-bundle-facts.json`。

新确认的工程缺口：`verification-summary.json` 目前固定返回 unavailable，说明冻结构建未保存整书级运行验证快照。工作台的实际代码收据不能凭外部文字引用冒充离线包内快照。后续应实现版本化、绑定工件/清单/输入哈希的冻结验证摘要，并处理过期、缺失和旧构建兼容；不得回填改变当前不可变构建。

完整计划继续执行，仍包括权利清单、独立试学与学习闭环、其它 audience、课程浏览器媒体验证、验证取消/重执行生命周期、历史工件权限、解析消费者恢复和 C++ 回归。此次没有产品代码变化，不重复运行无关全量测试；实际 API、浏览器、导出、渲染和一致性检查均完成，最后 `git diff --check` 通过。
