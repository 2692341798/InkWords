# 冻结图片嵌入与实际导出验收

日期：2026-09-13；执行者：Codex（delegated_ai）。本记录不是真人试学或出版批准。

## 修复与边界

原渲染器只输出资产注释，ZIP 即使携带 PNG，DOCX/PDF 中也没有图片。
本轮增加只接受冻结内容哈希的 `BookImageSource`，部署读取仍使用非 root 进程和只读卷。
读取后再次校验 SHA-256、文件大小、尺寸和完整解码；单图限制 10 MiB/4000 万像素，
每本书不同图片的原始字节总量限制 100 MiB。缺失、损坏、篡改或未绑定的图片引用导致
导出失败，禁止从当前修订补图或访问正文给定的远端 URL/宿主路径。

复用已安装 Goldmark 1.8.5 的图片解析器和源位置，支持内联、引用式、空替代文本、
转义和引用块图片。已定位图片在原位置替换为冻结 data URI；未定位资产在本章末尾
插入图片与资产身份。代码块与行内代码保持原字节，不靠全局替换 URL 改正文。
Markdown 可携带完整图片，Pandoc 将相同字节嵌入 DOCX；PDF 保留仅允许 data 图片的 CSP，
等待所有图片完成解码再打印，并显示说明图注。PNG/JPEG 已支持；WebP 解码依赖授权尚待答复，
本轮没有安装依赖或把 WebP 当作通过。

PDF 字体证据使用同一图片源重新计算实际 HTML 哈希，ZIP 同步验证该绑定。
新增图像不会自动通过字体来源门禁：Fontconfig 不能证明截图内部字体的权利与来源。
工具清单增加 `inkwords.frozen-book-images.v1`，PDF 布局版本为 `inkwords.pdf-layout.v3`。

## 实际验证

- 新增冻结图片测试先失败后通过，覆盖原位嵌入、正文不变、代码示例保留、缺失/篡改/
  非图片内容拒绝、外部引用拒绝、真实 Pandoc 图片字节、ZIP/字体证据绑定。
- 最终 `GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...` 通过；日志在本轮
  `embedded-images/backend-final-tests.log`。本轮没有前端改动，没有重报新的前端测试数。
- 本机 export-service 更新为 `inkwords/export-service:frozen-images-v3-20260913`，健康。
  保留各服务原 Compose 链、环境值、非 root 用户、只读资产卷与正常 Chromium 沙箱。
  参考 DOCX 哈希仍为 `39a60a28c70285ccd0276f87d4f6ce7aaefd14717e0ae18cea125b6df1a6423d`。
- 真实工作台点击“生成 DOCX”收到下载事件；四格式正式接口返回成功。
  使用同一构建 `966e247b-0473-4d54-936c-c1fd30ae92be`，冻结 manifest 为
  `sha256:d41ac94c7ca0a7784bcf19387c3d53ba499051ed8abe6da70315694e06195876`。
- 最终 ZIP 的 AST 与冻结 manifest 和修复前逐字节一致；24 项内容哈希和 manifest 哈希
  通过。三张 PNG 与原始采集文件完全相同；Markdown data URI 和 DOCX `word/media`
  均恢复出同一三张原图。最终 ZIP 的所有 `word/` 部件与 WPS 实际打开文件一致。
- 最终 PDF 22 页，按页面真实图片绘制操作核对第 16–18 页各一张图片；不是只数资源字典。
  对图片页逐页查看，图注中静态观察和非运行证据的限定保留；原始全窗口截图的代码较小，
  可复制代码仍由正文与代码附件提供，不把整窗图当作逐字抄录材料。
- WPS 实际打开带图 DOCX，26 页、没有新的缺失字体提示；第 22、23、24 页分别查看
  基线测试、方法隔离测试、源码结构图片与中文图注。未在 WPS 保存或改写母稿。
- 独立 `render_docx.py` 首次未传字体配置，中文缺失，该 18 页结果不算通过。改用此前
  已核对的 `content-rights-review/docx-fonts.conf` 后为 26 页；第 1–20 页与旧校样
  逐像素一致，第 21–26 页已逐页查看。

## 尚未完成

更正（2026-09-13）：再次打开 `docx-proof-fonts/page-22.png` 和 `page-23.png`，
两张原始校样均完整显示“第 22 页”“第 23 页”；直接操作 WPS 滚动至两页底部也
看到完整页脚。此前“只显示数字”的结论是 AI 观察错误，不是已复现的渲染缺陷。
无需字体实验、模板修改或重新部署；原始校样与导出文件保持不变。更正证据见
`embedded-images/footer-correction.json`。

文末许可声明较密，仍需要继续版式精校。本轮只证明冻结图片进入了各导出格式，
不宣称整书版式审校通过。

构建仍有 17 项出版阻断、没有新审阅；历史构建及其两条 AI 审阅没有复制到新构建。
截图保持 unverified、rights pending；个人六维学习、延迟保持、读者试学、代表性
hands_on 样章及其它整书验收继续未完成。成片录像不属于 V1 必交付项。

证据目录：`output/real-acceptance/2026-09-13/embedded-images/`。
最终文件为 `review-bundle-final.zip`、`book-final.pdf`；WPS 检查文件为 `book.docx`。
`verify.py` 和 `verification.json` 记录完整字节/哈希核对与限制；`docx-proof` 是失败的
默认字体试验，`docx-proof-fonts` 才是指定字体的独立校样。
