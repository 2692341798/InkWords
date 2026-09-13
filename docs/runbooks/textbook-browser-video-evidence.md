# 教材浏览器截图与视频证据 Runbook

本手册用于对已批准教材修订的教学工件采集浏览器截图、IDE 画面和副屏讲解视频。自动浏览器证据和人工录制是两类记录：前者由 course-runner 生成结构化 `RuntimeEvidence`，后者由操作者核对并保留原始媒体。任何一方都不能替代另一方。

## 1. 开始条件

只有同时满足下列条件才开始采集：

- 章节已有当前 `approved revision`，候选稿或旧修订不可用于验收。
- 教学工件的 revision ID、artifact ID、artifact SHA-256、manifest SHA-256 和 runner image digest 已记录。
- 只运行内容寻址的生成教学工件；不运行导入的上游仓库、用户目录或任意 shell 命令。
- 浏览器自动采集必须保留 Chromium 自身 sandbox；禁止 `--no-sandbox`、特权容器、Docker socket 或宿主用户目录挂载。

## 2. 自动浏览器预检

以真实 vault 路径和本地环境文件查看服务状态：

```bash
OBSIDIAN_VAULT_PATH=/absolute/path/to/vault \
  docker compose --env-file backend/.env ps course-runner

docker logs --since 10m inkwords-course-runner-1 2>&1 | \
  rg 'browser-page verification|chromium_sandbox_unavailable'
```

只有固定本地页面预检成功后，`browser_page` 执行器才会注入。当日志出现 `chromium_sandbox_unavailable` 时，自动截图验收立即停止，结果保持 `unverified`。人工录屏可以继续，但必须标记为人工观察，不得填充自动 `RuntimeEvidence`。

## 3. 人工截图和录屏

1. 从章节工作台读取当前 `VideoRunbookProjection`，确认格式、技术栈、观察目标、推荐工具和每一步的完成信号。
2. 记录操作系统、IDE/浏览器完整版本、窗口尺寸、缩放比例、字体和采集时间。
3. 从 Runbook 声明的 `StartState` 开始，按 `Action` 和 `ShortcutOrMenu` 执行；屏幕上同时保留当前源码、运行配置和观察窗口。
4. 每次到达 `CapturePoint` 时截取一张原始 PNG；视频旁白只陈述画面直接支持的事实，其他机制解释标为来源说明或有边界的推断。
5. 如果预期画面没有出现，先完成该步 `Recovery`，把失败及恢复一起保留；不用历史日志、旧截图或另一个工件补画面。
6. 到达 `CompletionSignal` 后结束该步，然后开始下一步。完成后保留原始录屏，剪辑版必须另存并记录所有裁剪、加速和遮挡。

登记截图时，必须在“截图关联运行证据”中明确选定记录，并核对修订号、工件 ID 与代码指纹。
系统按所选记录绑定修订，不按列表顺序猜测。历史修订记录可用于归档，但不能作为当前批准稿
的验收证据；上传图片本身不会把运行状态或权利状态改为通过。页面刷新后需重新选择。

批准动作会生成新的批准修订，工件可以保留在其直接来源候选上。遇到这种情况，先核对批准稿的
`parent_revision_id`、两份正文及教学文件哈希，并确认页面标为“当前稿件的代码来源”；登记时
保留运行收据原本的候选修订 ID，在来源说明中同时记录批准修订。不要把无关历史记录改成当前
证据，也不要仅因批准修订没有单独工件就重复运行。已有冻结包不会自动包含后来登记的图片。

## 4. 必须保留的记录

每个截图或录屏至少附带下列字段：

```text
project_id:
chapter_id:
approved_revision_id:
approved_revision_content_hash:
code_artifact_id:
code_artifact_sha256:
command_manifest_sha256:
runner_image_digest:
video_runbook_format:
step_number:
tool_and_version:
operating_system:
sampling_conditions:
observed_fact:
explanation_boundary:
captured_at:
original_media_path:
original_media_sha256:
edited_media_path:
edit_log:
reviewer:
reviewed_at:
status: unverified | accepted | rejected
```

用本机工具计算原始媒体哈希：

```bash
shasum -a 256 /absolute/path/to/original-media
```

不要将 API key、登录凭据、本机用户名、与教材无关的文件树或个人通知录入画面。需要遮挡时，先在采集前关闭相关窗口；若后期遮挡，必须写入 `edit_log`。

## 5. 验收规则

- 自动 `browser_page` 只在 URL/最终 URL、Playwright/Chromium 版本、PNG 哈希、DOM 断言、console 和本地网络摘要全部存在时可为 `verified`。
- 人工媒体只有在修订/工件/步骤哈希一致、画面可读、观察可重现且隐私检查完成后才可为 `accepted`。
- 任一输入哈希、批准修订或工具链变化后，旧媒体保留供审计，但不再证明当前修订。
- 未达到完成信号、只录到旁白、使用不同工件、跳过失败恢复或无法计算原始哈希时，状态必须为 `rejected` 或 `unverified`。

## 6. 当前 Docker Desktop 边界

2026-09-06 的本机固定页面诊断得到三条相互印证的结果：

- Bubblewrap 在当前非特权容器中挂载私有 `/proc` 返回 `Operation not permitted`。
- 使用空 `/proc` 时，嵌套 user namespace 因缺少 `/proc/self/uid_map` 失败。
- 只读绑定容器 `/proc` 时，嵌套 user namespace 因 `uid_map` 只读失败，Chromium 报告 `No usable sandbox`。

因此当前 `browser_page` 正确行为是 fail-closed。只有能同时提供私有 `/proc`、可用 Chromium sandbox 且继续满足断网/只读/非 root/资源上限的新运行环境，才能重新开启自动截图验收。
