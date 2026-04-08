# InkWords 架构优化与业务增强设计 (2026-04-09)

## 1. 目标与背景
基于前期发现的项目不足，本次优化重点解决以下四个维度的问题：
- **架构规范**：修复依赖注入（DI）失效问题，补充缺失的 Godoc/JSDoc 注释。
- **安全与健壮性**：解决 OAuth 账号劫持风险，完善用户输入的后端校验。
- **前端与体验**：修复由于 `searchNested` 闭包引起的 React 渲染性能隐患，统一下发中文错误提示。
- **商业闭环**：落地基于硬性拦截的 Tokens 额度与计费策略。

## 2. 详细设计方案

### 2.1 架构与规范优化
- **依赖注入重构**：
  - 后端 `api` 层（如 `AuthAPI`, `UserAPI`）不再直接内部实例化 `service`，而是通过构造函数 `NewAuthAPI(authService)` 传入依赖。
  - `service` 层的结构体将显式声明其依赖的组件，便于后续测试。
- **注释规范补全**：
  - 为所有公开的 API Handler、Service 方法补充标准 `Godoc`。
  - 为前端核心组件（如 `Dashboard.tsx`, `Sidebar.tsx`）补充 `JSDoc`，说明组件职责。

### 2.2 商业闭环（计费与额度拦截）
- **数据库同步**：在 `users` 表中增加/使用 `token_limit` 字段（若未设置，默认为 100,000）。
- **生成拦截器**：
  - 在调用大模型生成（`/api/v1/stream/generate` 与 `/api/v1/project/analyze` 及继续生成接口）前，校验当前用户的 `TokensUsed >= TokenLimit`。
  - 若超额，直接阻断 HTTP/SSE 请求，返回中文错误提示 `"您的 Token 额度已耗尽，请升级订阅或联系管理员"`。

### 2.3 OAuth 账号劫持防御（密码验证绑定）
- **业务流程**：
  - 当用户通过 GitHub 授权回调时，如果系统发现 `email` 已存在于本地常规注册用户中，但 `github_id` 尚未绑定。
  - 后端将不直接登录，而是重定向至前端，URL 携带 `?bind_required=true&email=xxx&github_id=yyy&avatar_url=zzz&username=uuu` 等临时信息（由于数据敏感，可通过签发一个 5 分钟有效期的临时 `bind_token` 传递给前端）。
  - 前端 `<Login />` 组件捕获到 `bind_token` 后，切换至“绑定已有账号”视图，要求用户输入该邮箱的本地密码。
  - 用户输入密码后，前端调用新增的 `/api/v1/auth/bind-github` 接口，后端校验密码成功后完成绑定并签发正式 JWT。

### 2.4 前端体验与中文化报错
- **后端中文报错统一下发**：
  - 全局排查 `c.JSON(..., gin.H{"message": "..."})`，将类似 "user not found"、"invalid request body" 等英文错误提示统一翻译为直观的中文。
- **React 性能修复**：
  - 在 `Sidebar.tsx` 中，针对“当前生成任务”卡片点击事件里的 `searchNested` 递归查询，引入 `useMemo` 构建扁平化的 `id -> BlogNode` 索引字典，消除每次点击/渲染时的深层遍历开销。
- **后端输入校验**：
  - 在 `UpdateProfile` 接口中对 `username` 进行长度限制（如 2-20 字符）和空值校验。

## 3. 影响范围与测试
- **测试重点**：
  - 第三方登录、本地注册、GitHub 绑定已有账号的完整状态机流转。
  - 额度耗尽时的拦截逻辑，确保不会继续扣费且前端能正确弹窗提示。
  - `Sidebar` 点击卡片后能正确联动并在历史记录中展开父节点。
