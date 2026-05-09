package parser

type FileChunk struct {
	Dir     string
	Content string
}

type GitFetcher struct{}

func NewGitFetcher() *GitFetcher {
	return &GitFetcher{}
}

const maxChunkChars = 2000000
const maxTotalChunks = 15

const largeRepoTruncationHint = "【系统提示】由于该项目体量极其庞大，系统已执行优雅降级，自动截断了后续文件（仅保留了前15个核心模块的分块）。请你在生成的博客引言或开头中，自然地向读者说明：由于项目过于庞大，本文仅抽取分析了其核心的若干模块代码，并未包含全量内容。"

type GitTreeResponse struct {
	Sha  string `json:"sha"`
	Url  string `json:"url"`
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}
