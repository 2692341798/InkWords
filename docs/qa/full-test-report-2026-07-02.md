# InkWords 全功能测试开发报告

## 结论

- 静态检查、单元测试、覆盖率、构建、包体积、确定性后端契约测试和浏览器回归均通过。
- 修复了首页在 Obsidian 不可用时的重复请求/错误刷屏，现在每次只发起一组请求，显示可读错误并由用户手动重试。
- review-service 不再向客户端泄露 Obsidian URL、`EOF` 等基础设施细节。
- 真实 DeepSeek、Obsidian 写入和 GitHub OAuth 的受保护工作流已建立，但本次未执行：当前没有专用 Secrets、`E2E_TEST_TOKEN`、真实验收 Runner 和人工 OAuth 授权。
- 后续使用本地 `.env` 启动了隔离真实验收栈，DeepSeek 任务已成功返回 `queued`；任务终态轮询与清理操作被平台执行用量限制拦截，因此不记为真实验收通过；Obsidian 写入和 OAuth 未执行。

## 新增测试能力

- Playwright：Chromium 桌面/移动核心流程，Firefox/WebKit 跨核冒烟，真实外部集成独立串行入口。
- 后端 `integration` build tag：ZIP 解析/去重/路径穿越、RabbitMQ generation 消息契约、PDF 产物契约、Obsidian HTTP 鉴权与读写往返。
- 隔离 Compose：独立 project、volume、network、Vault 和回环端口。
- CI：新增后端契约测试、Chromium E2E、跨浏览器冒烟和受保护的真实验收工作流。

## 实际验证

| 检查 | 结果 |
| --- | --- |
| `go test ./... -count=1` | 通过 |
| `go vet ./...` | 通过 |
| `./scripts/check_coverage.sh` | 通过：后端 37.90%，generation 54.20% |
| `go test -tags=integration ./integration` | 通过 |
| `npm test` | 通过：45 files / 168 tests |
| `npm run lint` / `npm run deadcode` | 通过 |
| `npm run test:coverage` | 通过：41.80 / 69.09 / 54.90 / 41.80 |
| `npm run build` / `npm run check:bundle` | 通过：主包 1,113,196 bytes，gzip 335,053 bytes |
| `npm run test:e2e` | 通过：7 passed / 1 expected skip |
| `npm run test:e2e:smoke` | 通过：Chromium、Firefox、WebKit 3/3 |
| E2E/real Compose 合并配置 | 通过 |
| GitHub Actions YAML / `git diff --check` | 通过 |
| 应用内浏览器 | 页面非空白、无框架错误层、错误提示与手动重试可见，当前 Vite 页面无新增 console error |

## 已知限制

- PR E2E 为确定性 UI 回归，外部 API 使用受控响应；Compose 健康和网关是独立门禁。
- 真实外部套件只能在 `real-acceptance` Environment 凭据完整且有人工审批时执行。在它通过前，不应声称 DeepSeek、Obsidian 和 GitHub OAuth 已完成本轮真实验收。
- 现有工作区在实施前已有未提交改动；本次未回滚、提交或推送任何文件。
