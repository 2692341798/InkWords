# 教学工件验证的工作台入口

日期：2026-09-10。

## 行为与接口

章节的教学工件卡片显示来源修订，并区分当前稿件代码来源和历史修订。页面打开时只查询既有任务；仅点击“运行隔离验证”才提交执行。任务活动期间每两秒查询一次，离开页面停止查询；后台任务继续运行，再次打开从服务端恢复。查询失败不会被当作任务失败，也不会触发自动重试。

| 操作 | 合同 |
| --- | --- |
| 查询既有任务（新增） | `GET /api/v1/textbook-projects/chapters/:chapterID/code-artifacts/:artifactID/verify`，返回 `{code:0,data:{id,status}}`；不存在任务时 data 为 null，不创建任务。 |
| 显式启动（既有） | 同一路径 POST，客户端仅发送空对象；服务器从本地 workspace 下的章节工件解析冻结身份。 |
| 查询任务快照（既有） | `GET /api/v1/textbook-projects/tasks/:taskID`，客户端检查教学验证 subtype。 |
| 失败重试（既有） | `POST /api/v1/textbook-projects/tasks/:taskID/retry`，复用原任务和冻结载荷。 |
| 取消活动任务（既有） | `POST /api/v1/tasks/:taskID/cancel`，由 installation workspace 授权。 |

查询首先验证工件属于当前 workspace 的指定章节；app 服务只接受 teaching_implementation。task 服务通过现有 workspace/type/idempotency 查询定位任务，再由 app 校验冻结 payload 的 artifact、revision、code hash 与 manifest hash 全部一致。响应不返回命令、路径或任务载荷。执行关闭时仍可读取历史。没有新的数据库结构、索引或迁移。

`ArtifactVerificationControls` 负责操作与观察，API 仍在 services。终态后重新读取章节的工件、证据与资产；不因任务 succeeded 就直接把工件标为 verified。父级工作区刷新时重建证据面板，防止旧局部快照覆盖新资产。页面不保存可被误当成业务事实的 localStorage 任务映射。

## 边界与后续

2026-09-10 后续修复：runner 已增加运行中取消观察与 context 停止，证据/工件状态/任务完成原子提交；核心任务条件更新防止取消与完成互相覆盖，取消响应回读实际终态。真实页面已取消正在执行的 Go 工件，确认进程退出、零运行证据，并在刷新后恢复同一取消记录。详见 `docs/qa/teaching-verification-cancellation-2026-09-10.md`。

当前失败任务可以重试，已取消任务仍受后端既有状态机限制，页面明确说明不能重试。已成功但证据后来过期的任务也没有新验证尝试生命周期，本次不把“刷新状态”冒充重新执行；后续需要带尝试身份和取消确认的重验证设计，不能简单放宽终态检查后重用正在退出的执行。

历史工件文件权限与当前生成工件不一定相同。r6 后续已通过受限恢复工具修复专用组读取权限，原代码/清单不变；原任务真实重试通过 Go 测试，但实际工具链与旧清单不匹配，收据不能直接沿用。另建绑定 Go 1.26.8 的独立 r6 工件用于真实取消测试，保留旧工件。其它未处理的历史工件不据此宣称恢复。

验证收据与具体测试范围见 `docs/qa/textbook-verification-ui-2026-09-10.md`。回退使用原 core-api 与前端镜像即可，无 schema 回退；旧任务与证据保留。
