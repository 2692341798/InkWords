package projectcourse

import (
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type ChapterDocument struct {
	ChapterID    string                    `json:"chapter_id"`
	ChapterType  sharedkernel.ChapterType  `json:"chapter_type"`
	Title        string                    `json:"title"`
	Markdown     string                    `json:"markdown"`
	Claims       []sharedkernel.Claim      `json:"claims"`
	EvidencePack EvidencePack              `json:"evidence_pack"`
	Lab          *sharedkernel.LabManifest `json:"lab,omitempty"`
}

type chapterContract struct {
	requiredSections []string
	requiresOfficial bool
	requiresLab      bool
}

var chapterContracts = map[sharedkernel.ChapterType]chapterContract{
	sharedkernel.ChapterProjectMap:      {requiredSections: []string{"项目地图", "主链路"}},
	sharedkernel.ChapterTechnicalTheory: {requiredSections: []string{"原理", "项目中的应用", "替代方案"}, requiresOfficial: true},
	sharedkernel.ChapterMainFlow:        {requiredSections: []string{"主链路", "数据流", "练习"}},
	sharedkernel.ChapterModuleDeepDive:  {requiredSections: []string{"模块职责", "源码证据", "练习"}},
	sharedkernel.ChapterDesignTradeoff:  {requiredSections: []string{"设计取舍", "替代方案", "边界"}},
	sharedkernel.ChapterHandsOnLab:      {requiredSections: []string{"任务", "提示", "验收"}, requiresLab: true},
	sharedkernel.ChapterTroubleshooting: {requiredSections: []string{"故障现象", "排查", "修复"}},
	sharedkernel.ChapterChallenge:       {requiredSections: []string{"挑战", "验收标准", "变式"}, requiresLab: true},
}

func (d ChapterDocument) ValidateContract() error {
	if strings.TrimSpace(d.ChapterID) == "" || strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.Markdown) == "" {
		return fmt.Errorf("chapter id, title and markdown are required")
	}
	if err := d.ChapterType.Validate(); err != nil {
		return err
	}
	contract, ok := chapterContracts[d.ChapterType]
	if !ok {
		return fmt.Errorf("chapter type %q has no contract", d.ChapterType)
	}
	for _, section := range contract.requiredSections {
		if !strings.Contains(d.Markdown, section) {
			return fmt.Errorf("chapter type %q requires section %q", d.ChapterType, section)
		}
	}
	if contract.requiresOfficial && len(d.EvidencePack.OfficialSources) == 0 {
		return fmt.Errorf("chapter type %q requires official sources", d.ChapterType)
	}
	if contract.requiresLab && d.Lab == nil {
		return fmt.Errorf("chapter type %q requires a lab manifest", d.ChapterType)
	}
	if err := ValidateClaimLedger(d.EvidencePack, d.Claims); err != nil {
		return err
	}
	if d.Lab != nil {
		if err := d.Lab.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func chapterContractFor(chapterType sharedkernel.ChapterType) (required []string, requiresOfficial, requiresLab bool, ok bool) {
	contract, ok := chapterContracts[chapterType]
	if !ok {
		return nil, false, false, false
	}
	return append([]string(nil), contract.requiredSections...), contract.requiresOfficial, contract.requiresLab, true
}

func ValidateChapterSet(chapters []ChapterDocument, dependencies map[string][]string) error {
	known := make(map[string]bool, len(chapters))
	for _, chapter := range chapters {
		if err := chapter.ValidateContract(); err != nil {
			return fmt.Errorf("chapter %s: %w", chapter.ChapterID, err)
		}
		if known[chapter.ChapterID] {
			return fmt.Errorf("duplicate chapter %q", chapter.ChapterID)
		}
		known[chapter.ChapterID] = true
	}
	for chapterID, prerequisites := range dependencies {
		if !known[chapterID] {
			return fmt.Errorf("dependency references unknown chapter %q", chapterID)
		}
		for _, prerequisite := range prerequisites {
			if !known[prerequisite] {
				return fmt.Errorf("chapter %q references unknown prerequisite %q", chapterID, prerequisite)
			}
		}
	}
	return nil
}
