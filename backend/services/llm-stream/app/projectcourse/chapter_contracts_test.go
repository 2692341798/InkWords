package projectcourse

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedkernel "inkwords-backend/shared/kernel/projectcourse"
)

func validChapterDocument() ChapterDocument {
	evidence := sharedkernel.EvidenceRef{EvidenceID: "ev-1", CommitSHA: "0123456789abcdef0123456789abcdef01234567", Path: "main.go", StartLine: 1, EndLine: 2, ContentHash: "sha256:main"}
	return ChapterDocument{ChapterID: "chapter-1", ChapterType: sharedkernel.ChapterMainFlow, Title: "主链路", Markdown: "# 主链路\n正文", Claims: []sharedkernel.Claim{{ClaimID: "claim-1", Text: "入口存在", ClaimType: "project_fact", Confidence: sharedkernel.ConfidenceObserved, EvidenceIDs: []string{"ev-1"}, Status: sharedkernel.ClaimVerified}}, EvidencePack: EvidencePack{ChapterID: "chapter-1", SourceEvidence: []sharedkernel.EvidenceRef{evidence}}}
}

func TestChapterContractAndQualityGate(t *testing.T) {
	document := validChapterDocument()
	document.Markdown = "# 主链路\n## 主链路\n## 数据流\n## 练习\n正文"
	require.NoError(t, document.ValidateContract())
	report := RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GatePass, report.Result)
	document.Claims[0].Status = sharedkernel.ClaimUnsupported
	report = RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GateHardFail, report.Result)
}

func TestEveryChapterTypeHasAnIndependentContract(t *testing.T) {
	for chapterType, expected := range map[sharedkernel.ChapterType][]string{
		sharedkernel.ChapterProjectMap:      {"项目地图", "主链路"},
		sharedkernel.ChapterTechnicalTheory: {"原理", "项目中的应用", "替代方案"},
		sharedkernel.ChapterMainFlow:        {"主链路", "数据流", "练习"},
		sharedkernel.ChapterModuleDeepDive:  {"模块职责", "源码证据", "练习"},
		sharedkernel.ChapterDesignTradeoff:  {"设计取舍", "替代方案", "边界"},
		sharedkernel.ChapterHandsOnLab:      {"任务", "提示", "验收"},
		sharedkernel.ChapterTroubleshooting: {"故障现象", "排查", "修复"},
		sharedkernel.ChapterChallenge:       {"挑战", "验收标准", "变式"},
	} {
		required, _, requiresLab, ok := chapterContractFor(chapterType)
		require.True(t, ok)
		require.Equal(t, expected, required)
		if chapterType == sharedkernel.ChapterHandsOnLab || chapterType == sharedkernel.ChapterChallenge {
			require.True(t, requiresLab)
		}
	}
}

func TestQualityGateRequiresOfficialSourceForTheoryAndLabVerification(t *testing.T) {
	document := validChapterDocument()
	document.ChapterType = sharedkernel.ChapterTechnicalTheory
	report := RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GateHardFail, report.Result)
	document.ChapterType = sharedkernel.ChapterHandsOnLab
	document.Lab = &sharedkernel.LabManifest{Language: "go", ToolchainVersion: "1.25", AllowedCommands: []string{"test"}, Starter: []sharedkernel.LabFile{{Path: "main.go"}}, Checkpoints: []sharedkernel.LabCheckpoint{{ID: "c1"}}, Solution: []sharedkernel.LabFile{{Path: "main.go"}}, Tests: []sharedkernel.LabFile{{Path: "main_test.go"}}}
	report = RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GateHardFail, report.Result)
}

func TestQualityGateRequiresExactCodeFenceProvenance(t *testing.T) {
	document := validChapterDocument()
	document.EvidencePack.SourceContent = map[string]string{"ev-1": "package main\nfunc main() {}"}
	document.Markdown = "# 主链路\n## 主链路\n## 数据流\n## 练习\n```source:ev-1\npackage main\nfunc main() {}\n```"
	require.Equal(t, sharedkernel.GatePass, RunChapterQualityGates(document, false).Result)
	document.Markdown = "# 主链路\n## 主链路\n## 数据流\n## 练习\n```go\npackage main\n```"
	require.Equal(t, sharedkernel.GateHardFail, RunChapterQualityGates(document, false).Result)
}

func TestQualityGateReportsSoftRisksWithoutHardBlocking(t *testing.T) {
	document := validChapterDocument()
	document.Markdown = "# 主链路\n## 主链路\n## 数据流\n## 练习\n待确认"
	report := RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GateSoftFail, report.Result)
	require.Contains(t, report.Checks, sharedkernel.GateReport{Name: "unresolved_language", Result: sharedkernel.GateSoftFail, Message: "chapter contains unresolved wording"})
}
