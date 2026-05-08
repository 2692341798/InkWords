package project

type Chapter struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Sort    int      `json:"sort"`
	Files   []string `json:"files"`
	Action  string   `json:"action,omitempty"`
}

type OutlineResult struct {
	SeriesTitle string    `json:"series_title"`
	Chapters    []Chapter `json:"chapters"`
	ParentID    string    `json:"parent_id,omitempty"`
}

type ModuleCard struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ScanRequest struct {
	GitURL string `json:"git_url" binding:"required"`
}

type AnalyzeRequest struct {
	GitURL string `json:"git_url" binding:"required"`
	SubDir string `json:"sub_dir"`
}
