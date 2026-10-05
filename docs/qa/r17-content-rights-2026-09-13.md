# r17 正文与教学代码补证验收

2026-09-13。Codex 用户委托 AI 复核与本机页面验收；不是真人同行审校、出版社认证或著作权归属证明。

当前冻结构建 `9c4dfc67-532a-46f6-906a-80a2a7876e67`，manifest `sha256:0dc7bedd7b784b3d399258da513dd00de9de91cd6117c2a96dfb3ef2551ed730`，批准 r17 `bbffa54d-37ad-4b49-910d-4470159a67af` 保持不变。本轮只有台账补证、委托复审和文档记录，无产品代码、部署、依赖安装、Provider 调用或教学代码执行。

## 核对范围与结论

实际审校 ZIP 与 9 月 10 日已审 r13 ZIP 比较：main.go、main_test.go、go.mod 与 THIRD-PARTY-NOTICES.txt 逐字节相同，16 项冻结引用完整一致。三份随稿声明按标题对应后文字相同，只有数组排列不同。正文差异已逐段审读，仅 GET/POST 句式与独立迁移练习及其题目、提示、答案和练习摘要标记更新；无新增来源或上游代码摘录。

本次再读三份工件文件，仍为静态分段教学树与六个测试。固定 Gin 源码许可和当前工件完整声明对应；旧有限源码比较用作来源追踪，不按相同行数判断权利。当前代码补证只涵盖这三份固定文件随教材用于教学、审阅及练习，并要求保留完整 Gin 版权、许可及免责声明，不覆盖截图或其它作品。[固定 Gin LICENSE](https://github.com/gin-gonic/gin/blob/73726dc606796a025971fe451f0aa6f1b9b847f6/LICENSE)

正文中 Gin/Go 已有来源、许可及改写声明继续保留。Go 正文和代码采用不同许可，本轮重读官方说明；Apple 操作资料的具体出版使用依据仍未确认，正文保持 pending。不因“官方公开”认定可出版，也不据此认定侵权。[Go 版权说明](https://go.dev/copyright)、[Go LICENSE](https://go.dev/LICENSE)、[Apple 网站条款](https://www.apple.com/legal/internet-services/terms/site.html)

真实页面追加两条 `inkwords.rights-amendment.v1`，均为首条补证、`delegated_ai`，原件不改写：

| 对象 | 原件 | 新补证 | 有效状态 |
| --- | --- | --- | --- |
| r17 正文 | `081edf39-c26b-4519-b668-d39b971ee1b8` | `35b92b50-64f9-4b0b-973e-03586d615167` | pending，具体剩余项为 Apple 操作资料出版依据 |
| 当前教学工件 | `5fd2d82b-064e-4a75-b65f-95c5b8b511e7` | `b454f563-b40b-4d85-b269-755259603e08` | ready，限定精确文件与随附声明 |

随后真实页面保存 rights v3 `83615c59-4023-4ee5-a238-99a5daed5f24`：needs_revision，1/4。保留 RIGHTS01（Apple 具体用途）、RIGHTS02（三张历史 VS Code 截图界面/商标等来源与复用条件）及 FONT02（历史截图实际字体身份缺失）。四项 PDF 字体已核对不替代截图字体身份；本轮没有提交资产字体声明。

## 实际验证

- 页面选择用户委托 AI、填写并保存两条补证；看到 code ready、prose pending。权利复审显示 v3 1/4、历史十条，真人记录仍为零。浏览截图显示实际预检仍阻断；没有使用未获扩展授权的本地浏览器截图脚本。
- API 原九条 RightsItem、原九条委托审阅及冻结 build 全部保持相等；只新增两条补证和一条复审。有效权利九项：五 ready（代码与四字体）、四 pending（正文与三图）。
- 新 ZIP `output/real-acceptance/2026-09-13/r17-content-rights/review-bundle-after.zip`，SHA-256 `5ac1000f5c8285fb4378008a167b1c888fd8c89eab0e982160343b9ff3f403e3`；24 项内容哈希及 manifest 哈希全部通过。完整账本、有效权利和十条委托审阅与 API 结构相同。
- 冻结 AST、构建清单、代码、三张图片、Markdown、视频教案、随稿声明和空真人记录均与上轮逐字节一致。verification-summary 只有导出重新评估时间改变，收据与其它字段完整相同。
- API 阻断 16→14、ZIP 阻断 18→16；出版候选仍 false。计数变化是补证结果，不代表整书已通过。
- 本轮未重跑产品全套测试或全页 PDF/WPS 校样：没有产品行为或冻结正文变化。前轮工程及版式验证保留其各自范围。

核对脚本的初次失败已诊断：声明顺序不同，改为按标题映射比较；复审实际字段为 verdict 而非 decision；验证摘要导出时间会更新，改为精确比较其余字段。均为本地核对脚本假设修正，没有修改生产门禁或冻结证据。

证据目录 `output/real-acceptance/2026-09-13/r17-content-rights/`：`inspect_content.py`、`content-comparison.json`、`r13-to-r17.diff`、`review-scope.md`、`editorial-before.json`、`editorial-amended.json`、`editorial-after.json`、`verify_export.py` 和 `verified-facts.json`。复用旧审阅限于实际相同字节与未变段落，不自动继承旧构建权利 ID。

动手实践代表章、真实运行/页面观察、独立冷读、个人六维学习和延迟保持仍缺，原计划三项综合验收不勾选。未把已生成、已读、AI 审阅或原示范运行等同于这些验收。
