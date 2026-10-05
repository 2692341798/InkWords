package textbook

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestDetectSourceConflictsNeverAutoMergesPrimaryAndOfficialDisagreement(t *testing.T) {
	conflicts := DetectSourceConflicts([]ClaimCandidate{
		{Key: "route-stage", Text: "启动登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary"},
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRoleOfficial, EvidenceID: "official"},
	})

	require.Equal(t, []SourceConflict{{Key: "route-stage", EvidenceIDs: []string{"official", "primary"}}}, conflicts)
}

func TestDetectSourceConflictsIgnoresSameRoleAndNonEligibleSources(t *testing.T) {
	conflicts := DetectSourceConflicts([]ClaimCandidate{
		{Key: "route-stage", Text: "启动登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary-a"},
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary-b"},
		{Key: "route-stage", Text: "请求期间登记", SourceRole: sharedtextbook.SourceRoleUserReference, EvidenceID: "reference"},
	})

	require.Empty(t, conflicts)
}

func TestDetectSourceConflictsAcceptsMatchingPrimaryAndOfficialText(t *testing.T) {
	conflicts := DetectSourceConflicts([]ClaimCandidate{
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary"},
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRoleOfficial, EvidenceID: "official"},
	})

	require.Empty(t, conflicts)
}

func TestDetectSourceConflictsReportsAdditionalDisagreementEvenWhenOneWordingMatches(t *testing.T) {
	conflicts := DetectSourceConflicts([]ClaimCandidate{
		{Key: "route-stage", Text: "启动登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary-a"},
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary-b"},
		{Key: "route-stage", Text: "启动登记", SourceRole: sharedtextbook.SourceRoleOfficial, EvidenceID: "official"},
	})

	require.Equal(t, []SourceConflict{{Key: "route-stage", EvidenceIDs: []string{"official", "primary-a", "primary-b"}}}, conflicts)
}
