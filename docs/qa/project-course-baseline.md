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
