# 学习者代码隔离执行：范围扩展提案

- 日期：2026-09-05
- 状态：已接受（2026-09-05）；2026-09-06 本机沙箱预检通过并显式启用，尚无真实学习者作答运行
- 关联：PRD 10.3、PR-14、`AGENTS.md:105`、FSRS ADR

## 需要决定的范围

现有规则为：`Run only generated teaching artifacts through the approved manifest and sandbox.`
设计文档 10.1 同样只允许执行 InkWords 生成的教学工件。学习者随作答提交的文件属于
另一种来源；不能仅将其放进教学工件目录、换一个 manifest 名称，就把它当作获准执行。

提议仅增加：用户在具体、已批准教材练习中亲自提交并已冻结的 Go 源文件，可在明确
点击验证后通过独立的学习者 manifest 和同等隔离要求运行。原导入仓库、任意宿主路径、
外部 URL、压缩包、模型指定命令、任意 shell、后台自动执行仍不在范围内。
用户已于 2026-09-05 明确接受这一来源范围扩展。该决定不等于沙箱已可用或验证通过。

本机 Docker Desktop 的 namespace 预检此前未通过。2026-09-06 已确认根因是默认容器
seccomp 在 Bubblewrap 建立内层沙箱前拦截必要系统调用；现使用受审外层 profile，仅为
Bubblewrap 放行建沙箱调用，并在执行固定或学习者进程前安装第二层 classic BPF 重新拒绝
mount 与 namespace 调用。容器仍为非特权、只读根文件系统、cap-drop all、
no-new-privileges，不新增云端、Linux 服务器或提权部署。

## 已完成的可审查部分

`shared/kernel/textbook/learner_verification.go` 的只读合同
`inkwords.learner-verification-plan.v1` 绑定 workspace、目标、作答、会话、章节、批准
revision、题目 ID、维度、原题目内容哈希、代码快照哈希、源文件集合哈希、完整题目
（含变式与 rubric）哈希、运行器 image digest、固定 Go 工具链版本和实际读取的外层
seccomp profile 摘要。

`NewLearnerVerificationPlan` 从有效代码快照与 v2 学习投影计算输入；`ValidateFor`
从所有者重新读取的数据重建并比较，拒绝换作答、换文件、换题或换环境。
`review-service/domain/mastery.PrepareLearnerVerificationPlan` 先核对 workspace 所有权
与原作答的完整代码身份，再组装输入。它没有 HTTP 路由、队列、执行器或文件系统写入。

这个哈希不是签名，不能单独证明批准、所有权或镜像真实性。后续消费者必须从受信任
服务重新读取并核对原数据。`files_hash` 是原源文件集合的摘要；它不是现有磁盘工件
的 `teachingartifact.TreeHash`，二者不得混用。计划也没有 `verified` 或 `passed` 状态。

## 接受范围扩展后的接入方式

1. **review-service 负责原事实与显式动作。** 从已保存 attempt、LearnerArtifact 和冻结
   PracticeSet 组装计划。预检只读；点击验证才创建具 request_id/input_hash 的任务。
   一个作答可有多个不同环境的运行，原文件和保存时间不改。重复提交复用同任务，取消、
   超时、丢失响应及进程重启都有持久化终态，不自动重跑。
2. **course-runner 负责执行。** 内部版本化接口只接受作答身份和已确认的 input_hash，
   不接收原始 shell、目录、环境变量或执行结果。它从 review-service 重新读取同一
   workspace 下的冻结文件和题目，核对运行器当前镜像、实际工具链和输入 hash。
   新路径不读取 review 数据库，也不复用 core-api 的教材 CodeArtifact/RuntimeEvidence 表。
3. **临时执行树与原快照分开。** 只从冻结字节构建临时树；可选 go.mod 仍是数据，不能
   授权下载依赖、替换到宿主路径或升级工具链。缺少 go.mod 时如需补入受信任的最小模块
   文件，必须在执行 manifest 中单列派生文件及完整执行树 hash，不能改写原快照或隐瞒
   注入。依赖不可用则报告未验证，不从网络补齐或运行安装脚本。
4. **命令与权限由服务端固定。** 拟用 `go test` 的受限参数模板，禁止请求自带 argv、
   正则过滤或 go generate；固定网络关闭、非 root、只读系统和源文件、临时工作区、
   cap-drop all、no-new-privileges、CPU/内存/pids/磁盘/输出/超时边界，不挂载宿主仓库、
   知识库或 Docker socket。Go 使用本地工具链、关闭 CGO/模块联网、禁用测试缓存复用。
   具体 argv/环境与限制必须进入已审核的 profile 和运行输入哈希，并通过实际预检。
   当前 `inkwords.learner-go-test-offline.v2` 固定 CPU 30 秒、内存 384 MiB、进程 64、
   单文件 64 MiB、输出 1 MiB、总超时 30 秒。256 MiB/10 MiB 在当前 Go 1.26 编译器下
   实测不足，因此不能继续沿用旧值或把资源失败误写为题目失败。
5. **事实和解释分开。** 新的学习者 VerificationRun 记录 plan hash、原快照 hash、
   实际执行树 hash、镜像/工具链、argv/环境白名单、起止时间、exit code、受限日志及
   截断状态。未开始、超时、隔离不可用和完成但命令失败不能混为一谈；exit code 未知
   就保持空值。学习者自行编写的测试标为 `learner_checks`，不冒充批准题目的独立验收。
6. **评分消费匹配的事实。** 只有本次作答、文件和计划均匹配的运行才能进入评分输入。
   学习者测试通过不自动证明题目正确、覆盖充分或掌握。没有独立验收依据时，明确
   限定结论；没有相应运行事实时，相关评分继续未知。之后若接入批准题目的验收测试，
   需单独冻结测试来源/hash，并将其与学习者代码逐项区分。

## 验收与回退

- 先用受控执行器验证上述身份、幂等、取消、失败、越权和重启合同，再使用真实已批准
  练习验证隔离环境；两种结果分开报告。测试不得在宿主执行学习者文件。
- 真实预检必须证明文件系统、网络、用户、资源和时间边界；未通过时保持未验证，
  不能为了通过而增加特权或绕开 browser/runner sandbox。
- review 后续新增运行表使用版本化迁移，不迁移或伪造已有记录；Down 在运行证据非空
  时拒绝删除。回退旧应用须暂停相关写入并保留 schema，优先向前修复。
- 前端显示准备、运行、取消、重试和恢复状态；验证、模型评分、用户纠正与应用仍为
  独立显式动作。自动工具记录不成为读者实测、人工审阅或教材批准。

范围规则已同步到 `AGENTS.md`。执行接入现已实现：`inkwords.learner-verification-report.v1`
和 review schema v33 保存显式任务、一次性 capability、取消/重试/中断终态及不可伪造的
运行身份；course-runner 重新向 review-service 解析冻结输入，在单独临时树中注入已列入
hash 的最小 `go.mod`，只允许固定离线 `go test` 策略。前端预览和状态恢复已接通，预览
本身不创建任务，开始和重新验证均要求明确点击。

自动合同、隔离 PostgreSQL、race、前端测试、Compose 更新和真实浏览器网关已通过。
2026-09-06，course-runner 使用内容摘要固定的 v2 profile 重建：外层 seccomp 文件以只读
方式挂载并由服务启动时实际计算 SHA-256，摘要不匹配即 fail-closed；启动预检先运行固定
`/bin/true`，再用生产 `ExecuteLearnerGoTest` 编译和运行服务内置测试，确认非 root、源码
只读、宿主应用路径隐藏、外网不可达、资源限制精确，并确认内层 BPF 对再次
`unshare(CLONE_NEWUSER)` 返回 EPERM。能力接口现返回 accepted=true、
available=true，镜像、Go 1.26.8 与沙箱 profile 摘要完整。`backend/.env` 已显式打开
`LEARNER_ARTIFACT_VERIFICATION_ENABLED` 并固定当前本地镜像摘要。

这次只执行了操作方固定自检，没有读取或执行学习者文件，也没有创建 VerificationRun。
评分 v3 继续只读消费同一作答最新的 passed/failed/timed_out 报告；其他终态保持未知。
首条真实运行仍必须来自用户在已批准练习中保存代码并明确点击验证，不能由自动自检代替。
