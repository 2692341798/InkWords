# r17 截图来源复核与出版待问事项

2026-09-13。Codex 用户委托 AI 实际像素审读与来源核对，不是真人审校或法律结论。范围是当前冻结构建 `9c4dfc67-532a-46f6-906a-80a2a7876e67` 的三张历史图。没有改变图片、正文、批准稿或冻结 manifest。

## 实际图像分层

三张原始 PNG 与当前 ZIP 内对应图像逐字节相同。前两图为 3164×1926，局部测试图为 2120×1460；精确哈希与当前 asset ID 见 `output/real-acceptance/2026-09-13/r17-screenshot-rights/asset-evidence.json`。

| 原文件 | 本次实际可见内容 | 已有证据边界 |
| --- | --- | --- |
| ide-source-layout-raw.png | 完整窗口轮廓、文件树、main.go 静态结构、受限模式提示、状态栏、中英文界面与图标 | 已核教学源文件，不等于界面/图标/字体全部可复用 |
| ide-test-baseline-raw.png | 完整窗口轮廓、main_test.go 方法隔离尾部和登记/斜杠测试、受限模式提示 | 没有运行输出，不能把可见测试代码当作执行证据 |
| ide-test-layout-raw.png | 编辑器局部、TestAPINotRegistered 与 TestMethodSpecific；多处可见 TraeCode Plugin 的解释/注释 CodeLens，还有 run test / debug test | 不能仅按 VS Code 本体和教学代码处理；不能从按钮推断已运行或调试 |

原采集清单明确写明局部屏幕区域，字体/缩放精确值没有记录。另两图本次未看到相同 TraeCode 文字，不等于证明没有第三方组件。不能从现有字体配置反推历史像素的实际字体。

## 可复核的一手来源及限制

微软的[通用版权使用规则](https://www.microsoft.com/en-us/legal/intellectualproperty/copyright/permissions)对截图用途附有条件，涉及完整画面、第三方内容、产品界面内使用及署名。当前局部图和扩展文字使适用性不能直接认定；这是待核对事项，不是侵权结论。完整画面或加署名本身也不保证其它条件已满足。

[Visual Studio Code 产品许可](https://code.visualstudio.com/license)与 [FAQ 的许可说明](https://code.visualstudio.com/docs/supporting/faq#_licensing)区分 Microsoft 发行版与 MIT 的 Code - OSS 源码，扩展有自己的许可。实际安装的 package.json 虽有 MIT 字段，不能据此覆盖成品界面。当前安装产品元数据为 Visual Studio Code 1.137.0、stable、提交 `645f29cc3176500b4b5762ba887cf2a7f0ffdf2c`；本机 LICENSE.rtf 是产品许可，相关文件哈希已保存。这些是当前安装观测，不是历史采集时的精确扩展或字体证明。

[官方名称与图标规范](https://code.visualstudio.com/brand)支持按其规则作教学介绍，要求首次全名并避免暗示背书；它并未消除截图中其它作品的条件。当前冻结 Markdown 中 Visual Studio Code 全名、微软规则中的特定署名句均未出现。暂不添加“已获许可”的断言，也不以名字相似推断 TraeCode 的准确发布者或许可证。

## 交给出版审阅者的具体问题

1. 对三张指定哈希图在本地学习软件、审校 PDF/DOCX、公开教材中的分别使用，采用哪些具体依据？通用截图规则是否足以覆盖局部图及软件内教材展示，或需其它适用依据？
2. 图中 TraeCode Plugin 的准确产品、发布者、版本与 UI 使用条件是什么？没有采集时证据，不能按当前安装猜测，也不能把教学代码的限定 ready 当作扩展许可。
3. 中英文字体、图标字形、系统窗口元素分别来自哪里？应记录实际字形来源和文件/许可，无法识别时需新的完整采集证据，另存资产并新冻结。
4. 在适用条件确认后，哪些作品需要怎样的署名、许可或不背书说明？声明必须绑定具体作品并随新冻结导出，不能改写旧冻结通知。

没有发出咨询邮件、上传截图或代表用户申请许可。新采集不能通过抹去标识把旧局部图包装成获准素材；原图及所有历史记录保留。

## 本次进度与剩余范围

新增证据将“截图来源待核”缩小到可定位的局部图、第三方 CodeLens、声明和字体问题；仍保持 pending。Apple 正文使用依据、动手章、真实观察、个人学习闭环与冷读者试学继续未完成。本轮没有产品代码、部署、依赖安装、Provider 调用或教学执行。

本机构建 API 实际追加补证 `b1636f41-2974-4000-b16f-efbb8c3cc37a` 返回 201，绑定原件 `7061e7c9-5961-42d2-911f-8f30f4b599f0`，有效状态 pending。真实浏览器刷新并重开项目后，新增具体依据完整恢复。页面实际填写并保存 rights v4 `ee50c4fc-b160-4086-8998-743e34d28fb3`，needs_revision 1/4，十一条委托 AI 审阅、零真人。

`verify_export.py` 实际通过：九条原件、原两条补证与原十条审阅不变；当前三条补证，有效五 ready/四 pending。新 ZIP 24 项内容哈希与 manifest 哈希通过，完整账本与十一条审阅和 API 相等，冻结 AST、代码、三图、Markdown、教案及随稿声明逐字节不变。ZIP SHA-256 `c810dc26b4960866b7adcec6c33dcce2d7584be1699121e59b4497fc006439f7`。API 14/ZIP 16 阻断保留、出版候选 false。未重跑产品全套测试或全页版式检查；没有相应实现或版式变动。
