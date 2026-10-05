package textbook

import (
	"fmt"
	"strings"
)

// GenerationTaskClass is the teaching decision that determines an approved
// model tier. Deterministic work (source parsing, hashing, and quality gates)
// intentionally has no class and must not call a model through this policy.
type GenerationTaskClass string

const (
	GenerationTaskBlueprint       GenerationTaskClass = "blueprint"
	GenerationTaskCoreConcept     GenerationTaskClass = "core_concept"
	GenerationTaskEditorialReview GenerationTaskClass = "editorial_review"
	GenerationTaskSampleChapter   GenerationTaskClass = "sample_chapter"
)

// TaskModelPolicy centralizes model selection so high-cost models are limited
// to blueprint, core-mechanism, and editorial-review work. A representative
// sample chapter uses the standard tier unless the caller explicitly models it
// as a core-concept task.
type TaskModelPolicy struct {
	CoreModel     string
	StandardModel string
}

func (policy TaskModelPolicy) ModelFor(task GenerationTaskClass) (string, error) {
	var model string
	switch task {
	case GenerationTaskBlueprint, GenerationTaskCoreConcept, GenerationTaskEditorialReview:
		model = policy.CoreModel
	case GenerationTaskSampleChapter:
		model = policy.StandardModel
	default:
		return "", fmt.Errorf("unsupported textbook model task %q", task)
	}
	if model = strings.TrimSpace(model); model == "" {
		return "", fmt.Errorf("textbook model policy has no model for %s", task)
	}
	return model, nil
}
