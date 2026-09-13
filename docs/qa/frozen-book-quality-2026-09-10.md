# 冻结质量报告的真实验收

日期：2026-09-10。范围：本地单用户现有 Gin r13 样章与 BookBuild 审校包；不宣称完整开发计划或出版验收完成。

## 实现与验证

新构建固定版本化章节质量快照，core/export 均使用快照做新构建质量预检。审校包汇总固定原报告、导出时自动检查与实际审阅记录，明确自动、真人、delegated_ai 三类来源，不合成评分。实现决策见 `docs/decisions/frozen-book-quality.md`。

共享合同测试覆盖原始警告保留、缺失/过期/失败报告、矛盾 failures、合同不匹配、外层章节替换、未知版本、报告哈希篡改、JSONB 键排序以及大整数精度。真实 PostgreSQL 夹具先冻结通过报告，再在隔离事务内改为失败：旧构建继续使用原报告，新冻结产生不同 ID 并显示失败，旧 manifest 不变，最后事务回滚。没有修改生产章节来制造结果。

使用 Go 1.26.8：共享 textbook、core textbook、export-service 所有相关包回归通过；相关教材/领域/共享边界架构检查通过。初次 exporter 回归有旧占位文案断言失败，改为新版字段及范围断言后通过。完整 `go test ./services` 仍因既有 `backend/pkg/jwt/jwt.go` 未移除而失败（`TestLegacyAuthenticationPackagesAreRemoved`）；未将完整架构或全仓测试标为通过，也未清理该用户遗留文件。无前端改动。最终日志在本目录对应的 output 证据目录。

## 部署与真实页面操作

core-api/export-service 顺序构建和更新为 `quality-snapshot-v1-20260910` 本机镜像，使用既有缓存基础镜像，无新依赖。真实知识库挂载、exporter 非 root/read-only/cap-drop/no-new-privileges/seccomp 以及 course-runner 隔离配置保留。网关实际 GET 成功，服务 healthy。

页面点击“冻结待审构建”产生 `fb19cf7e-b952-4388-90a4-53b57a35acf4`，重复点击 HTTP 201 返回同一 ID/哈希，未重复创建。manifest 为 `sha256:a744d724013b2ed9cf9bc7cd2d461fa7b9b03cca01258ac32fe7f047ae798e76`，input 为 `sha256:ab6774ad4b1ed8ad07e74dd58eeb8d467aca37080048fedd4bc6ede74125518e`。

新快照只有当前批准 r13 `1ee15e29-1304-4978-9e6f-d41421798c8b`，原始 v9 报告与章节记录逐字段一致：passed=true，manual_review_required=true，4 条 advisories 完整保留。自动章节门禁通过但出版预检 false。

本轮显式复核新构建及新增报告后，通过实际页面表单保存 8 条 delegated_ai v1：发展性、技术、自学性、一致性、文字、版式在单章范围通过 3/4；权利需要修改 0/4；读者试学未评估。新构建没有继承旧审批，真人审阅和权利项仍为 0，未晋级。

## 文件与兼容性

最终包：`output/real-acceptance/2026-09-10/book-quality/Gin-r13-质量与运行证据审校包.zip`。
最终部署版本重新下载并复核：实际 601575 bytes，SHA-256 `3f9aa5a0b26c957bc7384733523eb7345a96f0f9b45563718d766172d53b5b81`。

全部 17 文件及 manifest 哈希通过；新报告包含全部 8 条真实审阅，`verification-summary` 的冻结时/导出时检查通过。Canonical AST 的章节、Markdown 和 Go 文件与先前已审 r13 一致。PDF 16 页文本和页面内容流逐页相同；DOCX 所有 ZIP 成员除 core.xml 外一致，后者仅 created/modified 时间变化。本轮依据内容一致性复用此前 PDF 16 页、DOCX 21 页全页视觉 QA，没有冒称再次逐页视觉审阅。新审阅面板截图已实际查看。

旧 b268 构建的 editorial 前后 JSON 完全一致，全部章节 revisions 前后相同。旧构建重新导出仍 `chapter_snapshot_status=unavailable`、null 快照，原 8 条审阅保留；不回填旧 manifest。

证据目录：`output/real-acceptance/2026-09-10/book-quality/`，包括实际 POST 结果、构建 JSON、原始章节记录、审阅前后记录、最终质量报告、文件比较脚本、兼容检查、回归和部署日志；截图 `output/playwright/frozen-quality-reviews.png`。

## 剩余范围

本切片解决质量报告不可追溯的工程缺口；权利清单、独立试学、真实六维学习/延迟保持、多章节/读者档位验收、运行器浏览器能力及原计划其它缺口继续。四条阅读建议仍保留待精修；自动通过和委托 AI 单章审阅不能代表完整教材或出版认证。
