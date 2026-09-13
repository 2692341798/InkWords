# 教材 PDF 导航与浏览器页码

日期：2026-09-10。

实际 Chromium 124 CLI 校样无目录和页码，每个二级标题前强制分页，代码与来源条目跨页拆散。保留冻结 Canonical Book AST 和正常 Chromium 沙箱，将浏览器生命周期移到 `export-service/infra/bookrender`；domain 只生成投影 HTML 和返回投影合同，bootstrap 注入适配器。单服务实例最多同时启动一个 PDF 浏览器，45 秒期限包含等待、启动和打印。

Goldmark 解析标题并分配去重标识，生成静态可点击目录；代码仅按原有换行分段，拼接显示文本与原文一致，不改母稿或工件。保留默认 raw HTML 拒绝和禁网 CSP。来源使用已有脚注去重，打印为末尾来源注释；重复引用仍可回链。标题、代码说明、列表引导语和其后内容保持关联，长代码单元和来源条目避免跨页；特别长的内容仍需实际校样。

页码通过 Chrome DevTools Protocol 的 `Page.printToPDF` 页脚模板得到真实 pageNumber/totalPages。未在 CSS 中伪造计数；Chrome 的 [页边生成内容](https://developer.chrome.com/blog/print-margins?hl=en)从 131 才支持，当前 124 不具备。页脚不显示临时文件路径、系统打印日期或虚构出版信息。使用明确空页眉、中文页码、Tagged PDF 和文档大纲。

## 客户端选型

固定 `github.com/chromedp/chromedp v0.15.1`，对应 cdproto `v0.0.0-20260321001828-e3e3800016bc`；不手写 WebSocket/CDP 客户端。官方 [go.mod](https://raw.githubusercontent.com/chromedp/chromedp/v0.15.1/go.mod)要求 Go 1.26，与本仓库一致；[MIT 许可证](https://raw.githubusercontent.com/chromedp/chromedp/v0.15.1/LICENSE)允许此类使用并要求保留声明。上游维护者 [advisories](https://github.com/chromedp/chromedp/security/advisories)当日没有已发布条目，这不等同于完整漏洞扫描或安全认证。当前环境无 govulncheck 可执行文件，本轮未声称通过漏洞扫描。

仅新增该客户端及它的六项传递模块，未升级已有版本。Linux arm64 服务二进制由 40,232,555 增为 42,924,684 字节；没有 Node/Python 服务运行依赖、Chromium 更新或前端包体变化。真实旧 Chromium 124 上的打印兼容性通过一次性容器及主项目网关校样验证，不能从最新协议文档推断所有旧浏览器 API 都可用。

不复制 chromedp 的默认启动选项，因为其中存在关闭部分浏览器保护的选项；显式设置 `no-sandbox=false` 和 `disable-setuid-sandbox=false`，防止 root 自动回退。测试捕获真实启动命令核对没有绕过标记，实际运行仍使用此前已验证的非 root/seccomp/只读运行覆盖层。新增架构回归禁止业务层引入 chromedp。

旧 CLI 实现和测试已保存到本轮 QA 目录，旧服务镜像保留；回退部署至前一镜像可恢复原输出。无数据库迁移，BookBuild manifest 与已批准母稿不变。实际导出与审阅证据见 `docs/qa/textbook-pdf-layout-2026-09-10.md`。
