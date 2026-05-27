package review

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReviewFrontmatter_ReadsOptionalReviewConfig(t *testing.T) {
	t.Parallel()

	content := `---
type: concept
title: "并发控制与速率限制"
related:
  - "[[sources/InkWords 内容生成平台架构解析系列|InkWords 内容生成平台架构解析系列]]"
review:
  enabled: true
  preferred_mode: light_recall
  exclude_from_random: false
  min_interval_days: 3
---

# 并发控制与速率限制

正文内容`

	meta, body := parseFrontmatter(content)

	require.Equal(t, "concept", meta.Type)
	require.Equal(t, "并发控制与速率限制", meta.Title)
	require.Equal(t, []string{`[[sources/InkWords 内容生成平台架构解析系列|InkWords 内容生成平台架构解析系列]]`}, meta.Related)
	require.NotNil(t, meta.Review.Enabled)
	require.True(t, *meta.Review.Enabled)
	require.Equal(t, "light_recall", meta.Review.PreferredMode)
	require.False(t, meta.Review.ExcludeFromRandom)
	require.Equal(t, 3, meta.Review.MinIntervalDays)
	require.Contains(t, body, "正文内容")
}

func TestIsEligibleReviewNote_FiltersSeedAndIndexes(t *testing.T) {
	t.Parallel()

	validBody := strings.Repeat("有效正文", 50)

	require.False(t, IsEligibleReviewNote("wiki/index.md", ReviewFrontmatter{Type: "meta"}, "# index"))
	require.False(t, IsEligibleReviewNote("wiki/concepts/_index.md", ReviewFrontmatter{Type: "concept"}, validBody))
	require.False(t, IsEligibleReviewNote("wiki/concepts/种子页.md", ReviewFrontmatter{Type: "concept"}, "Context extracted from [[foo]]"))
	require.False(t, IsEligibleReviewNote("wiki/concepts/太短了.md", ReviewFrontmatter{Type: "concept"}, "太短"))
	require.False(t, IsEligibleReviewNote("wiki/concepts/显式排除.md", ReviewFrontmatter{
		Type: "concept",
		Review: ReviewConfig{
			ExcludeFromRandom: true,
		},
	}, validBody))
	require.True(t, IsEligibleReviewNote("wiki/concepts/并发控制与速率限制.md", ReviewFrontmatter{Type: "concept"}, validBody))
}
