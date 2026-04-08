package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"golang.org/x/crypto/bcrypt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/pkg/jwt"
)

var (
	ErrUnsupportedProvider     = errors.New("unsupported oauth provider")
	ErrOAuthCallback           = errors.New("oauth callback failed")
	ErrEmailExistsBindRequired = errors.New("email exists, bind required")
	store                      = base64Captcha.DefaultMemStore
)

type AuthService struct {
	db *gorm.DB
}

// NewAuthService 创建 AuthService 实例并注入数据库依赖
func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
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
			if errors.Is(err, ErrEmailExistsBindRequired) {
				return "", user, err
			}
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
	// 查询是否已存在对应的 GithubID 或邮箱
	err = s.db.Where("github_id = ?", ghIDStr).Or("email = ?", email).First(&user).Error
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
			if err := s.db.Create(&user).Error; err != nil {
				return nil, err
			}
			return &user, nil
		}
		return nil, err
	}

	// 如果找到的用户 GithubID 不匹配（即通过 Email 找到的，且尚未绑定）
	if user.GithubID == nil || *user.GithubID != ghIDStr {
		// 不自动绑定，返回需要绑定的错误
		// 临时将从 Github 获取的信息放在 user 结构体里返回（不上库），供前端展示使用
		tempUser := &model.User{
			Email:     email,
			Username:  ghUser.Login,
			AvatarURL: ghUser.AvatarURL,
			GithubID:  &ghIDStr,
		}
		return tempUser, ErrEmailExistsBindRequired
	}

	// 如果已绑定，则更新基本信息
	user.Username = ghUser.Login
	user.AvatarURL = ghUser.AvatarURL
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Register 注册新用户，使用邮箱和密码
func (s *AuthService) Register(req struct {
	Username     string `json:"username" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required"`
	CaptchaID    string `json:"captcha_id" binding:"required"`
	CaptchaValue string `json:"captcha_value" binding:"required"`
}) (string, error) {
	// 校验图形验证码
	if !s.VerifyCaptcha(req.CaptchaID, req.CaptchaValue) {
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

	// 生成密码哈希
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建新用户
	user := model.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return "", errors.New("创建用户失败，请稍后重试")
	}

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, nil
}

// Login 用户登录，返回 JWT Token 和用户信息
func (s *AuthService) Login(email, password, captchaID, captchaValue string) (string, *model.User, error) {
	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
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
		s.db.Save(&user)
		return "", nil, errors.New("邮箱或密码错误")
	}

	// 成功则重置状态
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	s.db.Save(&user)

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, &user, nil
}

// BindGithub 验证密码并绑定 GitHub 账号
func (s *AuthService) BindGithub(email, password, ghIDStr, username, avatarURL string) (string, *model.User, error) {
	var user model.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户不存在")
		}
		return "", nil, err
	}

	// 验证密码
	if user.PasswordHash == "" {
		return "", nil, errors.New("该账号未设置本地密码，无法绑定")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("密码错误")
	}

	// 更新用户信息
	user.GithubID = &ghIDStr
	if username != "" {
		user.Username = username
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	if err := s.db.Save(&user).Error; err != nil {
		return "", nil, err
	}

	// 生成 JWT Token
	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("生成 JWT 失败: %w", err)
	}

	return jwtToken, &user, nil
}
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
