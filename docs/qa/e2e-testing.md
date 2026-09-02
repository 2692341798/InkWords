# InkWords E2E 测试运行手册

## 环境契约

| 变量 | 用途 | 确定性测试默认值 |
| --- | --- | --- |
| `INKWORDS_E2E_BASE_URL` | Playwright 访问入口 | `http://127.0.0.1:18080` |
| `E2E_RUN_ID` | 数据和 Vault 子目录隔离标识 | `local` |
| `E2E_AUTH_MODE` | `ui`、`bypass` 或 `real` 鉴权模式 | `ui` |
| `E2E_EXTERNAL_MODE` | `stub` 或 `real` 外部集成模式 | `stub` |
| `E2E_PORT` | 仅绑定到回环地址的网关端口 | `18080` |
| `E2E_VAULT_PATH` | 可安全丢弃的临时 Vault 绝对路径 | 必填 |

Compose 的项目名必须包含当前运行标识，例如
`inkwords-e2e-${E2E_RUN_ID}`。命名卷和网络由 Compose 项目名隔离；Vault
仅允许指向本次运行新建的临时目录。清理时只执行当前项目的 `down -v`，
不得删除共享 Vault 或其他 Compose 项目。

## 本地确定性测试

```bash
export E2E_RUN_ID="local-$(date +%s)"
export COMPOSE_PROJECT_NAME="inkwords-e2e-${E2E_RUN_ID}"
export E2E_PORT=18080
export E2E_VAULT_PATH="/tmp/${COMPOSE_PROJECT_NAME}/vault"
export INKWORDS_E2E_BASE_URL="http://127.0.0.1:${E2E_PORT}"
export E2E_AUTH_MODE=ui
export E2E_EXTERNAL_MODE=stub
mkdir -p "${E2E_VAULT_PATH}/wiki/e2e/${E2E_RUN_ID}"
docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d --build
(cd frontend && npm run test:e2e)
docker compose -f docker-compose.yml -f docker-compose.e2e.yml down -v
rm -rf "${E2E_VAULT_PATH}"
```

失败时先保存 `docker compose ... logs --no-color`、Playwright report、trace、
截图和视频，再清理当前项目。测试日志不得包含 Token、API Key、OAuth code
或完整用户内容。

## Project Mastery Course 验收补充

项目精通课程的验收必须使用任务开始时解析得到的固定 commit SHA。测试夹具只允许使用
脱敏的仓库结构和短代码片段，不得执行被分析仓库的构建、测试、安装脚本或 hook。
蓝图阶段先验证文件处置、证据引用、覆盖率和依赖；只有批准后的蓝图才允许进入正文生成。

## 真实验收

真实验收使用带有 `inkwords-real-acceptance` 标签的专用 self-hosted Runner，
并通过 `docker-compose.real.yml` 恢复 Obsidian REST API 代理；工作流要求
受保护的 GitHub Actions Environment `real-acceptance` 人工审批。该 Environment
提供专用 `DEEPSEEK_API_KEY`、`E2E_TEST_TOKEN`、GitHub OAuth 凭据和 Obsidian API Key；测试数据
只能写入 `wiki/e2e/<run-id>/`。

GitHub OAuth 的 2FA/授权确认无法在无交互的托管 Runner 中完成。自动化负责
验证授权 URL、callback 错误和已准备会话后的应用状态；首次授权需在可交互、
专用测试账号环境中执行并保留脱敏证据。
