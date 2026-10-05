# 运行详情、静态采集与图片导出缺口（2026-09-13）

## 已完成并实测

补齐同源路由源码和测试画面：在临时 VS Code 窗口使用受限模式，仅静态查看 main.go 与
main_test.go；原文件哈希仍与 r15/r14 冻结代码相同。六份配套原始 PNG 覆盖源码结构、
查找与分段、main 入口及六项测试；五份为本轮采集，一份沿用上一轮。采集窗口已关闭，
原工作窗口保留，未执行教学程序或调试器。实际版本为 VS Code 1.137.0、macOS 15.7.3。

两张新增素材通过真实表单登记为 `rendered_ui`，连同既有方法隔离图共三张当前同源资产：

| 图像 | 资产 ID | SHA-256 |
| --- | --- | --- |
| 路由结构与文件树 | d0b619d3-730d-52bc-9852-c230d228fbb7 | 38d325df239a54d0682f9ad7f6a5e2d2dc44dbbe4cbd7a7e864243586ddad046 |
| 登记基线与斜杠边界 | 813aa25b-ebef-58fb-99b3-cc999de2c11d | c1224e0ee0f5db6c21c68d300b23822bf1354f336cf62a30d2e1a0e9a14b2eb6 |
| 方法隔离 | 8f11ec74-642b-577a-b2a7-b7045e8a8b82 | c915834bef74236aaeaef0fcaf4005e25bebc4cf4a7bd7302a134e063c1ad4cf |

全部保持 `manual_capture`、`unverified`、权利 `pending`，来源明确为 delegated_ai。
刷新后仍在 r14“当前稿件的代码来源”下，不改写为 r15 运行记录。

教案第三步要求读取退出码和原始输出，原 UI 仅显示数量摘要。新增 `RuntimeEvidenceDetails`
展示完整收据身份、工件/清单/输入/运行器指纹、原始 JSON，以及单命令退出码和文本输出。
缺失、旧格式和截断保持明确，内容作为纯文本显示；展开不启动任务、不改变验证状态。
真实页面展开收据 `38db421a-3978-4e0f-9726-6b6d53ea5b12` 后显示退出码 0 和
`ok example.com/inkwords/teaching 0.002s`，原始 JSON 与 API 相同。原始成功属于之前的
Go 1.26.8 隔离任务，本轮没有重跑代码。默认窄视口的既有侧栏遮挡仍在，详情内部长指纹和
JSON 正常换行；不据此宣称设备专项验收通过。

## ZIP 读权限修复

新构建首次下载 ZIP 返回 409：export-service 的非 root 用户无法读取 core-api 以 0600
保存的图片。为 `visualasset.Store` 增加显式 `WithReadGroup`，默认仍私有；core-api 和
course-runner 发布端沿用现有组 20001，目录 0750、文件 0640，无组写或全局读权限。
同字节重登记先校验哈希，再修复旧文件权限；拒绝符号链接和无效组。

本机已部署 core-api `inkwords/core-api:visual-read-v1-20260913` 及前端
`inkwords-frontend:runtime-details-v1-20260913`，保留各自 Compose 覆盖链和全部环境值。
export-service 的 UID 10001、附加组 20001 在只读卷直接读出三图，哈希均一致。真实表单
重登记前后资产、修订、运行收据 JSON 完全相同。course-runner 的发布端接线已改并通过
编译/定向测试，本轮未重建其容器；不宣称自动浏览器采集环境已改变。

前端 79 文件、315 项测试，lint/build 通过；新详情测试先失败再通过，覆盖失败退出码、
原文保留、HTML 不执行及缺失/截断。图像权限测试先失败再通过；相关导出/工件/架构回归、
后端 `go test ./...` 全部通过。最后的 runner 接线变更另经 bootstrap/架构测试通过。
`git diff --check` 通过。

## 新冻结与尚未完成的真实缺口

通过页面冻结 `966e247b-0473-4d54-936c-c1fd30ae92be`，manifest
`sha256:d41ac94c7ca0a7784bcf19387c3d53ba499051ed8abe6da70315694e06195876`。
它保留批准 r15、三份分发声明、同源工件及三图。权限修复后 ZIP 下载成功，24 项内容哈希
和 manifest 校验通过，三张图片与原图逐字节一致。ZIP SHA-256：
`9ca33fdc6fbead0ae468fa297dfb0c224217e98721042311e98bfad20cf08143`。

**图片嵌入尚未完成**：Markdown 只输出 `inkwords:asset` 注释，DOCX 的 `word/media/`
为空，19 页 PDF 图片数为 0。原因是当前 `RenderBookMarkdown` 未将冻结资产编排为实际
图像，DOCX/PDF 也没有装配图像资源。ZIP 携带原图不等于原图已进入教材；不能报告四格式
完整验收。下一步须接入仅按冻结哈希读取的图像资源、补失败测试，再检查实际图片与分页。
这一步必须保持 PDF 网络隔离、原始图片哈希和权利/图片内字体的独立审查边界。

新构建无审阅记录，出版预检 17 项阻断；旧 `66e5f4a4` 的 manifest 与两条 AI 审阅均保留，
不能把旧评分自动移植为新图版面通过。原计划三项综合验收继续未勾选。

范围核对：PR-13 的退出条件为真实终端/浏览器证据、IDE 截图清单及可跟录视频教案；PRD
第 9 节要求操作方案。成片录像不属于当前 V1 必交付项。此前检查点将“录像未完成”并列
为必需待办的表述过宽；继续完成图像编排、教案可跟录和原计划其余验收，不增加成片任务。

原始证据：`output/real-acceptance/2026-09-13/current-media/`；冻结导出、回读和逐项校验：
`output/real-acceptance/2026-09-13/media-build/verification.json`。本报告为 AI 本地执行记录。
