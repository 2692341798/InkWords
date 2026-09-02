# Project Mastery Course 基线

日期：2026-07-13
验收仓库：`https://github.com/2692341798/InkWords`
本地基线 commit：`8bc4bd4038d1525477157002b12227680d287a07`

## 范围

这是 Phase 0 的脱敏离线基线，不执行目标仓库的构建、测试、安装脚本或 hook。评测夹具位于
`backend/services/llm-stream/app/projectcourse/testdata/inkwords-fixture/manifest.json`。

## 当前生成链路观察

- 现有 Git 分析入口仍以文件清单和正文作为生成输入，Project Course 尚未接入旧链路。
- 当前文章生成和系列生成仍复用 generation task、RabbitMQ、SSE 与 blogs 持久化。
- 夹具覆盖 Go route/handler/service、RabbitMQ consumer、React hook/store、Compose、README、测试和脚本角色。
- 旧路径、符号存在性、固定 commit 内容 hash 和三类证据置信度已固化为 `known-failures.json` 断言。

## 回归测试

执行命令：

```bash
cd backend && go test ./services/llm-stream/... ./services/core-api/... ./services/export-service/...
cd frontend && npm test -- --run
```

结果：后端目标服务测试通过；Go 进程结束时尝试清理用户级 build cache 被沙箱拒绝，但不影响测试结果。前端全量测试通过：45 个测试文件、169 个测试。前端生产构建通过；Vite 报告既有的大 chunk 警告，未因本次改动新增处理。

## 后续补充

Task 2 完成后补充固定 SHA 解析结果、确定性清单文件数/角色分布、预算行为和大文件处置统计；Task 3 完成后补充知识图关键事实覆盖率。

## 当前实现增量（2026-07-13）

- 创建课程只提交仓库 URL、ref 和读者等级；分析任务负责解析并绑定 40 位 commit SHA。
- 蓝图批准后生成任务绑定 course ID、蓝图版本和 SHA；章节结果经 EvidenceRef、Claim Ledger 和质量门禁后才归并。
- 课程实验默认由 `PROJECT_COURSE_LAB_VERIFICATION_ENABLED=false` 阻断；开启时仅允许 bubblewrap 固定 Go 测试模板。
- 课程 ZIP 只接受已验证 checkpoint，包含 README、starter、checkpoints、hints、solution、tests、coverage 和文件 hash manifest。
- 最近验证：后端相关服务全量 Go 测试通过；前端全量测试 46 个测试文件、171 个测试通过；生产构建通过；Compose 配置在提供 `OBSIDIAN_VAULT_PATH` 后通过。
- 只读验收脚本锁定 `main` 为 `f14bd1dbc1e568a2335341dd4df0f6c0574bee35`，得到 60 个文件、485 个符号和 1037 条关系；脚本明确不执行目标仓库。
