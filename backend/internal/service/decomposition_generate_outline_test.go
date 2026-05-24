package service

import (
	"strings"
	"testing"

	"inkwords-backend/internal/prompt"
)

func TestOutlineScenarioHint(t *testing.T) {
	t.Run("uses beginner walkthrough learning path hint", func(t *testing.T) {
		got := outlineScenarioHint(prompt.ScenarioModeBeginnerWalkthrough)
		if got == "" || !containsAll(got, "环境准备", "目录结构", "排错") {
			t.Fatalf("expected beginner walkthrough hint, got %q", got)
		}
	})

	t.Run("uses open book review lookup hint", func(t *testing.T) {
		got := outlineScenarioHint(prompt.ScenarioModeOpenBookExamReview)
		if got == "" || !containsAll(got, "考点", "题型", "速查") {
			t.Fatalf("expected open book exam review hint, got %q", got)
		}
	})

	t.Run("falls back to interpretation hint", func(t *testing.T) {
		got := outlineScenarioHint(prompt.ScenarioModeEbookInterpretation)
		if got == "" || !containsAll(got, "篇章", "主题脉络", "连贯阅读") {
			t.Fatalf("expected ebook interpretation hint, got %q", got)
		}
	})
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
