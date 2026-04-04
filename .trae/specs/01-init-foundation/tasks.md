# Tasks

- [x] Task 1: 初始化前端 React 骨架
  - [x] SubTask 1.1: 在 `frontend` 目录下使用 Vite 创建 React(TS) 项目
  - [x] SubTask 1.2: 安装 Tailwind CSS 及其依赖并初始化配置
  - [x] SubTask 1.3: 初始化 Shadcn UI 配置并安装基础组件
  - [x] SubTask 1.4: 安装 Zustand 并在 `src/store` 下创建基础状态模块
- [x] Task 2: 建立后端 GORM 数据库模型
  - [x] SubTask 2.1: 在 `backend` 目录下安装 `gorm` 及 `gorm.io/driver/postgres` 依赖
  - [x] SubTask 2.2: 在 `backend/internal/model` 目录下创建 `user.go`, `blog.go`, `oauth_token.go` 文件并编写模型结构体
  - [x] SubTask 2.3: 在模型中添加 UUIDv4 主键支持、软删除（gorm.DeletedAt）及时间戳字段
  - [x] SubTask 2.4: 在模型中配置索引（如 `blogs` 表的 `user_id, parent_id, chapter_sort` 组合索引）
  - [x] SubTask 2.5: 在 `backend/internal/db` (或类似包) 中编写数据库连接与 AutoMigrate 逻辑，并在 main 中调用

# Task Dependencies
- [Task 1] 和 [Task 2] 可以并行执行。
