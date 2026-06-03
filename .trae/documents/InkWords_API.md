# 墨言知识训练平台 (InkWords Trainer) - API 接口文档

## 0. 变更记录
- 2026-06-03：稳定性与工程化优化（Task 1-5）。本次不新增、不删除任何后端 API 路由或请求/响应字段；主要变更为：后端启动链路补齐显式 `http.Server` 与优雅停机，`/api/v1/stream/scan`、`/api/v1/stream/analyze`、`/api/v1/blogs/:id/continue` 等流式主链路恢复遵守请求取消语义（客户端断开后默认停止后台任务）；系列生成链路补齐“前置草稿创建/清理 + 章节完成落库 + Token 记账”的事务边界与可观测错误；前端对 SSE 401 统一收口为清 token 并返回登录页，不再强制 `location.reload()`。
- 2026-06-02：系列生成失败原因可视化与 SSE 稳定性修复。本次不新增、不删除任何后端 API 路由、请求字段或响应字段；前端开始消费并展示既有系列生成 SSE 事件中的 `status=error` 与 `message`，后端 `stream` handler 统一为生成/分析类流增加缓冲并在每次写事件后主动 `flush`，降低慢客户端导致的流式背压与误超时风险。
- 2026-06-01：文件来源 Analyze 链路新增“动态提示词 profile”锁定机制。`POST /api/v1/stream/analyze` 在完成大纲分析后会额外返回 `resolved_prompt_profile`（含 `key`、`display_name`、`document_kind`、`reason`）；`POST /api/v1/stream/generate` 请求新增 `prompt_profile_key`、`document_kind`，用于让单篇/系列生成沿用同一次 Analyze 已锁定的内容类型提示词。
- 2026-06-01：知识漫游复习会话升级为“文章驱动提问 + 结构化反馈”；`POST /api/v1/review/sessions` 与 `GET /api/v1/review/sessions/:id` 新增 `session_outline`、`current_round_goal`，`POST /api/v1/review/sessions/:id/respond` 新增 `review_feedback` 与 `current_round_goal`，用于明确返回本轮目标、命中点、遗漏点与下一步建议。
- 2026-05-29：工程化结构拆分 Phase 1：review 领域与 Sidebar/export 逻辑完成模块化拆分，生成链路辅助逻辑拆分为更小文件；本次不新增、不删除、不修改任何对外 API 路由或请求结构。
- 2026-05-29：知识漫游复习入口调整为“随机抽题 + 手动选文”双入口；`POST /api/v1/review/pick` 的后端实现改为从候选集中真正随机选题，不再固定返回首个符合条件的笔记。`GET /api/v1/review/today` 路由保留以兼容既有客户端，但当前前端主入口不再展示“今日推荐”卡片。
- 2026-05-29：生成器前端工作流改为 `选择来源 -> 配置解析 -> 确认大纲` 三步模型；解析/分析进度内嵌在“配置解析”，写作进度内嵌在“确认大纲”。本次不新增、不修改任何后端 API 路由或请求结构，但文件上传前端交互调整为“先完成 `/api/v1/project/parse`，再由用户在配置页显式触发 `/api/v1/stream/analyze` 生成大纲”，避免上传 ZIP/课件后跳过场景选择。
- 2026-05-28：工程规范收尾与提交前同步：本次未新增或删除后端 API 路由，但统一收紧了部分既有接口的外部错误输出约定。`/api/v1/blogs` 相关接口不再直接透出内部错误详情，`/api/v1/blogs/:id` 在目标不存在时明确返回 `404 blog not found`；流式接口 `/api/v1/stream/generate`、`/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish`、`/api/v1/stream/analyze`、`/api/v1/stream/scan` 的 SSE `error` 事件统一返回稳定错误文案，避免泄漏底层数据库/系统异常文本。
- 2026-05-27：项目定位升级为“墨言知识训练平台（InkWords Trainer）”，口号“把资料变成知识，把知识变成能力”；本次仅同步文档命名口径，不新增、不修改任何后端 API 路由或请求结构。
- 2026-05-27：前端新增 `HomeEntry` 引导入口、共享 `StepStrip` 步骤条，以及“同一时间只显示当前主步骤”的流程式工作台；本次仅调整前端编排与交互，不新增、不修改任何后端 API 路由或请求结构。
- 2026-05-27：新增“知识漫游复习”接口族 `/api/v1/review/*`，覆盖今日推荐、随机抽题、手动选文、会话创建、追问、提示、结束训练与最近记录查询；所有接口均要求 JWT Bearer Token。
- 2026-05-25：将 `/api/v1/project/parse` 的文件上传上限从 100MB 提升到 888MB，并同步更新前端文件选择校验与 Nginx `client_max_body_size`，避免网关层和应用层限制不一致。
- 2026-05-25：修复 AI 思考/对话式前言混入正文的问题；流式正文输出链路新增统一清洗，默认剥离 `<think>...</think>` 与“好的，收到你的需求 / 作为高级全栈架构师”等开头套话，`/api/v1/stream/generate`、`/api/v1/blogs/:id/continue`、`/api/v1/blogs/:id/polish` 均受此约束。
- 2026-05-25：修复前端“创作场景”在文件上传与大纲生成过程中的交互歧义；保持 `/api/v1/stream/analyze` 与 `/api/v1/stream/generate` 的 `scenario_mode` 请求结构不变，但前端在上传分析时改为读取最新场景值，并在大纲生成后锁定该场景（仅 UI/请求时机修复，无 API 路由变更）。
- 2026-05-25：修复系列生成异常时历史博客只剩父级导读的问题；`/api/v1/stream/generate` 的后端实现改为先为每个章节创建子博客草稿，再在流式成功后回填正文、失败时标记错误状态。API 路由与请求结构不变，但 `/api/v1/blogs` 返回的系列 `children` 在章节失败场景下也会保留占位子节点。
- 2026-05-24：`/api/v1/stream/analyze` 与 `/api/v1/stream/generate` 新增 `scenario_mode` 请求字段，支持 `ebook_interpretation`、`open_book_exam_review`、`beginner_walkthrough` 三种创作场景；后端缺省按来源兜底（`git -> beginner_walkthrough`，其它来源 -> `ebook_interpretation`）。
- 2026-05-21：`/api/v1/project/parse` 新增 ZIP 课件包解析能力；支持返回 `data.archive_summary`，用于展示压缩包扫描、保留、去重、忽略与失败统计。
- 2026-05-21：新增用户写作模板接口 `/api/v1/user/prompt-settings`（GET/PUT），并为 `/api/v1/stream/generate` 增加 `article_style` 请求字段，用于控制文章类型/写作要求模板。
- 2026-05-21：修复本地 PDF/Word/Markdown 上传后触发 `git_url is required for git source type` 弹窗的问题；前端在 `/api/v1/stream/analyze` 显式发送 `source_type=file`，后端增加基于 `source_content` 的文件来源兼容推断（无 API 路由变更）。
- 2026-04-29：新增“写博客”入口配套接口 `/api/v1/blogs/draft`（创建手写草稿）。
- 2026-05-08：写博客编辑器新增“语音输入”（纯前端能力，无 API 变更）。
- 2026-05-08：新增“博客润色”流式接口 `/api/v1/blogs/:id/polish`（SSE 输出润色草稿，不落库）。
- 2026-05-08：工程化整理（无 API 路由变更，主要为仓库文件治理与文档同步）。
- 2026-05-08：目录结构工程化调整落地（无 API 路由变更）。
- 2026-05-08：后端 Blog Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 User Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 Auth Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-08：后端 Stream/Project Domain 垂直切片迁移（无 API 路由变更）。
- 2026-05-10：修复导出到 Obsidian 时“初始化知识库目录失败”（兼容 Obsidian Local REST API 目录列表 `{ "files": [...] }` 返回格式；无 API 路由变更）。
## 1. 认证模块 (AuthAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/auth/captcha` | GET | 获取图形验证码 | 无 -> 返回 `{ captcha_id, image }` |
| `/api/v1/auth/register` | POST | 邮箱密码注册 | `{ username, email, password, captcha_id, captcha_value }` |
| `/api/v1/auth/login` | POST | 邮箱密码登录 | `{ email, password, captcha_id, captcha_value, remember_me }` -> 返回 `token` |
| `/api/v1/auth/oauth/:provider` | GET | 第三方授权跳转 (如 `github`) | 无 |
| `/api/v1/auth/callback/:provider` | GET | OAuth回调 | `code`, `state` -> 重定向至前端 (带 `token` 或 `bind_required` 等参数) |
| `/api/v1/auth/bind-github` | POST | GitHub 登录发现邮箱冲突时绑定本地账号 | `{ email, password, github_id, username, avatar_url }` -> 返回 `token` |

## 2. 用户模块 (UserAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/user/profile` | GET | 获取当前登录用户信息 | JWT Bearer Token |
| `/api/v1/user/profile` | PUT | 更新当前登录用户名 | `{ username }` |
| `/api/v1/user/avatar` | POST | 上传用户头像图片 | `multipart/form-data` -> `avatar` |
| `/api/v1/user/stats` | GET | 获取用户仪表盘统计数据 (Token, 费用, 字数, 技术栈) | JWT Bearer Token |
| `/api/v1/user/prompt-settings` | GET | 获取文章类型默认模板与当前用户自定义覆盖 | JWT Bearer Token |
| `/api/v1/user/prompt-settings` | PUT | 更新当前用户的写作要求模板覆盖（空字符串表示恢复默认） | `{ overrides: { [styleKey]: string } }` |

## 3. 项目解析模块 (ProjectAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/project/analyze` | POST | 解析 Git 仓库生成大纲 (Legacy) | `{ git_url, sub_dir }` |
| `/api/v1/project/parse` | POST | 解析本地文件或 ZIP 课件包并提取 `source_content` | `multipart/form-data` -> `file` (最大支持 888MB；支持 `.pdf/.docx/.md/.markdown/.txt/.zip`) |

### 3.1 `/api/v1/project/parse` 返回说明
- 普通文件上传时，成功响应保持兼容：`data.source_content`
- ZIP 课件包上传时，成功响应会额外返回：
  - `data.archive_summary.total_files`
  - `data.archive_summary.supported_files`
  - `data.archive_summary.kept_files`
  - `data.archive_summary.duplicate_files`
  - `data.archive_summary.ignored_files`
  - `data.archive_summary.failed_files`
  - `data.archive_summary.kept_paths`
- ZIP 解析会自动完成白名单筛选、内容去重、顺序聚合，并在“无有效文本文件”或“存在非法压缩路径”时返回错误。

## 4. 流式生成模块 (StreamAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/stream/scan` | POST | 快速扫描 Git 仓库一级目录并通过 README 智能提取描述 | `{ git_url }` -> SSE Stream |
| `/api/v1/stream/analyze` | POST | 实时流式拉取 Git 或解析长文本文件生成大纲 | `{ git_url, selected_modules, source_type, source_content, scenario_mode }` -> SSE Stream；当请求仅包含 `source_content` 且未传 `git_url` 时，后端会兼容判定为 `file` 来源；文件来源完成时会在结果中返回 `resolved_prompt_profile` |
| `/api/v1/stream/generate` | POST | 根据大纲或内容流式生成博客章节 | `{ source_content, source_type, git_url, outline, series_title, parent_id, article_style, scenario_mode, prompt_profile_key, document_kind }` -> SSE Stream |
| `/api/v1/blogs/:id/continue` | POST | 继续生成被截断的单篇博客 (Legacy) | 无 -> SSE Stream |
| `/api/v1/blogs/:id/polish` | POST | 对当前草稿全文润色并返回“润色草稿” | `{ title, content }` -> SSE Stream |

### 4.1 `scenario_mode` 场景字段说明
- 支持枚举：
  - `ebook_interpretation`：电子书解读
  - `open_book_exam_review`：开卷复习
  - `beginner_walkthrough`：小白教程
- 设计边界：
  - `scenario_mode` 决定“这次要产出什么任务形态”。
  - `article_style` 继续决定“内容以什么写法呈现”。

### 4.2 `/api/v1/stream/analyze` 请求补充说明
- 新增字段：`scenario_mode`
- 缺省兜底：
  - `git -> beginner_walkthrough`
  - `file` 及其它来源 -> `ebook_interpretation`
- 作用：
  - 控制大纲拆解偏向“章节解读 / 考点速查 / 学习路径”中的哪一种结构。
- 文件来源补充能力：
  - 当 `source_type=file` 时，后端会在大纲生成前先做一次轻量内容分类，为当前文件锁定最匹配的动态提示词 profile。
  - Analyze 完成事件中的 `content` 会额外包含 `resolved_prompt_profile`，结构如下：

```json
{
  "series_title": "《非暴力沟通》解读",
  "chapters": [],
  "resolved_prompt_profile": {
    "key": "psychology_communication_book",
    "display_name": "心理学经典解读",
    "document_kind": "psychology_communication",
    "reason": "已根据文件内容自动匹配提示词。"
  }
}
```

- 回退策略：
  - 若分类器不可用、内容为空、返回非法 key 或 JSON 解析失败，后端会按 `scenario_mode` 回退到默认 profile，并在 `reason` 中明确标记“已回退到默认提示词”。
- 前端交互约束：
  - 用户可在发起 Analyze 前手动切换 `scenario_mode`。
  - 大纲返回后，前端会锁定本次 Analyze 使用的 `scenario_mode`，隐藏选择器并以只读标签展示当前场景，避免“大纲按 A 分析、正文按 B 生成”的歧义。
  - 当返回 `resolved_prompt_profile` 时，前端会在大纲区额外显示“当前提示词类型”只读标签，并将该 profile 锁定到后续 Generate 请求。

### 4.3 `/api/v1/stream/generate` 请求补充说明
- 新增字段：`scenario_mode`
- 新增字段：`prompt_profile_key`、`document_kind`
- 作用范围：
  - 单篇生成
  - 系列章节生成
  - 系列导读生成
- 字段作用：
  - `prompt_profile_key`：指定当前生成链路沿用的动态提示词 profile key。
  - `document_kind`：记录当前文件被识别出的文档类别，便于前后端保持一致语义。
- 兼容策略：
  - 旧前端不传 `scenario_mode` 仍可调用，后端按 `source_type` 自动回填默认值。
  - 旧前端不传 `prompt_profile_key` 或传非法值时，后端会按 `scenario_mode` 自动回退到默认 profile，保证旧链路可继续工作。
- 前端约束：
  - 当本次任务已经生成大纲时，Generate 会沿用该次 Analyze 已锁定的 `scenario_mode`，不再允许用户在大纲生成后修改。
  - 当本次任务来自文件 Analyze，Generate 会同时沿用 Analyze 返回的 `resolved_prompt_profile.key` 与 `resolved_prompt_profile.document_kind`，避免“大纲像心理学解读、正文又回退成通用技术博客”的漂移。

### 4.4 流式正文清洗约束
- 适用范围：
  - `/api/v1/stream/generate`
  - `/api/v1/blogs/:id/continue`
  - `/api/v1/blogs/:id/polish`
- 清洗目标：
  - 剥离 `<think>...</think>` 思考标签块
  - 跳过 `reasoning_content`
  - 去除开头的对话式前言/角色自述，例如“好的，收到你的需求”“作为高级全栈架构师……”“你是一位文本解读专家……”
- 设计目标：
  - 用户最终看到和落库的正文应只包含 Markdown 正文内容，不应混入模型思考过程或对话式套话

### 4.5 `/api/v1/stream/generate` 系列章节阶段事件
- 适用范围：
  - 仅系列章节生成链路；单篇生成与系列导读仍沿用既有事件语义。
- 新增状态：
  - `understanding`：章节理解阶段开始
  - `drafting`：章节草稿生成阶段开始
  - `reviewing`：章节技术审稿阶段开始
  - `revising`：终稿补强准备阶段开始
  - `streaming`：仅终稿补强阶段持续输出正文 chunk
  - `usage`：终稿补强完成后返回本章节的 DeepSeek usage 与 Prompt Cache 命中统计
- `usage` 事件载荷：
  - `prompt_tokens`
  - `completion_tokens`
  - `prompt_cache_hit_tokens`
  - `prompt_cache_miss_tokens`
- 典型 `event: chunk` 载荷示例：

```json
{
  "status": "understanding",
  "chapter_sort": 1,
  "title": "Gin 路由"
}
```

```json
{
  "status": "streaming",
  "chapter_sort": 1,
  "title": "Gin 路由",
  "content": "### 1. 请求先进入 Engine\\n"
}
```

```json
{
  "status": "usage",
  "chapter_sort": 1,
  "prompt_tokens": 1200,
  "completion_tokens": 500,
  "prompt_cache_hit_tokens": 900,
  "prompt_cache_miss_tokens": 300
}
```
- 兼容说明：
  - 路由、请求体、`completed/error` 终态事件不变。
  - 旧前端即使暂未消费新增阶段，也仍可通过 `streaming/completed/error` 维持基本链路。

## 5. 知识漫游复习模块 (ReviewAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/review/today` | GET | 获取今日推荐复习题卡 | JWT Bearer Token |
| `/api/v1/review/pick` | POST | 手动随机抽一篇可复习文章 | JWT Bearer Token |
| `/api/v1/review/notes` | GET | 获取可手动选择复习的文章列表 | `query`, `series_title`, `page`, `page_size` |
| `/api/v1/review/history` | GET | 获取最近复习记录摘要 | `limit` |
| `/api/v1/review/sessions` | POST | 创建一次复习会话 | `{ note_path, mode, entry_type }` |
| `/api/v1/review/sessions/:id` | GET | 获取复习会话当前状态与轮次 | 路径参数 `id` |
| `/api/v1/review/sessions/:id/respond` | POST | 提交一轮回答并推进会话 | `{ answer }` |
| `/api/v1/review/sessions/:id/hint` | POST | 请求一条提示 | `{}` |
| `/api/v1/review/sessions/:id/finish` | POST | 主动结束当前训练 | `{}` |

### 5.1 题卡与候选列表字段
- `GET /api/v1/review/today` 与 `POST /api/v1/review/pick` 返回：
  - `note_path`: Obsidian 笔记路径
  - `title`: 题卡标题
  - `source_title`: 所属来源或系列标题
  - `review_reason`: 推荐原因
  - `estimated_minutes`: 预估耗时
  - `available_modes`: 可选训练模式数组（`light_recall` / `detailed_qa`）
- `GET /api/v1/review/notes` 返回：
  - `items[].note_path`
  - `items[].title`
  - `items[].source_title`
  - `items[].last_reviewed_at`
  - `items[].preferred_mode`
  - `total`, `page`, `page_size`

### 5.2 会话与反馈字段
- `POST /api/v1/review/sessions`、`GET /api/v1/review/sessions/:id` 返回：
  - `session_id`, `status`, `mode`, `title`
  - `opening_prompt`: 开场提问
  - `initial_hints`: 初始提示列表
  - `session_outline.summary`: 当前文章的复习摘要
  - `session_outline.main_question`: 当前文章主问题
  - `session_outline.core_concepts / process_steps / application_cases / checkpoints`: 会话提炼出的文章关键点
  - `current_round_goal`: 当前这一轮最应该完成的回答目标
  - `latest_review_feedback`: 最近一轮回答的结构化判定（仅会话详情接口在有回答后返回）
  - `next_question`: 下一轮问题（可选）
  - `turn_index`: 当前轮次
  - `turns[]`: 已落库的轮次记录（仅会话详情接口返回）
- `POST /api/v1/review/sessions/:id/respond` 返回：
  - `session_id`, `session_status`, `turn_index`
  - `stage_feedback`: 当前阶段反馈（可选）
  - `current_round_goal`: 下一轮或当前轮的目标提示
  - `review_feedback.judgement`: 当前回答的判定（如 `答对较多 / 部分答对 / 偏题`）
  - `review_feedback.hit_points`: 当前回答已命中的文章关键点
  - `review_feedback.missed_points`: 当前回答尚未覆盖的关键点
  - `review_feedback.suggestion`: 下一步补充建议
  - `next_question`: 下一轮问题（可选）
  - `completed`: 是否已结束
  - `final_feedback.summary / strengths / gaps / next_focus`
- `POST /api/v1/review/sessions/:id/hint` 返回：
  - `session_id`, `hint_text`, `remaining_hint_count`
- `POST /api/v1/review/sessions/:id/finish` 返回：
  - `session_id`, `session_status`
  - `final_feedback.summary / strengths / gaps / next_focus`

### 5.3 复习枚举约束
- `entry_type`：
  - `today`：今日推荐入口
  - `manual_random`：手动随机抽题入口
  - `manual_select`：手动选文入口
- `mode`：
  - `light_recall`：轻提示复述
  - `detailed_qa`：细致问答
- `status`：
  - `created`
  - `in_progress`
  - `completed`
  - `abandoned`

## 6. 博客管理模块 (BlogAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/blogs` | GET | 获取用户的博客历史列表 (含系列结构) | 无 |
| `/api/v1/blogs/draft` | POST | 创建一篇手写草稿（顶级单篇，便于进入编辑器手写写作） | 无 |
| `/api/v1/blogs/:id` | PUT | 更新博客内容 (标题、内容等) | `{ title, content }` |
| `/api/v1/blogs` | DELETE | 批量删除博客 | `{ blog_ids: [] }` |
| `/api/v1/blogs/:id/export` | GET | 将系列博客或单篇博客导出为 Markdown Zip 包 | 无 -> application/zip |
| `/api/v1/blogs/:id/export/pdf` | GET | 将系列博客导出为合并 PDF（封面 + 目录 + 正文，无页码） | 无 -> application/pdf |
| `/api/v1/blogs/:id/export/obsidian` | POST | 将单篇博客导出到本地 Obsidian Vault（通过 Obsidian Local REST API） | 无 -> JSON `{ code: 200, message: "success" }` |
| `/api/v1/blogs/:id/export/obsidian/series` | POST | 批量同步系列到 Obsidian（遵循 Karpathy LLM Wiki Pattern：生成 sources/concepts/entities 并更新 index/log/hot；通过 Obsidian Local REST API） | 无 -> JSON `{ code: 200, message: "success" }` |
