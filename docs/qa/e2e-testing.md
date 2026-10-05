# InkWords E2E 测试运行手册

## 环境契约

| 变量 | 用途 | 确定性测试默认值 |
| --- | --- | --- |
| `INKWORDS_E2E_BASE_URL` | Playwright 访问入口 | `http://127.0.0.1:18080` |
| `E2E_RUN_ID` | 数据和 Vault 子目录隔离标识 | `local` |
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
export E2E_EXTERNAL_MODE=stub
mkdir -p "${E2E_VAULT_PATH}/wiki/e2e/${E2E_RUN_ID}"
docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d --build
(cd frontend && npm run test:e2e)
docker compose -f docker-compose.yml -f docker-compose.e2e.yml down -v
rm -rf "${E2E_VAULT_PATH}"
```

失败时先保存 `docker compose ... logs --no-color`、Playwright report、trace、
截图和视频，再清理当前项目。测试日志不得包含 Token、API Key
或完整用户内容。

## Project Mastery Course 验收补充

项目精通课程的验收必须使用任务开始时解析得到的固定 commit SHA。测试夹具只允许使用
脱敏的仓库结构和短代码片段，不得执行被分析仓库的构建、测试、安装脚本或 hook。
蓝图阶段先验证文件处置、证据引用、覆盖率和依赖；只有批准后的蓝图才允许进入正文生成。

## 可自学教材基线补充

教材平台在接入真实 Provider、官网抓取或隔离执行器前，先运行 Gin 离线夹具：

```bash
cd backend
GOCACHE=/tmp/inkwords-go-build go test ./services/llm-stream/app/projectcourse \
  -run 'TestTextbookBaselineFixturePinsGinEvidenceAndQualityContract|TestOfflineInkWordsAcceptance' -count=1 -v
```

夹具固定 Gin `v1.12.0` 的 commit、短证据片段及允许的官方资料来源，不访问网络也不执行 Gin 仓库。它验证证据与质量门禁合同，不能替代真实模型生成、目标读者试学、运行验证或出版排版校对；这些验收在有明确 opt-in 条件时单独记录。

浏览器层还覆盖 `frontend/e2e/textbook-sample-chapter.spec.ts`：候选稿在人工应用前没有导出入口，获取编辑锁并应用后才出现 Markdown 与含谱系 ZIP 下载链接。该测试使用拦截的本地确定性 API，不会把 UI 成功误当作真实模型、运行器或出版验收。

## 真实验收

真实验收使用带有 `inkwords-real-acceptance` 标签的专用 self-hosted Runner，
并通过 `docker-compose.real.yml` 恢复 Obsidian REST API 代理；工作流要求
受保护的 GitHub Actions Environment `real-acceptance` 人工审批。该 Environment
提供专用 `DEEPSEEK_API_KEY` 和 Obsidian API Key；测试数据
只能写入 `wiki/e2e/<run-id>/`。

工作流输入 `mastery_model` 冻结本次评分模型。Environment 获批后，先运行
`TestRealAssessmentSeparatesRepresentativeAnswers`：对完整因果答案、明显遗漏答案和明确
误解答案各执行一次评分，共 3 次 Provider 调用；要求输出满足证据合同、Token 遥测已知，
并且三类答案达到预先固定的分数、遗漏和误解判定。日志只记录输入/请求 hash、聚合分数、
反馈项数量、Token 和延迟，不记录密钥或 Provider 原始响应。每个成功样本另写一份权限为
`0600` 的结构化 JSON，保留经过合同校验的逐项反馈、证据引用与调用元数据，供人工复核。
结果保存为 `mastery-assessment-acceptance.log` 和 `real-acceptance-artifacts/mastery/`；任一
样本不满足门槛都会使工作流失败，并停止尚未开始的后续评分调用。

真实评分和真实教材生成分别由 `run_mastery_assessment`、`run_textbook_generation` 控制，
默认均为 false，避免一次审批隐含未选择的 Provider 费用。启用教材生成时，
`textbook_model` 冻结目标模型；foundation、programming、stack_familiar 依次各调用一次，
前一版本失败即停止，不继续消费后续调用。每份输出必须通过当前硬门禁、保留人工复核要求、
标记运行状态为 unverified，并具有不同的请求和正文 hash。完整候选 Markdown、结构化质量
与 Provider 遥测写入 `real-acceptance-artifacts/textbook/` 并随工作流上传，不能仅凭日志
摘要宣称章节可批准。

如果基础读者版本已经通过产品内的显式重试生成并持久化，本次 dispatch 必须把
`textbook_audiences` 选择为 `programming,stack_familiar`，避免重复调用 foundation。
默认选项仍是三个 audience，适用于没有可复核基础版候选的独立验收。工作流只提供这两个
固定选择，不能通过任意输入扩大调用集合。

V1 是无登录的本地单用户应用。E2E 不再准备账号、JWT、验证码或 OAuth 会话；
真实模式只用于明确 opt-in 的 Provider/Obsidian 等外部集成，并继续受 Environment
人工审批约束。
