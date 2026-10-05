# 教学代码验证工作台真实验收

日期：2026-09-10。代码、服务部署与浏览器实测均完成；完整开发计划继续执行。

## 改动与回归

新增按章节/工件读取既有验证任务的 GET，复用 workspace 幂等查询与冻结载荷校验。前端增加显式启动、定时状态查询、失败重试、活动任务取消及页面恢复；终态刷新真实证据。工件卡片增加 r12/r6 等来源修订与当前/历史标记，防止对同名 main.go 选错工件。完整合同见 `docs/decisions/textbook-verification-workbench.md`。

- 先运行新的只读查询测试，因方法尚未存在真实失败，记录 `red.log`。
- core task、textbookartifact、textbook、HTTP routes 全部包通过（`go-focused.log`）；随后增加 GET 无副作用测试，相关验证、路由、Peer/Runtime/Textbook 架构检查通过（`architecture.log`）。
- 前端全量 74 文件 / 288 测试通过；后续修订标签的小改动由最终 22 项聚焦测试覆盖。最终 lint 和生产构建通过；构建仍有部分 chunk 大于 500 kB 的提示，没有因此声称体积优化。
- 组件回归覆盖首次只读、显式启动、重新挂载恢复、活动轮询与终态证据刷新、失败重试、显式取消、查询失败不自动执行、错误任务类型拒绝及来源修订标记。
- PostgreSQL 对现有查询实际 EXPLAIN ANALYZE：57 行表使用顺序扫描，匹配 1 行、排除 56 行，执行约 0.197 ms；未新增索引，未把小表顺扫误报为索引命中。详见 `query-plan.txt`。

## 部署与实际操作

依次部署 core-api `inkwords/core-api:verification-ui-v1-20260910` 与 frontend `inkwords-frontend:verification-ui-v2-20260910`，两者 healthy；export-service 保持 `pdf-layout-v5-20260910`。真实 Obsidian 挂载与 PDF 覆盖层保留，没有依赖安装或全局环境修改。

1. 实际 GET 恢复 r12 工件任务 `34037fcf-690f-484a-a53a-f4edcd241f29`，页面显示当前稿件代码来源、已完成任务和仍有效的已验证证据。该工件由当前批准 r13 继承，未重新运行。
2. 检查历史 r6 教学代码和冻结清单后，通过真实按钮首次提交工件 `7f9df5f4-304b-58c6-9578-1f1c051cef0c`，HTTP 202，新任务 `22e4c9d4-736c-464f-8095-fefb871e2f38`。
3. 实际失败原因为不可变工件目录 permission denied，发生在进入代码执行之前。预检时已知它冻结旧 Go 1.26.0，不能把这次更早的目录错误写成工具链检查结果。页面显示“验证任务失败”、原始原因、“重试同一验证”，工件仍为未验证。
4. 完整刷新页面、重新进入工作台与章节，当前和失败任务均恢复原 ID；期间验证 POST 数为 0，证据在 `browser-refresh-proof.txt`。
5. 真实点击失败重试一次，HTTP 202，任务 ID 未变，retry_count 为 1；同一权限问题仍使执行失败。没有修改权限、换命令、关闭沙箱或循环重试。`browser-retry-proof.txt` 保留真实结果。

两张截图已逐张查看：`output/playwright/verification-ui-current.png`、`output/playwright/verification-ui-failure.png`，当前/历史身份、按钮、状态与错误文字均可读。取消功能由组件与现有后端合同测试覆盖；本轮未对一个实际运行中的任务执行取消，不能宣称真实取消验收通过。

## 未变事实与未完成项

操作前后 API 回读：批准 r13 `1ee15e29-1304-4978-9e6f-d41421798c8b`、全部修订与既有运行证据不变；当前构建 `70aadfa4-32d1-4c87-b235-e76ca5678197` manifest 仍为 `sha256:be7d2e46427bef674ead12e42a8091d3c3723d0ee73811988f32061a6a10fa3f`。无 Provider 调用、学习作答或 Git 提交。当前出版预检仍 false。

已取消任务的恢复、成功任务证据过期后的重新验证、历史工件目录可读性仍需后续处理。r13 的新排版/整书复审、权利、多读者与实际学习等原计划项也未完成。未运行全仓 Go 回归；本次只声明上述已覆盖范围。

全部本轮运行收据保存在 `output/real-acceptance/2026-09-10/verification-ui/`，摘要 `final-facts.json`。最终 `git diff --check` 通过。
