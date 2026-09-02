# Project Course 实验验证隔离决策

状态：首期落地执行器窄接口与 Linux bubblewrap 隔离实现；功能开关默认关闭，未配置可用 sandbox 时 fail-closed。

## 方案比较

| 方案 | 网络/资源隔离 | Compose 与本地兼容 | 首期决策 |
| --- | --- | --- | --- |
| 远程沙箱服务 | 强，依赖外部控制面 | 需要额外凭据和网络 | 暂不采用 |
| rootless 容器 | 强，需正确配置 runtime | Linux/Compose 友好，macOS 依赖 Docker | 作为服务边界 |
| gVisor | 强，运行时与镜像成本较高 | 需要专用 runtime | 暂不作为默认 |
| nsjail | 强，规则复杂、发行版集成成本较高 | 需要额外维护 | 暂不采用 |
| bubblewrap | namespace、禁网和 rlimit 直接可组合 | Linux 容器内轻量，macOS 通过 Docker 使用 | 首期选用 |

## 约束

- 课程实验只能运行系统生成且已验证的工件，不能运行目标仓库。
- 命令必须来自 manifest allowlist；拒绝 shell 链接、重定向、环境变量展开和未允许参数。
- 执行器必须显式提供工作目录、超时、资源限制和空环境；未注入执行器时任务直接失败。
- 生产实现使用独立 `course-runner` 容器中的 bubblewrap，不挂载宿主 `docker.sock`；在非 root、禁网、只读系统工具链和临时 workspace 中执行。
- bubblewrap 参数包含 user/mount/network namespace、CPU 30 秒、地址空间 256 MiB、进程数 64、单文件 10 MiB 和输出 1 MiB 上限。Go module 下载与 checksum 网络均关闭。

## 当前实现

`backend/services/course-runner/domain/verification` 负责合同校验、命令门禁、临时 workspace 复制和执行器接口编排。`BubblewrapExecutor` 只接受固定的 Go 测试模板；服务通过 `PROJECT_COURSE_LAB_VERIFICATION_ENABLED=false` 默认关闭，开启时找不到 `bwrap` 会拒绝启动。
