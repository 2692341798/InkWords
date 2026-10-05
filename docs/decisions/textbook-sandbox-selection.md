# ADR：教材运行证据的隔离执行选择

- 状态：接受；PR-12 分阶段实施中
- 日期：2026-09-03
- 关联：`docs/superpowers/plans/2026-09-02-textbook-platform-implementation.md` PR-12、PR-13

## 决策

### 2026-09-06 教学工件读取与运行版本

真实 r7 运行发现 core-api 的私有临时目录/文件权限不能被非 root course-runner 读取。
教学工件发布器现为内容 hash 校验通过的工件设置专用 GID 20001，目录 0750、文件 0640；
course-runner 仅加入该组，原 UID、只读文件系统/工件挂载、cap_drop、seccomp、禁网隔离
均保留。默认 Store 仍私有，Provider 拒绝稿存储不使用该组。按原 task/revision/content
hash 执行 textbook-artifact-reconcile 可修复既有工件的读取权限，不修改字节或母稿。

`TEXTBOOK_TEACHING_GO_VERSION` 指定运行器镜像内的精确版本（本次实际为 go1.26.8）。
go.mod 的 go 1.26.0 只表示最低语言版本，不能充当实际运行版本。v2 投影在 go.mod 添加
操作方指定的 toolchain 指令，并将版本纳入工件 ID 和命令清单；main.go/main_test.go
字节不变，构建元数据变化形成新代码树 hash，保留旧清单及其过期证据。运行结果继续
记录执行器实际版本；不匹配仍判为 stale，不放松判断。CLI `-toolchain go1.26.8` 可为
已有候选创建新投影；省略该参数仅恢复旧 v1 清单。

失败的 textbook_teaching_artifact_verify 任务现在可经既有任务 retry API 重试。
服务端先验证冻结工件身份/hash，再将完全相同的 payload 发到独立验证队列；不会转成
生成任务，也不能由客户端提供新路径、命令或运行参数。

InkWords 只在 Linux 主机的 Bubblewrap 预检完整通过时，执行系统生成、内容 hash 匹配的教学工件。默认关闭执行；macOS、Windows、Android、iOS 以及任意预检失败环境均返回 `unverified`，不得为了展示“运行成功”而退回宿主机执行、Docker socket、`privileged` 容器或目标仓库目录。

现有 `course-runner` 的旧 ProjectCourse 实验执行器不能直接当作教材执行器：其 manifest、artifact token 和持久化边界均不同。教材链路将以独立适配器读取 `textbook_code_artifacts`，验证 artifact/revision/input hash 后才把临时副本交给通用隔离执行边界。

## 约束

- 只接受由核心服务创建的 `teaching_implementation`；上游源码导读、用户目录和 Git 检出永远不可执行。
- 命令只能由语言模板生成；Go V1 仅允许固定的 `go test ./...` 与受限 `-run` 模式，不接受 shell 字符串、重定向、管道、环境注入或网络下载。
- Bubblewrap 必须使用新 namespace、禁网、非 root、只读系统目录、独立临时目录、CPU、内存、PID、文件大小与输出大小上限。内容寻址工件会先复制到临时目录，但在 namespace 内仍以只读方式挂载；唯一可写位置是 `/tmp` tmpfs，不会回写任何源工件。
- 只有带 artifact/input hash、命令清单、runner image digest、工具链版本、原始输出引用和采集时间的结果才允许为 `verified`。合同、代码、命令、工具链或运行器变化改变 input hash，旧结果立即不可作为当前证据。
- 浏览器页面证据必须来自受控教学页；IDE 画面始终由人按 Runbook 采集，附工具版本、采样条件、原始文件与解释边界，不能被自动化能力冒充。
- `browser_page` 只有在同一条结构化观察中保存本地页 URL 与最终 URL、实际 Playwright/Chromium 版本、内容寻址截图引用、至少一条 DOM 断言、显式 console 收集结果和本地网络摘要时才能标为 `verified`。course-runner 已实现受控 Playwright 执行器：它只在通过 Bubblewrap 预检的环境中启动 loopback 教学页并阻断非本地请求；当前 Docker Desktop 无法创建所需 user namespace，因此执行器仍 fail-closed，不能把合同测试当作真实浏览器运行证据。

## 平台策略

| 平台 | 当前行为 | 原因 |
| --- | --- | --- |
| Linux + 通过 Bubblewrap 预检 | 可显式启用 | 可建立最小权限 namespace 隔离。 |
| Linux + 预检失败 | `unverified` | 不扩大容器权限来绕过用户 namespace 或挂载限制。 |
| macOS / Windows | `unverified` | 本地没有与当前 Bubblewrap 合同等价、已验证的替代执行器。 |

## 验证标准

启用前必须以受控恶意 fixture 实测：网络不可达、目标仓库不可见、符号链接逃逸被拒绝、资源上限命中、输出截断被标识。Docker Compose 语法检查、单元测试和宿主机 `go test` 均不能替代该隔离证明。当前开发机若无法完成该实测，UI 应准确显示“待人工采集 / 未验证”。

## 当前开关与可观测行为

- `TEXTBOOK_TEACHING_ARTIFACT_VERIFICATION_ENABLED` 默认 `false`，且必须同时提供给 `core-api` 与 `course-runner`；关闭时 core-api 拒绝创建验证任务，避免产生没有消费者的排队任务。
- 启用时 `course-runner` 还必须找到 `bwrap`，并以不包含教材工件的最小 namespace、非 root、只读系统目录和私有 `/tmp` 预检通过。缺失或预检失败时服务启动失败；它不会退回到宿主机或普通容器执行。
- `TEXTBOOK_RUNNER_IMAGE_DIGEST` 必须是实际 runner 镜像的完整 SHA-256 摘要。缺失或不合法时，执行结果仍为 `unverified`，不会形成可声明为已验证的证据。
- 教材请求走 `textbook.verification.requested`，与旧 ProjectCourse 的 `course.verification.requested` 分离；消息只含 artifact/revision/hash/manifest hash，worker 会重新从数据库和内容寻址卷加载并核对这些值。
- `teaching-artifacts` 卷由 core-api 写入、course-runner 只读挂载。每次运行都会追加一条运行证据；只有通过时才设置 30 天有效期，其他结果保留失败或不可用原因。

## 后果

### 缺失工件的单任务恢复（2026-09-06）

`GetGeneratedRevisionContext` 与批准合同读取统一返回带 `sha256:` 的哈希，数据库仍保存
原格式，不做回填或 schema 变更。core-api 镜像内的 `textbook-artifact-reconcile` 默认
只预览，必须同时提供原生成 task、预期 candidate revision 和原正文 hash；不匹配即拒绝。
只有追加 `-apply` 才按已有生成工件服务登记。它不重写任务终态或正文、不取得编辑锁、
不批准、不运行代码，也不调用 Provider。重复执行使用同一内容寻址树和工件 ID。

```bash
docker compose --env-file backend/.env exec -T core-api /app/textbook-artifact-reconcile \
  -task <原任务UUID> -revision <原候选UUID> -content-hash <原稿64位SHA256>
# 核对预览后，同一命令追加 -apply。
```

该工具用于已保存候选的派生工件恢复，不能绕过人工批准和运行器的后续准入。回退程序
保留工件卷及数据库记录即可，不通过删除数据或重置任务持久化标记来回退。

### 母稿到代码工件的同源约束（2026-09-06）

样章保存后的投影必须读取该生成候选 revision 的 Markdown 并校验 content hash，不能
用内置示例替代其代码。当前 Gin 样章只接收母稿中两个顶层 Go 围栏，前一段必须明确标识
教学实现 `main.go` 或教学测试 `main_test.go`；来源、文件名、重复、包名、函数声明、
语法或非生产用途/省略说明不满足时不登记工件，已保存候选仍保留且未验证。

Goldmark 解析围栏，源文件与测试保留母稿原字节；工件 ID 绑定 revision/content hash，
内容寻址树绑定实际文件。投影只补固定 module 的 go.mod（最低 Go 1.26.0）并使用固定
go_test 命令，不读取正文 shell 指令或补造浏览器页面。源代码的编译和运行仍只在批准的
course-runner 沙箱内进行；静态提取不形成 RuntimeEvidence。生产运行器使用同一构建阶段
复制的 Go 工具链，报告版本取运行器的 runtime.Version，不能抄录 manifest 中的版本声明。

旧的固定 Gin 登记示例及浏览器页面已从生产投影移除。历史记录保持原样，不将旧工件重新
绑定到新母稿；真实库现存工件数量须在部署时核对。将来支持其他章节布局或浏览器投影时，
需要显式的母稿文件/运行命令合同，不能重新引入与正文无关的固定示例。

本决策牺牲了跨平台“立即运行”的便利性，换取不把不受信任代码升级为本机执行权限。未来若引入 macOS 等价隔离器，必须独立评审其文件、网络、进程与资源边界，并添加同等的恶意 fixture 证据后才能将平台状态改为可用。

## 2026-09-05 本机 Docker Desktop 复核

当前本机 `docker context` 为 `desktop-linux`，Engine 29.3.1 的 SecurityOptions 只报告
`seccomp` 与 `cgroupns`，没有 user namespace 或 Enhanced Container Isolation（ECI）。
Docker 官方说明确认：macOS 上普通容器运行在 Docker Desktop 的 Linux VM 内，这形成
宿主机边界；但只有 Docker Business 提供的 ECI 才会为每个容器自动建立专用 Linux
user namespace，并加强对 Docker daemon、VM 和 bind mount 的隔离。因此，普通的
`docker run --network none --read-only --cap-drop ALL` 虽然是有价值的纵深防御，仍不能
在本项目中冒充当前 Bubblewrap 合同的等价验收。

本机只允许从以下三条路线中显式选择，不自动降级：

1. 保持当前 fail-closed 行为：教材代码继续标为 `unverified`，不影响阅读和人工编辑；
2. 用户已有或明确选择 Docker Business ECI 后，另行实现“受信任控制面 + 无网络一次性
   执行容器”，业务服务仍不得挂载 Docker socket，并重新运行全部恶意 fixture；
3. 经用户明确授权安装后，评估独立本地 VM 执行器；该 VM 只接收内容寻址教学工件，
   不挂载仓库、知识库、凭据或业务数据库，并同样通过网络、文件逃逸和资源上限测试。

来源：

- [Docker Desktop for Mac permission requirements](https://docs.docker.com/desktop/setup/install/mac-permission-requirements/)
- [Enhanced Container Isolation](https://docs.docker.com/enterprise/security/hardened-desktop/enhanced-container-isolation/)
- [Container security FAQ](https://docs.docker.com/security/faqs/containers/)
- [Docker run resource constraints](https://docs.docker.com/engine/containers/run/)
- [Read-only bind mounts](https://docs.docker.com/engine/storage/bind-mounts/)

## 2026-09-06 Bubblewrap 可用性修正

进一步用一次性固定命令定位后，Docker Desktop 上非特权 user namespace 实际可用；旧
预检失败是容器默认 seccomp 在 Bubblewrap 建立内层 namespace 前拒绝了 `unshare`。
course-runner 现使用 `services/course-runner/seccomp/bubblewrap-outer.json`：它从 Moby
默认 profile 派生，只额外允许 Bubblewrap 建立沙箱所需的五类调用。Bubblewrap 随后在
启动工件进程前装入内层 classic BPF，重新拒绝 mount、pivot_root、setns、umount2、
unshare、clone3 和携带 namespace 标志的 clone。容器自身仍为 read-only rootfs、
cap-drop all、no-new-privileges、非 privileged，且没有 Docker socket。

生产执行器的固定 Go 自检已经在该环境编译并通过，同时证明二次
`unshare(CLONE_NEWUSER)` 返回 EPERM。这个结果纠正了上节“本机不能创建 namespace”的
诊断，但当前只显式启用了独立的学习者 Go profile。教学工件开关仍为 false，浏览器页、
截图、资源耗尽和文件逃逸等教材恶意 fixture 未完成前，教材 RuntimeEvidence 继续保持
`unverified`。

同日继续检查真实 Playwright 最终启动参数后发现，未显式设置 `chromiumSandbox: true`
时，Playwright 会为当前容器镜像加入 `--no-sandbox`；仅在 probe 源码中“没有写这个参数”
不能证明 Chromium sandbox 生效。probe 现强制启用 Chromium sandbox，补齐冷缓存 Go 页面
服务器的固定离线环境，并成为教材验证启动预检。临时打开教材开关时，页面服务器启动后
Chromium 报告 sandbox 不可用。执行器因此不注入 browser_page 能力；已通过预检的 Go
命令会先通过生产 `Execute` 路径的固定 Go 编译/测试预检，再可按 manifest 顺序执行并保存
verified 结果；遇到 browser_page 时保存 unverified，
工件整体保持 unverified。当前本机已在 core-api 和 course-runner 显式启用教材 Go 执行，
两个服务 healthy；没有批准代码工件可运行，也没有截图或 RuntimeEvidence 被保存。
browser_page 继续 fail-closed。

后续用固定操作方页面分别去除 Go 专用内层 BPF、尝试私有 `/proc`、空
`/proc` 和只读绑定容器 `/proc`。去除内层 BPF 仍不能启动 Chromium；私有
`/proc` 挂载被当前非特权 Docker Desktop 拒绝，空 `/proc` 缺少
`/proc/self/uid_map`，只读绑定则无法写入 `uid_map`。这三条路径最终都使
Chromium 报告 `No usable sandbox`，证明卡点是嵌套 sandbox 建立所需的 proc/userns 边界，
不是 Playwright 启动参数或 Go 专用 BPF 误伤。固定 probe 现将该类错误稳定归类为
`chromium_sandbox_unavailable`，详细采集与人工媒体验收步骤见
`docs/runbooks/textbook-browser-video-evidence.md`。

外层策略基于 [Moby 默认 seccomp profile](https://github.com/moby/profiles/blob/836ae4d37ef2ec995c77c99fc55f5b5f3af3a897/seccomp/default.json)，其来源版本、原始及派生摘要和许可证记录在 `backend/services/course-runner/seccomp/README.md`。Docker 的 seccomp 行为依据 [Docker seccomp 文档](https://docs.docker.com/engine/security/seccomp/)。
