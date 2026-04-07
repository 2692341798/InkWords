# 登录注册功能增强设计方案 (Auth Enhancement Design)

## 1. 背景与目标
当前 InkWords 的登录注册功能仅包含基础的邮箱/密码和 GitHub 授权。为了提升系统安全性和用户体验，需要对认证模块进行核心功能扩展、安全机制加固以及前端交互优化。

## 2. 核心功能扩展

### 2.1 数据库结构更新 (PostgreSQL)
- **`users` 表新增字段**:
  - `is_email_verified` (BOOLEAN): 标识邮箱是否已验证，默认 false。
  - `failed_login_attempts` (INTEGER): 连续登录失败次数，默认 0。
  - `locked_until` (TIMESTAMP): 账号锁定截止时间，为空表示未锁定。
- **新增 `verification_codes` 表**:
  - `id` (UUID/Serial): 主键。
  - `email` (VARCHAR): 目标邮箱。
  - `code` (VARCHAR): 6位数字验证码。
  - `type` (VARCHAR): 验证码类型 (枚举: `register`, `reset_password`)。
  - `expires_at` (TIMESTAMP): 过期时间（通常为发送后 10-15 分钟）。
  - `created_at` (TIMESTAMP): 创建时间。

### 2.2 后端 API 接口 (Go + Gin)
- **`GET /api/v1/auth/captcha`**
  - 功能：生成本地图形验证码（使用类似 `github.com/mojocn/base64Captcha` 的库）。
  - 响应：返回 `captcha_id` 和 Base64 格式的图片数据。
- **`POST /api/v1/auth/send-code`**
  - 功能：发送邮箱验证码。
  - 请求参数：`email`, `type` (register/reset_password), `captcha_id`, `captcha_value`。
  - 逻辑：校验图形验证码 -> 生成 6 位随机数 -> 存入数据库 -> 通过 SMTP 发送邮件。若 `.env` 中未配置 SMTP，则在控制台打印验证码（Mock 兜底）。
- **`POST /api/v1/auth/register`**
  - 请求参数：`username`, `email`, `password`, `code` (邮箱验证码)。
  - 逻辑：校验邮箱验证码 -> 验证密码强度 -> 创建用户（`is_email_verified = true`）-> 返回 Token。
- **`POST /api/v1/auth/login`**
  - 请求参数：`email`, `password`, `remember_me` (BOOLEAN), 选填：`captcha_id`, `captcha_value`。
  - 逻辑：
    1. 检查 `locked_until`，若处于锁定状态则拒绝。
    2. 若 `failed_login_attempts` >= 3，强制要求图形验证码。
    3. 校验密码。若失败，`failed_login_attempts` +1。若达到 5 次，设置 `locked_until` 为 15 分钟后。
    4. 校验成功，清空 `failed_login_attempts` 和 `locked_until`。
    5. 根据 `remember_me` 签发 Token（true=30天, false=24小时）。
- **`POST /api/v1/auth/reset-password`**
  - 请求参数：`email`, `code` (邮箱验证码), `new_password`。
  - 逻辑：校验验证码 -> 验证新密码强度 -> 更新密码。

## 3. 安全与校验机制
1. **密码强度校验**：
   - 前端：输入时实时校验，要求包含大小写字母、数字，长度 >= 8。展示“弱/中/强”的密码强度进度条。
   - 后端：严格校验上述规则，不符合则返回 400 错误。
2. **图形验证码防刷**：
   - 必须在“发送邮箱验证码”前完成图形验证码。
   - 登录失败超过 3 次时，强制要求图形验证码。
3. **登录失败防爆破锁定**：
   - 连续 5 次密码错误，锁定账号 15 分钟。

## 4. 前端交互体验 (React + Tailwind)
1. **单卡片平滑过渡**：
   - 维持现有的极简居中单卡片设计。
   - 引入状态机或简单条件渲染管理 `login` / `register` / `forgot_password` 三种视图。
   - 切换状态时加入平滑的高度变化和淡入淡出动画。
2. **组件细节优化**：
   - **登录页**：新增「记住我」复选框和「忘记密码？」链接。
   - **注册/重置页**：新增「获取验证码」输入框组合，按钮附带 60s 倒计时防频繁点击。
   - **密码框**：增加「显示/隐藏密码」的切换图标，下方嵌入密码强度指示条。
3. **状态管理与提示**：
   - 统一捕获后端错误并给予友好的 Toast 或内联红色提示。
   - 邮箱未验证、账号被锁定时，给予明确引导。
