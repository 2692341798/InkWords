# Go 依赖项目与来源显式选择验收

日期：2026-09-13。状态：操作工具入口、合同与登记校验通过；真实 hands_on 空章只读预检通过。

## 变化

- `TeachingDependencySelection` 固定工作区、项目、章节、主资料快照、模块版本、工具链
  与依赖清单哈希，不接收路径或命令。V1 只允许规范 HTTPS Git 主资料地址直接对应
  模块；虚拟模块地址、子模块等需另行制定来源映射合同，不能自动推断。
- 本地选择文件最多 16 KiB，必须提供独立审阅的文件 SHA-256；拒绝未知 JSON 字段、
  尾随内容、符号链接和超限文件。读取依赖包前校验实际章节与当前主资料归属及快照。
- 候选投影再次确认章节/工作区、当前来源及原生成任务冻结的证据包。原证据包哈希必须
  与候选一致，选择快照必须实际在该包中；不能从项目其它资料替换候选来源。
- 登记事务重复校验选择与候选来源；来源变化或错误归属拒绝。相同输入幂等，不覆盖
  既有工件或母稿。原大仓储文件的工件登记方法拆到 `code_artifact_repository.go`。
- 执行清单可选 `dependency_selection` 保存完整选择，并参与清单/验证哈希；v4 工件
  ID 同时绑定候选、内容、依赖和选择。旧字段缺省的序列化哈希兼容测试继续通过。
  runner 私有快照还核对依赖包 origin 与保存的选择一致。

## 实际验证

证据目录：`output/real-acceptance/2026-09-13/go-dependency-selection/`。

- `red.log` 保存实现前失败；`focused.log`、应用组合测试与 `backend-tests.log` 通过。
  全量命令：`GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...`，含架构检查。
- `postgres-verified.log`：独立 PostgreSQL 14 容器内实测工作区/项目/章节/快照归属、
  软删除、候选原证据不匹配、登记拒绝和幂等。夹具中未调用模型或执行教学代码。
  初始夹具缺就绪等待、排序及候选来源字段，按现有约束补齐后通过，失败日志保留。
- 应用测试证明选择随执行清单保存，工具链/包来源不一致拒绝；候选不匹配和读取后
  来源变化不会退回不带依赖的投影。共享合同测试证明来源变化使旧验证哈希失效。
- 真实只读检查器以临时容器运行新检查二进制，复用已有镜像与数据库连接，输入目录
  只读，未部署服务镜像或挂载工件写目录。`real-selection-preflight.json` 返回
  validated=true、activated=false、executed=false、registered=false。
- 选择绑定真实 hands_on 章 `862f68e0-8705-46ab-8473-1a197d6fab78`、主资料快照
  `cebb9dc8-da05-4475-9ca4-bdc6474ce199`，来自蓝图 r6 引用的固定 Gin 源码。
  项目/章节完整记录及修订、工件、任务数量前后相同，见 `real-selection-state-check.json`。
- `real-selection-rejections.json`：改绑另一真实项目、改动快照哈希的两次只读请求均被拒绝，
  拒绝后项目与章节记录仍不变。
- 选择文件哈希：`sha256:7de13c41790d664c8e98078947682160269be62029c086b1085ebedc76894544`；
  依赖清单哈希仍为 `sha256:79a707dfcddc51376f63339b866bb5d21b0922fa50f2a8b49a63c870bd884f5d`。
- `explain.sql`/`explain.log`：实际关联查询命中现有快照主键、来源/项目/章节索引，
  本机执行 0.127 ms。没有新增索引、数据库迁移或写放大。`git diff --check` 通过。
  原候选任务读取的 `explain-task.log` 命中已有章节任务复合索引，执行 0.047 ms。

## 操作入口与剩余工作

`textbook-dependency-inspect` 保留旧的包检查参数，增加 `--selection`、`--selection-hash`。
选择预检需要配置 DATABASE_URL；输出是可审阅的选择记录，不代表后台已启用该包。

`textbook-artifact-reconcile` 在原 task/revision/content-hash/toolchain 参数外增加
`--dependency-selection`、`--selection-hash`、`--dependency-root`、`--dependency-manifest`。
四个依赖参数与精确工具链须同时提供；默认预览，显式 `--apply` 才登记。预览/应用均
重新读取与校验，失败不回退。实际用法应先检查 selection JSON 与 preview，再应用。

当前没有为真实章登记或执行依赖工件；hands_on 仍是空章。页面自动生成预检/投影尚未
接入此显式选择，服务仍运行旧镜像。下一步完成教材文件/服务观测合同，并在生成、批准
和验证流程中使用选定包，实测 Gin 沙箱编译/测试/页面。来源哈希通过不代表版权获准，
33 份依赖许可仍待审；原三项综合验收、个人学习、冷读者与整书审校继续保留。
