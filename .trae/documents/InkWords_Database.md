# 墨言博客助手 (InkWords) - 数据库设计文档

## 1. 数据库选型
- **类型**: 关系型数据库 (RDBMS)
- **引擎**: PostgreSQL 14+
- **ORM**: GORM (Go)
- **连接字符串**: `postgres://inkwords:inkwords_password@db:5432/inkwords_db?sslmode=disable`
- **挂载卷**: Docker volume `pgdata` 持久化至 `/var/lib/postgresql/data`。

## 2. 表结构化设计

### 2.1 用户表 (`users`)
存储用户的基本信息与第三方授权状态。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 用户唯一标识 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |
| `deleted_at` | TIMESTAMP | Index | 软删除标识 |
| `email` | VARCHAR(255) | Unique | 用户邮箱 (普通注册) |
| `password` | VARCHAR(255) | | Bcrypt 哈希密码 |
| `github_id` | VARCHAR(255) | Unique | GitHub OAuth ID |
| `avatar_url` | VARCHAR(255) | | 头像地址 |
| `name` | VARCHAR(255) | | 用户昵称/显示名 |

### 2.2 博客表 (`blogs`)
存储通过解析文档或 Git 仓库生成的博客数据。支持树形结构（父节点 - 子章节）。

| 字段名 | 数据类型 | 约束/索引 | 描述 |
| ------ | -------- | --------- | ---- |
| `id` | UUID | Primary Key | 博客唯一标识 |
| `created_at` | TIMESTAMP | | 创建时间 |
| `updated_at` | TIMESTAMP | | 更新时间 |
| `deleted_at` | TIMESTAMP | Index | 软删除标识 |
| `user_id` | UUID | Foreign Key | 关联 `users.id` |
| `parent_id` | UUID | Foreign Key, Nullable | 关联 `blogs.id`，指向系列父节点 |
| `title` | VARCHAR(255) | | 博客/章节标题 |
| `content` | TEXT | | Markdown 格式正文内容 |
| `source_type` | VARCHAR(50) | | 来源类型 (`file`, `git`) |
| `status` | INTEGER | Default 0 | 状态 (0:生成中, 1:已完成, -1:失败) |
| `chapter_sort`| INTEGER | Default 1 | 在系列博客中的排序序号 |

## 3. 关联关系 (Associations)
- **User (1) <-> (N) Blog**: 一个用户可以拥有多篇博客历史记录。
- **Blog (1) <-> (N) Blog**: 自引用（Self-Referencing）。一个父级 Blog（代表系列入口，例如 "Hydrogen语言系列"）可以拥有多个子级 Blog（代表具体章节内容，例如 "第 1 篇：架构概览"）。通过 `parent_id` 建立一对多父子关系。

## 4. 迁移策略 (Migration)
- 系统在启动时，会通过 GORM 的 `AutoMigrate` 功能自动根据 Go 模型结构 (`internal/model`) 同步创建或更新数据库表。
- 敏感数据如密码，在存入数据库之前必须经过 `golang.org/x/crypto/bcrypt` 进行哈希加密。
