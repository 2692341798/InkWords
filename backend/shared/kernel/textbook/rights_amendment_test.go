package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func rightsAmendmentFixture() (RightsItem, RightsAmendment) {
	base := RightsItem{ID: "original", ProjectID: "project", BuildID: "build", SubjectRef: "chapter:r1", WorkType: RightsWorkTypeProse, RightsBasis: "待核对引用范围", AllowedUse: "本地审校", Attribution: "待核对署名", PublicationStatus: RightsStatusPending}
	next := base
	next.PublicationStatus = RightsStatusReady
	next.RightsBasis = "已经核对的许可证据"
	return base, RightsAmendment{ContractVersion: RightsAmendmentFormat, ID: "amendment-1", BuildID: base.BuildID, ManifestHash: "sha256:" + strings.Repeat("a", 64), BaseItemID: base.ID, Revision: 1, ReviewerKind: "delegated_ai", Reviewer: "测试代理", DelegationNote: "用户委托核查权利证据", Reason: "补齐引用范围与许可证明", EvidenceRefs: []string{"license:snapshot-1"}, EffectiveItem: next, CompletedAt: time.Unix(10, 0).UTC()}
}

func TestRightsAmendmentsPreserveOriginalAndResolveExplicitChain(t *testing.T) {
	base, first := rightsAmendmentFixture()
	second := first
	second.ID, second.PreviousAmendmentID, second.Revision = "amendment-2", first.ID, 2
	second.EffectiveItem.PublicationStatus = RightsStatusBlocked
	second.CompletedAt = time.Unix(20, 0).UTC()
	ledger, err := ResolveRightsLedger(base.BuildID, first.ManifestHash, []RightsItem{base}, []RightsAmendment{second, first})
	require.NoError(t, err)
	require.Equal(t, RightsStatusPending, ledger.OriginalItems[0].PublicationStatus)
	require.Equal(t, RightsStatusBlocked, ledger.EffectiveItems[0].PublicationStatus)
	require.Equal(t, first.ID, ledger.Amendments[0].ID)
	require.Equal(t, base.ID, ledger.EffectiveItems[0].ID)
}

func TestRightsAmendmentsRejectBrokenOrAmbiguousEvidence(t *testing.T) {
	base, first := rightsAmendmentFixture()
	for name, change := range map[string]func(*RightsAmendment){
		"cross build":         func(a *RightsAmendment) { a.BuildID = "other" },
		"cross manifest":      func(a *RightsAmendment) { a.ManifestHash = "sha256:" + strings.Repeat("b", 64) },
		"cross project":       func(a *RightsAmendment) { a.EffectiveItem.ProjectID = "other" },
		"subject alias":       func(a *RightsAmendment) { a.EffectiveItem.SubjectRef = "chapter:r2" },
		"work type":           func(a *RightsAmendment) { a.EffectiveItem.WorkType = RightsWorkTypeCode },
		"orphan":              func(a *RightsAmendment) { a.BaseItemID = "missing" },
		"missing predecessor": func(a *RightsAmendment) { a.PreviousAmendmentID = "missing"; a.Revision = 2 },
		"missing evidence":    func(a *RightsAmendment) { a.EvidenceRefs = nil },
		"missing delegation":  func(a *RightsAmendment) { a.DelegationNote = "" },
		"unknown provenance":  func(a *RightsAmendment) { a.ReviewerKind = "certified" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := first
			change(&bad)
			_, err := ResolveRightsLedger(base.BuildID, first.ManifestHash, []RightsItem{base}, []RightsAmendment{bad})
			require.Error(t, err)
		})
	}
	fork := first
	fork.ID = "fork"
	_, err := ResolveRightsLedger(base.BuildID, first.ManifestHash, []RightsItem{base}, []RightsAmendment{first, fork})
	require.Error(t, err)
	_, err = ResolveRightsLedger(base.BuildID, first.ManifestHash, []RightsItem{base, base}, nil)
	require.Error(t, err)
}

func TestRightsAmendmentRequiresReviewOfNewEvidence(t *testing.T) {
	base, amendment := rightsAmendmentFixture()
	input := PublicationPreflightInput{BuildID: base.BuildID, ManifestHash: amendment.ManifestHash, RightsItems: []RightsItem{base}, RightsAmendments: []RightsAmendment{amendment}, RequiredRightsSubjects: []RightsSubject{{SubjectRef: base.SubjectRef, WorkType: base.WorkType}}, AutomatedChecks: []AutomatedPublicationCheck{{ID: "check", Detector: "test", Status: QualityStatusPass}}}
	for _, stage := range RequiredPublicationReviewStages() {
		input.HumanReviews = append(input.HumanReviews, HumanPublicationReview{ContractVersion: HumanPublicationReviewFormat, ReviewerKind: "human", ManifestHash: amendment.ManifestHash, Revision: 1, Verdict: "pass", Score: 3, Scope: "这是明确标识的隔离测试范围", EvidenceRefs: []string{"test:fixture"}, ID: string(stage), BuildID: base.BuildID, Stage: stage, Reviewer: "测试审阅者", Notes: "这是明确标识的测试审阅记录", CompletedAt: time.Unix(1, 0)})
	}
	result := EvaluatePublicationPreflight(input)
	require.False(t, result.Passed)
	require.Contains(t, result.Blockers, "权利证据已补证，请重新审阅当前权利清单。")
	for _, review := range input.HumanReviews {
		if review.Stage == PublicationReviewRights {
			review.ID, review.Revision, review.CompletedAt = "rights-review-v2", 2, time.Unix(20, 0)
			input.HumanReviews = append(input.HumanReviews, review)
		}
	}
	require.True(t, EvaluatePublicationPreflight(input).Passed)
	input.RightsAmendments[0].EffectiveItem.PublicationStatus = RightsStatusBlocked
	require.False(t, EvaluatePublicationPreflight(input).Passed)
}
