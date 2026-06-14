package user

import (
	"context"
	"encoding/json"
	"errors"
	"sort"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"inkwords-backend/shared/kernel/prompt"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetUserByID(ctx context.Context, uid uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) GetProfile(ctx context.Context, uid uuid.UUID) (*ProfileData, error) {
	user, err := s.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	tokenLimit := user.TokenLimit
	if tokenLimit == 0 {
		tokenLimit = 1000000000
	}

	connectedPlatforms := make([]string, 0)
	if user.GithubID != nil && *user.GithubID != "" {
		connectedPlatforms = append(connectedPlatforms, "github")
	}
	if user.WechatOpenID != nil && *user.WechatOpenID != "" {
		connectedPlatforms = append(connectedPlatforms, "wechat")
	}

	return &ProfileData{
		Username:           user.Username,
		Email:              user.Email,
		AvatarURL:          user.AvatarURL,
		SubscriptionTier:   user.SubscriptionTier,
		TokensUsed:         user.TokensUsed,
		TokenLimit:         tokenLimit,
		ConnectedPlatforms: connectedPlatforms,
	}, nil
}

func (s *Service) UpdateUsername(ctx context.Context, uid uuid.UUID, username string) error {
	if len(username) < 2 || len(username) > 20 {
		return errors.New("用户名长度必须在 2 到 20 个字符之间")
	}
	return s.repo.UpdateUsername(ctx, uid, username)
}

func (s *Service) UpdateAvatarURL(ctx context.Context, uid uuid.UUID, avatarURL string) error {
	return s.repo.UpdateAvatarURL(ctx, uid, avatarURL)
}

func (s *Service) GetStats(ctx context.Context, uid uuid.UUID) (*StatsData, error) {
	user, err := s.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}

	totalArticles, err := s.repo.CountArticles(ctx, uid)
	if err != nil {
		return nil, err
	}

	totalWords, err := s.repo.SumWords(ctx, uid)
	if err != nil {
		return nil, err
	}

	blogs, err := s.repo.ListBlogsWithTechStacks(ctx, uid)
	if err != nil {
		return nil, err
	}

	stackMap := make(map[string]int)
	for _, blog := range blogs {
		var stacks []string
		if len(blog.TechStacks) > 0 {
			if err := json.Unmarshal(blog.TechStacks, &stacks); err == nil {
				for _, stack := range stacks {
					stackMap[stack]++
				}
			}
		}
	}

	techStackStats := make([]TechStackStat, 0, len(stackMap))
	for k, v := range stackMap {
		techStackStats = append(techStackStats, TechStackStat{Name: k, Count: v})
	}

	sort.Slice(techStackStats, func(i, j int) bool {
		return techStackStats[i].Count > techStackStats[j].Count
	})

	if len(techStackStats) > 20 {
		techStackStats = techStackStats[:20]
	}

	estimatedCost := (float64(user.TokensUsed) / 1000000.0) * 2.3

	return &StatsData{
		TokensUsed:     user.TokensUsed,
		EstimatedCost:  estimatedCost,
		TotalArticles:  totalArticles,
		TotalWords:     totalWords,
		TechStackStats: techStackStats,
	}, nil
}

func (s *Service) GetPromptSettings(ctx context.Context, uid uuid.UUID) (*PromptSettingsResponse, error) {
	defaults := map[string]string{
		string(prompt.ArticleStyleGeneral):          prompt.DefaultRequirements(prompt.ArticleStyleGeneral),
		string(prompt.ArticleStyleBeginnerTutorial): prompt.DefaultRequirements(prompt.ArticleStyleBeginnerTutorial),
		string(prompt.ArticleStyleExamReview):       prompt.DefaultRequirements(prompt.ArticleStyleExamReview),
	}

	styles := []PromptStyleItem{
		{Key: string(prompt.ArticleStyleGeneral), Label: "通用技术博客"},
		{Key: string(prompt.ArticleStyleBeginnerTutorial), Label: "小白手把手教程"},
		{Key: string(prompt.ArticleStyleExamReview), Label: "备考复习文章"},
	}

	overrides := map[string]string{}
	row, err := s.repo.GetPromptSettings(ctx, uid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil && row != nil && len(row.Overrides) > 0 {
		_ = json.Unmarshal(row.Overrides, &overrides)
	}

	return &PromptSettingsResponse{
		Styles:    styles,
		Defaults:  defaults,
		Overrides: overrides,
	}, nil
}

func (s *Service) UpdatePromptSettings(ctx context.Context, uid uuid.UUID, patch map[string]string) error {
	merged := map[string]string{}
	row, err := s.repo.GetPromptSettings(ctx, uid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && row != nil && len(row.Overrides) > 0 {
		_ = json.Unmarshal(row.Overrides, &merged)
	}

	for k, v := range patch {
		merged[k] = v
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return err
	}

	return s.repo.UpsertPromptSettings(ctx, uid, datatypes.JSON(b))
}
