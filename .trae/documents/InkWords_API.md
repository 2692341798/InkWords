# 墨言博客助手 (InkWords) - API 接口文档

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

## 3. 项目解析模块 (ProjectAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/project/analyze` | POST | 解析 Git 仓库生成大纲 (Legacy) | `{ git_url, sub_dir }` |
| `/api/v1/project/parse` | POST | 解析本地文件生成大纲 | `multipart/form-data` -> `file` (最大支持 100MB) |

## 4. 流式生成模块 (StreamAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/stream/scan` | POST | 快速扫描 Git 仓库一级目录并通过 README 智能提取描述 | `{ git_url }` -> SSE Stream |
| `/api/v1/stream/analyze` | POST | 实时流式拉取 Git 或解析长文本文件生成大纲 | `{ git_url, selected_modules, source_type, source_content }` -> SSE Stream |
| `/api/v1/stream/generate` | POST | 根据大纲或内容流式生成博客章节 | `{ source_content, source_type, git_url, outline, series_title, parent_id }` -> SSE Stream |
| `/api/v1/blogs/:id/continue` | POST | 继续生成被截断的单篇博客 (Legacy) | 无 -> SSE Stream |

## 5. 博客管理模块 (BlogAPI)
| 接口地址 | 请求方法 | 功能描述 | 参数 |
| -------- | -------- | -------- | ---- |
| `/api/v1/blogs` | GET | 获取用户的博客历史列表 (含系列结构) | 无 |
| `/api/v1/blogs/:id` | PUT | 更新博客内容 (标题、内容等) | `{ title, content }` |
| `/api/v1/blogs` | DELETE | 批量删除博客 | `{ ids: [] }` |
| `/api/v1/blogs/:id/export` | GET | 将系列博客或单篇博客导出为 Markdown Zip 包 | 无 -> application/zip |
