# PDF 字体观测、文件追溯与权利范围

日期：2026-09-10。审阅来源：用户委托 AI。当前状态：正文观测和导出绑定已实现；完整打印字体覆盖尚未完成。

后续检查点：受限 Fontconfig 来源、字符覆盖与同构建字体权利联验已完成实际部署，PDF 字体门禁
已通过；DOM 页脚观测语义仍保持 unobserved。见 `docs/qa/closed-pdf-font-profile-2026-09-10.md`。
本文以下保留初始 v1 观测与 rights v4 的历史事实。

## 实际发现

固定构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4` 的 r13 母稿与源码保持不变。
相同字体容器环境中，真实 Chromium print-media 观测了 449 个正文文本父节点，报告 5 个字体名称：

| 字体 | 观测字形累计 | Fontconfig 对应 |
| --- | ---: | --- |
| FreeMono | 5915 | 单一文件/face 候选 |
| FreeSans | 136 | 单一文件/face 候选 |
| NotoSansCJKsc-Bold | 364 | 单一文件/face 候选 |
| NotoSansCJKsc-Regular | 11258 | 单一文件/face 候选 |
| NotoSansMonoCJKtc-Regular | 92 | 单一文件/face 候选 |

5 个 face 对应 4 个实际字体文件；Noto Regular 集合同时包含简体正文字体与繁体等宽候选。
这是字体名称与字形使用观测，不把字形数称为唯一字符数或完整 PDF 字体清单。
62 个已安装 face 只代表候选库，不能把未观测字体都说成实际使用。

## 实现和范围

新增 `inkwords.pdf-font-evidence.v1`，绑定 Canonical Book AST、实际 canonical HTML 和最终 PDF 的 SHA-256。
每份新审校包增加 `projections/book.pdf.font-evidence.json`，包含 Chromium 版本、print-media 正文观测、
候选文件路径/哈希、TTC face index、Fontconfig 原始版本字段和明确限制。
package 校验会拒绝把其它母稿、HTML 或 PDF 的报告附到当前文件；字体来源不完整时不能生成
`publication_candidate=true`。缺报告的旧/未配置渲染路径也不能被当作字体已经核对。

适配器使用现有 chromedp/CDP，没有引入依赖或降低浏览器沙箱隔离。Fontconfig 输出限 1 MiB/2048 face，
命令 3 秒；仅允许部署字体目录的普通文件，每文件至多 64 MiB；同一个 TTC 只哈希一次。
字体节点观测最多 10000 节点、5 秒，整个渲染继续受已有 45 秒截止与单实例并发限制。
缺失/歧义/自定义字体继续标记 unresolved，观测失败不伪造清单。

CDP 的这个接口仅说明给定节点的子文本使用字体，并未覆盖浏览器内部生成的页眉/页脚。
文件匹配基于 PostScript 名称与实际 Fontconfig 候选，不冒充操作系统文件打开跟踪。
因此 v1 明确保留 `body_observation_status=partial`、`print_furniture_status=unobserved`，不宣布完整字体来源已通过。
协议依据：[Chromium CSS.getPlatformFontsForNode](https://chromedevtools.github.io/devtools-protocol/tot/CSS/#method-getPlatformFontsForNode)；
文件元数据依据：[Fontconfig 手册](https://fontconfig.pages.freedesktop.org/fontconfig/fontconfig-user.html)。

## 文件内许可核对与真实登记

在内存中直接读取实际容器字体文件，用现有 fontTools 提取 name 0/5/6/13/14；
取得的原字节 SHA-256 与观测候选一致，未下载或打包字体二进制。
Noto 字体内记载 Version 2.004 与 SIL OFL 1.1；FreeMono/FreeSans 内记载 Version 0412.2268、
GPL v3 或后续版本，并明确附有“未修改字体/部分嵌入文档不因此使文档受 GPL 约束”的例外。
GNU 站点本次访问超时，不把该站点标作成功获取；具体例外来自已哈希的实际字体文件元数据。
OFL 对文档/子集嵌入与独立字体分发作了区分，参见 [OFL FAQ 1.12–1.13](https://openfontlicense.org/ofl-faq/)。

经用户委托范围复核，实际浏览器登记 4 个以 `font-file:sha256:...` 为主体的 ready 项：
`fdcbf617-3b38-4b36-999c-79891b08e6c5`、`60d0556d-4241-480d-a361-5d2575f8c415`、
`de437352-e9ad-493c-abad-1445420749bc`、`17352adf-5f8a-4876-98ed-b6d19c55520b`。
4 次实际 HTTP 201，范围限定为对应未修改字体的文档渲染/嵌入，不包括单独分发或修改字体包，
不代表已覆盖全部打印字体，更不证明正文的来源权利通过。元数据与判断原件保存于
`observed-font-name-metadata.json`、`font-rights-decisions.json`、`saved-font-rights.json`。
原正文/代码两项仍 pending，代码补证历史不变。

## 验证与运行环境

真实隔离探针使用现有 exporter 镜像、非 root、read-only、cap-drop ALL、no-new-privileges、
现有 Chromium seccomp、网络 none、1 GiB 内存/192 pids 和固定本地母稿；没有 Provider 调用。
实际 PDF 渲染测试 0.70 秒通过。新旧 16 页逐页文本、72 dpi 像素全部相同，抽查页面 PNG 可读；
观测没有修改正文或排版。默认测试仍显式 opt-in，不把跳过探针当作已运行。

exporter 全包、架构、全量后端 57 测试包通过；最后补充 HTML 绑定与 Chromium 版本后，相关导出/渲染
回归再次通过。前端未改，本轮不重复声称重跑前端测试。
`inkwords/export-service:font-provenance-v2-20260910` 已部署，保留真实 vault、review 模型和 profile、
其它服务以及既有 PDF 沙箱参数。实际 ZIP 已返回含报告的 19 文件版本。

最终审校包 `Gin-r13-字体来源与权利审校包.zip`：616094 字节，SHA-256
`ccbb86c1d039de18a39e242e55bd77cedafa926bc651cdfc0ca3f9590f609824`。
19 个文件与 manifest 哈希全部通过，字体报告 PDF/AST 绑定通过，生产导出的 16 页文本和逐像素
仍与原包相同。core 与包内权利历史/审阅一致：6 权利项（原正文/代码 2 pending、字体 4 ready
限定用途）、原代码补证 1 条、委托审阅 11 条、真人 0 条。字体报告的 `not_reviewed` 是自动检测层
不作许可审批的状态；用户委托的限定权利决定单独保存在 rights-ledger，不混写自动记录。

实际浏览器追加 rights v4 `22aa0f2b-61d2-47f2-902f-9be59258a598`，HTTP 201，继续
needs_revision；旧权利项、补证、十条审阅与冻结母稿不变。导出 manifest 仍明确阻断完整字体覆盖，
未晋级出版候选。生产镜像 ID `069aac5d2c8a01a4321328dcab01a52cd7d1908f36853d91140abb3aa26a6da7`，
healthy；真实探针容器已退出且 exit 0。见 `final-facts.json` 与 `browser-final.log`。

## 下一步

为完整打印来源建立封闭的部署字体集合：Fontconfig 配置只能解析已哈希且已核对许可的字体文件，
正文观测仍保留，页脚即使无法逐节点调试也不能从集合外回退。必须验证实际可见字体集合、
缺字/布局与所有导出格式的范围；不能通过关沙箱或把未观测字段改成 true 完成。
此外继续正文/代码引用改写与分发声明、真人复审版本入口和原计划其它验收。
