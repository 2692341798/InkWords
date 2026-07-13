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
	if document.ChapterType == sharedkernel.ChapterTechnicalTheory && len(document.EvidencePack.OfficialSources) == 0 {
		add("official_source", sharedkernel.GateHardFail, "technical theory chapter requires an official source")
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
