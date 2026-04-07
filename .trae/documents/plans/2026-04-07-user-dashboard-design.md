# 用户个人中心与仪表盘设计方案

## 1. 需求背景与目标
- **目标**：为系统新增“用户个人主页（Dashboard）”，允许用户查看其生成的博客统计数据、管理个人信息。
- **核心功能**：
  1. 用户基本信息展示与修改（支持修改用户名、上传头像）。
  2. 统计卡片：显示已消耗的 Tokens 数量、预估费用（按固定汇率，如 1元/百万Token）、已生成的文章总数、总字数。
  3. 图表展示：使用 `recharts` 渲染柱状图或饼图，统计并展示该用户所生成博客中涉及到的“技术栈”使用频率排行。

## 2. 后端设计

### 2.1 数据库结构变更
**修改表：`Blog` (模型：`internal/model/blog.go`)**
- 新增字段 `WordCount` (`int`)：用于存储文章总字数。
- 新增字段 `TechStacks` (`jsonb` / `[]string`)：用于存储生成文章时识别到的相关技术栈列表。

### 2.2 核心业务逻辑变更
**生成服务 (`internal/service/generator.go`)**
- 在将大模型生成的 Markdown 文本存入数据库前，自动计算纯文本的字数并写入 `WordCount`。
- 生成主体内容结束后，执行一次轻量级的 LLM 调用（Prompt 例如：“请从以下文章中提取出涉及的核心技术栈名称，以 JSON 数组格式返回，不要有任何额外字符。”），将提取结果存入 `TechStacks` 字段。

### 2.3 新增 API 接口
- **GET `/api/user/stats`**：返回统计信息。
  - 返回体示例：`{ "tokens_used": 1500000, "estimated_cost": 1.5, "total_articles": 12, "total_words": 34000, "tech_stack_stats": [{"name": "React", "count": 5}, {"name": "Go", "count": 3}] }`
- **POST `/api/user/avatar`**：上传用户头像，图片保存至 `backend/uploads/avatars/`（并在 Nginx/Gin 中配置静态文件服务），返回图片的 URL。
- **PUT `/api/user/profile`**：更新用户基础信息（如 `username`，`avatar_url`）。

## 3. 前端设计

### 3.1 页面与路由
- 新增路由 `/dashboard`，并在左侧导航栏 `Sidebar.tsx` 中新增“个人中心 / 仪表盘”入口。
- 新建组件 `src/components/Dashboard.tsx`，采用 Tailwind CSS 与 Shadcn UI 组件搭建。

### 3.2 页面结构
1. **顶部：个人信息区**：
   - 展示当前头像与用户名。
   - 提供“编辑个人资料”弹窗（支持上传头像图片与修改昵称）。
2. **中间：统计卡片区**：
   - 4 个数据看板，分别展示：消耗的 Tokens、预计支付费用（RMB）、文章总数、文章总字数。
3. **底部：图表展示区**：
   - 引入 `recharts` 图表库。
   - 渲染“技术栈使用统计”柱状图（Bar Chart），直观展示用户最常用的 10 个技术栈。

## 4. 安全与性能
- 上传头像：必须限制图片格式（仅限 PNG/JPG/WebP 等）与文件大小（如限制 2MB 以内），防止滥用。
- 技术栈提取：提取过程应考虑异常处理，若 LLM 返回非法 JSON 不应阻断主流程。
- Nginx 静态服务：由于后端运行在容器内，且前端负责代理访问，需要在 `nginx.conf` 中增加针对 `/uploads/avatars/` 路径的反向代理或静态文件映射，或者让 Gin 服务直接开放静态资源路由。推荐：后端 Gin 配置 `r.Static("/uploads", "./uploads")`，并在 Nginx 统一代理 `/api` 与 `/uploads`。
