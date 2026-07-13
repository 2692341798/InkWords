# InkWords Project Mastery Course 验收报告

日期：2026-07-13  
验收仓库：`https://github.com/2692341798/InkWords`  
验收 ref：`main`  
锁定 SHA：`f14bd1dbc1e568a2335341dd4df0f6c0574bee35`

## 验收边界

本报告只记录当前工作区能够复现的证据。分析器可以读取锁定快照和生成证据，不执行目标仓库的构建、测试、安装脚本、hook 或服务启动。课程实验若启用，只允许运行系统生成的实验工件，并由独立 `course-runner` 隔离。

## 已验证

| 项目 | 证据 |
| --- | --- |
| 固定 SHA | `git ls-remote ... refs/heads/main` 返回上述 SHA |
| 只读分析夹具 | `go run ./scripts/test_project_course_e2e.go -repository https://github.com/2692341798/InkWords -ref main`：60 个文件、485 个符号、1037 条关系；`executed_target_repository=false` |
| 证据引用 | `EvidenceRef` 校验路径、行号、内容 hash 和快照 SHA |
| 章节合同 | 8 种章节类型有独立 required sections 和硬门禁 |
| 官方资料 | 首期 Go、Gin、React、Zustand、PostgreSQL、RabbitMQ、Redis、Docker Compose、Nginx、TypeScript registry；HTTPS allowlist、私网地址、跨域重定向、响应大小/类型均有测试 |
| 实验隔离 | bubblewrap 默认关闭；启用时禁网、限时、限进程/内存/文件大小，目标仓库不会进入 runner |
| 课程打包 | 仅接受已验证 checkpoint；manifest 锁定 commit、文件 hash 和排序 |
| 阶段事件 | `CourseCheckpoint` 校验 course、blueprint version、input/output hash 和完成状态 |
| 幂等查询索引 | PostgreSQL core migration 增加仅覆盖 `project_course_phase` 的 JSON 表达式索引；事务内临时同构表 `EXPLAIN (COSTS OFF)` 验证查询语法和排序路径。索引增加少量写放大/存储，适用相同输入哈希的结果复用查询 |
| 离线课程验收 | `go test ./services/llm-stream/app/projectcourse -run TestOfflineInkWordsAcceptance -v`：三种读者等级证据/覆盖稳定，夹具章节合同与硬门禁通过，variant manifest 合同通过 |
| 真实分析链路 | Docker Compose 中通过网关创建 `programming` 课程任务；首次运行暴露源码内嵌 `--- File: ... ---` 标记被误解析为路径的问题，修复为仅识别行首文件头后重跑成功，任务进入 `awaiting_approval`，固定 SHA 为 `f14bd1dbc1e568a2335341dd4df0f6c0574bee35`，生成 48 个章节蓝图与覆盖矩阵；覆盖接口显示 47/47 文件已覆盖 |
| 浏览器入口验收 | 内置浏览器打开 `http://127.0.0.1:4173/`：课程入口、仓库/ref/读者等级控件和分析按钮均唯一可操作；输入 239 字符长仓库/ref 后页面无横向溢出 |
| 浏览器课程流程 | 使用受控开发 bypass 入口 `http://127.0.0.1:4174/`：从工作入口进入课程，提交 InkWords 分析，等待 `awaiting_approval` 与固定 SHA，修改标题、交换前两章排序、关闭非核心章节并保存；页面重新显示修改后的蓝图状态。未点击“批准并继续生成”。 |
| 容器集成冒烟 | Colima 调整为 4 CPU/8 GiB 后，全部 7 个应用镜像构建成功；Compose 服务均健康，首页返回 200，`/api/v1/ping` 返回 `pong`，core-api、RabbitMQ、Redis、Postgres 与 worker 就绪。course-runner 已补入 Go 工具链，但默认 Compose 安全策略下 bubblewrap 无法创建 namespace，实验验证仍保持关闭 |
| 本地生成器合同 | `TestJSONChapterGeneratorBuildsEvidenceBoundChapterFromLocalMock`：本地 httptest JSON 响应经过 EvidenceRef、Claim Ledger、章节合同和质量门禁；不代表真实 DeepSeek 生成 |
| 任务编排合同回归 | `TestCourseTaskRunnerCoversEveryChapterContractWithDeferredLabs`：8 类章节合同全部进入 runner；6 类本地生成并通过硬门禁，2 类实验章节因隔离验证未完成而阻塞，所有 checkpoints 通过校验 |
| 硬门禁状态回归 | `TestCourseTaskRunnerDoesNotCompleteChapterAfterHardQualityFailure`：生成器返回不满足章节合同的文档时，章节进入 `blocked`，课程不会错误返回 `completed` |
| 自动化测试 | 后端全量 Go 测试、前端全量测试、lint、build 和 Compose config 已通过（详见 baseline）；路径解析回归测试通过，修复后的 `llm-stream` 镜像已重新构建并运行 |

## 尚未通过或无法执行

- 首次并行 Compose 构建曾触发 Docker daemon EOF；调整 Colima 资源后已用缓存完成全部镜像构建并通过容器健康与网关冒烟。课程实验验证仍按 fail-closed 配置保持关闭，尚未声称真实 sandbox 实验通过。
- 隔离冒烟实际确认 course-runner 以 `uid=100(runner)` 运行、Go 工具链和 bubblewrap 存在，但 Docker 默认安全策略返回 `bwrap: No permissions to create a new namespace`；即使在无卷临时特权容器中，嵌套 bubblewrap 也因 `RTM_NEWADDR` 权限失败。没有把特权配置写入 Compose，因此成熟 sandbox 的运行时验收仍未通过。
- DeepSeek 真实章节生成需要受控的 API 凭据和配额；当前只验证了生成器合同、证据门禁和任务路由，没有声称完整正文生成成功。
- 三种读者等级的真实生成对比、蓝图人工审阅批准、完整正文生成、人工完成三个累积 checkpoint、变式任务和故障排查 dogfood 尚未完成；批准操作会向外部 DeepSeek 服务发送仓库衍生内容，本轮未在未获用户明确授权时执行。
- 浏览器深度交互验收未完成：当前沙箱中的 Playwright Chrome 进程无法启动；已有页面截图和组件测试不替代真实浏览器流程证据。
- 本轮内置浏览器已打开 Docker 网关，但生产前端先显示登录页；注册流程包含图形验证码，未在没有现成测试账号或用户明确授权的情况下绕过/代解。此前开发服务器入口控件检查仍只作为未登录入口证据；移动视口覆盖未能在当前浏览器后端保持，因此未宣称移动端完整验收通过。
- 需要在 Docker 和受控凭据可用后补录完整 ZIP 的自动硬门禁结果；本报告不把“代码路径存在”当作运行时通过。

## 复现命令

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./...
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/projectcourse -run TestOfflineInkWordsAcceptance -v
cd frontend && npm test -- --run
cd frontend && npm run lint
cd frontend && npm run build
```
