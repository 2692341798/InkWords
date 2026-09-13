# r9 冻结引用、真实导出与学习入口验收

日期：2026-09-09 至 2026-09-10（Asia/Singapore）。执行者：Codex，用户授权 AI 审阅与电脑操作。本记录不代表真人试学、出版社审校或出版认证。

## 已完成

1. 承接用户委托审阅的 r9 批准母稿，浏览器先创建构建 `a7b37aac-2f26-4c4b-a83c-ae3785a8df8e`。真实导出发现内部引用标记泄漏、DOCX 表格错位和字体回退，保留失败样本后修复投影。
2. Canonical Book AST v2 冻结 8 份实际引用来源。浏览器创建新构建 `91655bf4-be54-4805-b88a-0e71fba9b0c2`，manifest hash `sha256:570b173926cac810e687cd5fa8e03cfc7dbf5c14cc936a5a0b7f786aab5242a0`。批准修订仍为 `30a83452-af8b-42cb-b254-3c7f37ce2443`，正文哈希 `c046f6ac963fe0e3c1d101b7c8e216e7a056cc9b224642d628588d9cb7bc1f79`，旧构建未改写。
3. 新 Markdown/DOCX 经真实网关导出 HTTP 200，内部 `[evidence:...]` 残留为 0。来源脚注保留固定 Gin SHA、文件、符号和行号。代码正文未改写；模板补齐 Pandoc 表格、脚注、代码样式，代码使用镜像现有 FreeMono。
4. 用已配置中文及等宽字体的 LibreOffice 渲染最终实际 DOCX，共 19 页，逐页查看 1–19。表格列、中文、代码和脚注可读，未见文字裁切、空白页或原先表格错位。此 PDF 是 DOCX 校样的本地 QA 产物，不是 export-service PDF 成功的证据。
5. 实际点击“从批准教材创建学习任务”，创建目标 `dc715aea-7fbb-4078-b70b-8097d4dc76de`，绑定 r9 和 explain/complete/reproduce/transfer/diagnose/retain 六维题目。页面打开独立复述题目和评分依据；GET 读回作答数 0，没有把 AI 操作写成用户掌握。

产物在 `output/real-acceptance/2026-09-09/delegated-review-v1/exports-citations/`：

| 文件 | 真实结果 |
| --- | --- |
| `Gin-样章-r9-引用校样.md` | 31,227 bytes；SHA-256 `01c7e9a63ad2f3af39c2c42bfbcbcb62a0a945b5409debcd22e2650edd491ba7` |
| `Gin-样章-r9-校样.docx` | 41,445 bytes；SHA-256 `0d6de5b8383281088868cea578d1237cbf8abbb568be1115a9b0321ea412e9cc` |
| `final-render/page-1.png` 至 `page-19.png` | 最终实际 DOCX 逐页校样 |
| `final-result.json` | 文件哈希、渲染方式和服务失败状态 |

## 实际测试与部署

- 共享教材合同、core-api 全服务包、export-service 全服务包、llm-stream 教材工作器回归通过，见 `book-citations-regression-v2.log`。初次命令误用了不存在的 `core-api/app/textbook`，更正为实际包路径后通过；未把初次失败当通过。
- PostgreSQL 真实容器测试通过，覆盖引用冻结、缺失别名、跨项目拒绝、仅官方资料引用和旧 manifest 不变；EXPLAIN 使用项目/快照/文档/片段现有索引，没有新增索引或迁移。初次测试缺少 official_confirmed fixture 字段触发数据库约束，补齐测试数据后通过，保留两份日志。
- Pandoc 实际 DOCX 测试检查 OOXML 表格、Table/Compact 样式、脚注关系和固定 SHA 链接；最终模板聚焦回归通过。代码/转义中的标记不被替换；缺失冻结引用和来源身份不匹配有失败测试。
- 关键架构边界与 `git diff --check` 通过。本轮较早的前端 72 文件 / 279 测试、lint、build 已通过；此导出切片没有再改前端。
- 全量 `go test ./...` 仍有本轮之前已存在的 JWT 遗留包依赖和 `course-runner/app 2`、`cmd 2`、`domain 2` 目录阻碍；没有删除既有工作来隐藏失败。
- 部署镜像：core-api `book-citations-20260910`；export-service `book-proof-20260910`；llm-stream `delegated-review-20260909`；frontend `delegated-approval-20260909`。采用已安装 Go 1.26.8 交叉编译与既有本地运行镜像，顺序构建，保留原 tag。
- Compose 使用真实知识库路径，core/export 健康，外部网关实际导出通过。对应覆盖文件 `runtime-build/compose.proof.yml`。未提交、推送或安装依赖。

## AI 审阅决定与未完成项

接受本切片的“冻结引用与可编辑 DOCX 导出”行为，保留构建 `ready_for_review`。实际服务端 5 项自动检查均通过，包括冻结母稿质量与当前代码工件验证；完整出版预检仍不通过。

- PDF 与审阅 ZIP 实际返回 503 `TEXTBOOK_BOOK_PDF_UNAVAILABLE`。容器内 Chromium 正常沙箱探测退出 133，namespace 创建返回 Operation not permitted。没有增加 `--no-sandbox`、特权配置或宿主执行回退。
- 版面仅完成 AI 校样复核。正式出版仍需页码、目录、跨页代码/练习排版，以及脚注编号独占行等细节精修；未登记“人工版式审校完成”。
- 两个权利主体（批准正文、教学代码）尚无完整权利记录；不把“官方公开”或 AI 生成自动认定为出版许可。
- 出版审核 API 当前只支持 `automated=false` 的人工记录。用户授权 AI 审阅已生效，但不能用该接口把 AI 审閱冒充真人；后续需独立委托审阅来源合同并保留真实出版/读者条件。
- 学习者作答、运行评分应用、独立迁移与至少 24 小时延迟保持尚未发生。学习入口已真实打通，不等于六维闭环完成。
- 三类 audience、实际截图/视频以及全部 PR-14/PR-16 要求继续按原计划核对，未将本切片冒充整个计划完成。

恢复点：从新构建与上述学习目标继续。先解决剩余操作验收的真实来源标记、出版委托审阅合同和运行环境，不重跑已通过测试来代替缺失证据。
