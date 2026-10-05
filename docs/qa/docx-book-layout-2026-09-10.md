# r11 DOCX 目录、页码与代码分页验收

日期：2026-09-10。审阅来源：用户委托 AI（Codex）；不是独立冷读者或出版社审校。

## 冻结对象与实际输出

- BookBuild：`8aa50b6b-45b3-477d-aece-4bbe1b164089`。
- manifest：`sha256:4164d754fc06350bf2a4bbb08e5d7b9281e8d7ba55a41d13af7075b9869e960b`。
- 母稿 r11：`5775cf06-e1f7-49e2-b6e4-744d2f23b19d`；本切片未修改母稿、来源、蓝图或运行证据。
- 最终实际网关 DOCX：`output/real-acceptance/2026-09-10/book-layout/Gin-r11-目录页码校样-v5.docx`，39,429 字节。
- SHA256：`3ed2bfa477d7fda8cc0838bd54fc635f614abdad92a526bd830fc2d7720dcb52`。
- 已部署镜像：`inkwords/export-service:book-layout-v5-20260910`；布局合同 `inkwords.docx-layout.v2`。

## 观察与验证

最终 DOCX 经本地 LibreOffice 渲染为 18 页，`final-v5-render/page-1.png` 至 `page-18.png` 全部逐页查看：书名与章节层级可区分，目录可读，页码 1–18 连续，正文、表格、代码和脚注未见裁切、重叠或缺字。第 8–10 页代码按原有空行分段，原来单独跨页的闭括号问题已消除；部分练习列表正常跨页。最后一页的留白未误判为内容缺失。

OOXML 与文本核对：9 个目录内部链接全部有对应书签；34 条脚注保留；原来的 2 个长代码段分为 7 个显示段，最长 28 行，重组后与修改前代码逐字一致（包括空行）。PAGE 域实际存在，校样每页页码由文本提取再次核对。机器事实见同目录 `final-v5-facts.json`。

首次分段实现被逐字断言发现丢失段尾空行，已修复后才部署 v5。导航回归覆盖中文重复标题与目标存在性；代码回归覆盖超长块和空行保留；相邻脚注编号有分隔。最终 `regression-v5.log` 中 export-service 全部包测试通过，`architecture.log` 中 Peer/Runtime/Textbook 相关架构测试通过。未把这些定向检查称为完整仓库回归；本轮无前端变更、无新依赖、无 Provider 调用。

## 审阅结论与未覆盖项

DOCX 当前单章校样范围通过，整书 layout 阶段仍记 `needs_revision`、2/4：真实服务端 PDF/ZIP 尚无可用校样，已有 Chromium 沙箱不可用问题未在本切片修复，未重试不变条件。`final-v5-render` 中 PDF 只用于 DOCX 视觉 QA，不能算服务端 PDF 导出成功。

已通过真实浏览器表单保存，POST 返回 201。审阅记录 `348a2048-562b-4bd7-9933-31a93aa68737`，`layout` v1，`reviewer_kind=delegated_ai`。随后 GET 与页面回读均确认该结论；当前新构建委托审阅 1 条、真人审阅 0、权利记录 0、preflight false，仍为 ready_for_review。提交与读回证据为同目录 `browser-review-submit.log`、`editorial-after.json`。最终 `git diff --check` 通过。

目录是静态可点击链接，离线修改标题需同步目录或重新从母稿导出；不同 Word/LibreOffice 版本可能重新分页，未验收每种客户端。代码分段不是语言语法解析，超长换行仍需校样。第 11 页母稿仍有“下面的动手节”这一方向指代待文字复核；本切片未在导出层偷偷改写已批准内容。

权利清单、独立读者试学、真实学习者六维表现及其它 audience 不在本次版面验收范围。整体开发计划仍在执行，不能晋级为出版就绪。
