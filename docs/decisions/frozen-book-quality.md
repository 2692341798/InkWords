# 冻结章节质量报告与整书审校证据汇总

日期：2026-09-10。

## 原因与行为

旧版审校 ZIP 的 `quality-report.json` 恒为 unavailable，章节质量预检查询当前数据库报告，离线审阅者无法查看冻结时的警告和人工复核要求。

新构建输入为 `inkwords.book-build-input.v3`，外层 BookBuild v1 和 Canonical Book AST 不变。`quality_snapshot` 使用 `inkwords.book-quality-snapshot.v1`，在构建事务中固定每个批准章节的章节 ID、修订 ID、内容哈希、写作合同 ID、原始质量 JSON 和报告哈希。构建整体同时固定批准的 BookContract 与 StyleSheet ID。报告中的 advisories、manual_review_required 和各项明细不裁剪、不改判。

报告哈希先按 Go JSON 值解析、排序键编码，再计算 SHA-256，以容忍 PostgreSQL JSONB 的空白和键顺序变化；数字使用 UseNumber 保留精度，避免大整数被 float64 舍入为同一值。这不是通用 RFC JSON 规范化协议。报告内容进入构建幂等哈希，采集时钟不进入。相同输入重复冻结复用原 ID，报告改变产生新构建，不向新构建迁移审批。

## 质量判断与范围

共享纯合同检查快照结构、版本、哈希、章节逐项绑定以及合同一致性。`Assess` 要求当前 sample-quality 合同版本、passed=true 且无 failures。空报告、旧版报告和失败报告可以固定为待审数据，但质量门禁不通过。未知/缺失声明版本、null 快照、被替换的章节或损坏哈希不能退回旧版路径。

core-api 和 export-service 对新构建只读取固定报告进行质量预检，数据库后续变化不改变旧构建快照。旧构建保持无快照，兼容原预检路径；不回填数据库、不改变旧 manifest 哈希。

`inkwords.book-quality-report.v1` 汇总构建 ID、manifest 哈希、导出时间、固定章节报告及其当前合同判断，以及当前构建的自动出版检查、真人审阅记录、用户委托 AI 审阅记录和出版预检。这些后续审阅是导出时的记录，明确区别于冻结章节报告。旧构建输出 `chapter_snapshot_status=unavailable`、null 快照及真实已有审阅，不能伪装拥有历史报告。

自动章节门禁通过不是整书审阅通过，不能推导整书 0–4 分、独立评审、真人冷读、权利许可或学习者掌握。报告不生成任何审阅分数；只收录实际记录，缺失阶段仍由既有 fail-closed 出版预检阻断。

## 存储与恢复

没有新依赖、数据库迁移或索引。原有工作区限定的批准章节连接查询仅增加三个投影列，过滤、排序和索引路径不变；SQL 继续参数化。JSONB 存储增加每次新构建所需的章节报告副本。独立 PostgreSQL 夹具验证事务冻结、输入幂等和后续变更隔离。

回退 core/export 镜像可读取外层 v1，但旧程序不会使用新增质量快照，预检会恢复旧动态行为；不视为等价验收。新快照仍留在数据库，重新升级后恢复读取。不能为回退删除构建或恢复旧 manifest。
