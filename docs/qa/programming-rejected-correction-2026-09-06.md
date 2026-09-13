# programming 真实拒绝稿的内容修订

日期：2026-09-06。执行者：Codex；本记录为自动静态复核，不是人工或读者验收。

原收据来自一次真实 DeepSeek 调用：
`7f9ec866508804bdc6d6670b054724da1cbb8c178419bfb360e6b230d75b00c5`，输入 11990、
输出 8575 Token，耗时 53063 ms。原文件/hash/用量保持，修订未再次调用模型。

## 内容变更

- 用固定 Gin SHA `73726dc606796a025971fe451f0aa6f1b9b847f6` 的 tree.go:61–68 和
  tree.go:133–249，推演 `/api/orders`、`/api/health` 的公共字节前缀，以及随后插入
  `/api` 时旧节点状态如何下移、父子处理函数如何共存。明确这是静态推演，没有运行
  导入仓库。不是只为门禁补两个引用。
- 对照教学分段树和生产压缩前缀树；补充字节与路径段的差异、字符串处理标识、
  重复登记覆盖、空路径段、非网络服务等边界，取消没有测量支持的速度断言。
- 纠正 getValue 与 handleHTTPRequest 的职责；405 需要配置及其他方法同路径匹配，
  并非所有未命中都由 getValue 直接返回 404/405。
- 去掉命令围栏，保留原教学代码；初始化只做一次，排错保留原目录和报错，不再建议
  删除整个目录。固定验证目标 Go 1.26.8，增加面向已有编程基础读者的 Go 记号说明。
- 纠正结构化练习：根路径原本已支持，改为补测试；复现改为实际从零编码；迁移改为
  只替换已登记路由且失败不创建节点的新需求；保留任务 ID/模式并重写相关答案和评分
  描述。固定运行要求按共享规则填写，正文与学习阶段同步，retain 明确实际间隔 72 小时。

## 可复核证据

本地目录：`output/real-acceptance/2026-09-06/programming-correction-v1/`。
最终输入 `correction-input-final.json`，完整提案 `proposal-final.json`，可读母稿
`proposal-final.md`，静态复核 `editorial-review.json`；早期输入/提案独立保留。

`textbook-recheck -correction-stdin` 两次均 exit 0，最终 JSON 逐字节一致。
质量合同 v9 无硬失败，3 条关键事实覆盖；修订正文 hash 为
`0519a4df084bb297c483682ca137d3e9a101a54c949fda7ea079d866c79c6d9a`。
提案 origin 为 automated_local_correction、ProviderCalls=0、CandidatePersisted=false，
runtime_verification 保持 unverified。原始调用的 11990/8575 用量独立保留。

两个 Go 围栏与原稿逐字节一致，未执行它们：

- main.go：`2328e2fc45901c4e69a98ed9a32975d534d9c6ea26550f1e7d5ebdd6c2c50b08`。
- main_test.go：`1af31c4f55625f6960f1eaf125aade10fe67a31c7b6c35dbda18bf31d1ae17c0`。

## 后续真实隔离验证

上述“未执行”描述的是修订提案生成时的状态；随后已独立完成一次真实代码验证。
新增显式启用的 `TestRealFrozenTeachingArtifact`，只接收操作方选定并校验 hash 的清单，
只允许 go.mod、main.go、main_test.go 三个普通文件和一个 go_test 命令。它复用现有
Runner/Bubblewrap 执行器，不创建 Provider、数据库或生产任务客户端。

工件从最终提案的两个 Go 围栏逐字节提取；仅新增 module route-demo 的 go.mod，固定
go 1.26.0 / toolchain go1.26.8。独立评测 revision ID
`6c6400ae-1047-5045-a47e-3aa9f4c016d4` 不是数据库章节修订。

- 代码树：`sha256:56139d05b4f1b8113b25f5792442aa11fab04c1e08f8768b5e10fbe6b596ff08`。
- 清单：`sha256:ef6cdf17ca21bd08277b07f99d589e3572f7426d14080a2ed1a77601b5024ad0`。
- Runner 镜像：`sha256:aac27f6ee6d2f7fb150414ebe4c72b98d33ee456d2da062ac9bdafd4e73f733c`。
- 验证输入：`sha256:3df3f782b21f2b17093a8ae9c307ea601f328b2e869e04cadfd77682b72ebf71`。
- 实际工具链：Go 1.26.8 linux/arm64；go_test 通过，输出 `ok route-demo 0.001s`，
  执行耗时 8422049004 ns（约 8.42 秒）。

采用同一 Compose course-runner 镜像和原有非 root、只读根、cap_drop ALL、
no-new-privileges 与外层 seccomp 配置启动一次性容器；验收程序使用清空后的环境，
Bubblewrap 预检通过后再执行。没有放宽隔离或运行 Gin 仓库。一次性容器完成后已自动
删除，事后 inspect 没有获取到该容器的运行时配置快照，不把 Compose 声明冒充实测快照。

`verification-v1/binding.json` 绑定原收据、提案、正文、每个文件和清单；`report.json`
保存完整机器结果，`run.log` 保存本次 exit 0 的原始日志。新增入口默认跳过，Runner
模块、执行器及 course-runner/架构回归通过。未重建或修改生产 course-runner。

该运行证明现有测试的断言通过，包括方法隔离、共享前缀、未登记父路由和后来加入父
路由时子路由保持；没有执行 main 输出、迁移题的新功能或真实学习者作答。
冻结提案本身没有被回写成已验证，正式母稿仍需绑定其对应工件与验证记录。

## 当前尚未完成

这份输入使用独立 evaluation 项目/章节身份，没有主项目的原生成 Task，不能假冒
主项目已批准章节或用旧任务走候选追加。原主项目仍为 r7 候选、批准蓝图 r3。
已针对该评测工件完成上述受控验证；继续另一 audience 时仍不得沿用 r7 或本稿
不同代码的运行证据。练习答案仅为待审答案，真实学习者编码、试学、延迟保持和人工
审校均未发生。三类读者整体验收仍未完成。
