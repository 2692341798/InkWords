# 遗留副本整理与全量工程验证

日期：2026-09-10。上一轮完成冻结质量报告后，本轮继续 PR-16 的迁移清理与完整验证，不把工程检查通过视为教材内容验收完成。

## 发现与处理

1. `backend/pkg/jwt/jwt.go` 仍是 HEAD 中的原文件，没有本地改动，也没有应用导入者。现有服务入口、路由、Compose 和 Docker 构建均不引用它；PRD 明确删除 JWT，架构测试要求其不存在。其依赖已经移除，因此它同时阻断 `go test ./...` 和架构检查。保存原文件后删除未调用的旧包，没有重新引入认证依赖。
2. course-runner 的 `app 2`、`cmd 2`、`domain 2` 共 9 个 Go 文件被 Go 自动发现为含空格的非法包路径。8 个与 HEAD 的旧标准路径文件逐字相同；另一个 Bubblewrap 副本有独立差异（文件权限和注释等），没有覆盖到当前运行器。全部按原始路径、字节和权限归档到 Go 模块外，保留独立变体。当前标准路径的运行器代码未变。
3. 前端有 8 个同类副本，包括配置、E2E、CodeBlock 和包体积脚本。其中 7 个与 HEAD 原文件相同，1 个旧 Playwright 配置有独立改动；全部原样归档。没有推定这些副本的产生原因，也没有丢弃独立改动。删除 Knip 对旧配置副本的特例忽略，保留原 UI 组件规则。
4. `mastery.ts` 和 `masteryAssessment.ts` 的 4 个类型只被各自模块内部引用，因此移除 export，仍保留结构定义和所有 API 返回类型；业务行为和网络协议不变。
5. 迁移测试把待迁移数写死为 3，后来加入第 35 号迁移导致失败，且固定次数 Down 会指向错误目标。该夹具改为显式 `UpTo(24)`：仍在有 OAuth 记录时拒绝、删除夹具阻断记录后仅执行尚未完成的清理、验证原数据及 workspace 字段保留，并直接验证该破坏性清理拒绝 Down。没有修改 SQL 或对生产数据库执行迁移。

## 恢复验证

归档目录：`output/real-acceptance/2026-09-10/legacy-cleanup/retained/`。
`archive-manifest.json` 和 `frontend-archive-manifest.json` 记录所有 18 个文件的原路径、保存路径、SHA-256、大小和权限。先复制全部文件、逐项校验，再移出源码路径；JWT 使用显式删除补丁。

已在单独临时目录运行恢复脚本，18 个文件全部恢复到原相对路径并与原 SHA-256 一致。另在隔离夹具中制造后来改动，确认恢复拒绝覆盖。没有在当前工作区重新引入旧代码。

需要回退时，从仓库根目录运行以下命令；它们仅在目标不存在或仍与归档完全一致时继续，否则报错要求保留冲突。恢复会重新引入旧检查阻断，因此不是等价的完成状态。

```bash
python3 output/real-acceptance/2026-09-10/legacy-cleanup/archive_legacy.py --restore
python3 output/real-acceptance/2026-09-10/legacy-cleanup/archive_legacy.py --restore-frontend
```

未提交、推送、修改 Git 元数据、安装依赖或调整沙箱。没有批量清理其余脏工作区。

## 实际检查

使用 Go 1.26.8、`GOCACHE=/tmp/inkwords-go-build`：

| 检查 | 结果与范围 |
| --- | --- |
| `go test ./... -count=1` | 57 个测试包通过，0 FAIL，包含完整架构与真实 PostgreSQL 迁移夹具 |
| `go test -tags=integration ./integration -count=1` | 通过；此目录是自包含合同测试，不等于真实 Provider、Obsidian 或 MQ 全链路 |
| `npm run lint` | 通过，E2E 夹具最终修改后再检通过 |
| `npm run deadcode` | 通过，没有新增忽略规则 |
| `npm run test:coverage` | 74 文件、289 测试通过；语句/行 58.83%，分支 77.32%，函数 62.80%；通过现有门槛不代表全路径覆盖 |
| `npm run build` | 通过 |
| `npm run check:bundle` | 通过；入口 JS raw 433334 bytes、gzip 137702 bytes |
| `npm run test:e2e` | Chromium 核心 8 条通过；浏览器真实运行、API 为路由夹具，不写真实审阅或学习记录 |
| Compose config | 使用真实知识库挂载和当前镜像叠加配置校验通过，仅输出去敏后的运行设置 |
| `git diff --check` | 通过 |

首次核心 E2E 有 3 个失败，保留原日志：出版工作台标题从“人工证据”改为“审阅证据”，其余两个夹具漏掉新增验证状态/预检 GET，实际网络 trace 定位到 404。修正标题断言、补齐明确的 GET 响应：无验证任务返回 null，未启动隔离运行器的夹具声明 available=false。未移除、忽略或放宽“浏览器零错误”断言；修复后全部 8 条通过。

## 本机事实和剩余项

真实网关 GET 当前构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4` 的 editorial，响应与前轮已经验证的完整记录一致；服务仍 healthy。构建、批准母稿、8 项 delegated_ai、权利项为 0 和 preflight=false 的事实均未变化。此次只清理未调用文件、类型导出与测试，现有运行镜像无需重启。

证据均在 `output/real-acceptance/2026-09-10/legacy-cleanup/`：初次失败、最终全量后端与前端日志、核心 E2E 修复日志、归档比对、恢复验证、真实 editorial 和 Compose 检查。完整开发目标仍未达成：独立读者试学、实际学习闭环/延迟保持、权利依据、多读者与整本教材验收，以及运行器浏览器能力等仍需单独推进。上述绿灯不能替代这些证据。
