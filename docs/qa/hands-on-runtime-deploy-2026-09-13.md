# 动手章合同本地部署与真实生成预检

日期：2026-09-13。状态：四服务部署、真实页面蓝图批准和只读预检完成；未调用模型或执行教学代码。

## 部署及复核

将当前代码编译为本地 arm64 二进制，依次更新 llm-stream、export-service、course-runner、core-api。
镜像标签均为 `inkwords/<service>:hands-on-contract-v1-20260913`，沿用各服务实际运行镜像的
基础层，仅替换应用二进制。没有下载依赖或重建其它应用栈。core-api Dockerfile 补充打包
只读 `textbook-dependency-inspect`，实际容器中的帮助入口通过。

runner 使用旧沙箱镜像自带的 Go 1.26.8 离线编译，避免宿主机 Go 1.26.0 使 runtime.Version
与沙箱工具链不符。部署环境仅更新 TEXTBOOK_RUNNER_IMAGE_DIGEST 为新镜像实际 ID：
`sha256:ef16d4933854ef22748a7ccf7daeac3b3eeba30ecfff4f85697df9e33e6fa48f`。

每个服务更新前确认没有 pending/queued/running/streaming 任务；Compose 预检核对完整
环境键、挂载和隔离设置。四容器均 healthy，容器二进制 SHA-256 与本轮构建相等，runner
二进制编译版本为 go1.26.8，环境镜像指纹等于 Docker 实际镜像 ID。

原始部署脚本在 core-api 的后置检查停下，原因是 Docker Desktop 将相同 bind 路径从
`/host_mnt/Users/...` 改为 `/Users/...` 并调整列表顺序。统一路径表示与顺序后，四服务
的权限、限制、网络、用户、命令和挂载均等价。没有因此放宽隔离或再次重建服务。

## 真实页面与预检

通过真实浏览器刷新 localhost 网关并打开“Gin 从零自学教材”，核对两章、r6 蓝图中
动手章的五条关键事实和 17 个证据选择。按用户委托执行蓝图批准，页面确认“当前蓝图
r6 · 它是样章生成的当前前提”。这是 AI 代操作的生成前置审批，不是独立真人审校。
没有批准样章或填写真人审校记录。

打开“亲手搭建并测试 Gin 订单接口”，仍为 r0 空章；点击生成入口只取得预检，未点击
“确认并创建任务”。真实页面和 GET 网关响应一致：

- provider/model：deepseek / deepseek-v4-flash。
- task v3、prompt inkwords.textbook.sample.v17、quality inkwords.sample-quality.v10。
- 17 段证据，保守输入 17,213 / 20,000 Token，输出预留 12,000；within_budget=true。
- 缓存执行前未知；没有经版本化核对的价格表，因此费用未知。
- 冻结输入：`sha256:093544ddfb31258f5977579ed9a725b1b08995de2b35398a066e7f72ffa77366`。

## 证据与恢复

证据目录：`output/real-acceptance/2026-09-13/hands-on-runtime-deploy/`。
`images.json` 保存各服务原镜像、原 Compose 文件链及保护设置；`deployment-plan.json`
为预检；`deployment-verified.json` 为最终四服务验证；`generation-preflight.json`
为实际网关响应；各服务 build/deploy 日志保留原始结果。`deployment.json` 只包含原
脚本停下前的前三服务，最终结论以 deployment-verified.json 为准。

`compose.hands-on.yml` 与 `compose.rollback.yml` 分别保存本轮/原镜像和 runner 指纹。
恢复时先确认没有活动任务，再对每个服务沿用 images.json 中自己的原 Compose 文件链，
最后追加 rollback 文件，以 no-deps/no-build/pull never 逐项恢复并验证。未执行恢复，
未改数据库结构或删除数据。不能直接重跑 deploy.py apply：它要求部署前的镜像身份。

上一轮完整后端/架构回归已通过；本轮仅新增检查器的 Dockerfile 打包及运行部署，实际
编译与镜像内入口验证通过，未重复全量测试。没有声称实际 Gin 编译、测试或浏览器探针通过。

## 下一步

取得这份预检的一次有界真实生成授权，失败不自动重试。生成候选后仍需审阅、显式依赖
选择/工件投影和 go_test/browser_page 隔离证据。页面自动依赖选择尚未接入；许可、个人
学习、冷读者及整书剩余审校和原三项综合验收继续未完成。
