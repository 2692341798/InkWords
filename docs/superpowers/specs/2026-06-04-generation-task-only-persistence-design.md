# InkWords 生成链路 Task-Only 持久化设计

**类型**：Technical Design / Backend Architecture  
**范围**：`generation` 任务链路的最终结果持久化边界收口  
**日期**：2026-06-04  
**状态**：待评审

## 1. 背景

InkWords 当前已经完成以下微服务化基线：

- 生产形态固定为 `core-api / llm-stream / parser-service / export-service / review-service`
- 外部入口固定为 Nginx 单入口 `http://localhost`
- 生成、解析、导出三类长任务已经进入 `job_tasks + job_task_events + RabbitMQ` 的统一任务模型
- `core-api` 与 `llm-stream` 已完成第一轮深拆分，服务自有 `bootstrap / routes / cmd` 结构已经落地

但生成链路仍存在一个关键技术债：虽然任务控制面已经拆分，但最终业务事实仍未完全回收到 `core-api`。

当前遗留问题主要表现为：

- `llm-stream` 仍通过共享 use case 间接直写 `blogs / users`
- `job_tasks.result_json` 对生成任务而言仍然过弱，无法作为 `core-api` 的稳定业务事实来源
- `INKWORDS_TASK_PERSISTENCE_MODE=task_only` 已存在，但还没有成为真正可用的默认边界
- 系列生成链路中的章节草稿、导读正文、续写正文、失败状态和 token 记账仍依赖 legacy 直接写库路径

因此，这一轮的目标不是“再拆新服务”，而是把生成链路真正推进到 `task_only`：

- `llm-stream` 只负责执行生成与写任务控制面
- `core-api` 负责把任务结果转换成最终业务事实并落库到 `blogs / users`

## 2. 目标与非目标

### 2.1 目标

- 让 `llm-stream` 在生成成功后只写 `job_tasks.result_json` 与 `job_task_events`
- 让 `core-api` 成为 `blogs / users.tokens_used` 的唯一最终写入方
- 为 `generate_single / generate_series / continue` 建立稳定的 generation result schema
- 保持前端入口、SSE 实时体验、对外 `/api/*` 路径不变
- 保持现有 `INKWORDS_TASK_PERSISTENCE_MODE` 作为回滚开关

### 2.2 非目标

- 不新增新的微服务
- 不新增 Kafka、第二套 MQ 完成事件总线或服务网格
- 不在这一轮推进独立数据库实例拆分
- 不修改前端主交互和页面流程
- 不把 `polish` 自动纳入业务表最终持久化

## 3. 当前问题

### 3.1 边界问题

- `blogs / users` 的事实归属已经明确属于 `core-api`
- 但 `llm-stream` 通过共享业务逻辑仍在更新章节正文、父博客导读、续写内容和 token 记账
- 这使得“服务已拆分，写入却仍未收口”的状态长期存在

### 3.2 数据流问题

- 当前生成任务完成后，`result_json` 可能只保存 `{"done":true}` 一类弱信号
- `core-api` 虽然已有 [result_persister.go](file:///Users/huangqijun/Documents/墨言博客助手/InkWords/backend/services/core-api/domain/task/result_persister.go) 雏形，但没有足够稳定的结果结构可供消费
- 结果导致 `task_only` 只能停留在“实验开关”，还不能承担真正业务落库职责

### 3.3 系列链路问题

- 系列链路不仅要处理正文，还要处理：
  - 系列父节点
  - 章节草稿
  - 章节成功/失败/跳过状态
  - 系列导读
  - token 记账
- 当前这些写点分散在 `decomposition_generate*.go` 多个文件中，是最需要收口的一组 legacy 写库路径

## 4. 目标边界

### 4.1 `llm-stream` 允许写入

- `job_tasks.status`
- `job_tasks.result_json`
- `job_task_events`

### 4.2 `llm-stream` 禁止写入

- `blogs`
- `users.tokens_used`
- 任何其他最终业务事实表

### 4.3 `core-api` 负责写入

- 单篇生成最终博客正文
- 系列父节点导读正文
- 系列章节正文与章节终态
- 继续生成后的最终正文
- `users.tokens_used` 记账

### 4.4 过渡策略

- 系列父节点和章节草稿允许继续采用“预创建”模式，以兼容前端当前对稳定 `blog_id` 和系列树的依赖
- 但预创建草稿应被视为“占位业务事实”，而不是 `llm-stream` 最终写入结果的理由
- 最终正文、失败终态和 token 记账必须回收到 `core-api`

## 5. 统一 generation result schema

`job_tasks.result_json` 需要从“完成标记”升级为“最终业务事实快照”。

### 5.1 外层统一结构

```json
{
  "result_version": 1,
  "task_type": "generation",
  "task_subtype": "generate_single | generate_series | continue | polish",
  "persistence_mode": "task_only",
  "final_status": "succeeded | failed",
  "usage": {
    "estimated_tokens": 1234
  },
  "payload": {}
}
```

设计约束：

- `result_version` 用于后续 schema 演进与兼容
- `task_type` 和 `task_subtype` 与任务表保持一致，避免 `core-api` 再靠推断判断类型
- `persistence_mode` 明确写明当前结果对应的边界语义，便于审计和回滚
- `usage` 至少承载本轮可稳定得到的 token 估算或累计值

### 5.2 单篇生成 `generate_single`

```json
{
  "payload": {
    "blog_id": "uuid，可选",
    "title": "文章标题",
    "content": "最终 Markdown",
    "source_type": "file | git | manual",
    "word_count": 2345,
    "tech_stacks": ["Go", "Docker"]
  }
}
```

约束：

- 若任务创建前已预建草稿，则 `blog_id` 必填
- 若未预建，则 `core-api` 可基于结果创建新博客

### 5.3 系列生成 `generate_series`

```json
{
  "payload": {
    "parent_blog": {
      "blog_id": "系列父节点 uuid",
      "title": "系列标题",
      "content": "系列导读最终正文"
    },
    "chapters": [
      {
        "blog_id": "章节草稿 uuid",
        "chapter_sort": 1,
        "title": "第 1 章",
        "content": "最终 Markdown",
        "word_count": 1800,
        "tech_stacks": ["Gin", "PostgreSQL"],
        "status": "succeeded | failed | skipped",
        "error_message": ""
      }
    ]
  }
}
```

约束：

- `chapters` 中每一章都必须有显式 `status`
- 部分章节失败不应阻止整个系列写回已有成功章节
- `error_message` 只在失败时填写
- `parent_blog.content` 对应最终导读正文

### 5.4 继续生成 `continue`

```json
{
  "payload": {
    "blog_id": "目标博客 uuid",
    "appended_content": "新增正文",
    "final_content": "拼接后的完整正文"
  }
}
```

约束：

- `final_content` 是 `core-api` 持久化的直接事实来源
- `appended_content` 主要用于审计、调试和可选回放

### 5.5 润色 `polish`

```json
{
  "payload": {
    "title": "可选标题",
    "content": "润色预览正文"
  }
}
```

约束：

- `polish` 任务结果仍可写入 `result_json`
- 但 `core-api` 不自动将其写回 `blogs`
- 原因是当前产品语义仍然是“预览后由用户显式应用”

## 6. 目标数据流

### 6.1 单篇生成

1. 前端向 `core-api` 创建 generation task
2. `core-api` 写入 `job_tasks`
3. `llm-stream` 消费任务并持续写 `job_task_events`
4. `llm-stream` 生成最终正文并写入结构化 `result_json`
5. `core-api` 调用 `ResultPersister`，把结果写入 `blogs / users.tokens_used`
6. 任务进入成功终态

### 6.2 系列生成

1. 前端向 `core-api` 创建 generation task
2. `core-api` 可选预创建系列父节点和章节草稿，确保前端历史树和任务过程中存在稳定 `blog_id`
3. `llm-stream` 并发生成章节并持续写入事件流
4. `llm-stream` 汇总章节结果和导读结果到统一 `result_json`
5. `core-api` 根据结果对系列父节点、章节正文、失败终态和 token 记账做最终持久化
6. 前端对外体验与当前 SSE/历史树模型保持兼容

### 6.3 继续生成

1. `llm-stream` 只产出 `appended_content / final_content`
2. `core-api` 负责把 `final_content` 更新到目标博客
3. token 记账由 `core-api` 完成

### 6.4 润色

1. `llm-stream` 产出润色预览结果
2. `result_json` 只保存预览结果
3. `core-api` 不自动落库
4. 前端仍通过既有显式保存路径应用润色内容

## 7. 核心设计决策

### 7.1 为什么不直接继续使用共享 persistence 注入

- 共享 persistence 只能让直写点“更像接口化”，但不能改变 `llm-stream` 仍在写最终业务表的事实
- 这与当前服务写入归属矩阵不一致
- 对 InkWords 当前阶段来说，这只能算“隐藏耦合”，不能算真正收口边界

### 7.2 为什么不在这一轮引入完成事件二次消费

- 当前仓库已经有 `job_tasks.result_json` 和 `job_task_events` 作为跨服务事实面
- 若再新增一套 `generation.completed` 二次消费链路，会引入额外的重试、幂等、排障和运维复杂度
- 对当前目标而言，直接让 `core-api` 消费任务结果已经足够

### 7.3 为什么保留系列草稿预创建

- 前端当前对系列树与稳定 `blog_id` 仍有依赖
- 一次性取消预创建会把“边界收口”变成“前后端交互语义重做”
- 本轮以“最终正文与记账回收”为主，草稿预创建保留为受控过渡方案

### 7.4 为什么 `polish` 不自动落库

- 当前产品语义是“生成润色预览 -> 用户确认应用”
- 若自动持久化，会改变用户已有心智和前端交互
- 因此本轮只让 `polish` 享受任务结果 schema 统一化，不改变其最终业务写入语义

## 8. 文件级改动面

### 8.1 `llm-stream` 侧

重点文件：

- `backend/internal/domain/stream/task_consumer.go`
- `backend/internal/service/generator.go`
- `backend/internal/service/decomposition_generate.go`
- `backend/internal/service/decomposition_generate_persistence.go`
- `backend/internal/service/decomposition_generate_intro.go`
- `backend/internal/service/decomposition_generate_continue.go`

计划改动：

- 定义 generation result DTO
- 让单篇、系列、续写在成功完成时返回结构化结果
- 让 `task_consumer` 在 `MarkSucceeded(...)` 时写入完整 `result_json`
- 在 `task_only` 下彻底关闭对 `blogs / users` 的 legacy 直接写入

### 8.2 `core-api` 侧

重点文件：

- `backend/services/core-api/domain/task/result_persister.go`
- `backend/services/core-api/app/bootstrap/bootstrap.go`
- `backend/internal/domain/task/service.go`

可新增文件：

- `backend/services/core-api/domain/task/generation_result.go`
- `backend/services/core-api/domain/task/generation_result_repository.go`

计划改动：

- 将 `ResultPersister` 从空壳依赖接成真实写入用例
- 在 generation task 成功后显式消费 `result_json`
- 按 `task_subtype` 分发到单篇、系列、续写、润色四类持久化处理
- 将 token 记账统一收口到 `core-api`

### 8.3 测试面

重点测试：

- `task_consumer` 成功写入完整 generation result
- `task_only` 下单篇、系列、续写不再直写 `blogs / users`
- `ResultPersister` 能正确解析并持久化 `generate_single / generate_series / continue`
- `polish` 虽然有结果 schema，但不会自动写业务表

## 9. 实施顺序

虽然目标是“让生成链路真正进入 `task_only`”，但代码落地建议分三步进行。

### Step 1：打通 `generate_single`

- Why：单篇结果模型最简单，最适合先验证 schema 与 `ResultPersister` 闭环

### Step 2：打通 `continue`

- Why：只涉及已有博客的正文追加，持久化语义清晰

### Step 3：打通 `generate_series`

- Why：系列链路最复杂，涉及父节点、章节数组、部分失败、导读与 token 汇总

说明：

- 设计层面一开始就定义最终目标边界
- 代码层面按 `单篇 -> 续写 -> 系列` 递进，符合最小改动与可回滚原则

## 10. 风险与回滚

### 10.1 风险

- `result_json` schema 不稳定，导致 `core-api` 无法正确解析
- 系列部分成功、部分失败时，系列树与终态可能不一致
- 同一任务结果被重复持久化，造成重复记账或覆盖异常
- 预创建草稿与最终持久化切换过快，造成前端历史树回归

### 10.2 缓解策略

- 为 result schema 引入 `result_version`
- 要求系列章节必须显式包含 `status / error_message`
- `ResultPersister` 设计成幂等更新，以 `task_id / blog_id` 作为稳定事实锚点
- 保留系列草稿预创建的过渡形态，只回收最终正文和 token 记账

### 10.3 回滚策略

- 保留 `INKWORDS_TASK_PERSISTENCE_MODE`
- 若任务结果持久化闭环异常，可切回 legacy 模式
- 对外 API、前端路径和 SSE 协议保持不变，回滚仅限服务内部实现

## 11. 验证方案

### 11.1 自动化验证

- `llm-stream` 在任务成功后写入完整 `result_json`
- `task_only` 下不再直写 `blogs / users`
- `core-api` 能正确解析并落库 `generate_single / generate_series / continue`
- `core-api` 能正确累计 `users.tokens_used`
- `polish` 不自动落库

### 11.2 集成验证

```bash
cd backend && go test ./... -count=1
cd frontend && npm run build
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
curl -I http://localhost
curl -sS http://localhost/api/v1/ping
```

### 11.3 人工验证重点

- 单篇生成完成后，正文正常保存
- 系列生成完成后，父节点导读和章节正文正常保存
- 系列某章失败时，系列树仍存在，失败章状态清晰
- 继续生成后，正文被正确追加
- 润色仍只作为预览，不自动覆盖正文

## 12. 待实现后的文档同步要求

当实现真正开始后，如涉及 API、数据库语义、任务 schema 或 Compose 行为变化，需要同步更新：

- `README.md`
- `.trae/documents/InkWords_API.md`
- `.trae/documents/InkWords_Architecture.md`
- `.trae/documents/InkWords_Database.md`
- `.trae/documents/InkWords_Development_Plan_and_Log.md`
- `.trae/documents/InkWords_Conversation_Log.md`

## 13. 结论

对 InkWords 当前阶段而言，最合适的推进路线不是“再拆更多服务”，而是把已经存在的任务中心和多服务边界真正做实：

- `llm-stream` 只负责生成执行与任务控制面写入
- `core-api` 负责把任务结果持久化成最终业务事实
- `task_only` 从实验开关升级为真实边界

这条路线与现有 `5` 服务形态、单入口网关、任务中心基础设施、写入归属矩阵以及当前前端交互语义保持一致，是当前最小改动、最可验证、最适合继续推进微服务化的方案。
