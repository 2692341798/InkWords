# r11 PDF 版式复核

日期：2026-09-10。审阅来源：用户委托 AI。范围：当前冻结单章校样，不代表真人试学、整本教材或出版社验收。

## 实现与部署

PDF 由冻结 Canonical Book AST 生成带目录的 HTML，再经基础设施层 chromedp/CDP 打印。保留当前 Chromium 沙箱配置、非 root、只读根、零 capabilities 和 no-new-privileges；未使用默认浏览器参数集合或关闭沙箱。打印具有 45 秒超时和单实例并发上限，等待字体就绪后输出真实页码、可点击目录和文档书签。

采用正文连续排版、标题与下一段保持、代码按原有空行分成可容纳的显示单元、来源条目保持和相邻脚注分隔。HTML 分隔仅作用于实际相邻上标，回归覆盖被正文或空白隔开的引用，避免 CSS 兄弟选择器误加逗号。两段原始代码重组后逐字一致，含全部空行；没有修改母稿。

部署镜像：`inkwords/export-service:pdf-layout-v5-20260910`。新增固定版本 chromedp v0.15.1 及其依赖，许可证随镜像保存；既有依赖没有升级。实现取舍、依赖和回退说明见 `docs/decisions/textbook-pdf-pagination.md`。

## 权威校样

输出根：`output/real-acceptance/2026-09-10/pdf-layout/`。

- BookBuild：`8aa50b6b-45b3-477d-aece-4bbe1b164089`。
- 冻结 manifest：`sha256:4164d754fc06350bf2a4bbb08e5d7b9281e8d7ba55a41d13af7075b9869e960b`。
- 最终文件：`Gin-r11-排版校样.pdf`，真实网关 HTTP 200，1,036,253 字节。
- SHA256：`3ae2ee7566fb5eb989e5c77eb88743b7fb7913b188d90bc82e7d5b047ff56e71`。
- 全部 14 页已渲染并逐页查看；目录、正文、代码、练习和来源可读，无裁切、重叠或孤立代码尾行。9 个 PDF 目录链接实际解析到有效页面，全部页码正确。
- HTML 中 16 条去重来源注释、9 个目录目标有效；两段代码分为 9 个显示单元后重组字节一致。DOCX 使用 34 条逐次引用脚注，计数差异来自投影规则。

详细断言见 `final-facts.json`，逐页校样在 `render-v5/`。前序 v2/v4 和旧 CLI 19 页校样保留作缺陷证据，不作为最终交付。

## 实际验证与边界

export-service 全部包测试通过（`service-tests-v5.log`）；浏览器 SDK 层级及相关架构检查通过（`architecture-final.log`）。非 root、只读、无网络、相同 seccomp 的一次性 Chromium 测试实际运行并通过（`real-chromium-test.log`），另有真实网关导出证据。`go mod verify` 返回 all modules verified。没有声称执行依赖漏洞扫描或全仓 Go 回归。

当前单章 DOCX 18 页既有校样和 PDF 14 页均满足版式 3/4。第 9 页母稿仍有“下面的动手节”这一方向措辞问题，属于文字审校范围；导出器不会暗改已批准内容。权利、真人试学、延迟保持、其他读者版本及完整计划仍未完成。本切片没有 Provider 调用、学习作答或 Git 提交。

## 已落库审阅与审校包

真实页面保存 layout v3，HTTP 201，记录 `8256948b-c47e-4ced-a57f-c2398248fc74`，`delegated_ai / pass / 3`；页面回显“版式审校 · 委托 AI 通过 · v3 · 3/4”。GET 回读确认三条历史保留、manifest 不变、真人记录 0、权利项 0、preflight false。

随后真实导出 `Gin-r11-版式审校包.zip`：545,071 字节，SHA256 `8380c534d6ac7630c157a0ef2d2d1089849abb4aa759889e5ad7cdab6fbd63c0`。17 个文件 CRC、manifest.sha256 和每项文件哈希全部通过；包中含新增 v3 审阅。冻结 AST 与 Markdown 未变，批准 r11 原文 SHA256 仍为 `6431a0c9f1a4323c52f6688926571a1711f2074d9775d59e76dee9714fb22bf3`。

包内 PDF 与最终独立 PDF 的 14 页提取文本和每页内容流逐一相同；DOCX 与已通过的 v5 仅 `docProps/core.xml` 元数据不同。未把动态生成时间导致的文件哈希差异误判为内容变化。详细记录见 `bundle-facts.json` 与 `editorial-after-v3.json`。最终 `git diff --check` 通过。
