package stream

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
	SourceContent   string    `json:"source_content"`
	SourceType      string    `json:"source_type"`
	Outline         []Chapter `json:"outline"`
	GitURL          string    `json:"git_url"`
	SubDir          string    `json:"sub_dir"`
	SelectedModules []string  `json:"selected_modules"`
	SeriesTitle     string    `json:"series_title"`
	ParentID        string    `json:"parent_id"`
	ArticleStyle    string    `json:"article_style"`
}

type PolishRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}
