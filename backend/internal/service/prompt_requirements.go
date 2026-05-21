package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"inkwords-backend/internal/model"
	"inkwords-backend/internal/prompt"
)

type PromptRequirementsService struct {
	db *gorm.DB
}

func NewPromptRequirementsService(db *gorm.DB) *PromptRequirementsService {
	return &PromptRequirementsService{db: db}
}

func (s *PromptRequirementsService) Resolve(ctx context.Context, userID uuid.UUID, style prompt.ArticleStyle) (string, error) {
	if !style.IsValid() {
		style = prompt.ArticleStyleGeneral
	}

	var row model.UserPromptSettings
	if err := s.db.WithContext(ctx).First(&row, "user_id = ?", userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return prompt.DefaultRequirements(style), nil
		}
		return "", err
	}

	if len(row.Overrides) == 0 {
		return prompt.DefaultRequirements(style), nil
	}

	var overrides map[string]string
	if err := json.Unmarshal(row.Overrides, &overrides); err != nil {
		return prompt.DefaultRequirements(style), nil
	}

	if v, ok := overrides[string(style)]; ok {
		if v == "" {
			return prompt.DefaultRequirements(style), nil
		}
		return v, nil
	}

	return prompt.DefaultRequirements(style), nil
}

