package projectcourse

import (
	"fmt"
	"strings"

	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

type QualityGateReport struct {
	Result sharedkernel.GateResult   `json:"result"`
	Checks []sharedkernel.GateReport `json:"checks"`
}

func RunChapterQualityGates(document ChapterDocument, verifiedLab bool) QualityGateReport {
	report := QualityGateReport{Result: sharedkernel.GatePass}
	add := func(name string, result sharedkernel.GateResult, message string) {
		report.Checks = append(report.Checks, sharedkernel.GateReport{Name: name, Result: result, Message: message})
		if result == sharedkernel.GateHardFail {
			report.Result = sharedkernel.GateHardFail
		}
	}
	if err := document.ValidateContract(); err != nil {
		add("chapter_contract", sharedkernel.GateHardFail, err.Error())
	} else {
		add("chapter_contract", sharedkernel.GatePass, "contract valid")
	}
	if err := validateCodeBlockProvenance(document); err != nil {
		add("code_block_provenance", sharedkernel.GateHardFail, err.Error())
	} else {
		add("code_block_provenance", sharedkernel.GatePass, "code blocks are bound to evidence or lab artifacts")
	}
	_, requiresOfficial, _, _ := chapterContractFor(document.ChapterType)
	if requiresOfficial && len(document.EvidencePack.OfficialSources) == 0 {
		add("official_source", sharedkernel.GateHardFail, "chapter type requires an official source")
	} else {
		add("official_source", sharedkernel.GatePass, "official source requirement satisfied")
	}
	if document.Lab != nil && !verifiedLab {
		add("lab_verification", sharedkernel.GateHardFail, "lab artifact has not passed isolated verification")
	} else {
		add("lab_verification", sharedkernel.GatePass, "lab verification satisfied or not applicable")
	}
	if !strings.Contains(document.Markdown, document.Title) {
		add("title_in_document", sharedkernel.GateHardFail, fmt.Sprintf("markdown does not contain chapter title %q", document.Title))
	} else {
		add("title_in_document", sharedkernel.GatePass, "title present")
	}
	return report
}

func validateCodeBlockProvenance(document ChapterDocument) error {
	lines := strings.Split(document.Markdown, "\n")
	var inFence bool
	var sourceType, sourceRef string
	var body []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFence {
			if !strings.HasPrefix(trimmed, "```") {
				continue
			}
			marker := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			parts := strings.SplitN(marker, ":", 2)
			if len(parts) != 2 || (parts[0] != "source" && parts[0] != "artifact") || strings.TrimSpace(parts[1]) == "" {
				return fmt.Errorf("code fence must declare source:<evidence_id> or artifact:<path>")
			}
			inFence, sourceType, sourceRef, body = true, parts[0], strings.TrimSpace(parts[1]), nil
			continue
		}
		if trimmed == "```" {
			content := strings.Join(body, "\n")
			if sourceType == "source" {
				expected, ok := document.EvidencePack.SourceContent[sourceRef]
				if !ok || content != expected {
					return fmt.Errorf("code fence source %q does not match evidence content", sourceRef)
				}
			} else if !labFileContentMatches(document.Lab, sourceRef, content) {
				return fmt.Errorf("code fence artifact %q does not match a lab file", sourceRef)
			}
			inFence = false
			continue
		}
		body = append(body, line)
	}
	if inFence {
		return fmt.Errorf("unterminated code fence")
	}
	return nil
}

func labFileContentMatches(lab *sharedkernel.LabManifest, path, content string) bool {
	if lab == nil {
		return false
	}
	groups := [][]sharedkernel.LabFile{lab.Starter, lab.Solution, lab.Tests}
	for _, checkpoint := range lab.Checkpoints {
		groups = append(groups, checkpoint.Files)
	}
	for _, files := range groups {
		for _, file := range files {
			if file.Path == path && file.Content == content {
				return true
			}
		}
	}
	return false
}
