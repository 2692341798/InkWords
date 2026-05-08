package auth

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

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

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

func (s *Service) GetAuthURL(provider string) (string, error) {
	config, err := getOAuthConfig(provider)
	if err != nil {
		return "", err
	}
	state := "inkwords-state"
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *Service) HandleCallback(ctx context.Context, provider, code string) (string, *model.User, error) {
	config, err := getOAuthConfig(provider)
	if err != nil {
		return "", nil, err
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return "", nil, fmt.Errorf("%w: failed to exchange token: %v", ErrOAuthCallback, err)
	}

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

	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %v", err)
	}

	return jwtToken, user, nil
}

func (s *Service) fetchGithubUser(ctx context.Context, accessToken string) (*model.User, error) {
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

	email := ghUser.Email
	if email == "" {
		email = fmt.Sprintf("%s@github.local", ghUser.Login)
	}

	user, err := s.repo.GetByGithubIDOrEmail(ctx, ghIDStr, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newUser := model.User{
				ID:        uuid.New(),
				Username:  ghUser.Login,
				Email:     email,
				GithubID:  &ghIDStr,
				AvatarURL: ghUser.AvatarURL,
			}
			if err := s.repo.Create(ctx, &newUser); err != nil {
				return nil, err
			}
			return &newUser, nil
		}
		return nil, err
	}

	if user.GithubID == nil || *user.GithubID != ghIDStr {
		tempUser := &model.User{
			Email:     email,
			Username:  ghUser.Login,
			AvatarURL: ghUser.AvatarURL,
			GithubID:  &ghIDStr,
		}
		return tempUser, ErrEmailExistsBindRequired
	}

	user.Username = ghUser.Login
	user.AvatarURL = ghUser.AvatarURL
	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (string, error) {
	if !s.VerifyCaptcha(req.CaptchaID, req.CaptchaValue) {
		return "", errors.New("图形验证码错误或已过期")
	}

	if len(req.Password) < 8 {
		return "", errors.New("密码长度必须至少为 8 位")
	}

	count, err := s.repo.CountByEmailOrUsername(ctx, req.Email, req.Username)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", errors.New("邮箱或用户名已被注册")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := model.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.Create(ctx, &user); err != nil {
		return "", errors.New("创建用户失败，请稍后重试")
	}

	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, nil
}

func (s *Service) Login(ctx context.Context, email, password, captchaID, captchaValue string) (string, *model.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("邮箱或密码错误")
		}
		return "", nil, err
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return "", nil, errors.New("账号已锁定，请稍后再试")
	}

	if user.PasswordHash == "" {
		return "", nil, errors.New("请使用第三方登录或重置密码")
	}

	if user.FailedLoginAttempts >= 3 {
		if captchaID == "" || !s.VerifyCaptcha(captchaID, captchaValue) {
			return "", nil, errors.New("请输入正确的图形验证码")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			lockTime := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockTime
		}
		_ = s.repo.Save(ctx, user)
		return "", nil, errors.New("邮箱或密码错误")
	}

	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	_ = s.repo.Save(ctx, user)

	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate jwt: %w", err)
	}

	return jwtToken, user, nil
}

func (s *Service) BindGithub(ctx context.Context, email, password, ghIDStr, username, avatarURL string) (string, *model.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("用户不存在")
		}
		return "", nil, err
	}

	if user.PasswordHash == "" {
		return "", nil, errors.New("该账号未设置本地密码，无法绑定")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, errors.New("密码错误")
	}

	user.GithubID = &ghIDStr
	if username != "" {
		user.Username = username
	}
	if avatarURL != "" {
		user.AvatarURL = avatarURL
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return "", nil, err
	}

	jwtToken, err := jwt.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return "", nil, fmt.Errorf("生成 JWT 失败: %w", err)
	}

	return jwtToken, user, nil
}

func (s *Service) GenerateCaptcha() (string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := c.Generate()
	return id, b64s, err
}

func (s *Service) VerifyCaptcha(id string, verifyValue string) bool {
	return store.Verify(id, verifyValue, true)
}

