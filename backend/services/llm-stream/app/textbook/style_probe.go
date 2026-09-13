package textbook

import (
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
	"strings"
)

// StyleProbe records two independently reviewable aspects of a representative
// sample: explaining a concept from a real problem and guiding a cold-start
// operation. It is a deterministic structural check, not a replacement for
// the author's approval or a cold-reader trial.
type StyleProbe struct {
	ConceptExplanationPassed bool     `json:"concept_explanation_passed"`
	HandsOnProcedurePassed   bool     `json:"hands_on_procedure_passed"`
	Failures                 []string `json:"failures,omitempty"`
}

func ProbeSampleStyle(markdown string) StyleProbe {
	return probeSampleStyleForProfile(markdown, sharedtextbook.ChapterProfileConcept)
}

func probeSampleStyleForProfile(markdown string, profile sharedtextbook.ChapterProfile) StyleProbe {
	conceptMissing := missingStyleMarkers(markdown, headingsForProfile(profile, []string{
		"## 先遇到一个真实问题",
		"## 先看全貌",
		"## 为什么要先登记路由",
		"## 跟着源码走三步",
	}))
	handsOnMissing := missingStyleMarkers(markdown, []string{
		"## 动手",
		"执行",
		"预期",
		"若",
	})
	if !containsAny(markdown, "打开", "新建") {
		handsOnMissing = append(handsOnMissing, "打开或新建")
	}
	failures := make([]string, 0, len(conceptMissing)+len(handsOnMissing))
	for _, marker := range conceptMissing {
		failures = append(failures, "concept_explanation_missing: "+marker)
	}
	for _, marker := range handsOnMissing {
		failures = append(failures, "hands_on_procedure_missing: "+marker)
	}
	return StyleProbe{ConceptExplanationPassed: len(conceptMissing) == 0, HandsOnProcedurePassed: len(handsOnMissing) == 0, Failures: failures}
}

func missingStyleMarkers(markdown string, markers []string) []string {
	missing := make([]string, 0)
	for _, marker := range markers {
		if !strings.Contains(markdown, marker) {
			missing = append(missing, marker)
		}
	}
	return missing
}
