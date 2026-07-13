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
	require.NoError(t, document.ValidateContract())
	report := RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GatePass, report.Result)
	document.Claims[0].Status = sharedkernel.ClaimUnsupported
	report = RunChapterQualityGates(document, false)
	require.Equal(t, sharedkernel.GateHardFail, report.Result)
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
