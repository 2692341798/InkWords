# 学习者验证 HTTP、PostgreSQL 与真实沙箱往返验收

日期：2026-09-10。审阅来源：`delegated_ai`。PR-14 有界工程验收。

## 本次补齐的证据

此前真实代码执行使用固定 Resolver，数据库一次性领取另有测试；两者尚未通过实际 HTTP 连通。
本次新增两个默认跳过的 opt-in 测试，使用生产服务的路由、客户端、应用服务、数据库 Store、
Stager 和 BubblewrapExecutor 串起完整往返。原测试不替换、不改写，生产行为未修改。

新增文件：

- `backend/services/course-runner/app/bootstrap/real_learner_http_test.go`：测试程序先运行原配置的
  沙箱与固定 Go 自检，再启动生产 capability/run 路由，使用生产 HTTP Resolver 反向领取输入。
- `backend/services/review-service/infra/learnerverification/real_http_roundtrip_test.go`：临时 PostgreSQL
  执行 review 版本化迁移，启动生产验证路由与应用/Store，用真实 HTTP 请求预检、启动、读取结果；
  最后从 Store 取回匹配报告并传入 `AttachLearnerVerification`。

输入提供方明确为操作方冻结评测数据；数据库只保存夹具目标和作答，不冒充用户在批准教材中的
真实练习。真实沙箱执行的文件与前轮评测相同，运行器镜像变化进入新的 plan/input hash。

## 实际执行结果

| 样本 | 收据 ID | 真实执行 | 持久化任务 | 执行时间 |
|---|---|---|---|---|
| 正确方法选择 | `61e10ddb-2cd1-4689-b93e-38d4992e065f` | passed / exit 0 | 1 | 8.144 秒 |
| 总返回第一项 | `643f0256-34aa-4f56-a0c4-ec40db3c46a4` | failed / exit 1 | 1 | 8.104 秒 |

错误样本真实输出指出 POST 返回 GET root、DELETE 返回 GET root 而非 nil。两份报告均记录
execution_started、完整原快照/执行树/输入 hash、Go 1.26.8、固定策略、起止时间和原始输出。
测试整体 29.81 秒，真实测试进程 exit 0。

逐项通过：

1. GET 预检不创建任务；错误 input_hash 的显式 POST 返回 409。
2. 相同 request_id/input_hash 的重复 POST 返回同一任务，实际 runner Execute 各一次。
3. runner 通过生产 HTTP 客户端回到 review 的 resolve 路由；PostgreSQL 中 queued 原子变为 running。
4. 领取凭证消费后，再向相同 resolve 路由重放返回 404，不重新开放冻结代码。
5. 正确/错误终态和真实报告均由服务保存；每份作答只有一个任务。
6. `LatestAssessmentEvidence` 读回的输入等于原冻结输入，报告等于持久化结果；评分证据 ID、
   verification_run_id、artifact_hash 与运行收据逐项匹配，未调用模型。
7. 在**同一进程**重新构建 Service/Store 并调用 Recover 后，终态报告不变。
   首次原始结果字段名 `restart_preserves_report` 不够准确，实际没有重启 OS 进程；原件保持，
   `final-review.json` 明确这一边界，后续测试输出改名为 `service_recreation_preserves_report`。

## 隔离与本地状态

评测镜像 `inkwords/learner-http-evaluation:20260910-v1`，ID
`sha256:27dec3fece39880ec768e03169ba208091f6b3fe80020970bd8f3630ff2dea17`。
它仅在现有 course-runner 镜像上加入测试程序；保留非特权、只读根、runner 用户、cap-drop ALL、
no-new-privileges、既有外层 seccomp、512 MiB/128 pids、`/tmp` noexec/nosuid。
唯一宿主绑定是只读 seccomp 文件，没有绑定代码目录、知识库或 Docker socket。

外层容器使用 bridge 网络完成服务间 HTTP，端口仅发布在 127.0.0.1；学习者程序仍在原 v2
unshared 网络的 Bubblewrap 内执行。没有给学习者网络权限，没有下载依赖、改变命令或主机执行回退。
review 评测 HTTP listener 临时监听随机端口供 Docker 回调，测试结束后关闭；runner 测试自身有
三分钟上限，协调器在完成后停止容器。

临时数据库 `learner_http_evaluation` 已由 Testcontainers 删除；评测 runner 已停止，保留镜像、
停止的容器及证据。主学习目标和构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4` 的审阅响应
与上一轮逐项相等，网关 HTTP 200。Provider 调用、生产写入、真人学习记录新增均为 0。
当前线上服务镜像未更改，没有生产迁移、提交或推送。

## 检查与证据入口

相关七个 Go 包（review 验证 app/infra/transport、runner bootstrap/domain/transport、架构）通过，
其中 Store 原有临时 PostgreSQL 测试继续通过。未改前端，没有重复跑前端全量。
新增真实测试在无 opt-in 时跳过；协调脚本创建排他 scope 文件，避免同目录隐式重复执行。
`git diff --check` 通过。

证据根目录：`output/real-acceptance/2026-09-10/learner-http-roundtrip/`。

- `run.py`、`runtime-build/`：实际协调流程和镜像构建。
- `capability.json`、`*-input.json`：通过真实 capability 后创建的固定输入。
- `review-http.log`、`runner-http.log`、`review-exit.json`：实际调用及终态。
- `*-http-result.json`：原始持久化 Job 和评分输入，含真实收据。
- `runner-scope.json`、`final-review.json`：隔离条件、原件 hash、清理和审阅边界。
- `regression.log`、`architecture.log`：实际回归结果。

## 剩余范围

这证明生产组件在隔离评测环境中的 HTTP → 一次性领取 → 真实执行 → PostgreSQL → 评分证据绑定。
尚未覆盖主应用的批准母稿读取/练习会话/原始作答保存，也未完成同一真实学习者的模型纠正应用、
六维表现和延迟保持。没有真人冷读或出版权利结论，PR-14/PR-16 整体验收保持未勾选。
