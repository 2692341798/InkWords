# r17 PDF 字体权利与历史截图缺口

2026-09-13，用户委托 AI。构建 `9c4dfc67-532a-46f6-906a-80a2a7876e67`。
本轮只补证、追加复审及核对导出；没有修改产品代码、安装依赖或重新采集图片。

## 本次文件核验

从正在运行的 exporter 读取当前受限目录中的四个实际字体文件，重新计算 SHA-256。
它们恰好等于本构建 PDF source_set 的四个文件，22 个 face 全部读取 name 0/5/6/13/14。
与 9 月 10 日既存 metadata 中对应 face 的版权、版本、名称和许可文字逐字相同；
不是仅检查 CSS 字体名称或复制旧构建的权利 ID。

| 文件 | 哈希 | 本构建权利 ID |
|---|---|---|
| FreeMono.otf | `daf720e115bc462f721cdf3065697b75fea033bf46d4598d3a727fce4b036032` | `6f6e9373-cf50-47dc-8095-88ee1db5785e` |
| FreeSans.otf | `d3e9138ccf76ef516cc6af867e4e7ed8776331947235824f924c5886186b0c80` | `f208979e-0212-453f-b978-f823cf883cf2` |
| NotoSansCJK-Bold.ttc | `faa5f3656a78b2e2d450d27fe8382c778bc2b6bb5ea29c986664a6a435056ceb` | `840b3775-7498-4578-bf81-ae21e8b65e2c` |
| NotoSansCJK-Regular.ttc | `b76b0433203017ca80401b2ee0dd69350349871c4b19d504c34dbdd80541690a` | `fcfbe531-a358-4ea9-82b7-1d434a527fcf` |

FreeFont 文件内包含 GPL v3 或后续版本声明及文档嵌入例外；本轮 GNU 网页两次读取超时，
依据为实际文件内声明与既存同哈希证据，没有声称官网刷新成功。
Noto 全部 face 内包含 OFL 1.1 声明，并对照了
[SIL 官方 OFL 1.1 文本](https://openfontlicense.org/open-font-license-official-text/)及
[FAQ 1.12、1.13](https://openfontlicense.org/ofl-faq/)的文档嵌入说明。
本轮不处理字体修改、单独字体再分发或其它扩展用途。

四项记录为 `ready`，严格限定固定文件用于当前教材 PDF 字形渲染、文档嵌入和打印。
不据此确认正文、截图的整体出版权利，也不主张这些就是历史 VS Code 截图使用的字体。
原五个必需主体仍 pending，旧记录保留；现在共九项作品权利。

## 历史截图无法从现有证据确认字体

三张原始 PNG 的哈希仍与冻结包相同。PNG 可见 sRGB/EXIF 信息，嵌套 EXIF 只有
ColorSpace、ExifImageWidth、ExifImageHeight，没有字体身份或文字元数据。
三项原始 asset.source 记录只有工具/系统、源码关联和截图定位，没有字体文件及实际使用
记录；capture-manifest 更明确写明字体和缩放沿用现有设置、精确值未记录。

因此不能用现在的 VS Code 配置、系统已安装字体、WPS 安装字体或 PDF 的四个字体文件
反推历史图片中的字形来源。本轮没有补写虚构的图片字体声明。现有资产字体合同已可用，
但三张图的完整来源证据尚未齐备。后续需要取得与实际捕获绑定的字体观测/文件依据及
全部图中文字区域的覆盖记录；新捕获必须另存新资产，保留旧图并重新冻结引用。

## 真实页面与导出

页面刷新后四项字体 ready 可见。指定截图的补证仍默认“尚未核对”；切换有文字后显示
四个未勾选候选，未做选中或保存，不能把候选清单当成该图实际字体使用证明。
真实表单追加 rights v2 `16ff6a27-b45b-4714-9442-311b25718f15`，仍需要修改 1/4。
旧 v1 及其余七条记录保留，共九条 AI、零真人。

v2 撤下旧 FONT-01 中“合同未实现、缺四项 PDF 字体权利”的判断，改为 FONT-02：
历史截图完整字形来源未确认、没有有效资产字体声明。正文/代码/界面等权利缺口保留。

新 ZIP SHA-256 `c07627f79840f51f402584f9157e4c206aa29dd5261252c64981977119ac7d13`。
24 项内容及 manifest 自身哈希通过；权利账本/有效项/九条 AI 审阅与 API 一致。
冻结清单、AST、Markdown、代码、图片、Runbook、声明和真人记录逐字节不变。
API 16/ZIP 18 项阻断，publication_candidate=false，字体整体门禁因图片缺证据继续拒绝。
本轮未重跑代码测试或全文版式校对，不改写此前测试和 WPS/PDF 校样范围。

证据目录 `output/real-acceptance/2026-09-13/r17-font-rights/`：实际字体 name 表、PNG
metadata 字段清单、四项请求/回读、复审和 ZIP 核对。首选捆绑 Python 缺 fontTools，
改用已安装 fontTools 的系统 Python 成功；没有安装或下载依赖。
原动手章、独立冷读/个人六维学习及三项综合验收仍未完成。
