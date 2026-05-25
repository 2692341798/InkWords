# 墨言博客助手 (InkWords) - API 接口文档

## 0. 变更记录
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
| `/api/v1/project/parse` | POST | 解析本地文件或 ZIP 课件包并提取 `source_content` | `multipart/form-data` -> `file` (最大支持 100MB；支持 `.pdf/.docx/.md/.markdown/.txt/.zip`) |

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
| `/api/v1/stream/analyze` | POST | 实时流式拉取 Git 或解析长文本文件生成大纲 | `{ git_url, selected_modules, source_type, source_content, scenario_mode }` -> SSE Stream；当请求仅包含 `source_content` 且未传 `git_url` 时，后端会兼容判定为 `file` 来源 |
| `/api/v1/stream/generate` | POST | 根据大纲或内容流式生成博客章节 | `{ source_content, source_type, git_url, outline, series_title, parent_id, article_style, scenario_mode }` -> SSE Stream |
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

### 4.3 `/api/v1/stream/generate` 请求补充说明
- 新增字段：`scenario_mode`
- 作用范围：
  - 单篇生成
  - 系列章节生成
  - 系列导读生成
- 兼容策略：
  - 旧前端不传 `scenario_mode` 仍可调用，后端按 `source_type` 自动回填默认值。

## 5. 博客管理模块 (BlogAPI)
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
