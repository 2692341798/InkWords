# 墨言博客助手 (InkWords) - API 接口规范与设计文档

## 1. 规范与约定
- **基础 URL**：所有后端接口均以 `/api/v1` 开头。（本地开发环境下，前端 Vite 配置了代理，将 `/api` 转发至 `http://localhost:8080`）
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

### 2.1 用户注册 (Register)
- **POST** `/api/v1/auth/register`
- **描述**：使用邮箱和密码注册新用户。
- **Request Body (JSON)**:
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword123",
    "username": "nickname"
  }
  ```
- **Response Data**:
  ```json
  {
    "user": {
      "id": "1234567890",
      "username": "nickname",
      "email": "user@example.com"
    }
  }
  ```

### 2.2 用户登录 (Login)
- **POST** `/api/v1/auth/login`
- **描述**：使用邮箱和密码进行登录，返回 JWT Token。
- **Request Body (JSON)**:
  ```json
  {
    "email": "user@example.com",
    "password": "securepassword123"
  }
  ```
- **Response Data**:
  ```json
  {
    "token": "eyJhbGciOiJIUzI1...",
    "user": {
      "id": "1234567890",
      "username": "nickname",
      "avatar_url": "https://...",
      "subscription_tier": 0,
      "tokens_used": 15000
    }
  }
  ```

### 2.3 第三方一键登录重定向 (OAuth Redirect)
- **GET** `/api/v1/auth/oauth/:provider`
- **Path Params**: `provider` (`github` 或 `wechat`)
- **描述**：重定向到 GitHub 或微信的授权页面。

### 2.4 第三方登录回调 (OAuth Callback)
- **GET** `/api/v1/auth/callback/:provider`
- **Query Params**: `code`
- **描述**：后端处理完第三方授权后，不直接返回 JSON，而是通过 `HTTP 307` 重定向回前端系统。
- **Response (Redirect)**:
  - **成功**：`307 Temporary Redirect` -> `http://<FRONTEND_URL>/?token=eyJhbGci...`
  - **失败**：`307 Temporary Redirect` -> `http://<FRONTEND_URL>/?error=错误信息`

### 2.5 获取个人中心配置与额度
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
- **描述**：用户上传文件或提交 Git URL，后端解析完毕后返回解析出的纯文本或任务 ID。
- **Request (Multipart/form-data 或 JSON)**:
  - `file` (File, Optional): 本地上传的文档 (目前支持 `.pdf`, `.docx`, `.md`, `.txt`)。
  - `git_url` (String, Optional): Git 仓库地址。
- **Response Data**:
  ```json
  {
    "source_content": "提取出的文本...",
    "task_id": "uuid-v4-ticket-1234",
    "estimated_series": 1 // 1: 单篇, >1: 大项目拆解系列
  }
  ```

### 3.2 大项目分析与大纲生成流 (Project Analyze Stream)
- **POST** `/api/v1/stream/analyze`
- **描述**：提交 Git 仓库 URL，系统通过 SSE 协议实时下发克隆、提取、生成大纲的各个阶段进度，缓解用户的等待焦虑。
- **Request Body (JSON)**:
  ```json
  {
    "git_url": "https://github.com/..."
  }
  ```
- **SSE Event 格式**:
  ```text
  // 阶段 1: 克隆进度
  event: chunk
  data: {"step": 0, "message": "正在克隆并拉取仓库..."}
  
  // 阶段 2: 分析源码
  event: chunk
  data: {"step": 1, "message": "分析仓库源码与结构完成"}

  // 阶段 3: 大纲生成 (包含 Map-Reduce 进度)
  event: chunk
  data: {"step": 2, "message": "评估大模型并生成项目大纲..."}
  // 额外可能触发的分块进度事件:
  // {"status": "chunk_analyzing", "dir": "cmd/api", "index": 1, "total": 10, "worker_id": 0}
  // {"status": "chunk_failed", "dir": "cmd/api", "index": 1, "attempt": 1, "worker_id": 0}
  // {"status": "chunk_done", "dir": "cmd/api", "index": 1, "worker_id": 0}
  
  // 阶段 4: 完成处理（携带最终数据）
  event: chunk
  data: {
    "step": 3, 
    "message": "正在完成最后处理...",
    "data": {
      "outline": [...],
      "source_content": "提取后的文档或源码内容..."
    }
  }

  // 结束标识
  event: done
  data: [DONE]
  ```

### 3.3 建立流式生成连接 (SSE)
- **POST** `/api/v1/stream/generate`
- **描述**：前端必须使用 `@microsoft/fetch-event-source` 库通过 POST 请求携带大文本 Payload，并**必须设置 `openWhenHidden: true` 防止浏览器后台挂起时断流**。系统在将 `source_content` 传给大模型前，已加入**字符截断保护**（强制截断超过 300,000 字符的文本），以防止 API 抛出 `invalid_request_error` 导致生成中断。若携带 `outline`，则进入系列生成模式，后端会主动创建并持久化一个 Parent 节点以避免数据孤岛，并**串行**生成多个章节并打字机渲染；否则进行单篇生成。支持前端使用 `AbortController` 随时中断请求，后端会优雅中止生成任务以节省 Token。
- **Request Body (JSON)**:
  ```json
  {
    "series_title": "用户编辑后的系列标题（可选）",
    "source_content": "提取后的文档或源码内容...",
    "outline": [/* 章节大纲数组 */],
    "source_type": "git", // "git" 或 "file"
    "parent_id": "已存在系列ID（可选，断点续传时使用）"
  }
  ```
- **SSE Event 格式**:
  ```text
  event: chunk
  data: {"content": "这是一段", "chapter_sort": 1}
  
  // 系列生成模式下的进度事件
  event: progress
  data: {"status": "generating", "chapter_sort": 1, "title": "基础篇"}

  event: done
  data: {"blog_id": "9876543210"}
  ```

## 4. 博客管理接口 (Blog API)

### 4.1 获取历史博客列表
- **GET** `/api/v1/blogs`
- **Query Params**: `page=1&size=20`
- **Response Data**: 返回按 `parent_id` 组织的列表，支持前端树状展开。在生成任务完成后，前端会主动调用此接口刷新历史记录，并联动展开/高亮新生成的文章。

### 4.2 保存/更新用户编辑的内容
- **PUT** `/api/v1/blogs/:id`
- **Request**:
  ```json
  {
    "title": "修改后的标题",
    "content": "修改后的 Markdown 内容..."
  }
  ```

### 4.3 继续生成未写完的博客 (Continue Generating)
- **POST** `/api/v1/blogs/:id/continue`
- **描述**：针对生成截断或内容不全的博客，调用此接口会让大模型基于已有内容“继续完成上文未写完的内容”。采用 SSE 流式下发，并在完成后自动追加到数据库中。
- **SSE Event 格式**:
  ```text
  event: message
  data: {"content": "继续输出", "status": "generating"}

  event: done
  data: [DONE]
  ```
- **Heartbeat Event:**
  ```text
  event: ping
  data: {}
  ```
  *(注：前端接收到 ping 事件应直接忽略，此机制仅用于防止 Nginx 等代理因长连接空闲而主动切断 SSE 链接)*

### 4.4 导出系列博客 (Export Series)
- **GET** `/api/v1/blogs/:id/export`
- **描述**：将指定的系列博客（或带子节点的文件夹）打包导出为 ZIP 压缩包下载。
- **Response**: 返回 `application/zip` 二进制流。

### 4.5 批量删除博客 (Batch Delete)
- **DELETE** `/api/v1/blogs`
- **描述**：批量删除用户选中的多篇博客（及其子节点）。
- **Request Body (JSON)**:
  ```json
  {
    "blog_ids": [
      "uuid-1",
      "uuid-2"
    ]
  }
  ```
- **Response Data**:
  ```json
  {
    "code": 200,
    "message": "success",
    "data": null
  }
  ```