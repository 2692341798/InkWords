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

	t.Run("ebook uses chapter interpretation hint", func(t *testing.T) {
		got := outlineScenarioHint(prompt.ScenarioModeEbookInterpretation)
		if got == "" || !containsAll(got, "篇章", "主题脉络", "只做文本解读") {
			t.Fatalf("expected ebook interpretation hint, got %q", got)
		}
		// 经典文本导读：不引导模型做技术拆分
		if containsAny(got, "架构", "源码", "模块", "项目") {
			t.Fatalf("ebook hint should not contain tech-split language, got %q", got)
		}
	})
}

func TestOutlineBaseInstruction(t *testing.T) {
	t.Run("ebook gets text interpreter persona not architect", func(t *testing.T) {
		got := outlineBaseInstruction(prompt.ScenarioModeEbookInterpretation)
		if got == "" || !containsAll(got, "逐章", "原文") {
			t.Fatalf("expected ebook base instruction, got %q", got)
		}
		if containsAny(got, "架构师", "源码", "模块", "高级") {
			t.Fatalf("ebook base instruction should not be architect persona, got %q", got)
		}
	})

	t.Run("beginner gets architect persona", func(t *testing.T) {
		got := outlineBaseInstruction(prompt.ScenarioModeBeginnerWalkthrough)
		if got == "" || !containsAll(got, "源码", "模块") {
			t.Fatalf("expected beginner walkthrough base instruction, got %q", got)
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

func containsAny(text string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}
