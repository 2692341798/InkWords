# [Auth Enhancement] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增强 InkWords 的登录注册功能，引入邮箱验证、图形验证码防刷、密码强度校验及登录失败锁定机制。

**Architecture:** 
1. **数据库层**：通过 GORM 自动迁移更新 `users` 表结构，并新增 `verification_codes` 表。
2. **后端 API 层**：在 `internal/api/auth.go` 和 `internal/service/auth.go` 中新增/修改接口（图形验证码生成、发送邮件、增强版注册/登录、重置密码）。
3. **前端 UI 层**：在 `frontend/src/components/Login.tsx` 中使用状态机制实现单卡片的登录/注册/重置密码无缝切换，并增加密码强度条、倒计时等交互。

**Tech Stack:** Go 1.21+, Gin, GORM, PostgreSQL 14+, React 18, Tailwind CSS, Shadcn UI.

---

### Task 1: 数据库模型与迁移更新

**Files:**
- Modify: `backend/internal/service/auth.go` (或其他定义数据模型的文件，根据实际项目结构调整)
- Create: 增加邮箱验证码模型

- [x] **Step 1: 编写/更新 User 模型**

修改 `backend/internal/service/auth.go` (或 `backend/internal/models/user.go`，由于规范中未明确模型独立，暂时放在 service/models 层)。

```go
// 假设原有 User 结构体在 models 或 service 中，更新如下：
type User struct {
	ID                  uint       `gorm:"primarykey" json:"id"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	Username            string     `gorm:"uniqueIndex;not null" json:"username"`
	Email               string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash        string     `gorm:"not null" json:"-"`
	GithubID            *string    `gorm:"uniqueIndex" json:"github_id,omitempty"`
	AvatarURL           string     `json:"avatar_url,omitempty"`
	// 新增字段
	IsEmailVerified     bool       `gorm:"default:false" json:"is_email_verified"`
	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`
}
```

- [x] **Step 2: 创建 VerificationCode 模型**

```go
type VerificationCode struct {
	ID        uint      `gorm:"primarykey"`
	Email     string    `gorm:"index;not null"`
	Code      string    `gorm:"not null"`
	Type      string    `gorm:"not null"` // "register" or "reset_password"
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}
```

- [x] **Step 3: 确保 GORM AutoMigrate 包含新模型**

在 `backend/cmd/server/main.go` 中，更新 AutoMigrate：
```go
// 在 db 初始化后
err = db.AutoMigrate(&User{}, &VerificationCode{})
if err != nil {
    log.Fatalf("Failed to auto migrate: %v", err)
}
```

- [x] **Step 4: Commit**
```bash
git add backend/
git commit -m "feat(db): update users table and add verification_codes table"
```

---

### Task 2: 后端 - 图形验证码与邮件发送服务

**Files:**
- Modify: `backend/go.mod` (引入第三方库)
- Modify: `backend/internal/service/auth.go`
- Modify: `backend/internal/api/auth.go`

- [x] **Step 1: 安装依赖库**
```bash
cd backend
go get github.com/mojocn/base64Captcha
go get gopkg.in/gomail.v2
```

- [x] **Step 2: 实现 Captcha 生成与校验逻辑 (service)**
在 `backend/internal/service/auth.go` 中：
```go
import "github.com/mojocn/base64Captcha"

var store = base64Captcha.DefaultMemStore

func GenerateCaptcha() (id, b64s string, err error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err = c.Generate()
	return id, b64s, err
}

func VerifyCaptcha(id string, VerifyValue string) bool {
	return store.Verify(id, VerifyValue, true)
}
```

- [x] **Step 3: 编写发送验证码逻辑及 Mock 兜底 (service)**
```go
import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"gopkg.in/gomail.v2"
)

func GenerateRandomCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

func SendVerificationEmail(email, code, codeType string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		// Mock 兜底
		fmt.Printf("======== MOCK EMAIL ========\nTo: %s\nType: %s\nCode: %s\n============================\n", email, codeType, code)
		return nil
	}
	
	// 实际发送逻辑
	smtpPort := 465 // 或从 env 获取
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "InkWords 验证码")
	m.SetBody("text/html", fmt.Sprintf("您的验证码是：<b>%s</b>，有效期15分钟。", code))

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	return d.DialAndSend(m)
}
```

- [x] **Step 4: 暴露 API 接口 (api)**
在 `backend/internal/api/auth.go` 中添加：
```go
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
	id, b64s, err := service.GenerateCaptcha()
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "验证码生成失败"})
		return
	}
	c.JSON(200, gin.H{"code": 200, "data": gin.H{"captcha_id": id, "image": b64s}})
}

func (h *AuthHandler) SendCode(c *gin.Context) {
	var req struct {
		Email        string `json:"email" binding:"required,email"`
		Type         string `json:"type" binding:"required"`
		CaptchaID    string `json:"captcha_id" binding:"required"`
		CaptchaValue string `json:"captcha_value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if !service.VerifyCaptcha(req.CaptchaID, req.CaptchaValue) {
		c.JSON(400, gin.H{"code": 400, "message": "图形验证码错误"})
		return
	}

	code := service.GenerateRandomCode()
	// 保存到 DB
	err := h.db.Create(&VerificationCode{
		Email:     req.Email,
		Code:      code,
		Type:      req.Type,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}).Error
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	service.SendVerificationEmail(req.Email, code, req.Type)
	c.JSON(200, gin.H{"code": 200, "message": "验证码已发送"})
}
```
并在路由中注册这些 Endpoint。

- [x] **Step 5: Commit**
```bash
git add backend/
git commit -m "feat(api): implement captcha and email verification sending"
```

---

### Task 3: 后端 - 注册、登录与密码重置逻辑强化

**Files:**
- Modify: `backend/internal/api/auth.go`

- [x] **Step 1: 强化注册接口逻辑**
修改 `Register` 接口，要求接收 `code`，并验证密码强度与验证码：
```go
// 校验验证码
var vc VerificationCode
if err := h.db.Where("email = ? AND code = ? AND type = ? AND expires_at > ?", req.Email, req.Code, "register", time.Now()).First(&vc).Error; err != nil {
	c.JSON(400, gin.H{"code": 400, "message": "验证码错误或已过期"})
	return
}
// 校验密码强度 (简单示例，可抽离为独立函数)
if len(req.Password) < 8 {
    c.JSON(400, gin.H{"code": 400, "message": "密码长度必须大于8位"})
    return
}
// 创建用户时设置 IsEmailVerified = true
```

- [x] **Step 2: 强化登录防爆破逻辑**
修改 `Login` 接口：
```go
// 查询用户
var user User
if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
	c.JSON(400, gin.H{"code": 400, "message": "邮箱或密码错误"})
	return
}

// 检查锁定状态
if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
	c.JSON(403, gin.H{"code": 403, "message": "账号已锁定，请稍后再试"})
	return
}

// 如果失败次数 >= 3，强制要求 Captcha
if user.FailedLoginAttempts >= 3 {
	if req.CaptchaID == "" || !service.VerifyCaptcha(req.CaptchaID, req.CaptchaValue) {
		c.JSON(400, gin.H{"code": 400, "message": "请输入正确的图形验证码"})
		return
	}
}

// 校验密码
if !service.CheckPasswordHash(req.Password, user.PasswordHash) {
	user.FailedLoginAttempts++
	if user.FailedLoginAttempts >= 5 {
		lockTime := time.Now().Add(15 * time.Minute)
		user.LockedUntil = &lockTime
	}
	h.db.Save(&user)
	c.JSON(400, gin.H{"code": 400, "message": "邮箱或密码错误"})
	return
}

// 成功则重置状态
user.FailedLoginAttempts = 0
user.LockedUntil = nil
h.db.Save(&user)

// 根据 remember_me 签发不同时长的 Token (具体看 JWT 实现)
```

- [x] **Step 3: 编写重置密码接口**
```go
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Code        string `json:"code" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	var vc VerificationCode
	if err := h.db.Where("email = ? AND code = ? AND type = ? AND expires_at > ?", req.Email, req.Code, "reset_password", time.Now()).First(&vc).Error; err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "验证码错误或已过期"})
		return
	}

	var user User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "用户不存在"})
		return
	}

	user.PasswordHash = service.HashPassword(req.NewPassword)
	h.db.Save(&user)
    
    // 将验证码标记为已使用（或直接删除）
    h.db.Delete(&vc)

	c.JSON(200, gin.H{"code": 200, "message": "密码重置成功"})
}
```

- [x] **Step 4: Commit**
```bash
git add backend/
git commit -m "feat(api): enhance login security, register validation and add reset password"
```

---

### Task 4: 前端 - 验证码组件与基础交互修改

**Files:**
- Modify: `frontend/src/components/Login.tsx`

- [x] **Step 1: 定义状态与枚举**
```tsx
type AuthMode = 'login' | 'register' | 'forgot_password'
const [mode, setMode] = useState<AuthMode>('login')
const [captcha, setCaptcha] = useState({ id: '', image: '', value: '' })
const [countdown, setCountdown] = useState(0)
const [rememberMe, setRememberMe] = useState(false)
const [showPassword, setShowPassword] = useState(false)
```

- [x] **Step 2: 获取验证码逻辑**
```tsx
const fetchCaptcha = async () => {
    try {
        const res = await fetch('/api/v1/auth/captcha')
        const data = await res.json()
        if (data.code === 200) {
            setCaptcha(prev => ({ ...prev, id: data.data.captcha_id, image: data.data.image, value: '' }))
        }
    } catch (e) {
        console.error(e)
    }
}
// 在 useEffect 和需要刷新验证码的地方调用 fetchCaptcha()
```

- [x] **Step 3: 发送邮箱验证码逻辑**
```tsx
const handleSendCode = async () => {
    if (!formData.email) return setError('请输入邮箱')
    if (!captcha.value) return setError('请输入图形验证码')
    
    try {
        const res = await fetch('/api/v1/auth/send-code', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                email: formData.email,
                type: mode === 'register' ? 'register' : 'reset_password',
                captcha_id: captcha.id,
                captcha_value: captcha.value
            })
        })
        const data = await res.json()
        if (data.code === 200) {
            setCountdown(60)
            // 开始倒计时逻辑
        } else {
            setError(data.message)
            fetchCaptcha() // 失败刷新图形验证码
        }
    } catch (e) {
        setError('发送失败')
    }
}
```

- [x] **Step 4: Commit**
```bash
git add frontend/
git commit -m "feat(ui): add captcha fetching and email code sending logic"
```

---

### Task 5: 前端 - UI 视图切换与密码强度条

**Files:**
- Modify: `frontend/src/components/Login.tsx`

- [ ] **Step 1: 实现密码强度检测 Hook / 函数**
```tsx
const getPasswordStrength = (pwd: string) => {
    let score = 0
    if (pwd.length > 8) score += 1
    if (/[a-z]/.test(pwd) && /[A-Z]/.test(pwd)) score += 1
    if (/[0-9]/.test(pwd)) score += 1
    if (/[^a-zA-Z0-9]/.test(pwd)) score += 1
    return score // 0: 弱, 1-2: 中, 3-4: 强
}
```

- [ ] **Step 2: 构建表单动态渲染 (单卡片切换)**
根据 `mode` 渲染不同的表单字段：
- **`login`**: 邮箱、密码、记住我、忘记密码链接。需要验证码时（可由后端 400 提示触发，或默认显示）展示图形验证码框。
- **`register`**: 昵称、邮箱、图形验证码、发送邮件按钮、邮箱验证码、密码（带强度条）。
- **`forgot_password`**: 邮箱、图形验证码、发送邮件按钮、邮箱验证码、新密码（带强度条）。

- [ ] **Step 3: 完善 Submit 逻辑**
在 `handleSubmit` 中根据 `mode` 发送请求到对应的 `/login`, `/register`, `/reset-password`，携带正确的 Payload。重置成功后 `setMode('login')`。

- [ ] **Step 4: Commit**
```bash
git add frontend/
git commit -m "feat(ui): implement dynamic auth forms and password strength meter"
```

---

### Task 6: 更新项目文档 (强制规范)

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Database.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`

- [ ] **Step 1: 更新 API 文档**
添加 `/api/v1/auth/captcha`, `/api/v1/auth/send-code`, `/api/v1/auth/reset-password` 的接口说明，更新 `/login` 和 `/register` 的参数。

- [ ] **Step 2: 更新数据库文档**
增加 `verification_codes` 表，补充 `users` 表的新增字段 `is_email_verified`, `failed_login_attempts`, `locked_until`。

- [ ] **Step 3: 写入开发日志**
在 `InkWords_Development_Plan_and_Log.md` 中记录本次 Auth 功能增强的开发完成情况。

- [ ] **Step 4: Commit**
```bash
git add .trae/documents/
git commit -m "docs: update API, Database and Dev Log for Auth enhancement"
```
