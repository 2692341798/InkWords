package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

type PromptRequirementsService struct {
	db *gorm.DB
}

// NewPromptRequirementsService 创建 Prompt 组装服务。
func NewPromptRequirementsService(db *gorm.DB) *PromptRequirementsService {
	return &PromptRequirementsService{db: db}
}

// Resolve 统一合并场景层、风格层与用户覆盖后的最终 Prompt 要求。
func (s *PromptRequirementsService) Resolve(
	ctx context.Context,
	userID uuid.UUID,
	scenario prompt.ScenarioMode,
	style prompt.ArticleStyle,
) (string, error) {
	if !scenario.IsValid() {
		scenario = prompt.ScenarioModeEbookInterpretation
	}
	if !style.IsValid() {
		style = prompt.ArticleStyleGeneral
	}

	styleRequirements := prompt.DefaultStyleRequirements(scenario, style)
	userStyleOverride := styleRequirements

	var row model.UserPromptSettings
	if err := s.db.WithContext(ctx).First(&row, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return strings.TrimSpace(strings.Join([]string{
				prompt.DefaultScenarioRequirements(scenario),
				userStyleOverride,
			}, "\n\n")), nil
		}
		return "", err
	}

	if len(row.Overrides) > 0 {
		var overrides map[string]string
		if err := json.Unmarshal(row.Overrides, &overrides); err == nil {
			if v, ok := overrides[string(style)]; ok {
				if v == "" {
					userStyleOverride = styleRequirements
				} else {
					userStyleOverride = v
				}
			}
		}
	}

	// Why: 场景层表达“这次任务要产出什么内容”，风格层表达“如何呈现”，
	// 两者需要统一返回给生成链路，避免调用方重复拼接并导致行为漂移。
	return strings.TrimSpace(strings.Join([]string{
		prompt.DefaultScenarioRequirements(scenario),
		userStyleOverride,
	}, "\n\n")), nil
}
