package stream

import (
	"testing"

	"inkwords-backend/internal/prompt"
)

func TestNormalizeScenarioMode_DefaultsBySourceType(t *testing.T) {
	t.Run("git defaults to beginner walkthrough", func(t *testing.T) {
		got := normalizeScenarioMode("", "git")
		if got != prompt.ScenarioModeBeginnerWalkthrough {
			t.Fatalf("expected beginner walkthrough, got %q", got)
		}
	})

	t.Run("file defaults to ebook interpretation", func(t *testing.T) {
		got := normalizeScenarioMode("", "file")
		if got != prompt.ScenarioModeEbookInterpretation {
			t.Fatalf("expected ebook interpretation, got %q", got)
		}
	})

	t.Run("valid explicit mode wins", func(t *testing.T) {
		got := normalizeScenarioMode(string(prompt.ScenarioModeOpenBookExamReview), "file")
		if got != prompt.ScenarioModeOpenBookExamReview {
			t.Fatalf("expected explicit open book exam review, got %q", got)
		}
	})
}
