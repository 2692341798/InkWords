# 蓝图章节引用边界修复

日期：2026-09-13。

## 问题与行为

动手章规划时，后端接受了不存在的未来章节 ID，而工作台通过项目的实际章节行展示
蓝图，导致该条目无法完整审阅。现在创建草稿和批准草稿均在项目事务中核对每一个
章节 ID：必须是规范 UUID、属于当前项目、且未软删除。失败返回已有的
`400 INVALID_STATE`，不会保存新草稿或改动原批准指针。

批准入口独立检查旧草稿；无需迁移或改写历史记录。蓝图内的标题仍可调整，正常跨卷
先修依赖照常通过。没有要求蓝图必须涵盖项目所有空章节；代表章覆盖仍由原有门禁判断。

蓝图相关持久化方法移至 `backend/services/core-api/domain/textbook/blueprint_repository.go`，
避免继续增长 1,845 行的原 repository.go。证据归属、合同身份、事实覆盖及审批事务
仍沿用原逻辑。本次没有更改母稿、生成器、沙箱或依赖。

## 实际验证

- 真实 PostgreSQL 14 临时容器红绿回归：不存在、跨项目、软删除、非 UUID、非规范
  UUID 五类引用，修复前创建与批准均错误通过，修复后均拒绝。断言创建行数不增、
  旧草稿保持 draft、原批准指针不变；正常两卷、两章、不同标题及跨卷先修通过。
- 新增参数化查询的 EXPLAIN ANALYZE 使用已有
  `idx_textbook_chapters_project_status_sort`，测试夹具中共享缓冲命中 2、执行
  0.012 ms。该小样本不代表规模性能；未增加索引、存储或写放大。
- `GOPROXY=off GOCACHE=/tmp/inkwords-go-build go test ./...` 通过，包含架构与教材
  仓储回归。测试通过已有依赖缓存执行，没有安装依赖。
- 本地 core-api 更新为 `inkwords/core-api:blueprint-membership-v1-20260913`，仅替换
  编译后的产品二进制，基于原运行镜像离线构建；Compose 环境逐键核对不变，健康通过。
- 实际 Nginx 入口拒绝含未来章节 ID 的创建请求与旧 r5 的批准请求，均为
  `400 INVALID_STATE`。完整 workspace、冻结构建 editorial JSON 前后相等：批准
  r4、首章 r15、r6 草稿、新章 r0、966e247b 构建与三条 AI 审阅均保持。
- 真实浏览器刷新并重新打开基础项目：显示 2 章、动手实践 r0、当前蓝图 r6、第二章
  关键事实与证据选择区。代表章缺口仍正确显示，未批准任何新蓝图。

证据目录：`output/real-acceptance/2026-09-13/blueprint-membership/`，包括
`red.log`、`green.log`、`backend-tests.log`、`image-build.log`、`deploy.log`、
`live-rejections.json`、`workspace-before.json`/`workspace-after.json` 和
`editorial-before.json`/`editorial-after.json`。

## 剩余范围

这次修复不等于 hands_on 代表章完成。r6 仍是未批准草稿，真实 Gin 的固定版本离线
依赖供给及服务观测清单仍待实现；没有新模型调用或教学运行。原个人学习、试学、
整书审校与权利门禁保留。WebP 依赖和浏览器运行详情本地截图的授权仍待答复。

回退仅需恢复上一 core-api 镜像及原 Compose 覆盖链；无数据迁移。回退会重新暴露
无效章节引用缺口，正常情况下应保留本修复。不要为回退删除草稿或修改批准记录。
