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
| 自动化测试 | 后端全量 Go 测试、前端全量测试、lint、build 和 Compose config 已通过（详见 baseline） |

## 尚未通过或无法执行

- 本机 Docker daemon 未启动，因此没有伪造 `docker compose up -d --build`、容器内实验验证和微服务冒烟结果。
- DeepSeek 真实章节生成需要受控的 API 凭据和配额；当前只验证了生成器合同、证据门禁和任务路由，没有声称完整正文生成成功。
- 三种读者等级的真实生成对比、人工完成三个累积 checkpoint、变式任务和故障排查 dogfood 尚未完成。
- 需要在 Docker 和受控凭据可用后补录完整 ZIP 的自动硬门禁结果；本报告不把“代码路径存在”当作运行时通过。

## 复现命令

```bash
cd backend && GOCACHE=/tmp/inkwords-go-build go test ./...
cd frontend && npm test -- --run
cd frontend && npm run lint
cd frontend && npm run build
```
