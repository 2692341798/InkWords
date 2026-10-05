package textbook

import (
	"fmt"
	"strings"
)

type LearningArcStage struct {
	Stage            LearningStage `json:"stage"`
	Objective        string        `json:"objective"`
	Prerequisites    []string      `json:"prerequisites,omitempty"`
	AllowedHintLevel int           `json:"allowed_hint_level"`
	SuccessEvidence  []string      `json:"success_evidence"`
	RecoveryRoute    string        `json:"recovery_route"`
}

func (stage LearningArcStage) Validate() error {
	if err := stage.Stage.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(stage.Objective) == "" || strings.TrimSpace(stage.RecoveryRoute) == "" || len(stage.SuccessEvidence) == 0 {
		return fmt.Errorf("learning arc stage %q is incomplete", stage.Stage)
	}
	if stage.AllowedHintLevel < 0 || stage.AllowedHintLevel > 3 {
		return fmt.Errorf("learning arc stage %q has an invalid hint level", stage.Stage)
	}
	return nil
}

type LearningArc struct {
	Stages []LearningArcStage `json:"stages"`
}

func DefaultLearningArc() LearningArc {
	order := defaultLearningStageOrder()
	stages := make([]LearningArcStage, 0, len(order))
	for _, stage := range order {
		stages = append(stages, LearningArcStage{Stage: stage})
	}
	return LearningArc{Stages: stages}
}

func (arc LearningArc) Validate() error {
	expected := defaultLearningStageOrder()
	if len(arc.Stages) != len(expected) {
		return fmt.Errorf("learning arc requires %d stages", len(expected))
	}
	for index, stage := range arc.Stages {
		if stage.Stage != expected[index] {
			return fmt.Errorf("learning arc stage %d must be %q", index, expected[index])
		}
		if err := stage.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type LearningObjective struct {
	ID            string             `json:"id"`
	ChapterID     string             `json:"chapter_id"`
	Text          string             `json:"text"`
	RequiredModes []LearningTaskMode `json:"required_modes"`
}

func (objective LearningObjective) Validate() error {
	if strings.TrimSpace(objective.ID) == "" || strings.TrimSpace(objective.ChapterID) == "" || strings.TrimSpace(objective.Text) == "" || len(objective.RequiredModes) == 0 {
		return fmt.Errorf("learning objective is incomplete")
	}
	for _, mode := range objective.RequiredModes {
		if err := mode.Validate(); err != nil {
			return err
		}
	}
	return nil
}
