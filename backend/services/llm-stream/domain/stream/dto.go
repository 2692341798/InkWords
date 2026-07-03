package stream

import "strings"

type Chapter struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Sort    int      `json:"sort"`
	Files   []string `json:"files"`
	Action  string   `json:"action,omitempty"`
}

type ModuleCard struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GenerateRequest struct {
	SourceContent    string    `json:"source_content"`
	SourceType       string    `json:"source_type"`
	Topic            string    `json:"topic"`
	Outline          []Chapter `json:"outline"`
	GitURL           string    `json:"git_url"`
	SubDir           string    `json:"sub_dir"`
	SelectedModules  []string  `json:"selected_modules"`
	SeriesTitle      string    `json:"series_title"`
	ParentID         string    `json:"parent_id"`
	ArticleStyle     string    `json:"article_style"`
	ScenarioMode     string    `json:"scenario_mode"`
	PromptProfileKey string    `json:"prompt_profile_key"`
	DocumentKind     string    `json:"document_kind"`
}

// Normalize fills backward-compatible aliases so old task payloads still work
// after the task-center contract moved to source_content/source_type.
func (r GenerateRequest) Normalize() GenerateRequest {
	if strings.TrimSpace(r.SourceContent) == "" && strings.TrimSpace(r.Topic) != "" {
		r.SourceContent = strings.TrimSpace(r.Topic)
	}
	if strings.TrimSpace(r.SourceType) == "" && strings.TrimSpace(r.Topic) != "" {
		r.SourceType = "topic"
	}
	return r
}

type PolishRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}
