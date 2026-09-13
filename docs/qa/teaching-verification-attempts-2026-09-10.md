# 教学代码独立验证尝试：2026-09-10 真实验收

本轮在已获授权的本地 InkWords 中完成实现、迁移、部署和浏览器操作。操作员为 delegated_ai；没有创建真人审阅或个人学习记录，也没有 Provider 调用。没有提交或推送 Git。

## 交付行为

- 取消、失败或成功后的显式重跑创建新任务，前次任务及证据保持原样。
- `request_id` 固定一次请求，`expected_previous_task_id` 固定用户看到的前序；精确重提复用原任务，旧前序/改写同请求身份被拒绝。
- cancelled 表示取消已受理；已经领取的执行器必须另外确认退出，才能创建下一次尝试。worker 的退出回执在执行和观察器结束后写入，重复释放不改首次时间。
- 页面恢复最新状态和尝试历史，支持重新运行、重新验证、取消、查询和响应中断后的同请求重提。旧 retry 接口不能重置版本化任务；旧空 POST 只恢复已有任务。
- 过期收据继续显示未验证；成功任务不会阻止用户显式收集新收据，旧收据仍保留。

## 实际环境与迁移

schema 37 → 38。core `inkwords/core-api:verification-attempts-v2-20260910`，runner `inkwords/course-runner:verification-attempts-v1-20260910`，frontend `inkwords-frontend:verification-attempts-v1-20260910`，均 healthy。

runner 实际摘要 `sha256:7517d0fabaa40e9881746acb8db84608d126a762b4f76f4f8dde85ea8ed6c7ea` 与声明、两条新收据一致；仍是 runner 用户、只读根文件系统、全部 capabilities 移除、原 seccomp、工件只读挂载和专用读取组。真实 Obsidian vault、review 模型配置保持，export 镜像不变。前端实际 HTTP 资源字节与本地 dist 相同。

迁移前保存 `core-before-migration38.dump`，6,004,851 字节，文件权限 0600；`pg_restore --list` 确认 job_tasks 与教材工件等目录项，未执行生产恢复覆盖。7 个原任务迁为各自第 1 次，剔除新增列后逐字段与迁移前 JSON 完全相同。旧成功执行标记 `legacy_terminal`，旧运行中取消不根据 finished_at 伪造退出。

唯一索引按 artifact/attempt 服务最新读取并约束重复序号，每次追加增加一个小 B-tree 写入；workspace 所有权仍单独校验。真实 60 行任务表的 `EXPLAIN (ANALYZE, BUFFERS)` 选择顺序扫描与小排序，执行 0.137ms，证据在 `latest-query-explain.txt`，没有声称小数据下已使用索引。Down 在已有验证序列时拒绝丢弃证据。

## 真实浏览器与运行结果

历史 r6 工件 `572c6411-5013-5e06-8df0-19afe29f747f`，代码树 `sha256:02c5cda70efa04ded67313e61d1b8e7e28a0265c45f5e6ff1008360731f04669`，清单 `sha256:a05849af83c0409f4345c04822e1379c6e22eee0a9932070ebbd48894dee6c2d`。所有尝试的冻结 payload 完全一致，工具链 Go 1.26.8。

| 尝试 | 任务 | 实际结果 | 退出来源 |
| --- | --- | --- | --- |
| 1，原记录 | `c9d9aca2-dd4f-4a5e-9b37-297a0d1fe58c` | 上轮运行中取消，原字段不变 | 本轮有来源的 operator_observed_exit |
| 2 | `ce780337-48b7-4ffc-9ba8-a3a7dbba9f1d` | Go 测试 verified，8.457s | worker |
| 3 | `70b8f4e5-79dc-4cb1-b04c-e9266dfc1419` | 从成功任务点击重新验证，运行中取消 | worker |
| 4 | `5593b31f-67b4-4133-a1e9-c072536b4916` | 从取消任务重跑，Go 测试 verified，8.739s | worker |

浏览器先观察到旧取消缺少退出确认，未显示重新运行按钮。核对上轮 `processes-after-cancel.txt`、旧 runner 镜像已无对应容器及当前仅 runner 主进程后，通过精确 CAS 登记第 1 次的操作员观察时间；`legacy-exit-ack.json` 保存来源文件哈希与解释，未把本次确认时间冒充历史退出时间。

第 3 次实际运行时观察到 Bubblewrap/Go 子进程；取消终态写于 00:44:41.812588 UTC，worker 释放写于 00:44:41.967576 UTC，随后只剩 runner 主进程。该次结果为空、没有新运行证据。两次成功分别新增 `3b32e523-db13-4c7f-b074-2c45dda17c52`、`222f69ac-2f7e-47e8-83fb-9d581029f403`，raw_evidence_ref 各自绑定任务。

教学验证任务总数 7 → 10，新增 3 次，四次 retry_count 均为 0。精确重放旧成功/最新成功均复用原 ID；新请求旧前序、同 ID 改前序返回 409，夹带 path 返回 400，取消任务旧 retry 返回 409；空 POST 恢复第 4 次。上述请求前后任务全量 JSON 完全相同。

整页刷新、重新打开工作台和章节，恢复第 4 次及四条历史；监听确认 0 次验证 POST/retry。桌面截图已逐张查看。390px 窄视口截图显示现有桌面侧栏仍遮挡内容，因此不记为移动端验收通过，也未扩展为移动应用改造。

## 验证与保留范围

- 后端全量 57 包通过；新增任务/应用、textbook、runner 域和 bootstrap 的 race 验证通过，随后 POST 历史投影修正的应用及架构回归通过。
- 隔离 PostgreSQL：旧字段保真、取消不补造退出、8 个相同请求仅创建 1 次、8 次领取仅 1 次成功、释放后新建、保留首次释放时间、旧前序拒绝、Down 拒绝证据丢失，均通过。
- broker 结果不确定后显式重提同请求，只恢复该任务；领取后重提不再次发布的测试通过。
- 前端全量 308 项通过；随后新增过期证据联验，最终验证面板/控制器 15 项通过。lint/build 通过。`git diff --check` 通过。
- 生产没有调整时间或收据有效期。过期入口使用自动化到期夹具验证；真实浏览器检验的是明确新建与新证据采集。

主章节、全部正文修订、批准记录、原运行证据、资产、锁和 generation target 均保持不变。当前批准 r13 `1ee15e29-1304-4978-9e6f-d41421798c8b` 及出版构建 `0f150bbe-b80a-4586-8997-320df2c35ae0` 未变，editorial 完整响应相同，真人审阅仍为 0。此次验证的是历史 r6 教学工件，不把它冒充当前批准稿的代码来源。

完整证据位于 `output/real-acceptance/2026-09-10/verification-attempts/`，核心清单为 `final-facts.json`、`tasks-final.json`、`chapter-final.json`、`runtime-final.json`、`live-api-checks.json` 和 `browser-refresh-final.txt`。实际操作脚本会明确检查错误，截图失败不能计为通过。

本垂直切片完成。原开发计划仍有三类 audience 代表章、真实评分/学习闭环、整书试学与审校等未完成项；执行器崩溃而没有退出回执时继续 fail-closed，需要独立运行环境证据才能恢复，不能自动放行。
