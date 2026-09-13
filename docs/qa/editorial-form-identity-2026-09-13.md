# 出版审阅表单的构建身份修复

日期：2026-09-13。范围：现有本地出版工作台；不改变审阅合同、审批状态或冻结内容。

## 复现与修复

上一轮 Vite 开发日志报告两个同级组件使用相同构建 ID 作为 React key。源码核对
确认是 `EditorialWorkspace` 内的 `RightsPanel` 与 `HumanReviewPanel`，并非
服务器重复创建了 BookBuild。

新增组件回归：在两个表单填写各自草稿，同构建重新渲染，然后改为另一构建。
修复前实际失败：切换后“权利依据”输入框数量从应有的 1 变为 2，旧权利表单
残留。修复为 `rights:<build-id>` 与 `human-review:<build-id>` 两个独立 key。

修复后的回归验证：同构建刷新保留两份草稿；新构建清空两份草稿且每个输入框
只存在一次；没有重复 key 错误，没有触发表单提交。没有通过移除构建身份来
掩盖问题，跨构建重置行为保留。

## 实际验收

- 先失败后通过的目标测试：`EditorialWorkspace.test.tsx`，最终 2 项通过。
- 前端全量：80 文件、320 项通过；lint、build、`git diff --check` 通过。
  构建仍有既存大 chunk 提示，未扩展到打包优化。
- 正式网关刷新后打开 Gin 从零自学项目，在“权利依据”和“真人审校范围”填入
  明确标注“不提交”的临时草稿，切换到熟悉技术栈项目再返回。两个字段各一份且
  都为空；控制台 error/warn 列表为空。未点击登记或审校提交按钮。
- 当前构建 `9c4dfc67-532a-46f6-906a-80a2a7876e67` 的 editorial API
  验收前后完整 JSON 相等，证明此次操作没有新增权利项或审校记录。
- 同构建刷新草稿保留与不经卸载切换构建的边界由组件回归覆盖；真实页面覆盖
  当前项目导航，不把项目切换测试夸大为创建新冻结构建的验收。

## 本地部署

只更新 frontend，使用已有本地镜像，无网络构建和拉取。新镜像
`inkwords/frontend:editorial-form-identity-v1-20260913`，ID
`sha256:92757efba304260efab67d74fc43e4464a7ae55315368f1da6f7768211e0ed8f`。
服务健康，环境、挂载和隔离设置等价；网关 index 与本次构建字节相同。
旧 `workspace-scroll-v1-20260913` 镜像保留，回退需在原 Compose 链最后追加
证据目录内 `compose.rollback.yml`，仅替换 frontend。

证据目录：`output/real-acceptance/2026-09-13/editorial-form-identity/`，包含
测试/lint/build 日志、部署预检和结果、前后 editorial JSON、回退配置与
`verified-facts.json` 的网关 SHA-256。无 Provider、教学执行、依赖安装或提交推送。

## 计划边界与文档纠正

当前 r17 的 WPS 27 页打印校样与服务端 PDF 21 页，已有全部编号总览和密集页
放大检查，见 `r17-layout-review-2026-09-13.md`；早期“全篇待做”仅为历史检查点。
本轮读取并核对该记录，没有重复校样或新增版式通过结论。
动手代表章、真实六维学习与延迟保持、独立试学及正文/截图权利仍未完成。
