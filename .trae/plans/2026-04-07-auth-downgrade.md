# Auth Downgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 移除复杂的邮箱验证和密码重置流程，简化为仅凭密码与图形验证码注册登录，同时保留防爆破机制。

**Architecture:** 
1. **数据库层**：删除 `verification_codes` 表及相关操作，移除 `users` 表的 `is_email_verified` 字段。
2. **后端 API 层**：删除邮件发送依赖 `gomail.v2`，移除 `/send-code` 和 `/reset-password` 接口，精简 `/register` 逻辑仅验证图形验证码和密码强度。
3. **前端 UI 层**：从 `Login.tsx` 中彻底移除验证码发送、倒计时、忘记密码模式等冗余 UI，保留单纯的登录与注册表单。

**Tech Stack:** Go 1.21+, Gin, GORM, PostgreSQL 14+, React 18, Tailwind CSS.

---

### Task 1: 数据库与模型瘦身

**Files:**
- Modify: `backend/internal/model/user.go`
- Delete: `backend/internal/model/verification_code.go`
- Modify: `backend/internal/db/db.go`

- [x] **Step 1: 移除 User 模型中的 IsEmailVerified 字段**
修改 `backend/internal/model/user.go`：
```go
package model

import "time"

type User struct {
	ID                  uint       `gorm:"primarykey" json:"id"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Username            string     `gorm:"uniqueIndex;not null" json:"username"`
	Email               string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash        string     `gorm:"not null" json:"-"`
	GithubID            *string    `gorm:"uniqueIndex" json:"github_id,omitempty"`
	AvatarURL           string     `json:"avatar_url,omitempty"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`
}
```

- [x] **Step 2: 删除 VerificationCode 模型**
执行命令删除文件：
```bash
rm backend/internal/model/verification_code.go
```

- [x] **Step 3: 更新 GORM 自动迁移配置**
修改 `backend/internal/db/db.go`，从 `AutoMigrate` 中移除 `&model.VerificationCode{}`：
```go
	// 自动迁移数据库结构
	err = db.AutoMigrate(
		&model.Blog{},
		&model.User{},
	)
```
注意：GORM 的 AutoMigrate 不会自动删除列或表，所以如果想要真正从数据库中删除表和列，需要写原生的 SQL，由于这里是开发环境，只需代码上移除即可，后续可以手动在 DB 执行 `DROP TABLE verification_codes;` 和 `ALTER TABLE users DROP COLUMN is_email_verified;`。为了安全，我们在代码中先不加自动 Drop，而是仅去除映射。

- [x] **Step 4: Commit**
```bash
git add backend/internal/model/ backend/internal/db/
git commit -m "refactor(db): remove verification code model and email verified field"
```

---

### Task 2: 后端 API 与服务逻辑瘦身

**Files:**
- Modify: `backend/internal/api/auth.go`
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/go.mod` & `backend/go.sum`

- [x] **Step 1: 移除 Service 层的邮件与验证码逻辑**
修改 `backend/internal/service/auth.go`，删除 `GenerateRandomCode`、`SendVerificationEmail`、`SaveVerificationCode` 函数及其相关的依赖导入。

- [x] **Step 2: 移除 API 层的废弃接口**
修改 `backend/internal/api/auth.go`，删除 `SendCode` 和 `ResetPassword` 函数。

- [x] **Step 3: 精简注册接口逻辑**
修改 `backend/internal/service/auth.go` 的 `Register` 函数，移除对邮箱验证码的查询校验，改为直接校验图形验证码（与之前登录防刷共用一套图形验证码组件）：
```go
func (s *AuthService) Register(req struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	CaptchaID    string `json:"captcha_id" binding:"required"`
	CaptchaValue string `json:"captcha_value" binding:"required"`
}) (string, error) {
	// 校验图形验证码
	if !VerifyCaptcha(req.CaptchaID, req.CaptchaValue) {
		return "", errors.New("图形验证码错误或已过期")
	}

	// 密码强度校验：长度必须大于等于 8 位
	if len(req.Password) < 8 {
		return "", errors.New("密码长度必须至少为 8 位")
	}

	// 检查邮箱或用户名是否已存在
	var count int64
	s.db.Model(&model.User{}).Where("email = ? OR username = ?", req.Email, req.Username).Count(&count)
	if count > 0 {
		return "", errors.New("邮箱或用户名已被注册")
	}

	// 创建新用户
	user := model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: HashPassword(req.Password),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return "", errors.New("创建用户失败，请稍后重试")
	}

	// 签发 Token
	return GenerateToken(user.ID, false)
}
```
并在 `backend/internal/api/auth.go` 中的 `Register` Handler 更新对应的 `req` 结构体：
```go
func (h *AuthAPI) Register(c *gin.Context) {
	var req struct {
		Username     string `json:"username" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required"`
		CaptchaID    string `json:"captcha_id" binding:"required"`
		CaptchaValue string `json:"captcha_value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数格式错误"})
		return
	}

	token, err := h.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "注册成功", "data": gin.H{"token": token}})
}
```

- [x] **Step 4: 清理路由与依赖**
修改 `backend/cmd/server/main.go`，移除 `/send-code` 和 `/reset-password` 的路由注册：
```go
// 移除这两行：
// authGroup.POST("/send-code", authAPI.SendCode)
// authGroup.POST("/reset-password", authAPI.ResetPassword)
```
执行命令清理无用依赖（如 `gomail.v2`）：
```bash
cd backend && go mod tidy
```

- [x] **Step 5: Commit**
```bash
git add backend/
git commit -m "refactor(api): remove email verification and reset password endpoints"
```

---

### Task 3: 前端 UI 与状态瘦身

**Files:**
- Modify: `frontend/src/components/Login.tsx`

- [x] **Step 1: 移除废弃状态与枚举**
修改 `frontend/src/components/Login.tsx`，将 `AuthMode` 简化，并删除 `countdown` 状态：
```tsx
type AuthMode = 'login' | 'register' // 移除 'forgot_password'
const [mode, setMode] = useState<AuthMode>('login')
// 删除 const [countdown, setCountdown] = useState(0)
```

- [x] **Step 2: 移除废弃的 handleSendCode 函数**
从文件中彻底删除 `handleSendCode` 函数以及与倒计时相关的 `useEffect` 代码。

- [x] **Step 3: 调整 handleSubmit 逻辑**
移除 `forgot_password` 的判断，更新 Payload 组装逻辑：
```tsx
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    const endpoint = mode === 'login' ? '/api/v1/auth/login' : '/api/v1/auth/register'
    
    // 统一 Payload，注册时也需要验证码
    const payload = mode === 'login'
      ? {
          email: formData.email,
          password: formData.password,
          remember_me: rememberMe,
          captcha_id: captcha.id,
          captcha_value: captcha.value
        }
      : {
          username: formData.name,
          email: formData.email,
          password: formData.password,
          captcha_id: captcha.id,
          captcha_value: captcha.value
        }

    try {
        // ... (保持 fetch 和后续错误处理逻辑不变)
```

- [x] **Step 4: 精简渲染层 (JSX)**
删除所有关于 `mode === 'forgot_password'` 的条件渲染。
删除所有的 `code` (邮箱验证码) 输入框组件。
删除“忘记密码？”的文字链接。
将“获取验证码”倒计时相关的 UI 从图形验证码组件旁彻底删除，现在图形验证码只需要单纯的输入框和刷新图片：
```tsx
{/* 渲染图形验证码 (登录需要时，或者注册时必须) */}
{(mode === 'register' || requireCaptcha) && (
  <div className="space-y-1.5">
    <label className="text-sm font-medium text-zinc-700">验证码</label>
    <div className="flex gap-2">
      <div className="relative flex-1">
        <input
          type="text"
          value={captcha.value}
          onChange={(e) => setCaptcha({ ...captcha, value: e.target.value })}
          required
          className="w-full px-3 py-2 border border-zinc-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-zinc-900/20 focus:border-zinc-900 transition-colors"
          placeholder="请输入图形验证码"
        />
      </div>
      <div
        className="w-32 h-10 rounded-lg overflow-hidden border border-zinc-200 cursor-pointer bg-white flex-shrink-0"
        onClick={fetchCaptcha}
        title="点击刷新验证码"
      >
        {captcha.image ? (
          <img src={captcha.image} alt="captcha" className="w-full h-full object-cover" />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-xs text-zinc-400">
            加载中...
          </div>
        )}
      </div>
    </div>
  </div>
)}
```
*注意：`requireCaptcha` 这个状态目前是登录失败时自动开启的，如果是注册态 `mode === 'register'` 则必须渲染。*

- [x] **Step 5: Commit**
```bash
git add frontend/src/components/Login.tsx
git commit -m "refactor(ui): simplify login and register forms, remove email verification"
```

---

### Task 4: 更新项目文档 (强制规范)

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `README.md`

- [x] **Step 1: 更新基准文档中的冗余描述**
- 从 API 文档中移除 `/send-code` 和 `/reset-password`，修改 `/register` 参数。
- 从数据库文档中移除 `verification_codes` 表和 `is_email_verified` 字段。
- 从 PRD 和 README 中将“邮件验证码”和“重置密码”相关的宣传字眼移除。

- [x] **Step 2: 写入开发日志与对话记录**
记录本次进行 Auth 流程瘦身（移除邮件验证，精简流程）的决策与行动。

- [x] **Step 3: Commit**
```bash
git add .trae/documents/ README.md backend/.env
git commit -m "docs: sync all documentation with auth downgrade changes"
```
