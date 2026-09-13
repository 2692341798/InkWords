package mastery

import "strings"

// AssessmentCodeSpanFormat identifies inclusive, one-based physical file lines.
const AssessmentCodeSpanFormat = "inkwords.code-line-range.v1"

// AssessmentCodeSpan selects original bytes from one immutable learner file.
// It authorizes neither execution nor a claim that the selected code is correct.
type AssessmentCodeSpan struct {
	Format    string `json:"format"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (input AssessmentInput) resolveJudgmentQuote(judgment AssessmentJudgment) (string, error) {
	span := judgment.AnswerSpan
	if span == nil {
		return judgment.AnswerQuote, nil
	}
	invalid := &AssessmentDecisionError{Rule: "decision_answer_span", CriterionID: judgment.CriterionID}
	if input.LearnerArtifact == nil || judgment.AnswerPath == "" || judgment.AnswerQuote != "" || span.Format != AssessmentCodeSpanFormat || span.StartLine < 1 || span.EndLine < span.StartLine {
		return "", invalid
	}
	for _, file := range input.LearnerArtifact.Files {
		if file.Path != judgment.AnswerPath {
			continue
		}
		lines := strings.SplitAfter(file.Content, "\n")
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		if span.EndLine > len(lines) {
			return "", invalid
		}
		quote := strings.Join(lines[span.StartLine-1:span.EndLine], "")
		if !boundedText(quote, 20000) {
			return "", invalid
		}
		return quote, nil
	}
	return "", invalid
}
