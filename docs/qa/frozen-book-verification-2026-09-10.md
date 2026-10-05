# 冻结运行证据与真实审校包验收

日期：2026-09-10。继续执行原开发计划；审阅来源为用户委托 AI。

## 已实现与部署

新 BookBuild 冻结完整教学清单和现存运行收据，core-api 与 export-service 使用共享纯合同检查每个命令的覆盖、修订/代码/清单/输入绑定、工具链、运行器和到期时间。审校包增加版本化摘要，分别显示冻结时与导出时的判断。旧构建不回填；新构建不自动继承旧审阅。

通过原有本地 Compose 覆盖层顺序部署：

- `inkwords/core-api:verification-snapshot-v1-20260910`
- `inkwords/export-service:verification-snapshot-v1-20260910`

两个服务均 healthy，外部 Nginx 项目接口成功。真实 Obsidian 挂载保持 `/Users/huangqijun/Documents/obsidian_knowledge/knowledge`。exporter 保留 UID/GID 10001、只读根、cap_drop ALL、no-new-privileges 与既有 Chromium seccomp；course-runner 未重建或改动。没有新增依赖、数据库迁移、索引或 Provider 调用。

决策与恢复边界见 `docs/decisions/frozen-book-verification.md`。证据目录为 `output/real-acceptance/2026-09-10/book-verification/`。

## 回归证据

先运行新增合同测试，因缺失实现得到预期编译失败，随后实现通过。最终实际运行：

```bash
GOTOOLCHAIN=go1.26.8 GOCACHE=/tmp/inkwords-go-build go test ./shared/kernel/textbook ./services/core-api/domain/textbook ./services/export-service/... -count=1
GOTOOLCHAIN=go1.26.8 GOCACHE=/tmp/inkwords-go-build go test ./services -run 'Test.*(Peer|Runtime|Textbook)' -count=1
git diff --check
```

均通过。没有前端代码修改，没有重复跑无关前端或全仓库测试，不能据此称完整仓库全绿。

共享合同测试覆盖过期、缺失收据、来源修订错配、输入哈希错配、清单哈希错配、失败状态、缺到期时间、未来采集时间、命令模式不匹配、合同改变、错误证据类型，以及终端成功不能覆盖浏览器命令。浏览器观察必须匹配本地页面路径和文本断言；快照读取覆盖旧版缺字段、null/未知/畸形版本、遗漏或异源工件。exporter 测试覆盖摘要身份、完整快照保留、日期判断和旧版 unavailable。

真实 PostgreSQL Testcontainers 夹具证明：在构建冻结后增加成功收据，旧构建的代码预检仍失败；重新冻结产生新 ID 且有完整收据；重复冻结返回同一 ID；新构建没有旧审阅/权利记录。事务内改变源收据后，已冻结 JSON 和其代码判断不变，随后回滚夹具。两小时后的判断失败但快照字节不变。新查询 EXPLAIN 使用现有 `idx_textbook_runtime_evidence_artifact` Index Scan 加 Sort（夹具一条记录），没有新增索引或写放大。

## 真实页面与导出

使用当前本机页面“冻结待审构建”按钮，POST 返回 201：

- 新构建：`b2687220-106a-4ddd-bc8d-b31dfd18e157`，ready_for_review。
- manifest：`sha256:ced4b0fc44dded5c28ec4b6245eac23d60d9b8d75689f7e4b1d188ac5b053254`。
- input：`sha256:ebde1bfffcf12460f3ff81e50bf5341edb2046dd4277881f156d09493d1edfe9`。
- 母稿仍为 r13 `1ee15e29-1304-4978-9e6f-d41421798c8b`，没有新母稿修订。

第二次真实点击返回同一构建、input 和 manifest，证明本次运行的幂等行为，不只依赖单元测试。

真实 GET 导出 ZIP，包内 `inkwords.book-verification-summary.v1` 的 snapshot 与数据库冻结 manifest 对应字段完全一致。包含工件 `e769de34-dfbc-5edd-b040-b7517c310d31` 的一条收据 `b97512de-6cee-4796-81c9-b3e80c53e3a1`，完整原始结构化观察、输入哈希、命令清单、Go 1.26.8/Bubblewrap 与到期日均在包内；`at_freeze.passed` 与本次 `at_export.passed` 为 true。只证明冻结清单中的 go_test，不证明本轮又执行过代码、Mac 安装或 go run 输出。

正文、Markdown 与代码文件均与此前 r13 完全一致；实际新 PDF 的 16 页文本和页面内容流与已逐页审阅校样一致，DOCX 内仅 `docProps/core.xml` 时间元数据变化。此切片没有重新声称逐页人工观察新图像，而是以明确的内容一致性核对复用原版面证据。旧构建 `70aadfa4-32d1-4c87-b235-e76ca5678197` 的 API 状态与八条审阅操作前后完全一致，旧构建实际导出仍为 unavailable 摘要，未回填。

基于上述新快照与同源投影核对，通过页面重新记录八项 delegated_ai v1 审阅，八次 POST 均 201；记录为新 ID，并在说明中明确此次重新核对范围。六项限当前单章通过 3/4，权利需要修改，独立试学尚未评估。新旧构建各自保留记录，没有数据库迁移式复制审批。截图 `output/playwright/frozen-verification-reviews.png` 已查看。

记录后再次导出 `Gin-r13-含运行证据与审阅记录.zip`：596,180 字节，SHA256 `0339af57bbf0d2d98fe1cdbdb5552c1ae86fd5bcc8ca46fde49caee11eb307f9`。全部 17 个文件通过 ZIP CRC、内容清单哈希与 manifest 校验，包内八条审阅 ID 与 API 一致。详见 `final-facts.json`、`verification-summary.json`、浏览器操作与部署日志。

## 剩余范围

新构建真人审阅 0、权利记录 0、preflight false，未晋级出版候选。权利清单、独立读者/真实学习证据、其它 audience、整书质量报告快照、课程媒体与验证生命周期等原计划项继续。此次解决的是离线包缺少固定运行证据的问题，不能据此宣布完整计划完成。
