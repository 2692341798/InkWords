# InkWords 知识漫游复习设计

## 1. 背景与目标
- 当前 InkWords 已经形成较完整的内容生产与沉淀主链路：`解析素材 -> 生成系列博客/解读/复习资料 -> 导出到 Obsidian`。
- 当前问题不在于“没有内容”，而在于“生成后的内容缺少再次调起与主动回忆机制”，用户很容易把知识库当作存档仓库，而非能力内化系统。
- 本设计的目标是在不推翻现有主链路的前提下，为 InkWords 新增一条“知识漫游复习”闭环，让系统能够从 Obsidian 知识库中抽取可复习内容，通过费曼式复述与温和追问，倒逼用户输出，从而强化记忆、理解与迁移能力。

## 2. 范围与非范围
### 2.1 本次范围（做）
- 新增一级业务能力：`知识漫游复习`。
- 支持三个入口：
  - `今日推荐`：每天给出一篇推荐复习内容。
  - `手动随机抽一篇`：用户随时主动开始漫游。
  - `手动选择文章复习`：用户从可复习笔记池中主动指定一篇文章开始训练。
- 首版只从 Obsidian `wiki/concepts/` 中筛选可复习笔记。
- 支持两种训练模式：
  - `light_recall`：轻提示复述
  - `detailed_qa`：细致提问
- 新增独立后端领域 `review`，管理抽题、会话、提示、反馈与记录。
- 新增会话记录与轮次记录表，沉淀复习过程数据。
- 给出前端页面结构、后端 API、数据库表设计、Obsidian frontmatter 扩展、提示词策略与实施顺序。

### 2.2 本次非范围（不做）
- 首版不将 `wiki/sources/`、`wiki/entities/`、`wiki/meta/` 纳入随机复习池。
- 首版不实现复杂的间隔重复算法（Spaced Repetition）或精细权重引擎。
- 首版不强制接入语音输入，仅支持文本作答。
- 首版不实现积分、排行、打卡榜或社交分享。
- 首版不做百分制评分，不将产品设计成考试工具。
- 首版不强制把全部历史 Obsidian 笔记迁移到新 frontmatter 结构。

## 3. 设计原则
- **输出优先**：复习不是再次阅读，而是让用户先讲出来，再补盲点。
- **温和陪练**：系统默认扮演“陪练教练”，不使用高压审判式措辞。
- **受控随机**：用户感知为漫游，系统内部做最小筛选，避免抽到垃圾题或重复题。
- **知识源单一**：Obsidian 继续作为知识正文源，PostgreSQL 只存复习会话和结果，不复制整库正文。
- **最小闭环优先**：先跑通 `抽题 -> 复述/提问 -> 反馈 -> 记录`，再做高阶调度。
- **可演进架构**：首版 API 和数据结构从一开始就为后续“更聪明的推荐”“写回复盘”“聊天式导师”预留边界。

## 4. 用户问题与产品定位
### 4.1 用户问题
- 用户已经能生产大量系列博客、电子书解读和复习资料，但这些内容进入 Obsidian 后缺少复用机制。
- 被动阅读知识库很难形成长期记忆，更难验证是否真正理解。
- 如果系统只是再次展示原文，用户会继续停留在“知道我看过”，而不是“我能讲明白”。

### 4.2 产品定位
- 新能力的定位不是“随机看一篇笔记”，而是“随机调起一篇值得练的知识卡，让用户用自己的话重新讲出来”。
- InkWords 因此从“写作助手”扩展为“知识复用与能力内化平台”。

建议的一句话定义：

```text
InkWords 不仅帮用户产出内容，还会把已经沉淀到 Obsidian 的知识重新拉回前台，通过费曼式复述与追问，把知识从存档变成能力。
```

## 5. 方案选择
- 采用 **方案 A：轻量漫游复习作为 MVP，独立 review 域实现，并为后续调度与导师模式预留扩展点**。
- 不采用“直接在现有生成器页面里硬塞复习功能”的原因：
  - 生成与复习属于不同业务心智模型；
  - 复习需要独立的会话状态与历史记录；
  - 强塞到生成器会让入口与目标混淆。
- 不采用“首版直接上复杂间隔重复系统”的原因：
  - 当前更需要先验证用户是否愿意使用这条复习闭环；
  - 复杂调度会显著抬高建模、推荐和数据治理成本。
- 不采用“首版直接做聊天式导师”的原因：
  - 对话状态与 prompt 策略会大幅复杂化；
  - 在没有验证基本复习留存前，不宜同时引入太多交互变量。

## 6. 信息架构与页面流程
### 6.1 信息架构
- 现有一级能力继续保留：`智能生成博客`、`历史博客`、`仪表盘`。
- 新增一级能力：`知识漫游复习`。

`知识漫游复习` 首版页面包含四个主块：
- `今日推荐`
- `开始漫游`
- `选择文章复习`
- `最近复习记录`

### 6.2 页面状态
首版主流程建议控制在 5 个状态：

1. `空状态`
- 文案解释复习能力。
- 提供 `开始今日复习`、`随机抽一篇`、`选择文章复习` 三个按钮。

2. `抽题卡`
- 展示标题、所属系列、预计耗时、推荐理由。
- 让用户选择 `轻提示复述` 或 `细致提问`。

3. `训练中`
- 用户根据模式进行一轮或多轮文本输入。
- 原文不先展示，只在需要时逐步给轻提示。

4. `反馈页`
- 展示用户讲清楚的部分、遗漏的部分、下一次优先补的点。

5. `结束动作`
- 提供 `查看原文精华`、`记录这次复习`、`再抽一篇` 等动作。

### 6.3 关键交互约束
- 默认不在训练开始前展示原文全文。
- 先鼓励输出，再给提示，再给总结。
- 单次只练一篇，降低启动与完成门槛。
- 所有前端展示文案统一使用中文。

### 6.4 手动选择文章复习
- 除了“系统推荐”和“随机抽题”，首版还应支持用户主动选择某一篇文章进入复习。
- `选择文章复习` 的页面形态建议为轻量列表或抽屉，而不是复杂知识库浏览器。
- 首版只展示通过后端筛选后的可复习 `concept` 页面，避免用户选到索引页、seed 页或系统页。
- 列表项建议至少展示：
  - 文章标题
  - 所属系列
  - 最近是否复习过
  - 推荐模式（如有）
- 列表支持最小可用检索：
  - 按标题关键字搜索
  - 按系列标题做基础筛选
- 用户从列表中点选某篇文章后，后续流程与“今日推荐/随机抽题”一致，仍然先进入题卡，再选择训练模式，再创建 session。

## 7. 抽题池与受控随机策略
### 7.1 首版抽题范围
- 只从 `wiki/concepts/` 抽题。
- 默认不从 `sources/` 抽题，因为 source 常是长篇父文，不适合轻量主动回忆。

### 7.2 默认排除规则
后端统一排除以下内容：
- `wiki/index.md`
- `wiki/hot.md`
- `wiki/log.md`
- 任意 `_index.md`
- 非 `concept` 类型页面
- 空白页
- 仅包含模板化占位内容的 seed 页
- 字数过少、信息密度明显不足的页面
- 显式在 frontmatter 中声明不参与复习的页面
- 最近刚复习过、且未达到最小间隔的页面

### 7.3 seed 占位页识别
- 现有 Obsidian 导出会自动生成一些 seed 类型或模板化概念页。
- 这类页面正文可能只有 `Context extracted from ...` 一类内容，不具备有效复述价值。
- 复习筛选器必须识别模板化正文、低字数和低信息量，避免用户抽到无效题目。

### 7.4 推荐优先级
首版采用轻量受控随机：
- 优先：最近导入但从未复习过的内容。
- 其次：较久未复习的内容。
- 再其次：上次表现一般或较弱的内容。
- 最后：随机补位。

### 7.5 推荐理由
每次抽题卡需要展示简短推荐理由，例如：
- `这是你最近导入但还没复习过的一篇内容。`
- `这篇内容你已经有一段时间没回顾了。`
- `你上次对这个主题的复述还可以更完整。`

## 8. 两种训练模式设计
### 8.1 `light_recall` 轻提示复述
- 定位：低压力进入，先让用户尽量自己讲出来。
- 系统初始仅提供：
  - 一句鼓励式开场
  - 2 到 4 个轻提示
  - 一个明确动作：请先讲第一遍
- 用户卡住时可请求提示升级，但提示不能直接泄露原文核心结论。
- 建议一轮主回答 + 一次补充回答即结束，避免拖长。

### 8.2 `detailed_qa` 细致提问
- 定位：比轻提示模式更深入，但仍然保持“教练式”而非“审判式”。
- 固定为 3 轮最稳：
  1. 主旨轮：这篇文章最核心在讲什么。
  2. 细节轮：追问关键概念、步骤、关系或边界。
  3. 迁移轮：要求用户举例、对新手解释、说明适用与不适用场景。
- 每轮只推进一个重点问题，降低用户认知负担。

### 8.3 反馈维度
首版不输出总分，统一输出四类观察结果：
- 是否抓住主线
- 关键概念是否准确
- 表达是否清楚
- 是否出现例子或迁移

## 9. 总体技术架构
### 9.1 领域边界
- `frontend/` 新增独立复习页面与对应 hooks/services/store。
- `backend/` 新增独立 `review` 领域切片。
- Obsidian 继续作为知识正文源。
- PostgreSQL 继续作为训练记录与统计源。

### 9.2 后端目录建议

```text
backend/internal/domain/review/
  handler.go
  service.go
  repository.go
  dto.go
  picker.go
  session_manager.go
  feedback_builder.go
```

### 9.3 为什么独立 review 域
- `stream` 解决的是长文本生成与 SSE 输出。
- `review` 解决的是抽题、会话、提示、反馈、会话恢复与记录。
- 两者状态机与职责边界不同，不应混用。

## 10. API 设计
### 10.1 设计原则
- 题卡推荐与正式训练会话分离。
- 首版全部采用普通 JSON 接口，不引入 SSE。
- 只有用户真正开始训练时才创建 session。

### 10.2 接口列表
#### 10.2.1 `GET /api/v1/review/today`
- 获取今日推荐题卡。
- 不创建 session。

建议返回：

```json
{
  "note_path": "wiki/concepts/并发控制与速率限制.md",
  "title": "并发控制与速率限制",
  "source_title": "InkWords 内容生成平台架构解析系列",
  "review_reason": "这是你最近导入但还没复习过的一篇内容。",
  "estimated_minutes": 5,
  "available_modes": ["light_recall", "detailed_qa"]
}
```

#### 10.2.2 `POST /api/v1/review/pick`
- 用户手动随机抽一篇。
- 不创建 session。

建议请求：

```json
{
  "scope": "concepts",
  "exclude_recent_days": 3,
  "prefer_unreviewed": true
}
```

#### 10.2.3 `GET /api/v1/review/notes`
- 获取“可手动选择复习”的文章列表。
- 用途：为 `选择文章复习` 页面或抽屉提供候选数据。
- 不创建 session。

建议查询参数：

```text
query=并发
series_title=InkWords
page=1
page_size=20
```

建议返回：

```json
{
  "items": [
    {
      "note_path": "wiki/concepts/并发控制与速率限制.md",
      "title": "并发控制与速率限制",
      "source_title": "InkWords 内容生成平台架构解析系列",
      "last_reviewed_at": "2026-05-20T12:00:00Z",
      "preferred_mode": "light_recall"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

#### 10.2.4 `POST /api/v1/review/sessions`
- 基于题卡创建一次复习 session。

建议请求：

```json
{
  "note_path": "wiki/concepts/并发控制与速率限制.md",
  "mode": "light_recall",
  "entry_type": "today"
}
```

建议返回：

```json
{
  "session_id": "uuid",
  "status": "created",
  "mode": "light_recall",
  "title": "并发控制与速率限制",
  "opening_prompt": "先别看原文，试着用自己的话讲讲这篇内容。",
  "initial_hints": [
    "这篇内容主要在解决什么问题？",
    "它最核心的两个概念是什么？"
  ],
  "turn_index": 1
}
```

说明：
- `entry_type` 支持至少三种值：
  - `today`
  - `manual_random`
  - `manual_select`

#### 10.2.5 `GET /api/v1/review/sessions/:id`
- 获取会话详情，支持刷新或稍后继续。

#### 10.2.6 `POST /api/v1/review/sessions/:id/respond`
- 用户提交一轮回答。
- 返回下一问、阶段反馈或结束引导。

#### 10.2.7 `POST /api/v1/review/sessions/:id/hint`
- 用户请求更具体提示。
- 返回 `hint_text` 与剩余提示次数。

#### 10.2.8 `POST /api/v1/review/sessions/:id/finish`
- 显式结束一次训练。
- 返回最终总结、优点、缺口与下一步关注点。

### 10.3 可选二期接口
- `GET /api/v1/review/history`
- `POST /api/v1/review/sessions/:id/abandon`
- `POST /api/v1/review/sessions/:id/writeback`

## 11. 会话状态机设计
### 11.1 题卡与 session 分离
- `today/pick` 只负责题卡推荐。
- `notes` 只负责列出可手动选择复习的笔记。
- 用户从推荐、随机或手动选择中确定目标文章，并选择模式后，才创建 `review_session`。

### 11.2 状态枚举
- `created`
- `in_progress`
- `completed`
- `abandoned`

### 11.3 状态流转

```text
题卡推荐 -> 创建 session(created) -> 用户开始回答(in_progress)
  -> 完成反馈(completed)
  -> 或中途退出(abandoned)
```

### 11.4 模式推进策略
#### `light_recall`
- 创建 session
- 系统给开场语 + 轻提示
- 用户回答
- 系统给阶段反馈
- 用户可补充一次、请求提示或直接结束
- 完成后进入 `completed`

#### `detailed_qa`
- 创建 session
- 系统发主旨问题
- 用户回答
- 系统发细节问题
- 用户回答
- 系统发迁移问题
- 用户回答
- 系统汇总反馈并完成

## 12. 数据模型设计
### 12.1 `review_sessions`
用于记录一次完整训练的主记录。

建议字段：

```sql
id                  uuid primary key
user_id             uuid not null
note_path           text not null
note_title          text not null
source_title        text null
entry_type          varchar(32) not null
mode                varchar(32) not null
status              varchar(32) not null
review_reason       text null
estimated_minutes   int null

content_digest      text null
summary_snapshot    text null
key_points_snapshot jsonb not null default '[]'
metadata_snapshot   jsonb not null default '{}'

hint_used_count     int not null default 0
max_hint_count      int not null default 2
turn_count          int not null default 0

final_summary       text null
strengths           jsonb not null default '[]'
gaps                jsonb not null default '[]'
next_focus          jsonb not null default '[]'
feedback_tags       jsonb not null default '[]'

started_at          timestamptz not null default now()
completed_at        timestamptz null
abandoned_at        timestamptz null
created_at          timestamptz not null default now()
updated_at          timestamptz not null default now()
```

### 12.2 `review_turns`
用于记录一次训练中的轮次交互。

建议字段：

```sql
id                  uuid primary key
session_id          uuid not null references review_sessions(id)
turn_index          int not null
role                varchar(16) not null
turn_type           varchar(32) not null
content             text not null
evaluation_tags     jsonb not null default '[]'
extra_payload       jsonb not null default '{}'
created_at          timestamptz not null default now()
```

建议约束：

```sql
unique(session_id, turn_index)
```

### 12.3 二期可选表 `review_item_stats`
- 首版可不落地。
- 二期用于支持更聪明的受控随机与简单间隔策略。

建议字段：

```sql
id                    uuid primary key
user_id               uuid not null
note_path             text not null
review_count          int not null default 0
completed_count       int not null default 0
last_mode             varchar(32) null
last_reviewed_at      timestamptz null
last_completed_at     timestamptz null
last_result_level     varchar(32) null
consecutive_successes int not null default 0
suggested_next_at     timestamptz null
created_at            timestamptz not null default now()
updated_at            timestamptz not null default now()
```

### 12.4 数据边界
- Obsidian：存知识正文与可选复习 frontmatter。
- PostgreSQL：存复习 session、轮次、结果与长期统计。
- 首版不将 Obsidian 全量正文复制进数据库。

## 13. Obsidian 兼容与 frontmatter 扩展
### 13.1 首版兼容策略
- 如果笔记没有 `review` 字段，不强制迁移。
- 默认规则：
  - `type=concept` 且内容满足最低有效标准 -> 参与复习池
  - `type=source/entity/meta/domain` -> 默认不参与
  - `review.enabled=false` -> 强制不参与
  - `review.exclude_from_random=true` -> 不参与随机抽题

### 13.2 建议的 frontmatter 扩展

```yaml
review:
  enabled: true
  difficulty: medium
  preferred_mode: light_recall
  min_interval_days: 3
  exclude_from_random: false
```

### 13.3 字段语义
- `review.enabled`：是否参与复习
- `review.difficulty`：难度分级，二期可用于权重
- `review.preferred_mode`：更适合复述还是提问
- `review.min_interval_days`：最小抽题间隔
- `review.exclude_from_random`：不参与随机池
- `review.exclude_from_random`：不参与随机池
- 如果未来需要更细粒度控制，可增补 `review.allow_manual_select`；但首版不作为必填字段。

### 13.4 与现有导出链路的衔接
- 后续可以在 `ExportSeriesToObsidian` 写入 `concepts` 时，顺手附带默认 `review` frontmatter。
- 这样可以让“生成出来的知识卡”天然成为“可复习素材”。
- 这属于值得做的二期增强，但不是首版阻塞项。

## 14. Prompt 与追问策略
### 14.1 系统角色
- 固定为“温和陪练教练”。
- 不能在用户作答前直接暴露原文答案。
- 不能使用高压评价语气。
- 每轮只推进一个动作：提问、提示、追问、反馈或总结。

### 14.2 `light_recall` Prompt 骨架
- 开场：
  - `先别看原文，试着用自己的话讲讲这篇内容。你不需要一字不差，只要抓住主线。`
- 初始只输出：
  - 一句鼓励
  - 2 到 4 个轻提示
  - 一个明确行动指令
- 回答后的反馈结构：
  - 你已经讲到的部分
  - 还可以补的部分
  - 是否需要再给一点提示

### 14.3 `detailed_qa` Prompt 骨架
- 第 1 轮：先确认主旨
- 第 2 轮：再确认关键细节
- 第 3 轮：最后确认迁移应用
- 每轮之间只给短反馈，不做大段总结。

### 14.4 提示升级策略
- `Level 1`：方向提示
- `Level 2`：结构提示
- `Level 3`：关键点提示
- 提示逐步具体，但始终不直接贴原文结论。

### 14.5 最终反馈结构
首版最终反馈统一成三段：
- `你已经抓住的部分`
- `你还欠缺的部分`
- `下次优先补哪一点`

## 15. 错误处理与边界场景
### 15.1 API 错误建议
- `404`：`note_path` 不存在
- `400`：文章不符合复习条件（如空内容、系统页、字数太少）
- `403`：用户无权访问该 session
- `409`：session 已完成但继续作答，或提示次数已用尽
- `500`：Obsidian 读取失败、快照生成失败、内部异常

### 15.2 设计处理原则
- 对内保留根因，便于排查。
- 对外返回稳定中文错误信息。
- 不把 Obsidian 结构化异常直接暴露给前端。

## 16. MVP 切分与实施顺序
### 16.1 MVP 必做
- 独立入口 `知识漫游复习`
- `今日推荐` + `手动随机抽一篇` + `手动选择文章复习`
- 只抽 `wiki/concepts/`
- 两种模式：`light_recall` 与 `detailed_qa`
- 文本作答
- session 与 turn 记录
- 训练结束反馈
- 最近复习记录列表

### 16.2 可后补
- 写回复盘到 Obsidian
- 更聪明的受控随机
- 难度权重
- 下一次建议时间
- 最近薄弱主题再次推荐
- 语音输入

### 16.3 先不要做
- 百分制打分
- 排行榜
- 社交分享
- 复杂间隔重复算法
- 多人协作或比赛模式

### 16.4 推荐实施顺序
1. 新增后端 `review` 域骨架与基础 API。
2. 完成 Obsidian 复习笔记筛选器。
3. 完成前端独立页面与主训练流程。
4. 补最近复习记录页块。
5. 增加提示升级与手动随机优化。
6. 二期再考虑写回复盘到 Obsidian。

## 17. 验证方案
### 17.1 后端验证
- 题卡接口能稳定返回有效 `concept` 笔记。
- 不会抽到 `_index.md`、`hot.md`、`log.md`、seed 占位页。
- session 生命周期符合 `created -> in_progress -> completed/abandoned`。
- hint 次数与模式轮次约束正常工作。

### 17.2 前端验证
- 用户 10 秒内可以开始一次复习。
- 两种模式在文案和流程上差异明显。
- 页面刷新后可恢复未完成 session。
- 最近复习记录能正确展示最近完成内容。

### 17.3 端到端验证
- 生成或导入一批 `concept` 笔记后，可直接进入复习池。
- 完整跑通：
  - 获取题卡
  - 创建 session
  - 提交回答
  - 请求提示
  - 完成反馈
  - 查询最近记录

## 18. 风险与后续演进
### 18.1 主要风险
- 抽题规则过宽会让用户抽到无效题，直接破坏体验。
- 反馈如果太像考试，用户会感到压力，不利于长期坚持。
- 如果不做训练快照，原文更新可能导致同一次 session 的判断标准漂移。

### 18.2 对应缓解
- 先把筛选规则做好，再谈推荐算法。
- 首版坚持温和陪练语气，不输出百分制。
- session 创建时读取 Obsidian 内容并生成快照，保证会话内判断稳定。

### 18.3 演进方向
- 引入 `review_item_stats` 做简单间隔推荐。
- 支持将复习总结写回 Obsidian，形成“复习日志”。
- 支持按主题或系列做专题复习，而不只是一篇一篇抽。
- 在验证留存后，再评估是否演进为聊天式费曼导师。

## 19. 总结
- 本设计通过新增独立的 `知识漫游复习` 能力，为 InkWords 补上了从“知识生成”走向“知识内化”的关键一环。
- 首版坚持最小闭环：从 Obsidian `concepts` 中抽取值得练的知识卡，围绕复述与追问建立温和陪练式训练流程，并把训练结果沉淀到 PostgreSQL。
- 在此基础上，InkWords 的产品主链路将从：

```text
解析 -> 生成 -> 导出到 Obsidian
```

演进为：

```text
解析 -> 生成 -> 导出到 Obsidian -> 抽题复习 -> 主动输出 -> 反馈补盲
```

- 这条链路的核心价值在于：帮助用户把“看过、存过、生成过”的知识，转化为“能够讲清楚、能迁移、能使用”的真正能力。
