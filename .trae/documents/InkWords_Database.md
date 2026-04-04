# 墨言博客助手 (InkWords) 数据库设计文档

## 1. 数据库选型与规范

### 1.1 基础选型
- **数据库引擎**：PostgreSQL 14+。
- **字符集与排序规则**：强制使用 `UTF8` 编码，以完美支持全球多语言、特殊符号及 Emoji 的存储（尤其是博客正文内容）。

### 1.2 核心设计规范
- **主键策略 (Primary Key)**：核心业务表（如 `users`, `blogs`）使用分布式唯一 ID（例如**雪花算法 Snowflake**）或 `UUIDv4`，不使用数据库 `SERIAL` 自增 ID，避免 ID 被恶意猜测或爬取。
- **软删除机制 (Soft Delete)**：所有核心数据表增加 `deleted_at` (TIMESTAMP) 字段。删除历史博客时仅做逻辑删除，便于后续的数据追溯、恢复以及审计。
- **时间戳审计**：每张表必须包含 `created_at` 与 `updated_at` (TIMESTAMP) 字段。

## 2. 核心业务表设计 (E-R 关系)

### 2.1 实体关系说明
1. **User (用户)**：系统的核心访问实体。
2. **Blog (博客)**：用户生成的文章实体。一个 User 可以拥有多个 Blog (1:N)。
3. **父子系列关联 (Series/Chapter)**：为了支持大项目拆解，`blogs` 表内部设计了**自引用关联**。即，通过 `parent_id` 字段指向某个总系列的 ID。当 `parent_id` 为空时，表示这是一个单篇博客或一个系列的合集节点。
4. **OAuth Token (第三方授权)**：一个 User 可以绑定多个第三方平台的授权 (1:N)，以支持一键分发。

## 3. 表结构详细设计

### 3.1 用户表 (users)
存储使用系统的基本用户信息及付费体系属性。
- `id` (BIGINT/UUID, PK): 雪花 ID 或 UUID。
- `username` (VARCHAR): 用户名或昵称。
- `email` (VARCHAR, Unique): 绑定的邮箱。
- `github_id` / `wechat_openid` (VARCHAR, Nullable): 第三方登录唯一标识。
- `avatar_url` (VARCHAR): 头像链接。
- `subscription_tier` (SMALLINT): 会员等级（0: 免费版, 1: Pro, 2: 团队版）。
- `tokens_used` (INTEGER): 本月已使用的 Token 数量。
- `created_at`, `updated_at`, `deleted_at`: 标准时间戳 (TIMESTAMP)。

### 3.2 博客主表 (blogs)
核心业务表，存储生成的 Markdown 内容及大项目拆解结构。
- `id` (BIGINT/UUID, PK): 雪花 ID。
- `user_id` (BIGINT/UUID, FK): 关联作者。
- `parent_id` (BIGINT/UUID, Nullable): **父级系列 ID**。若为大项目生成的系列博客，此字段指向代表“整个系列”的根节点记录。若是单篇独立博客，则为 NULL。
- `chapter_sort` (INTEGER): **章节序号**。在大项目拆解场景中，表示当前文章在整个系列中的排序位置。
- `title` (VARCHAR): 博客或系列的标题。
- `content` (TEXT): **Markdown 正文内容**。因为单篇内容限制约 5000 字且包含 Mermaid 代码块，PostgreSQL 中的 `TEXT` 类型支持无限长度存储，确保容量充裕且不截断。
- `source_type` (VARCHAR): 生成来源（如 `github`, `pdf`, `word`, `markdown`）。在 Git 仓库拆解场景中，前端会默认下发标记为 `git` 或对应的 Git 平台。
- `status` (SMALLINT): 生成状态（0: 生成中, 1: 已完成, 2: 生成失败/中断）。
- `created_at`, `updated_at`, `deleted_at`: 标准时间戳 (TIMESTAMP)。

### 3.3 第三方授权表 (oauth_tokens)
用于管理用户在掘金、CSDN等平台的一键发文授权。
- `id` (BIGINT/UUID, PK)
- `user_id` (BIGINT/UUID, FK)
- `platform_type` (VARCHAR): 平台标识（如 `juejin`, `csdn`, `zhihu`）。
- `access_token` (TEXT): 经过 AES 加密存储的访问凭证。
- `refresh_token` (TEXT): 经过 AES 加密存储的刷新凭证。
- `expires_in` (INTEGER): Token 过期时间。
- `created_at`, `updated_at`, `deleted_at`

## 4. 索引与性能优化
- **外键约束**：在物理层不设置强外键（`user_id`、`parent_id`），由业务代码（Go Service）保障数据一致性，提升插入和更新性能。
- **组合索引**：在 `blogs` 表中，针对 `(user_id, parent_id, chapter_sort)` 建立组合索引（B-Tree），极大提升侧边栏（Sidebar）拉取“用户某项目系列文章”列表时的查询速度。
- **部分索引 (Partial Index)**：可以针对 `deleted_at IS NULL` 创建部分索引，在查询未删除记录时性能大幅提升。