# 受限 PDF 字体来源与构建权利联验

日期：2026-09-10。操作与审阅来源：用户委托 AI。

## 已验证行为

在固定 r13 构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4` 上，将 Fontconfig 候选集合从 62 个 face
限定为 4 个已核对文件中的 22 个 face。每次渲染生成独立 Fontconfig 配置，仅提取部署主配置和
19 个指定规则文件中的 match/alias，记录 20 个配置来源哈希。不加载 user/local include，拒绝
规则中的额外目录、include、file 改写。字体仅从只读镜像的专用目录选取，四个链接必须指向
规定的原文件，字节必须匹配已核对 SHA-256。没有下载、替换或修改生产字体。

字体清单与 Chromium 使用同一配置。渲染前后校验配置哈希、每个字体文件哈希和完整 face 清单；
集合外、缺失或变化的文件使渲染失败。`TEXTBOOK_PDF_FONT_PROFILE=inkwords.pdf-font-source-set.v1`
由现有 PDF Compose 覆盖层开启，未配置时仍走带明确限制的旧观测路径，未知 profile 拒绝执行。
字体升级必须重新核对文件并更新 profile，不能因名称相同自动继承旧许可。

正文每个被观测节点使用的字体候选必须覆盖该节点基础字符；不是用“其它已安装字体有这个字”
代替实际候选。页脚使用的 Noto SC Regular 须覆盖中文固定字样和全部数字。
Canonical HTML 禁止加载外部或 data 字体；图片、SVG、对象等载体仍需独立核对，存在时 PDF
字体门禁不通过。字符范围检查不等于复杂文字塑形或变体选择正确性，实际校样仍是必要证据。

## 出版门禁边界

`inkwords.pdf-font-evidence.v1` 保留正文 partial 和页脚 unobserved 的真实 DOM 观测语义，新增独立
`inkwords.pdf-font-source-set.v1` 来源约束证据，不把配置推断改名为逐文件打开或页脚 DOM 观测。
最终 AST、HTML 和 PDF 绑定照常校验。无构建与权利上下文的 `PublicationReady()` 仍返回 false。

审校包使用 `PublicationReadyForBook` 联验：受限来源、正文字符检查、稳定清单、无未核对载体，
并且每个实际允许的字体哈希恰有一个同 project/build 的有效 font 类型 ready 权利项。
缺失、pending、错误作品类型、其它构建/项目、重复或不完整记录都不能通过。
仅移除 PDF 字体这一项阻断，其它 preflight 阻断继续保留；不将字体许可扩大为正文或整书出版许可。

## 真实失败与修复证据

- 只保留字体目录、去掉系统排版规则的初始探针仍为 16 页，但所有页像素变化，因此未采用。
- 保留系统规则的静态配置探针 16 页文本/像素一致，但 user/local include 仍存在，因此继续收紧。
- 独立配置初版被 `FC_DEBUG=0` 输出前缀及空 `FONTCONFIG_SYSROOT` 导致的空清单阻断。
  Fontconfig 命令使用清理后的环境，错误/空清单明确失败；Chromium 环境合并中的 debug 值单独归零。
- `FONTCONFIG_SYSROOT=/` 时 fc-list 仍正常，但主样章第 8–10 页代码中文变成方框。PDF 提取文本
  仍相同，图片对照才暴露问题。临时字体路径/只读权限调整未解决，不保留权限放宽尝试。
  最终不设置 SYSROOT；部署环境若继承此变量则拒绝该 profile。16 页像素全部恢复相同。
- 正文字符检查发现脚注回链的 U+FE0E：变体选择符并非独立字形。按 Unicode 语义检查基础字符，
  不为选择符要求单独 glyph，也不因此宣称变体显示正确。
- 修复 stdout 上限的 `bytes.Buffer.ReadFrom` 快路径绕过：限制写入器不再匿名嵌入 Buffer，
  增加实际 `io.Copy` 回归。输出仍限制 1 MiB，命令 3 秒，整个渲染 45 秒/并发 1。

这些失败探针全部保存在 `output/real-acceptance/2026-09-10/closed-font-profile/`，不覆盖原成功稿。
官方依据：[Fontconfig 配置手册](https://fontconfig.pages.freedesktop.org/fontconfig/fontconfig-user.html)、
[Unicode 变体选择符与不支持字符显示](https://www.unicode.org/faq/unsup_char.html)。
SYSROOT 与当前 Chromium 的组合结果来自本机对照实验，不外推为所有版本的 Fontconfig 行为。

## 验证与实际交付

最终 v12 隔离探针 0.76 秒通过：449 节点检查通过、22 face/4 文件、前后清单稳定。
16 页文本与 72 dpi 像素逐页全部等于原始 PDF，第 8 页放大图确认代码中文与页脚清晰。
故意修改隔离镜像中的 FreeMono 文件后，真实运行在打印前拒绝，容器 exit 1、无 PDF；
这是预期拒绝，不冒充正常渲染通过。两者均使用现有非 root、只读根、cap-drop ALL、
no-new-privileges、Chromium seccomp、无网络、1 GiB/192 PID 的配置。

后端全量 57 个测试包通过；最后字符规则与门禁联验修改后，export-service 全包和架构回归再通过。
前端未修改。生产第一版真实浏览器 ZIP 下载成功，19 个文件哈希通过，母稿/代码/构建 manifest
原字节不变，16 页文本/像素全部相同。最终门禁联验版的生产交付事实见本目录后续检查点和
`output/real-acceptance/2026-09-10/closed-font-profile/final-facts.json`。

原正文/代码权利仍 pending，冷读/实际个人学习、其它整书验收及原计划剩余开发继续处理。
本轮没有 Provider 调用、个人掌握记录写入、真人审校记录伪装或出版候选晋级。

### 最终生产事实

`inkwords/export-service:closed-font-profile-v2-20260910` 已部署 healthy，镜像 ID
`3753348a6fd5df448e342a3e0ef7c64d91eca02ae0a92854e87c59c08a11cce1`。
保留原 review 模型/profile、真实 vault、其它服务及全部沙箱参数。实际页面下载最终包
`Gin-r13-字体来源联验审校包.zip`，616298 字节，SHA-256
`fa7d80f3b28f7ddd0bd6b8230b3506757582adfc154084ffc7ce76b8d56db483`。

19 文件及 manifest 自身哈希通过，原母稿/代码/冻结 manifest 不变，16 页文本/像素逐页一致。
导出 rights.json 与 core 权利账本的 effective_items 一致；原始登记、补证历史及审阅全部一致。
四个允许字体哈希与同构建四条 ready 项完全相等，PDF 字体阻断已从生产 ZIP 移除。
浏览器实际 HTTP 201 保存 rights v5 `5f809088-c4e3-4b3f-8f71-af98fbf48836`，撤下旧字体缺口，
保留正文/代码需修改；共有 12 条 delegated_ai、0 真人审阅。两条原正文/代码仍 pending，
读者试学未通过，publication_candidate=false。最终事实和可重复核对脚本分别是
`final-facts.json`、`verify-final.py`；全局计划未勾选完成。
