package project

type ScanRequest struct {
	GitURL string `json:"git_url" binding:"required"`
}

type AnalyzeRequest struct {
	GitURL string `json:"git_url" binding:"required"`
	SubDir string `json:"sub_dir"`
}

