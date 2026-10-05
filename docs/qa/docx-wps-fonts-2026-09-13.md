# r15 DOCX 字体与 WPS 真实校样验收

日期：2026-09-13。执行者：Codex（用户委托 AI，`delegated_ai`）。
对象：冻结构建 `66e5f4a4-00ac-449e-82c1-1647d8dfd248` 的批准 r15。
本记录是实际工具执行与版面观察，不是人工试学、出版社审校或出版批准。

## 发现与修复

首次在用户电脑 WPS 打开旧 DOCX，出现 6 组缺失字体提示、标题使用 Calibri 主题字体，
页面为 28 页。参考模板虽然有 Noto 字体声明，但同时保留 Office 主题字体属性和
过期字体表；目标账户也未安装模板需要的 Noto CJK / FreeMono。

- `normalize_reference_fonts.py` 清除 `w:rFonts` 主题属性，显式设置中文/等宽字体，
  将字体表收敛为 Noto Sans CJK SC、Noto Sans Mono CJK SC、FreeMono。
- 用户单独授权后，通过字体册“将字体添加到当前用户”安装 5 个已整理的本地文件，
  合计 40,415,212 字节。5 个安装后文件均与候选 SHA-256 一致，位于 `~/Library/Fonts`。
  没有下载、购买云字体、替换已有字体或安装编程依赖。
- 重启 WPS 后缺字提示消失，页数为 23。继续发现短表格最后一行被拆到两页；
  参考模板的 `Table` 样式增加 `w:trPr/w:cantSplit`，真实 WPS 复验中 `nil` 行
  完整移至第 9 页，重复表头保留，各列内容完整。
- 每次 Pandoc 调用冻结实际参考文件字节并返回 `docx_reference_sha256`；审校 ZIP
  的 `manifest.json.renderer_tool_versions` 保存 DOCX/PDF 实际工具版本，避免模板变化
  无法追溯。它与不可变 `book-build-manifest.json` 分开，不修改冻结母稿。

## 实际验证

- 字体声明、跨页行规则和 ZIP 渲染版本回归均先失败、修复后通过。
- `GOCACHE=/tmp/inkwords-go-build go test ./services/export-service/domain/export ./services -count=1`
  通过：export 2.923 秒、架构 0.407 秒。本轮未重复全后端或前端全量测试。
- 本地 export-service 以 `docx-fonts-v3-20260913` 更新；保留原 Compose 配置链和全部
  已配置环境值，其余服务未重建。
- `book-fonts-v2.docx` 经独立 LibreOffice 渲染为 23 页；第 1–20 页已逐页查看，
  第 21–23 页补齐查看。v2 全部页图与前一字体修复版逐像素一致；没有发现缺字、
  裁切或内容重叠。长声明较密、部分段落跨页，仍属于可继续精校的排版范围。
- WPS 实际检查标题、安装命令、多行 Go 代码、比较表第 8/9 页及文末来源声明。
  缺失字体警告已消失；这不是声称在 WPS 逐页检查了全部 23 页。
- v2 与 v1 的 `word/document.xml`、`word/footnotes.xml`、`word/footer1.xml` 字节相同；
  v1 这些内容部件亦与旧导出相同。历史导出文件保留。
- 最终 ZIP 正式接口 HTTP 200，21 项内容哈希及 manifest 哈希全部通过；DOCX 的全部
  `word/` 部件与 WPS 实查校样相同。冻结 manifest、AST、Markdown 和两份教案文件
  与此前审校包完全相同。清单记录 Pandoc 3.1.9 及实际模板哈希。
- `git diff --check` 通过。

## 证据位置

本轮目录：`output/real-acceptance/2026-09-13/docx-fonts/`。

- 安装候选与校验：`install-candidates/manifest.json`、`installed-fonts.json`。
- 交付校样：`book-fonts-v2.docx`，SHA-256
  `1c9ffcb8d8a10da675f5ec34f6387c7efb8c75a309568417cf707712e179e09d`。
- 参考模板 SHA-256：`39a60a28c70285ccd0276f87d4f6ce7aaefd14717e0ae18cea125b6df1a6423d`。
- 独立校样：`render-v2/book-fonts-v2.pdf` 与 `page-1.png` 至 `page-23.png`。
- 最终审校包与哈希复核：`review-bundle-fonts-v3.zip`、`validation.json`。
- WPS/字体册原生截图已在任务中实际展示；没有声称这些截图已保存为本地原图。

## 保留的边界

字体安装只解决该账户的客户端显示，不等于字体出版使用证据已经登记。
构建仍为待审，审校包仍有 13 项阻断，未晋级出版候选。
个人六维学习、当前静态素材/录屏、hands_on 代表章、权利补证与整书试学审校仍待完成。
本轮未新调用模型、未在宿主机运行教学代码、未放宽沙箱、未提交或推送 Git。
