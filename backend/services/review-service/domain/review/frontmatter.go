package review

import (
	"strings"

	"gopkg.in/yaml.v3"
)

const minimumEligibleReviewBodyRunes = 120

// ReviewConfig 描述单篇笔记可选的复习配置。
type ReviewConfig struct {
	Enabled           *bool  `yaml:"enabled"`
	Difficulty        string `yaml:"difficulty"`
	PreferredMode     string `yaml:"preferred_mode"`
	ExcludeFromRandom bool   `yaml:"exclude_from_random"`
	MinIntervalDays   int    `yaml:"min_interval_days"`
}

// ReviewFrontmatter 表示复习筛选所需的最小 frontmatter 字段集合。
type ReviewFrontmatter struct {
	Type    string       `yaml:"type"`
	Title   string       `yaml:"title"`
	Status  string       `yaml:"status"`
	Related []string     `yaml:"related"`
	Review  ReviewConfig `yaml:"review"`
}

func parseFrontmatter(content string) (ReviewFrontmatter, string) {
	const separator = "\n---\n"
	if !strings.HasPrefix(content, "---\n") {
		return ReviewFrontmatter{}, content
	}

	parts := strings.SplitN(content, separator, 2)
	if len(parts) != 2 {
		return ReviewFrontmatter{}, content
	}

	var meta ReviewFrontmatter
	if err := yaml.Unmarshal([]byte(strings.TrimPrefix(parts[0], "---\n")), &meta); err != nil {
		return ReviewFrontmatter{}, content
	}

	return meta, parts[1]
}

// IsEligibleReviewNote 返回当前笔记是否适合进入首版复习池。
func IsEligibleReviewNote(notePath string, meta ReviewFrontmatter, body string) bool {
	trimmedBody := strings.TrimSpace(body)

	// Why: 复习入口只面向真实概念页，必须屏蔽索引页、系统页和模板 seed 页，
	// 否则用户会抽到空洞或不可复述的内容，直接破坏“知识漫游”的体验。
	if isSystemNotePath(notePath) {
		return false
	}
	if meta.Type != "concept" {
		return false
	}
	if meta.Review.Enabled != nil && !*meta.Review.Enabled {
		return false
	}
	if meta.Review.ExcludeFromRandom {
		return false
	}
	if isSeedLikeBody(trimmedBody) {
		return false
	}
	return len([]rune(trimmedBody)) >= minimumEligibleReviewBodyRunes
}

func isSystemNotePath(notePath string) bool {
	switch notePath {
	case "wiki/index.md", "wiki/hot.md", "wiki/log.md":
		return true
	}
	return strings.HasSuffix(notePath, "/_index.md") || strings.HasSuffix(notePath, "_index.md")
}

func isSeedLikeBody(body string) bool {
	return strings.HasPrefix(body, "Context extracted from [[")
}
