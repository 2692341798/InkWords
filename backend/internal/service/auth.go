package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/gomail.v2"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"gorm.io/gorm"

	"inkwords-backend/internal/db"
	"inkwords-backend/internal/model"
	"inkwords-backend/pkg/jwt"
)

var (
	ErrUnsupportedProvider = errors.New("unsupported oauth provider")
	ErrOAuthCallback       = errors.New("oauth callback failed")
	store                  = base64Captcha.DefaultMemStore
)

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// getOAuthConfig 获取 OAuth 配置
func getOAuthConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "github":
		return &oauth2.Config{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		}, nil
	default:
		return nil, ErrUnsupportedProvider
	}
}

// GetAuthURL 生成 OAuth 授权地址
func (s *AuthService) GetAuthURL(provider string) (string, error) {
	config, err := getOAuthConfig(provider)
	if err != nil {
		return "", err
	}

	// state 推荐使用随机数防止 CSRF，这里简单处理
	state := "inkwords-state"
	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return url, nil
}

// HandleCallback 处理 OAuth 回调并生成本系统 JWT
func (s *AuthService) HandleCallback(ctx context.Context, provider, code string) (string, *model.User, error) {
	config, err := getOAuthConfig(provider)
	if err != nil {
		return "", nil, err
	}

	// 1. 获取 Access Token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to exchange token: %v", ErrOAuthCallback, err)
	}

	// 2. 根据 provider 获取用户信息
	var user *model.User
	if provider == "github" {
		user, err = s.fetchGithubUser(ctx, token.AccessToken)
		if err != nil {
			return "", nil, err
		}
	}

	// 3. 生成 JWT token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %v", err)
	}

	return jwtToken, user, nil
}

// fetchGithubUser 获取 Github 用户信息并保存/更新到数据库
func (s *AuthService) fetchGithubUser(ctx context.Context, accessToken string) (*model.User, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status: %d", resp.StatusCode)
	}

	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, err
	}

	ghIDStr := fmt.Sprintf("%d", ghUser.ID)

	// 处理 email 可能为空的情况（如果 Github 用户隐藏了邮箱）
	// 这里简化处理，真实环境可能需要再调用 https://api.github.com/user/emails
	email := ghUser.Email
	if email == "" {
		email = fmt.Sprintf("%s@github.local", ghUser.Login)
	}

	var user model.User
	// 使用 GORM 的 FirstOrCreate 实现 UPSERT
	err = db.DB.Where("github_id = ?", ghIDStr).Or("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新用户
			user = model.User{
				ID:        uuid.New(),
				Username:  ghUser.Login,
				Email:     email,
				GithubID:  &ghIDStr,
				AvatarURL: ghUser.AvatarURL,
			}
			if err := db.DB.Create(&user).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		// 更新用户信息
		user.Username = ghUser.Login
		user.AvatarURL = ghUser.AvatarURL
		user.GithubID = &ghIDStr
		if err := db.DB.Save(&user).Error; err != nil {
			return nil, err
		}
	}

	return &user, nil
}

// Register 注册新用户，使用邮箱和密码
func (s *AuthService) Register(email, username, password, code string) (string, *model.User, error) {
	// 校验验证码
	var vc model.VerificationCode
	if err := db.DB.Where("email = ? AND code = ? AND type = ? AND expires_at > ?", email, code, "register", time.Now()).First(&vc).Error; err != nil {
		return "", nil, errors.New("验证码错误或已过期")
	}

	// 校验密码强度
	if len(password) < 8 {
		return "", nil, errors.New("密码长度必须大于或等于8位")
	}

	// 检查邮箱是否已存在
	var existingUser model.User
	if err := db.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		return "", nil, errors.New("邮箱已被注册")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, err
	}

	// 生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:              uuid.New(),
		Email:           email,
		Username:        username,
		PasswordHash:    string(hashedPassword),
		IsEmailVerified: true,
	}

	if err := db.DB.Create(user).Error; err != nil {
		return "", nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 验证成功后删除验证码
	db.DB.Delete(&vc)

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, user, nil
}

// Login 用户登录，返回 JWT Token 和用户信息
func (s *AuthService) Login(email, password, captchaID, captchaValue string) (string, *model.User, error) {
	var user model.User
	if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("邮箱或密码错误")
		}
		return "", nil, err
	}

	// 检查锁定状态
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return "", nil, errors.New("账号已锁定，请稍后再试")
	}

	// 检查用户是否有密码（可能是仅通过第三方登录注册的用户）
	if user.PasswordHash == "" {
		return "", nil, errors.New("请使用第三方登录或重置密码")
	}

	// 如果失败次数 >= 3，强制要求 Captcha
	if user.FailedLoginAttempts >= 3 {
		if captchaID == "" || !s.VerifyCaptcha(captchaID, captchaValue) {
			return "", nil, errors.New("请输入正确的图形验证码")
		}
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			lockTime := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockTime
		}
		db.DB.Save(&user)
		return "", nil, errors.New("邮箱或密码错误")
	}

	// 成功则重置状态
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	db.DB.Save(&user)

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, &user, nil
}

// ResetPassword 重置密码
func (s *AuthService) ResetPassword(email, code, newPassword string) error {
	var vc model.VerificationCode
	if err := db.DB.Where("email = ? AND code = ? AND type = ? AND expires_at > ?", email, code, "reset_password", time.Now()).First(&vc).Error; err != nil {
		return errors.New("验证码错误或已过期")
	}

	if len(newPassword) < 8 {
		return errors.New("密码长度必须大于或等于8位")
	}

	var user model.User
	if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return errors.New("用户不存在")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil

	db.DB.Save(&user)
	db.DB.Delete(&vc)

	return nil
}

// GenerateCaptcha 生成图形验证码
func (s *AuthService) GenerateCaptcha() (string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := c.Generate()
	return id, b64s, err
}

// VerifyCaptcha 校验图形验证码
func (s *AuthService) VerifyCaptcha(id string, verifyValue string) bool {
	return store.Verify(id, verifyValue, true)
}

// GenerateRandomCode 生成 6 位随机验证码
func (s *AuthService) GenerateRandomCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n)
}

// SaveVerificationCode 记录验证码到数据库
func (s *AuthService) SaveVerificationCode(email, code, codeType string) error {
	vc := &model.VerificationCode{
		Email:     email,
		Code:      code,
		Type:      codeType,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	return db.DB.Create(vc).Error
}

// SendVerificationEmail 发送验证邮件（含 Mock）
func (s *AuthService) SendVerificationEmail(email, code, codeType string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		// Mock 兜底
		fmt.Printf("======== MOCK EMAIL ========\nTo: %s\nType: %s\nCode: %s\n============================\n", email, codeType, code)
		return nil
	}

	smtpPort := 465 // 或从 env 获取，默认 465
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
