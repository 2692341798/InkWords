# Project Course 实验验证隔离决策

状态：首期只落地执行器窄接口和安全门禁，暂不把宿主机 shell 作为沙箱。

## 约束

- 课程实验只能运行系统生成且已验证的工件，不能运行目标仓库。
- 命令必须来自 manifest allowlist；拒绝 shell 链接、重定向、环境变量展开和未允许参数。
- 执行器必须显式提供工作目录、超时、资源限制和空环境；未注入执行器时任务直接失败。
- 后续生产实现需要成熟隔离运行时（独立容器或等价沙箱），并在非 root、禁网、只读基础镜像和临时 workspace 中执行。

## 当前实现

`backend/services/course-runner/domain/verification` 只负责合同校验、命令门禁和执行器接口编排。它不会自行调用 `os/exec`，避免误把 core-api 或 llm-stream 容器当成实验沙箱。
