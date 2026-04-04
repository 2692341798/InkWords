# 墨言博客助手 (InkWords) - API 接口规范与设计文档

## 1. 规范与约定
- **基础 URL**：所有后端接口均以 `/api/v1` 开头。
- **流式接口**：大模型生成接口以 `/api/v1/stream` 开头，采用 SSE (Server-Sent Events) 协议。
- **数据格式**：非流式接口统一采用 `application/json`。
- **鉴权方式**：请求头携带 `Authorization: Bearer <JWT_TOKEN>`。
- **统一响应结构 (Standard Response)**:
  ```json
  {
    "code": 200,          // 业务状态码 (200 成功, 4xx/5xx 错误)
    "message": "success", // 描述信息
    "data": {}            // 核心载荷 (可能为 null)
  }
  ```

## 2. 用户与鉴权接口 (Auth API)

### 2.1 第三方一键登录重定向 (OAuth Redirect)
- **GET** `/api/v1/auth/oauth/:provider`
- **Path Params**: `provider` (`github` 或 `wechat`)
- **描述**：重定向到 GitHub 或微信的授权页面。

### 2.2 第三方登录回调 (OAuth Callback)
- **GET** `/api/v1/auth/callback/:provider`
- **Query Params**: `code`
- **Response Data**:
  ```json
  {
    "token": "eyJhbGciOiJIUzI1...",
    "user": {
      "id": "1234567890",
      "username": "user_github",
      "avatar_url": "https://...",
      "subscription_tier": 0,
      "tokens_used": 15000
    }
  }
  ```

### 2.3 获取个人中心配置与额度
- **GET** `/api/v1/user/profile`
- **Response Data**:
  ```json
  {
    "subscription_tier": 1,
    "tokens_used": 45000,
    "token_limit": 100000,
    "connected_platforms": ["juejin", "csdn"]
  }
  ```

## 3. 博客生成与解析接口 (Generator API)

### 3.1 提交解析任务 (获取生成 Ticket)
- **POST** `/api/v1/generator/parse`
- **描述**：用户上传文件或提交 Git URL，后端解析完毕后返回一个任务 ID (Ticket)，前端持此 Ticket 去建立 SSE 连接。
- **Request (Multipart/form-data 或 JSON)**:
  - `file` (File, Optional): 本地上传的文档 (PDF/MD/Word)。
  - `git_url` (String, Optional): Git 仓库地址。
- **Response Data**:
  ```json
  {
    "task_id": "uuid-v4-ticket-1234",
    "estimated_series": 1 // 1: 单篇, >1: 大项目拆解系列
  }
  ```

### 3.2 建立流式生成连接 (SSE)
- **GET** `/api/v1/stream/generate?task_id={task_id}`
- **描述**：前端通过 `EventSource` 发起请求，后端流式返回 Markdown 文本。
- **SSE Event 格式**:
  ```text
  event: chunk
  data: {"content": "这是一段", "chapter_sort": 1}

  event: done
  data: {"blog_id": "9876543210"}
  ```

## 4. 博客管理接口 (Blog API)

### 4.1 获取历史博客列表
- **GET** `/api/v1/blogs`
- **Query Params**: `page=1&size=20`
- **Response Data**: 返回按 `parent_id` 组织的列表，支持前端树状展开。

### 4.2 保存/更新用户编辑的内容
- **PUT** `/api/v1/blogs/:id`
- **Request**:
  ```json
  {
    "title": "修改后的标题",
    "content": "修改后的 Markdown 内容..."
  }
  ```