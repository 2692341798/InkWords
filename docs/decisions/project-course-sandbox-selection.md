# Project Course 实验验证隔离决策

状态：ProjectCourse 已退出当前产品路径；可复用的 bubblewrap 执行器现仅服务于教材教学制品验证，并继续在未配置可用 sandbox 时 fail-closed。

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
- bubblewrap 参数包含 user/mount/network namespace、CPU 30 秒、地址空间 384 MiB、进程数 64、单文件 64 MiB 和输出 1 MiB 上限。Go module 下载与 checksum 网络均关闭；旧 256 MiB/10 MiB 限制在 Go 1.26 编译下实测不足。

## 当前实现

`backend/services/course-runner/domain/verification` 只保留共享的 Bubblewrap 执行器、路径与命令门禁；教材合同、临时 workspace 和执行编排由 `domain/textbookverification` 负责。服务通过 `TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED=false` 默认关闭真实执行，开启时找不到可用的 `bwrap` 会拒绝启动。旧 `PROJECT_COURSE_LAB_VERIFICATION_ENABLED` 配置已移除。

2026-09-06，共享执行器在 Docker Desktop 上通过受审外层 seccomp profile 建立 Bubblewrap
namespace，并在工件进程启动前安装内层 BPF，重新拒绝 mount 与 namespace 调用。该结果
本地学习者固定 Go profile 和教材 Go 执行均已显式启用；真实库尚无批准代码工件，因此
没有教材 RuntimeEvidence。Playwright/Chromium 自身 sandbox 预检仍失败，browser_page
按逐命令结果保持 unverified，不能据此补造截图证据。
