# 学习者代码运行证据评分验收

日期：2026-09-10。审阅来源：`delegated_ai`。范围：PR-14 的固定 Go 文件快照、真实隔离运行收据与真实模型评分绑定。

## 结果与修改

两个操作方编写的评测样本在真实 Bubblewrap 沙箱执行，再以原始冻结输入和收据调用 DeepSeek。
正确 `findRoot` 按方法返回原 root；错误版本总返回第一项。测试覆盖空列表、GET、第二项 POST、
缺失 DELETE 与登记不被修改。正确样本通过 exit 0，错误样本的 POST/DELETE 断言失败 exit 1。
这些样本没有写入个人学习数据库，不能计作用户掌握、真人试学或出版审核。

发现并修复两处实际问题：

- 前端学习者执行能力判断与两个类型仍固定 v1，实际服务已使用 `inkwords.learner-go-test-offline.v2`。
  更新到 v2，并继续拒绝 v1/v999；执行仍需显式操作。修改涉及 `LearnerCodeSnapshot.tsx`、
  `services/mastery.ts`、组件测试和 E2E 夹具。
- 首次真实评分的 runtime 正分引用了运行收据，却缺少受测代码的路径/行号，被 `feedback_answer_quote`
  拒绝。仅在带工件的评分请求中补充说明：正分同时引用原受测代码和对应 `learner-runtime:*`。
  代码引文标明受测对象，运行收据证明执行结果。领域门禁未放宽，文本/旧合同分支不变。
  新请求 hash 改变；旧输入、收据和失败结果均保留。

## 真实执行与模型结果

| 样本/批次 | 真实运行 | runtime/tests | 模型调用 | 输入/输出 Token | 延迟 |
|---|---|---|---|---|---|
| 正确，原提示 | passed，exit 0 | 合同拒绝，无有效评分 | 1 | 4592 / 636 | 3699 ms |
| 正确，新提示 | 同一冻结收据 | 4 / 4 | 1 | 4705 / 711 | 3765 ms |
| 错误，新提示 | failed，exit 1 | 0 / 0 | 1 | 4788 / 1121 | 5920 ms |

总计 3 次真实 Provider 调用，输入 14085、输出 2468 Token；无隐式重试。原批首例失败后停止，
未发送错误样本；修正后新批才执行两例。错误新批返回缓存 Token 512，其它缺失字段保持未知；
未返回货币成本，不推算为零。模型为 `deepseek-v4-flash`，代码配置关闭推理、最大输出 6000，
适配器超时 60 秒，与部署的 `local-evaluation-v1` 对应。

正确收据 `3cdd0798-47cb-471c-94c3-4f824cf6f348`，错误收据
`805c22ad-26ed-4207-8330-0776a50ab482`。评测镜像以现有 runner 镜像为基础，仅加入测试可执行文件，
使用实际 Stager、Runner 和 BubblewrapExecutor；Resolver 返回操作方冻结夹具，并非生产 HTTP resolver。
评测镜像 ID：`sha256:6613db4d632ba0d90cdf97ca5f7f5f9524f6f10b663c35f8c8d5aa243fe9036f`。
容器保持无网络、只读、非特权、cap-drop ALL、no-new-privileges、原 seccomp、512 MiB/128 pids，
`/tmp` 保持 noexec；内层使用固定 v2 离线策略、Go 1.26.8。没有宿主机执行、依赖下载或放宽沙箱。

最初向运行容器的 tmpfs 复制文件不可见；随后 `/tmp` 可执行文件被 noexec 拒绝，两次均未执行
学习者代码。改为独立评测镜像 `/app` 内的测试程序后成功。原失败日志保留。

对两份成功反馈逐项检查原文件行号、解析后的原文引文、对应收据 ID、分数与理由：正确性区别成立，
错误样本明确指出 POST/DELETE 失败及“自称通过”与证据不符。正确样本提示仍重复已测空列表/缺失场景，
仅作为可选回顾，不能认定缺失；该评测的 runtime/tests 项评价执行结果，不代表通用测试质量评价。
结论是两例有界语义验收通过，不能推广为全面评分质量通过。

## 检查与部署

- v2 能力测试先出现 1 失败/4 通过，修复后 7 通过，包含旧版/未知版拒绝。
- 运行评分测试证明“仅收据缺代码”和“仅代码缺运行收据”继续拒绝，双绑定可接受。
  `runtime-quote-red.log` 的首次失败含夹具 hash 设置错误，不能当作有效 TDD 红灯证据。
- review-service 全包、course-runner 学习者验证包及架构回归通过。
- 前端 74 文件/291 测试、lint、build、bundle 检查通过；核心 Chromium 8 条通过。
  这些 E2E 使用路由夹具，未冒充生产学习者执行。入口包 gzip 137702 字节；构建仍有普通大分块提示。
- 前端已部署 `inkwords-frontend:learner-profile-v2-20260910`；网关实际 JS 与构建字节/hash 相等。
- review-service 已部署 `inkwords/review-service:runtime-quote-v2-20260910`，镜像
  `sha256:0d8c6aabead2f2482550423462060380e0219fc2df69bb87c53a1fec8fe8b6e1`。
  容器二进制与构建 SHA-256 `d8e3a0476879afefb95b4064a9b6f8c7c5a4b076a03aec3a04ea5175ea76fe02` 一致。
  首次 Dockerfile 错将本地 image ID 当 registry 引用，镜像元数据请求 403；改为已核对 ID 的
  本地 tag 后构建成功，保留两次日志，未安装新依赖。
- 部署仅更新对应服务，保留已有 Compose 所有覆盖层、实际知识库挂载、模型/profile/key 配置。
  review-service healthy；网关 due、原学习目标和冻结构建 editorial 均 HTTP 200；真实浏览器刷新成功。
  原目标尝试仍为 0，部署前后目标响应相等；构建 `fb19cf7e-b952-4388-90a4-53b57a35acf4`
  的 editorial 与部署前相等。未改教材、批准修订或审阅记录，未执行生产迁移。
- `git diff --check` 通过。未提交或推送。

## 可恢复证据与剩余范围

证据根目录：`output/real-acceptance/2026-09-10/runtime-assessment/`。
`image-run/` 保留真实运行与首次失败评分；`prompt-v2/` 保留同一输入/收据的新评分。
`final-scope.json` 为最终调用数、原件 hash、评分和语义边界；`scope.json` 是首次失败后的历史快照。
`review-deploy/verification.json`、部署脚本与构建日志记录实际服务；`frontend-deploy/` 记录前端更新。
新增 opt-in Go 测试默认跳过真实执行；每次 Provider 评测先创建排他性 scope 文件，避免隐式重复调用。

PR-14 仍保持未勾选：尚未证明生产 HTTP claim/store 的完整运行评分和个人纠正/应用流程，
也未取得真实学习者六维/延迟保持证据。独立冷读、权利账本、整书代表性与其它 PR-16 条件保持。
继续处理可实现的工程验收，不将这些样本标为真人学习完成或出版就绪。
