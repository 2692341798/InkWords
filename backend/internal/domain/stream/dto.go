package stream

import "inkwords-backend/internal/service"

type GenerateRequest struct {
	SourceContent   string            `json:"source_content"`
	SourceType      string            `json:"source_type"`
	Outline         []service.Chapter `json:"outline"`
	GitURL          string            `json:"git_url"`
	SubDir          string            `json:"sub_dir"`
	SelectedModules []string          `json:"selected_modules"`
	SeriesTitle     string            `json:"series_title"`
	ParentID        string            `json:"parent_id"`
}

type PolishRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}

