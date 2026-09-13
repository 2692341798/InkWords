# stack_familiar 拒绝稿修订与真实代码验证

日期：2026-09-06。执行者：Codex。内容复核为自动静态复核；未进行人工审阅或读者试学。

## 修订内容与来源

沿用真实 v16 DeepSeek 收据
`650c439fd33967275d379385ebac9313a76670404e502989bd276db08e680a70`。
原调用输入 12006、输出 8526 Token，约 49.25 秒；缓存和费用未知。本轮新增 Provider
调用为 0，原收据、失败原因和用量均保留，未把修订文本改写成模型原始输出。

- 纠正 HTTP 无状态与进程路由内存的错误因果关系，改由同一 Engine 的登记写入与
  分发读取解释；查不到路由时分别检查实例、方法、完整路径与登记是否执行。
- 对照固定 Gin 提交 `73726dc606796a025971fe451f0aa6f1b9b847f6` 的
  tree.go:61–68、133–249，区分 `i < len(n.path)` 的旧状态下移和
  `i < len(path)` 的剩余路径下降，推演两个子路径后插入父路径的状态。未运行导入仓库。
- 区分 getValue 的匹配结果、handleHTTPRequest 的处理链执行、重定向与条件式
  405/404；教学程序只返回字符串和布尔值。补全重复登记、空 handler、空分段、
  路径约定、无网络服务等边界，去掉未经测量的性能结论。
- 补全题改为增加根路径和失败查询不改树的测试；复现题要求从零写程序；迁移题改为
  原代码尚无的 removeRoute，且只移除终点、不剪枝；排错题使用两个实例作对照。
  正文、答案、五项评分与学习阶段同步；保留原任务 ID、模式及固定运行要求。
  retain 明确要求实际间隔至少 72 小时，未预填学习结果。
- main.go 与原稿逐字节一致。main_test.go 新增 healthHandler 在父路径登记前后的
  两个返回值断言，补足原稿“两个子路径都保留”的测试证据。

## 实际验证

目录：`output/real-acceptance/2026-09-06/stack-familiar-correction-v1/`。
`correction-input-final.json` 为结构化输入，`proposal-final.json` 为最终提案，
`proposal-final.md` 为可读母稿。第一轮门禁失败的输入及提案独立保留为 iteration1；
完善独立作答指令和类比边界后，两次 textbook-recheck 均 exit 0，最终 JSON 逐字节一致。
质量合同 v9 无硬失败，3 条关键事实、10 段证据覆盖通过。长段落、长句、重复来源段落
及流程图机会仍是自动编辑建议，不冒充八维人工审校已通过。

最终正文 hash：
`sha256:add03801d8e1d00c04e333ca37e012ae7b60c9ff517165ea44cf33166637d01a`。
提案 origin 为 automated_local_correction、ProviderCalls=0、CandidatePersisted=false，
runtime_verification 仍为 unverified。独立运行证据保存在 verification-v1，不回写冻结提案。

从两个 Go 围栏逐字节提取文件，另加 module route-demo 的 go.mod，固定 go 1.26.0 /
toolchain go1.26.8。独立评测 revision ID `0dd477a6-011b-5d67-9c87-89d8f0bf1931`
不是数据库章节修订。

- main.go：`sha256:af4d70f3a156999cfddd640c0d28a6c445052185ce44e79f9286046a8b59a3a5`。
- main_test.go：`sha256:443410f1ea7d28a82ba1ddea4e8ffd064e08d006ceb52f3706b7f79cd7aea8d2`。
- 代码树：`sha256:26a858e6b2d5658cdbb841270c8598f4b9b2b9fb7c5ec573630cf2ee1bec8a20`。
- 清单：`sha256:1f25280b1abb501444a38cb71d79e0bd47400f1b277c0b5b71f786cc55e99951`。
- 验证输入：`sha256:6b737811f889435bc01141ec1720fcf2d30727acc9e0e561cc48ac24bc8d1d80`。
- Runner 镜像：`sha256:aac27f6ee6d2f7fb150414ebe4c72b98d33ee456d2da062ac9bdafd4e73f733c`。
- 实际 Go 1.26.8，Bubblewrap 预检通过，go_test 输出 `ok route-demo 0.001s`，
  执行耗时 8031688337 ns（约 8.03 秒），验收入口和一次性容器均 exit 0。

复用未改变的 TestRealFrozenTeachingArtifact 与已编译验收程序；逐项 hash 核对与上一轮
Runner 源码、内外 seccomp 和测试二进制一致。入口清空环境后只运行批准清单中的 go_test。
实际 inspect 保存 runner 用户、只读根、cap_drop ALL、非 privileged、
no-new-privileges、seccomp、128 进程、512 MiB、1 CPU、只读工件与二进制挂载。
真实知识库路径在 Compose 配置中保持。未重建服务、安装依赖或放宽隔离；本轮创建的
一次性容器在保存配置与退出状态后删除。

verification-v1/binding.json、report.json、run.log、container-profile.json 和
runner-source-hashes.json 分别保存输入绑定、机器结果、原始输出、实际容器配置和
执行器身份。editorial-review.json 保留自动复核和未覆盖边界。

## 完成边界

这次运行覆盖现有教学测试：方法隔离、未知路径、共享节点、父路径登记前后和两个
子路径保持。没有执行 main 的打印、问题体验及新增练习实现，也没有学习者提交。
不能将基线测试输出复用于 removeRoute 或其他修改后的代码。

真实 API 回读确认主项目仍为 r7 候选 `018e61ab-c48f-4eff-ae87-9aeca4a3209e`，
蓝图 r3 `94e45a08-f9cf-40cf-8d93-fb2e917936c1`。本评测没有主项目生成 Task，
不借旧任务创建候选、不伪造章节身份或批准。三类读者整体验收、人工审阅、真实试学、
延迟保持及整书导出审校仍未完成，PR-16 保持未勾选。
