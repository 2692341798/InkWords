# Gin 视频教案的同源代码定位（2026-09-13）

本轮完成教案生成器修复与真实编辑器静态验证。批准 r13 保持不变；新教案是本地只读预览，尚未成为新候选修订或正式导出内容。

## 问题与变更

Gin 教案写死 `TestPathOnlyBaseline`、`TestRouteTree`，而实际 r13 包含另一组测试。将教学文件解析提取到 `shared/platform/teachingartifact/ManuscriptGoFiles`，供 core-api 工件派生与 llm-stream 教案共同使用。来源标签、顶层 Go 代码块、文件名、语法与非生产限制规则保持原样。

教案从原字节 Go AST 提取实际类型、函数、接收器、测试声明及文件行号，附母稿与文件 SHA-256。注释中的假测试名不会入选。母稿哈希不符时拒绝生成；教学文件缺失、重复或损坏时保留候选稿，并返回明确的 `source-review-pending` 教案，不编造源码定位或运行通过证据。

## 实际 r13 证据

- 修订：`1ee15e29-1304-4978-9e6f-d41421798c8b`，approved。
- 母稿：`d8c57b17ba1242d4b0f370225a40b273f8df88af60a89a9dc616ee6ce816804d`。
- main.go：`e2940cf8040cd3bc0acb28d9e2250b202fb840fd7ef34d8ffd90a7e74e58eb13`。
- main_test.go：`ad38d95f9fb9d797741fb02cc0199d3a2472c11e1e114d38729f5e501269862a`。
- 实际测试：TestFindRoute:5、TestSharedAPINode:21、TestAPINotRegistered:34、TestMethodSpecific:49、TestRegistrationBaseline:62、TestTeachingSlashBoundary:69。

通过电脑界面打开现有 Visual Studio Code 1.137.0 中的同源 `main_test.go`，使用“转到行”定位 49。AX 显示 `func TestMethodSpecific(t *testing.T) {`，实际屏幕可见第 49–60 行 GET/POST 同路径调用与结果断言。采集者为 delegated_ai；这是静态源码观察，未启动教学程序、IDE 调试器或安装扩展。截图已在会话显示，尚未登记为持久媒体资产。

预览与原字节文件位于 `output/real-acceptance/2026-09-13/runbook-source-binding/`：`r13-runbook-preview.json`、`r13-manuscript.md`、`main.go`、`main_test.go`。`preview.go` 仅调用解析/投影代码，没有执行提取出的教学文件。

## 验证与未完成项

新增测试先因旧签名不支持母稿输入而编译失败；实现后相关包、工件解析回归和架构检查通过。`GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...` 全部通过。实际预览发现并修正 SHA-256 前缀重复问题后，llm-stream 教材包再次全量通过；文件摘要另用独立 SHA-256 复核。`git diff --check` 通过。

未部署新的 llm-stream 镜像，未调用真实模型，未改写批准稿或数据库。后续仍需把新教案经过候选流程落库，在真实工作台复核并登记静态截图；录制步骤还需对应实际输入、预期画面与有效隔离运行证据。IDE 调试证明、冷读者、个人六类学习证据与出版验收均未完成，原计划三项综合验收继续保持未勾选。
